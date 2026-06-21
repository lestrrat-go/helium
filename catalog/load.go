package catalog

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	helium "github.com/lestrrat-go/helium"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// goosWindows is the runtime.GOOS value for Windows. Drive-letter handling in
// file: URIs is gated on this so POSIX behavior is never altered.
const goosWindows = "windows"

// internalLoader implements icatalog.Loader using helium's parser.
type internalLoader struct {
	errorHandler helium.ErrorHandler
	maxBytes     int
}

func (l internalLoader) Load(ctx context.Context, filename string) (*icatalog.Catalog, error) {
	return loadInternal(ctx, filename, l.errorHandler, l.maxBytes)
}

// Load parses an OASIS XML Catalog file and returns a Catalog.
// It is a convenience wrapper around NewLoader().Load().
func Load(ctx context.Context, filename string) (*Catalog, error) {
	return NewLoader().Load(ctx, filename)
}

// Load parses an OASIS XML Catalog file and returns a Catalog.
func (l Loader) Load(ctx context.Context, filename string) (*Catalog, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	cfg := l.cfg
	if cfg == nil {
		cfg = &loaderConfig{}
	}

	var eh helium.ErrorHandler
	if cfg.errorHandler != nil {
		eh = cfg.errorHandler
	} else {
		eh = helium.NilErrorHandler{}
	}

	ic, err := loadInternal(ctx, filename, eh, cfg.maxBytes)
	if err != nil {
		closeHandler(eh)
		return nil, err
	}

	closeHandler(eh)
	return &Catalog{cat: ic}, nil
}

func loadInternal(ctx context.Context, filename string, eh helium.ErrorHandler, maxBytes int) (*icatalog.Catalog, error) {
	path, isFileURI, err := catalogFilePath(filename)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to resolve path %q: %w", filename, err)
	}

	data, err := readCatalogFile(absPath, maxBytes)
	if err != nil {
		return nil, err
	}

	// Read from the local filesystem path, but resolve relative URIs in the
	// catalog against the catalog's URI. When the catalog was referenced via a
	// "file:" URI, downstream relative URIs must stay in "file:" URI space (so
	// "asset.xml" in /tmp/catalog.xml resolves to "file:///tmp/asset.xml", not
	// the bare local path "/tmp/asset.xml"). For a plain OS path input, the
	// filesystem path itself is the base, preserving the original behavior.
	baseURI := absPath
	if isFileURI {
		baseURI = localPathToFileURI(absPath)
	}

	return loadFromBytes(ctx, data, baseURI, eh, maxBytes)
}

// readCatalogFile reads a catalog file at absPath through a bounded reader so an
// unbounded or pathological source (e.g. /dev/zero) cannot exhaust memory. The
// cap is maxBytes, or [MaxCatalogSize] when maxBytes is less than or equal to
// zero. LimitReader allows one extra byte so a file exactly at the cap is
// accepted while anything larger is detected and rejected with
// [ErrCatalogTooLarge]. The extra byte is suppressed when the cap is already
// math.MaxInt64 so it cannot overflow into a zero-byte read.
func readCatalogFile(absPath string, maxBytes int) ([]byte, error) {
	limit := int64(maxBytes)
	if limit <= 0 {
		limit = MaxCatalogSize
	}

	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}
	defer f.Close()

	// Read one byte past the cap so a file that is exactly at the cap is
	// accepted while anything larger is detected, but guard against overflow:
	// limit+1 wraps to a negative value for limit==math.MaxInt on 64-bit
	// platforms, which would make io.LimitReader read zero bytes and silently
	// reject a valid catalog.
	readLimit := limit
	if readLimit < math.MaxInt64 {
		readLimit++
	}

	data, err := io.ReadAll(io.LimitReader(f, readLimit))
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("catalog: %q exceeds maximum size of %d bytes: %w", absPath, limit, ErrCatalogTooLarge)
	}

	return data, nil
}

// catalogFilePath converts a catalog reference into a local filesystem path.
// Bare paths are returned unchanged. "file:" URIs are parsed and percent-decoded
// into a local path. Any other URI scheme is unsupported and rejected.
//
// The second return value reports whether ref was a "file:" URI. Callers use it
// to decide the catalog's baseURI: a "file:" reference keeps relative downstream
// URIs in "file:" URI space, while a plain OS path resolves them as OS paths.
func catalogFilePath(ref string) (string, bool, error) {
	// A Windows path such as "C:\tmp\catalog.xml", "C:/tmp/catalog.xml", or a
	// "\\host\share" UNC path must be treated as a local OS path, not as a URI
	// whose scheme is the drive letter "C". Check this before HasScheme so the
	// leading "C:" is not mistaken for a scheme.
	//
	// filepath.VolumeName only recognizes these forms when running on Windows,
	// so a portable drive-letter check is needed as well to keep the behavior
	// consistent (and tested) on the POSIX CI host.
	if filepath.VolumeName(ref) != "" || hasDriveLetterPrefix(ref) {
		return ref, false, nil
	}

	if !icatalog.HasScheme(ref) {
		return ref, false, nil
	}

	u, err := url.Parse(ref)
	if err != nil {
		return "", false, fmt.Errorf("catalog: failed to parse URI %q: %w", ref, err)
	}

	if u.Scheme != "file" {
		return "", false, fmt.Errorf("catalog: unsupported URI scheme %q in %q", u.Scheme, ref)
	}

	// For "file:///abs/path" the host is empty and Path holds the (already
	// percent-decoded) absolute path. For "file://host/path" a non-localhost
	// host is not addressable on the local filesystem. URI hosts are
	// case-insensitive, so an empty host and "localhost" in any case both
	// denote the local machine.
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", false, fmt.Errorf("catalog: non-local file URI host %q in %q", u.Host, ref)
	}

	// An opaque file: URI such as "file:next.xml" has u.Opaque set and an empty
	// u.Path, and a "file://localhost" URI has no path at all. Neither denotes a
	// local filesystem path, so reject them rather than letting an empty path
	// fall through and read the process working directory.
	if u.Opaque != "" || u.Path == "" {
		return "", false, fmt.Errorf("catalog: invalid file URI %q: no local path", ref)
	}

	return fileURIPath(u.Path), true, nil
}

// localPathToFileURI builds a "file://" URI from an absolute local filesystem
// path. It is used as the catalog baseURI when the catalog was referenced by a
// "file:" URI, so relative URIs in the catalog resolve back into "file:" URI
// space rather than as bare filesystem paths.
//
// (&url.URL{Scheme: "file", Path: ...}).String() percent-encodes as needed and,
// on Windows, converts the OS separator to "/" via filepath.ToSlash, yielding
// "file:///C:/tmp/catalog.xml". On POSIX the absolute path already uses "/".
func localPathToFileURI(absPath string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	return u.String()
}

// fileURIPath converts the (already percent-decoded) path component of a "file:"
// URI into a local filesystem path. On POSIX an absolute URI path such as
// "/abs/x" — including "/C:/tmp/x", which is a legitimate POSIX absolute path —
// is returned unchanged. On Windows a URI such as "file:///C:/tmp/x" yields a
// path of "/C:/tmp/x"; the leading slash before the drive letter is stripped
// ("C:/tmp/x") and slashes are converted to the OS separator.
func fileURIPath(p string) string {
	return fileURIPathFor(runtime.GOOS, p)
}

// fileURIPathFor is the OS-parameterized implementation of fileURIPath. The
// drive-letter slash strip only applies on Windows; on POSIX "/C:/tmp/x" is a
// valid absolute path and must be left untouched. goos is threaded explicitly
// so the conversion is deterministically testable on a non-Windows CI host.
func fileURIPathFor(goos, p string) string {
	// On Windows, detect a drive-letter path of the form "/C:/...": a leading
	// slash followed by a single ASCII letter and a colon, and strip the slash.
	if goos == goosWindows && len(p) >= 3 && p[0] == '/' && isASCIILetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// hasDriveLetterPrefix reports whether s begins with a Windows drive-letter
// prefix such as "C:\" or "C:/". This is checked independently of the host OS
// so a drive-letter reference is never mistaken for a URI scheme.
func hasDriveLetterPrefix(s string) bool {
	return len(s) >= 3 && isASCIILetter(s[0]) && s[1] == ':' &&
		(s[2] == '\\' || s[2] == '/')
}

func loadFromBytes(ctx context.Context, data []byte, baseURI string, eh helium.ErrorHandler, maxBytes int) (*icatalog.Catalog, error) {
	p := helium.NewParser()
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to parse %q: %w", baseURI, err)
	}

	root := documentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("catalog: no root element in %q", baseURI)
	}

	if root.URI() != lexicon.NamespaceCatalog {
		return nil, fmt.Errorf("catalog: root element namespace %q is not %q in %q",
			root.URI(), lexicon.NamespaceCatalog, baseURI)
	}

	cat := &icatalog.Catalog{
		Prefer:  icatalog.PreferPublic, // default per OASIS spec
		BaseURI: baseURI,
		Loader:  internalLoader{errorHandler: eh, maxBytes: maxBytes},
	}

	if v := getAttr(root, lexicon.AttrPrefer); v != "" {
		cat.Prefer = icatalog.ParsePrefer(v)
	}

	parseEntries(ctx, root, cat.Prefer, baseURI, &cat.Entries, eh)

	return cat, nil
}

// parseEntries walks child elements of parent and appends catalog entries.
func parseEntries(ctx context.Context, parent *helium.Element, prefer icatalog.Prefer, baseURI string, entries *[]icatalog.Entry, eh helium.ErrorHandler) {
	if v := getAttrNS(parent, lexicon.AttrBase, lexicon.NamespaceXML); v != "" {
		if resolved, err := icatalog.ResolveURI(baseURI, v); err != nil {
			eh.Handle(ctx, fmt.Errorf("catalog: %s attribute: %w", lexicon.AttrBase, err))
		} else {
			baseURI = resolved
		}
	}

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}

		if elem.URI() != lexicon.NamespaceCatalog {
			continue
		}

		localName := elem.LocalName()

		elemBase := baseURI
		if v := getAttrNS(elem, lexicon.AttrBase, lexicon.NamespaceXML); v != "" {
			if resolved, err := icatalog.ResolveURI(baseURI, v); err != nil {
				eh.Handle(ctx, fmt.Errorf("catalog: %s attribute: %w", lexicon.AttrBase, err))
			} else {
				elemBase = resolved
			}
		}

		elemPrefer := prefer
		if v := getAttr(elem, lexicon.AttrPrefer); v != "" {
			elemPrefer = icatalog.ParsePrefer(v)
		}

		switch localName {
		case lexicon.ElemPublic:
			pubID := icatalog.NormalizePublicID(getAttr(elem, lexicon.AttrPublicID))
			uri, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrURI))
			if !ok {
				continue
			}
			if pubID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, pubID, lexicon.AttrPublicID, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryPublic,
					Name:   pubID,
					URL:    uri,
					Prefer: elemPrefer,
				})
			}
		case lexicon.ElemSystem:
			sysID := getAttr(elem, lexicon.AttrSystemID)
			uri, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrURI))
			if !ok {
				continue
			}
			if sysID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, sysID, lexicon.AttrSystemID, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntrySystem,
					Name: sysID,
					URL:  uri,
				})
			}
		case lexicon.ElemRewriteSystem:
			startString := getAttr(elem, lexicon.AttrSystemIDStartString)
			prefix, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrRewritePrefix))
			if !ok {
				continue
			}
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrSystemIDStartString, prefix, lexicon.AttrRewritePrefix)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteSystem,
					Name: startString,
					URL:  prefix,
				})
			}
		case lexicon.ElemDelegatePublic:
			startString := icatalog.NormalizePublicID(getAttr(elem, lexicon.AttrPublicIDStartString))
			catFile, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrCatalog))
			if !ok {
				continue
			}
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrPublicIDStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryDelegatePublic,
					Name:   startString,
					URL:    catFile,
					Prefer: elemPrefer,
				})
			}
		case lexicon.ElemDelegateSystem:
			startString := getAttr(elem, lexicon.AttrSystemIDStartString)
			catFile, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrCatalog))
			if !ok {
				continue
			}
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrSystemIDStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateSystem,
					Name: startString,
					URL:  catFile,
				})
			}
		case lexicon.ElemURI:
			name := getAttr(elem, lexicon.AttrName)
			uri, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrURI))
			if !ok {
				continue
			}
			if name == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, name, lexicon.AttrName, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryURI,
					Name: name,
					URL:  uri,
				})
			}
		case lexicon.ElemRewriteURI:
			startString := getAttr(elem, lexicon.AttrURIStartString)
			prefix, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrRewritePrefix))
			if !ok {
				continue
			}
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrURIStartString, prefix, lexicon.AttrRewritePrefix)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteURI,
					Name: startString,
					URL:  prefix,
				})
			}
		case lexicon.ElemDelegateURI:
			startString := getAttr(elem, lexicon.AttrURIStartString)
			catFile, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrCatalog))
			if !ok {
				continue
			}
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrURIStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateURI,
					Name: startString,
					URL:  catFile,
				})
			}
		case lexicon.ElemNextCatalog:
			catFile, ok := resolveEntryURI(ctx, eh, elemBase, getAttr(elem, lexicon.AttrCatalog))
			if !ok {
				continue
			}
			if catFile == "" {
				eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", localName, lexicon.AttrCatalog))
			} else {
				if !icatalog.HasNextCatalog(*entries, catFile) {
					*entries = append(*entries, icatalog.Entry{
						Type: icatalog.EntryNextCatalog,
						URL:  catFile,
					})
				}
			}
		case lexicon.ElemGroup:
			parseEntries(ctx, elem, elemPrefer, elemBase, entries, eh)
		}
	}
}

// documentElement returns the first child element of a Document.
func documentElement(doc *helium.Document) *helium.Element {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if elem, ok := helium.AsNode[*helium.Element](child); ok {
			return elem
		}
	}
	return nil
}

// resolveEntryURI resolves a catalog entry's URI/prefix/catalog attribute
// against base. When the value is a syntactically malformed URI that url.Parse
// rejects, the error is reported through eh and ok is false so the caller skips
// the entry rather than storing the raw, unresolved value as a usable mapping.
func resolveEntryURI(ctx context.Context, eh helium.ErrorHandler, base, value string) (string, bool) {
	resolved, err := icatalog.ResolveURI(base, value)
	if err != nil {
		eh.Handle(ctx, fmt.Errorf("catalog: %w", err))
		return "", false
	}
	return resolved, true
}

// catalogMissingAttr reports which required attributes are missing on a catalog entry.
func catalogMissingAttr(ctx context.Context, eh helium.ErrorHandler, elemName, val1, attr1, val2, attr2 string) {
	if val1 == "" {
		eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", elemName, attr1))
	}
	if val2 == "" {
		eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", elemName, attr2))
	}
}

// closeHandler closes the error handler if it implements io.Closer.
func closeHandler(eh helium.ErrorHandler) {
	if c, ok := eh.(io.Closer); ok {
		_ = c.Close()
	}
}

// getAttr returns the value of the attribute with the given local name.
func getAttr(elem *helium.Element, name string) string {
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return ""
	}
	return attr.Value()
}

// getAttrNS returns the value of the attribute with the given local name
// and namespace URI.
func getAttrNS(elem *helium.Element, name, nsURI string) string {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: nsURI})
	if !ok {
		return ""
	}
	return attr.Value()
}
