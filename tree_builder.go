package helium

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"math"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/sax"
)

// MaxExternalDTDSize is the maximum number of bytes read from an external
// DTD subset. Loading is gated by LoadExternalDTD/ValidateDTD/
// DefaultDTDAttributes, so an unbounded read of a hostile or pathological
// source (e.g. /dev/zero) could exhaust memory before any entity or parse
// limits apply. The DTD is read through a strict byte cap and rejected when
// it is exceeded.
const MaxExternalDTDSize = 10 << 20 // 10 MiB

// fileParseInput wraps an os.File as a sax.ParseInput.
type fileParseInput struct {
	io.ReadCloser
	uri string
}

func (f *fileParseInput) URI() string { return f.uri }

// TreeBuilder is a SAX2 handler that builds a DOM tree from SAX events,
// analogous to libxml2's default SAX handler (xmlSAX2InitDefaultSAXHandler).
type TreeBuilder struct{}

func (t *TreeBuilder) pctx(ctxif context.Context) *parserCtx {
	return getParserCtx(ctxif)
}

// NewTreeBuilder creates a new TreeBuilder that builds a DOM tree from SAX events.
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{}
}

func (t *TreeBuilder) SetDocumentLocator(ctxif context.Context, loc sax.DocumentLocator) error {
	return nil
}

func (t *TreeBuilder) StartDocument(ctxif context.Context) error {
	ctx := t.pctx(ctxif)
	ctx.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	ctx.doc.idsSkip = ctx.loadsubset.IsSet(SkipIDs)
	ctx.doc.url = ctx.baseURI
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif context.Context) error {
	ctx := t.pctx(ctxif)
	if ctx.doc != nil && ctx.wellFormed {
		ctx.doc.properties |= DocWellFormed
		if ctx.valid {
			ctx.doc.properties |= DocDTDValid
		}
	}
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif context.Context, target, data string) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	pi := doc.CreatePI(target, data)

	// Track external entity base URI for base-uri() resolution.
	if ctx.currentEntityURI != "" {
		pi.entityBaseURI = ctx.currentEntityURI
	}

	switch ctx.inSubset {
	case 1:
		if err := doc.IntSubset().AddChild(pi); err != nil {
			return err
		}
	case 2:
		if err := doc.ExtSubset().AddChild(pi); err != nil {
			return err
		}
	}

	parent := ctx.elem
	if parent == nil {
		if err := doc.AddChild(pi); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(pi); err != nil {
			return err
		}
	} else {
		if err := parent.AddSibling(pi); err != nil {
			return err
		}
	}
	return nil
}

func (t *TreeBuilder) StartElementNS(ctxif context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	//	ctx := t.pctx(ctxif)
	ctx := t.pctx(ctxif)
	doc := ctx.doc

	e, err := doc.CreateElement(localname)
	if err != nil {
		return err
	}

	e.SetLine(ctx.LineNumber())

	// When this element is being created as part of external entity
	// expansion, record the entity's URI so base-uri() returns the
	// correct value without needing a synthetic xml:base attribute.
	if ctx.currentEntityURI != "" {
		e.entityBaseURI = ctx.currentEntityURI
	}

	if uri != "" {
		if err := e.SetActiveNamespace(prefix, uri); err != nil {
			return err
		}
	}

	for _, ns := range namespaces {
		if err := e.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
			return err
		}
	}

	for _, attr := range attrs {
		if attr.IsDefault() && !ctx.loadsubset.IsSet(CompleteAttrs) {
			continue
		}
		if p := attr.Prefix(); p != "" {
			// Prefixed attribute — look up the namespace from the
			// element itself or from the parent context (the element
			// hasn't been linked to the tree yet).
			ns := lookupNSByPrefix(e, p)
			if ns == nil && ctx.elem != nil {
				ns = lookupNSByPrefix(ctx.elem, p)
			}
			if ctx.replaceEntities {
				// When replaceEntities is true (ParseNoEnt), entity
				// references are already resolved in the attribute
				// value. Use the literal setter to avoid re-parsing &
				// as new entity reference starts.
				_ = e.SetAttributeNS(attr.LocalName(), attr.Value(), ns)
				carrySyntheticBase(e, attr, ns)
			} else {
				// The attribute value still carries raw entity syntax that
				// must be parsed into the attribute's child list.
				if err := e.SetParsedAttributeNS(attr.LocalName(), attr.Value(), ns); err != nil {
					return err
				}
			}
		} else {
			if ctx.replaceEntities {
				_ = e.SetAttribute(attr.Name(), attr.Value())
			} else {
				// The attribute value still carries raw entity syntax that
				// must be parsed into the attribute's child list.
				if err := e.SetParsedAttribute(attr.Name(), attr.Value()); err != nil {
					return err
				}
			}
		}
	}

	// Propagate attribute types from DTD declarations and register IDs.
	elemName := localname
	if prefix != "" {
		elemName = prefix + ":" + localname
	}
	registerIDs := !ctx.loadsubset.IsSet(SkipIDs)
	e.ForEachAttribute(func(a *Attribute) bool {
		aLocalName := a.LocalName()
		aPrefix := a.Prefix()
		if decl := lookupAttributeDecl(doc, aLocalName, aPrefix, elemName); decl != nil {
			a.SetAType(decl.AType())
		}
		if registerIDs {
			if a.Name() == lexicon.QNameXMLID || a.AType() == enum.AttrID {
				doc.RegisterID(a.Value(), e)
			}
		}
		return true
	})

	var parent MutableNode
	if e := ctx.elem; e != nil {
		parent = e
	}
	if parent == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(e); err != nil {
			return err
		}
	} else {
		if err := parent.AddSibling(e); err != nil {
			return err
		}
	}

	ctx.elem = e

	return nil
}

func (t *TreeBuilder) EndElementNS(ctxif context.Context, localname, prefix, uri string) error {
	ctx := t.pctx(ctxif)
	cur := ctx.elem
	if cur == nil {
		return errors.New("no context node to end")
	}

	p := cur.Parent()
	if e, ok := p.(*Element); ok {
		ctx.elem = e
	} else {
		ctx.elem = nil
	}
	return nil
}

func (t *TreeBuilder) Characters(ctxif context.Context, data []byte) error {
	ctx := t.pctx(ctxif)
	n := ctx.elem
	if n == nil {
		return errors.New("text content placed in wrong location")
	}

	if err := n.AppendText(data); err != nil {
		return err
	}
	// A character-reference delivery marks the Text node it lands in (whether
	// freshly created or merged into the last Text child) so element-content
	// validity can treat its whitespace as non-ignorable. Sticky: once set, a
	// later literal append into the same node does not clear it.
	if ctx.charDataFromCharRef {
		if last, ok := AsNode[*Text](n.LastChild()); ok {
			last.fromCharRef = true
		}
	}
	return nil
}

// CDataBlock mirrors xmlSAX2Text(ctxt, value, len, XML_CDATA_SECTION_NODE)
// in libxml2's SAX2.c. Unlike text nodes, adjacent CDATA sections are NOT
// merged — each callback creates a new CDATASection node.
func (t *TreeBuilder) CDataBlock(ctxif context.Context, data []byte) error {
	ctx := t.pctx(ctxif)
	parent := ctx.elem
	if parent == nil {
		return nil
	}

	doc := ctx.doc
	cdata := doc.CreateCDATASection(data)

	return parent.AddChild(cdata)
}

// Comment mirrors xmlSAX2Comment in libxml2's SAX2.c, which delegates
// parent selection to xmlSAX2AppendChild. When inside a DTD subset the
// comment is added to the DTD, not the document.
func (t *TreeBuilder) Comment(ctxif context.Context, data []byte) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	if doc == nil {
		return errors.New("comment placed in wrong location")
	}

	e := doc.CreateComment(data)

	// Mirror xmlSAX2AppendChild parent selection (SAX2.c:899-907).
	switch ctx.inSubset {
	case inInternalSubset:
		return doc.IntSubset().AddChild(e)
	case inExternalSubset:
		return doc.ExtSubset().AddChild(e)
	}

	n := ctx.elem
	if n == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
	} else if n.Type() == ElementNode {
		if err := n.AddChild(e); err != nil {
			return err
		}
	} else {
		if err := n.AddSibling(e); err != nil {
			return err
		}
	}
	return nil
}

func (t *TreeBuilder) InternalSubset(ctxif context.Context, name, eid, uri string) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc

	dtd, err := doc.InternalSubset()
	if err == nil {
		// HTML mode would skip freeing the DTD here.
		dtd.Free()
		doc.intSubset = nil // hmm, do we need this?
	}

	_, err = doc.CreateInternalSubset(name, eid, uri)
	if err != nil {
		return err
	}

	return nil
}

// catalogOpenName converts a catalog-resolved system id / URI into the name
// handed to [fs.FS.Open]. A catalog may map an identifier to a "file:" URI
// (e.g. "file:///tmp/x.dtd"); that URI is not a filesystem path and must be
// converted before opening, mirroring how the XInclude processor handles
// "file:" hrefs (see internal/iofs.FileURIToPath). Non-file URIs and plain
// paths are returned unchanged so existing handling is preserved. A "file:"
// URI that cannot be converted to a local path (opaque or non-local host) is
// also returned unchanged, letting the subsequent Open fail as before.
func catalogOpenName(ref string) string {
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != lexicon.SchemeFile {
		return ref
	}
	p, err := iofs.FileURIToPath(ref)
	if err != nil {
		return ref
	}
	return p
}

// networkAccessForbidden reports whether opening name must be refused because it
// names a network resource and the parser was configured to forbid network
// access (Parser.AllowNetwork(false) → parseNoNet / libxml2 XML_PARSE_NONET,
// the default). It is the enforcement point for NONET: helium has no dedicated
// network loader, so every external-resource load goes through fs.FS.Open, and
// this guard refuses a network-scheme name before it reaches the (possibly
// network-capable) caller-supplied fs.FS. A name with no scheme, a "file:"
// scheme, or a bare path is never a network resource and loads as usual; scheme
// matching is case-insensitive (uripath.URIScheme lowercases its result).
//
// Leading/trailing XML whitespace (space, tab, CR, LF) is stripped before the
// scheme check: a SYSTEM literal is a URI reference (XML §4.2.2), and RFC 3986
// requires surrounding whitespace to be ignored when a URI is extracted. Without
// this a whitespace-prefixed id (" http://x") reads as scheme-less to
// uripath.URIScheme (whose first byte must be an ALPHA) and would slip past the
// NONET gate to a network-capable caller fs.FS.
func networkAccessForbidden(ctx *parserCtx, name string) bool {
	if !ctx.options.IsSet(parseNoNet) {
		return false
	}
	switch uripath.URIScheme(strings.Trim(name, xmlWhitespace)) {
	case "http", "https", "ftp":
		return true
	default:
		return false
	}
}

// xmlWhitespace is the XML whitespace set (S production: #x20 #x9 #xD #xA), used
// as a cutset to strip whitespace surrounding a SYSTEM-literal URI reference.
const xmlWhitespace = " \t\r\n"

// systemIDRetryEligible reports whether a declared SYSTEM id is eligible for the
// confined-FS base-relative retry (openExternalResource). Only an ORIGINALLY
// relative reference is eligible: the id must be neither an absolute path (POSIX
// or Windows, GOOS-independent), nor carry a URI scheme (e.g. "file:", "http:"),
// nor carry a colon anywhere in its FIRST path segment (the part before the
// first '/' or '\\'). It is evaluated on the id AS DECLARED — before URI
// resolution against the base and before any catalog mapping — because the
// confined-FS retry only ever wants to recover the relative name a caller who
// rooted the FS at the document directory expects; an absolute or file-URI
// SYSTEM id is never retried.
//
// The first-segment colon check catches every scheme-looking id that
// [uripath.HasURIScheme] deliberately does not: a one-letter URI scheme
// ("x:opaque" — valid per RFC 3986, scheme = ALPHA *( ALPHA / DIGIT / "+" /
// "-" / "." )) reads as a Windows drive letter to that helper (whose 2-char
// minimum disambiguates "C:\\path"), and a bare drive letter itself. A genuine
// relative path never carries a colon in its first segment — RFC 3986 makes
// such a reference scheme-ambiguous — so excluding it cannot misclassify a real
// relative id.
func systemIDRetryEligible(declaredSystemID string) bool {
	if declaredSystemID == "" {
		return false
	}
	if uripath.IsAbsolutePath(declaredSystemID) || uripath.HasURIScheme(declaredSystemID) {
		return false
	}
	firstSeg := declaredSystemID
	if i := strings.IndexAny(firstSeg, `/\`); i >= 0 {
		firstSeg = firstSeg[:i]
	}
	return !strings.Contains(firstSeg, ":")
}

// baseRelativeFSName derives a valid [fs.ValidPath] name for primary by making
// it relative to the directory of base. primary is the resolved resource in
// local-path form (post [catalogOpenName]); base is the fixed top-level document
// base URI (parserCtx.documentBaseURI), NOT the moving per-resource baseURI — so
// a nested resource in a subdirectory relativizes against the document root and
// yields the correct root-relative name (a nested SYSTEM id resolved to
// "/dir/sub/x.dtd" against document base "/dir/doc.xml" becomes "sub/x.dtd").
//
// The parser resolves a relative SYSTEM id against the base URI into an absolute
// path (see ExternalSubset / EntityDecl), which an fs.FS that enforces
// [fs.ValidPath] — [os.DirFS], [os.Root.FS] (os.OpenRoot), [testing/fstest.MapFS]
// — rejects. Re-relativizing against the document base recovers the name a caller
// who rooted such an FS at the document's own directory expects (a SYSTEM id
// "sub.dtd" resolved against "/dir/doc.xml" becomes "sub.dtd" again).
//
// Eligibility (the reference was ORIGINALLY relative) is enforced by the caller
// via [systemIDRetryEligible] BEFORE this runs — filepath.Rel here would happily
// re-relativize an in-root absolute id ("/dir/sub.dtd" against "/dir/doc.xml"
// becomes "sub.dtd"), so the path-shape guard below is NOT what keeps an absolute
// SYSTEM id from being retried; the caller's eligibility gate is.
//
// It reports ok=false — no usable fallback — when base is empty, when either name
// still carries a URI scheme, when the two are not both path-shaped on the same
// volume, or when the result would ascend above the FS root: a leading "/" or a
// surviving ".." after cleaning makes it an invalid fs.ValidPath. That result
// guard blocks a "../"- or absolute-path escape above the FS root; it does NOT
// stop a symlink inside the root from pointing outside (only [os.Root.FS]
// confines symlinks — see openExternalResource).
func baseRelativeFSName(primary, base string) (string, bool) {
	if base == "" || uripath.HasURIScheme(primary) {
		return "", false
	}
	basePath := catalogOpenName(base)
	if uripath.HasURIScheme(basePath) {
		return "", false
	}
	rel, err := filepath.Rel(filepath.Dir(basePath), primary)
	if err != nil {
		return "", false
	}
	rel = path.Clean(filepath.ToSlash(rel))
	if !fs.ValidPath(rel) {
		return "", false
	}
	return rel, true
}

// openExternalResource opens primary through the parser's fs.FS. primary is the
// resolved resource name: for the direct entity/DTD paths it is the raw
// (historical) name — a "file:" URI or an absolute path — tried verbatim first,
// so [iofs.PermissiveRoot] (which wants the absolute path for os.Open) and a
// caller FS keyed on the file-URI name are unchanged.
//
// This is the single enforcement point for the NONET network-access guard: it
// runs [networkAccessForbidden] on primary AND on the derived retry name, so a
// base-relative retry that turns into a network-scheme name is refused just like
// a network-scheme primary. A forbidden name returns [ErrNetworkAccessForbidden];
// callers must treat that error distinctly from an ordinary open failure.
//
// When the primary open fails specifically because the FS rejected the name as an
// invalid io/fs path ([fs.ErrInvalid] — reported by [os.DirFS], [os.Root.FS] and
// [io/fs.Sub], but never by PermissiveRoot or DenyAll, which use os.Open /
// return fs.ErrNotExist), it retries with the name made relative to the fixed
// top-level document base's directory (parserCtx.documentBaseURI) via
// [baseRelativeFSName]. This lets a confined fs.FS rooted at the document's
// directory resolve a SYSTEM id that resolution turned into an absolute path.
//
// The retry fires ONLY when retryEligible is true — i.e. the reference's declared
// SYSTEM id was ORIGINALLY relative (see [systemIDRetryEligible]). An originally
// absolute or file-URI SYSTEM id is never retried, even when it names an in-root
// file whose base-relative form would be a valid fs.ValidPath: filepath.Rel would
// re-relativize such an id, so eligibility — not the fs.ValidPath shape of the
// retry name — is what enforces that promise.
//
// The supported confined-FS document base is an ABSOLUTE path or "file:" URI (as
// ParseFile always sets): re-relativizing needs an absolute base to recover the
// original relative name. A RELATIVE document base is out of scope — BuildURI
// yields a valid-but-absent relative path that fails with fs.ErrNotExist (not
// fs.ErrInvalid), so the retry never fires; serving an fs.FS rooted elsewhere than
// the document directory is the job of the root-aware [DirFS] adapter, which
// serves an in-root absolute name directly, not this retry.
//
// The retry name is a validated fs.ValidPath, so a "../"- or absolute-path escape
// above the FS root is blocked. That path-shape guard does NOT confine symlinks:
// [os.DirFS] follows an in-root symlink that points outside its root, so os.DirFS
// is path-escape-safe but not a symlink sandbox. For symlink-safe confinement use
// [os.Root.FS] (os.OpenRoot, Go 1.24+), which refuses any open that escapes the
// root through a symlink.
func (ctx *parserCtx) openExternalResource(primary string, retryEligible bool) (fs.File, error) {
	if networkAccessForbidden(ctx, primary) {
		return nil, ErrNetworkAccessForbidden
	}
	f, err := ctx.fsys.Open(primary)
	if err == nil {
		return f, nil
	}
	if !retryEligible || !errors.Is(err, fs.ErrInvalid) {
		return nil, err //nolint:wrapcheck // resolver errors propagate to caller verbatim
	}
	rel, ok := baseRelativeFSName(catalogOpenName(primary), ctx.documentBaseURI)
	if !ok || rel == primary {
		return nil, err //nolint:wrapcheck // resolver errors propagate to caller verbatim
	}
	if networkAccessForbidden(ctx, rel) {
		return nil, ErrNetworkAccessForbidden
	}
	f2, err2 := ctx.fsys.Open(rel)
	if err2 == nil {
		return f2, nil
	}
	return nil, err //nolint:wrapcheck // report the primary (resolved-name) error
}

func (t *TreeBuilder) ExternalSubset(ctxif context.Context, name, eid, uri string) error {
	ctx := t.pctx(ctxif)

	if ctx.options.IsSet(parseNoXXE) {
		return nil
	}

	// Resolve the load decision once from three independent intents, matching
	// libxml2 (parser.c xmlSAX2ExternalSubset / SAX2.c): the external subset is
	// loaded iff DTD validation (parseDTDValid), external-DTD loading (DetectIDs,
	// from LoadExternalDTD), or default-attribute application (CompleteAttrs,
	// from DefaultDTDAttributes) was requested. Reading all three here — rather
	// than a single bit that the setters coupled together — makes the decision
	// independent of the order the setters were called.
	if !ctx.options.IsSet(parseDTDValid) &&
		!ctx.loadsubset.IsSet(DetectIDs) &&
		!ctx.loadsubset.IsSet(CompleteAttrs) {
		return nil
	}

	// Capture retry eligibility from the SYSTEM id AS DECLARED — before catalog
	// mapping and before BuildURI resolution below. Only an originally relative
	// SYSTEM id is eligible for the confined-FS base-relative retry; an absolute
	// or file-URI id is never retried (see openExternalResource).
	retryEligible := systemIDRetryEligible(uri)

	// Try catalog resolution first. A catalog may resolve the identifier to a
	// "file:" URI, which is not a filesystem path; convert it before it reaches
	// the base-URI joining and fsys.Open below (CAT-001).
	if ctx.catalog != nil {
		if catalogURI := ctx.catalog.Resolve(ctxif, eid, uri); catalogURI != "" {
			uri = catalogOpenName(catalogURI)
		}
	}

	if uri == "" {
		return nil
	}

	// Resolve the system URI against the document's base URI. Use BuildURI (the
	// same GOOS-independent, forward-slash/file:-URI-aware resolver used for
	// entity URIs) rather than filepath.Dir/Join: on Windows filepath.Join
	// mangles a "file:///C:/dir/doc.xml" base (it cleans "file://" to "file:/"
	// and emits '\' separators), so a nested external DTD declared with a
	// relative SYSTEM id ("inc.dtd") never resolves and is silently dropped.
	// BuildURI keeps a drive-rooted result wrapped as "file:///C:/dir/inc.dtd",
	// which the fsys (normalizingFS) converts back to a native path. POSIX
	// output is unchanged: "file:///tmp/doc.xml" + "inc.dtd" -> "file:///tmp/inc.dtd".
	resolved := uri
	if !uripath.IsAbsolutePath(uri) && !uripath.HasURIScheme(uri) && ctx.baseURI != "" {
		if built := BuildURI(uri, ctx.baseURI); built != "" {
			resolved = built
		}
	}

	// resolved may be a "file:" URI (e.g. "file:///C:/dir/inc.dtd" from a
	// drive-rooted base); convert it to a native path before Open, the same way
	// a catalog-resolved file: URI is handled. A plain path is returned verbatim.
	openName := catalogOpenName(resolved)
	f, err := ctx.openExternalResource(openName, retryEligible)
	if errors.Is(err, ErrNetworkAccessForbidden) {
		// A network-scheme name (primary or the base-relative retry) is refused
		// hard while network access is forbidden, not downgraded to a warning.
		return ErrNetworkAccessForbidden
	}
	if err != nil {
		// Loading was requested (the resolve-once gate above passed), so a failed
		// open is a requested-but-failed load, not an absent DTD. Surface it as a
		// non-fatal warning rather than swallowing it silently — a caller that
		// asked to load/validate against the external subset gets a signal, while
		// the parse stays lenient (matching libxml2, which warns but continues).
		// Under DTD validation the missing content model then surfaces downstream
		// as a validation failure. The warning is gated by parseNoWarning.
		_ = ctx.warning(ctxif, "failed to load external DTD subset %q: %s", resolved, err)
		return nil
	}

	// fs.FileInfo.Size() is only reliable for regular files: a valid fs.FS
	// may stream or synthesize DTD content and report a non-regular,
	// unknown, under-reported, or over-reported size. Stat is therefore
	// never used to reject — the authoritative cap is the actual number of
	// bytes read below. Read through a strict byte cap, allowing one extra
	// byte so a source that under-reports (or lies about) its size is still
	// caught.
	// ctx.maxExtDTDSize is already resolved: MaxExternalDTDSize for the default,
	// a configured positive cap verbatim, or 0 (the resolveLimit sentinel for
	// "no limit", from Parser.MaxExternalDTDBytes with a negative argument). A
	// zero here means the caller explicitly disabled the cap for trusted input,
	// so read without a byte bound.
	limit := int64(ctx.maxExtDTDSize)
	if limit <= 0 {
		limit = math.MaxInt64
	}
	data, exceeded, readErr := iolimit.ReadAll(f, limit)
	// Close the file immediately once the bounded read completes, before the
	// already-buffered DTD is parsed, so the descriptor is not held open for
	// the lifetime of the parse.
	f.Close()

	// Enforce the cap authoritatively against the bytes actually read, before
	// inspecting the read error: a reader that returns n>0 alongside a
	// non-EOF error on the cap-crossing read must still be rejected.
	if exceeded {
		return ErrExternalDTDTooLarge
	}
	// A non-EOF read error (e.g. io.ErrUnexpectedEOF or a transport failure)
	// means the DTD was only partially read. Treating that as an absent DTD
	// would silently accept a truncated subset, so surface it. io.EOF is the
	// normal terminator for a fully consumed stream and is not an error.
	if readErr != nil && readErr != io.EOF {
		return readErr
	}

	// An external subset may begin with a TextDecl
	// ('<?xml' VersionInfo? EncodingDecl S? '?>'). Consume it (and honor any
	// declared encoding) before the declaration loop, which would otherwise
	// reject the '<?xml' as a processing instruction whose target may not be
	// "xml". This is the same treatment external parameter/general entities get.
	data, err = ctx.decodeExternalPEContent(ctxif, resolved, data)
	if err != nil {
		return err
	}

	doc := ctx.doc

	// Create the external subset DTD
	dtd := newDTD()
	dtd.name = name
	dtd.externalID = eid
	dtd.systemID = uri
	dtd.doc = doc
	doc.extSubset = dtd

	// Parse markup declarations from the DTD content.
	// Push content onto the input stack and loop until exhausted.
	savedExternal := ctx.external
	savedBaseURI := ctx.baseURI
	savedDTDInputFloor := ctx.dtdInputFloor
	ctx.external = true
	ctx.baseURI = resolved

	baseLen := ctx.inputTab.Len()
	ctx.pushInput(strcursor.NewByteCursor(bytes.NewReader(data)))
	// The DTD cursor we just pushed is the enclosing content cursor for the
	// shared declaration step: it lives one level above baseLen.
	dtdFloor := ctx.inputTab.Len()
	// skipBlanksPE expands parameter-entity references inside/adjacent to markup
	// declarations by pushing their padded replacement text ABOVE this cursor and
	// crossing back when the PE input is spent; it must never pop below this base
	// (into the main document input), so record its depth as the floor.
	ctx.dtdInputFloor = dtdFloor

	// Restore parser state on every exit path, including the error returns
	// below, and ensure our pushed input is always removed from the stack.
	defer func() {
		for ctx.inputTab.Len() > baseLen {
			ctx.popInput()
		}
		ctx.external = savedExternal
		ctx.baseURI = savedBaseURI
		ctx.dtdInputFloor = savedDTDInputFloor
	}()

	// Parse the external subset declaration-by-declaration through the SHARED
	// step used for INCLUDE-section bodies (parseExternalSubsetDeclStep), so a
	// parameter-entity reference expands identically in both contexts: a
	// blank-only skip (NOT skipBlanks, whose handlePEReference would consume a
	// "%pe;" reference without expanding it), explicit parsePEReference
	// expansion, spent-cursor cleanup, and a forward-progress guard that surfaces
	// a malformed "<!BOGUS" while the external DTD cursor/baseURI are still
	// active (so its location, not the main doctype's, is reported). A malformed
	// or unterminated conditional section propagates as a fatal error.
	for ctx.inputTab.Len() > baseLen {
		top := ctx.adaptCursor(ctx.inputTab.PeekOne())
		if top == nil || top.Done() {
			break
		}

		stop, err := ctx.parseExternalSubsetDeclStep(ctxif, dtdFloor)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

func (t *TreeBuilder) HasInternalSubset(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) HasExternalSubset(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) IsStandalone(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) GetEntity(ctxif context.Context, name string) (ent sax.Entity, err error) {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	x, ok := doc.GetEntity(name)
	if !ok {
		err = errors.New("entity not found")
	} else {
		ent = x
	}
	return
}

func (t *TreeBuilder) GetParameterEntity(ctxif context.Context, name string) (sax.Entity, error) {
	if ctxif == nil {
		return nil, ErrInvalidParserCtx
	}

	ctx := t.pctx(ctxif)
	doc := ctx.doc
	if doc == nil {
		return nil, ErrInvalidDocument
	}

	if ret, ok := doc.GetParameterEntity(name); ok {
		return ret, nil
	}

	return nil, ErrEntityNotFound
}

// carrySyntheticBase re-marks a replayed xml:base attribute as parser-synthesized.
// When an external-entity subtree is replayed under replaceEntities, the tree
// builder rebuilds each attribute fresh, so the syntheticBase marker on the cached
// source attribute (set by parseExternalEntityPrivate) must be copied onto the
// newly built one — otherwise DTD validation would flag the synthetic xml:base as
// an undeclared attribute. An authored xml:base is never marked, so it is not
// carried and remains subject to the "attribute must be declared" VC.
func carrySyntheticBase(e *Element, src sax.Attribute, ns *Namespace) {
	srcAttr, ok := src.(*Attribute)
	if !ok || !srcAttr.syntheticBase {
		return
	}
	var uri string
	if ns != nil {
		uri = ns.URI()
	}
	if a := e.GetAttributeNodeNS(srcAttr.LocalName(), uri); a != nil {
		a.syntheticBase = true
	}
}

func (t *TreeBuilder) AttributeDecl(ctxif context.Context, eName string, aName string, typ enum.AttributeType, deftype enum.AttributeDefault, value string, enumif sax.Enumeration) error {
	ctx := t.pctx(ctxif)

	if aName == lexicon.QNameXMLID && typ != enum.AttrID {
		// libxml2 says "raise the error but keep the validity flag"
		// but I don't know if we can do that..
		return errors.New("xml:id: attribute type should be enum.AttrID")
	}
	var prefix string
	var local string
	if p, l, ok := strings.Cut(aName, ":"); ok {
		prefix = p
		local = l
	} else {
		local = aName
	}

	enum := enumif.(Enumeration) //nolint:forcetypeassert

	doc := ctx.doc
	switch ctx.inSubset {
	case 1:
		if _, err := ctx.addAttributeDecl(doc.intSubset, eName, local, prefix, typ, deftype, value, enum); err != nil {
			return err
		}
	case 2:
		if _, err := ctx.addAttributeDecl(doc.extSubset, eName, local, prefix, typ, deftype, value, enum); err != nil {
			return err
		}
	default:
		return errors.New("TreeBuilder.AttributeDecl called while not in subset")
	}
	// NOTE: Attribute declaration validation (xmlValidateAttributeDecl in
	// libxml2) is now handled post-parse via validateDocument() when
	// ParseDTDValid is set.
	return nil
}

func (t *TreeBuilder) ElementDecl(ctxif context.Context, name string, typ enum.ElementType, content sax.ElementContent) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.ElementDecl called while not in subset")
	}

	_, err := dtd.AddElementDecl(name, typ, content.(*ElementContent)) //nolint:forcetypeassert
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeBuilder) IgnorableWhitespace(ctxif context.Context, content []byte) error {
	ctx := t.pctx(ctxif)
	if ctx.keepBlanks {
		return t.Characters(ctxif, content)
	}

	return nil
}

func (t *TreeBuilder) NotationDecl(ctxif context.Context, name string, publicID string, systemID string) error {
	ctx := t.pctx(ctxif)
	dtd := ctx.doc.intSubset
	if dtd == nil {
		return nil
	}
	_, err := dtd.AddNotation(name, publicID, systemID)
	return err
}

func (t *TreeBuilder) Reference(ctxif context.Context, name string) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	var n Node
	var err error
	if name[0] == '#' {
		if n, err = doc.CreateCharRef(name); err != nil {
			return err
		}
	} else {
		if n, err = doc.CreateReference(name); err != nil {
			return err
		}
	}

	parent := ctx.elem
	return parent.AddChild(n)
}

func (t *TreeBuilder) ResolveEntity(ctxif context.Context, publicID string, systemID string) (sax.ParseInput, error) {
	ctx := t.pctx(ctxif)
	if ctx.catalog != nil {
		if resolved := ctx.catalog.Resolve(ctxif, publicID, systemID); resolved != "" {
			// A catalog may resolve to a "file:" URI; convert it to a local
			// path before opening (CAT-001).
			openName := catalogOpenName(resolved)
			f, err := ctx.openExternalResource(openName, ctx.extRefRelative)
			if errors.Is(err, ErrNetworkAccessForbidden) {
				return nil, ErrNetworkAccessForbidden
			}
			if err == nil {
				return &fileParseInput{ReadCloser: f, uri: resolved}, nil
			}
		}
	}

	// Fall back to direct file-based resolution. The systemID at this point
	// is the entity's resolved URI (built from system ID + base URI in
	// EntityDecl). Open it verbatim first — a "file:" URI or absolute path — so a
	// caller FS keyed on that historical name still resolves; openExternalResource
	// normalizes it only for the confined-FS base-relative retry.
	if systemID != "" {
		f, err := ctx.openExternalResource(systemID, ctx.extRefRelative)
		if errors.Is(err, ErrNetworkAccessForbidden) {
			return nil, ErrNetworkAccessForbidden
		}
		if err == nil {
			return &fileParseInput{ReadCloser: f, uri: systemID}, nil
		}
	}

	return nil, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) EntityDecl(ctxif context.Context, name string, typ enum.EntityType, publicID string, systemID string, notation string) error {
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.EntityDecl called while note in subset")
	}

	ent, err := dtd.AddEntity(name, typ, publicID, systemID, notation)
	if err != nil {
		return err
	}

	// Build the full URI for external entities by resolving the system ID
	// against the document's base URI (mirrors libxml2's xmlSAX2EntityDecl).
	if ent.uri == "" && systemID != "" {
		base := ctx.baseURI
		if base != "" {
			resolved := BuildURI(systemID, base)
			if resolved != "" {
				ent.uri = resolved
			}
		}
		if ent.uri == "" {
			ent.uri = systemID
		}
	}

	return nil
}

func (t *TreeBuilder) UnparsedEntityDecl(ctxif context.Context, name string, publicID string, systemID string, notation string) error {
	// Mirror xmlSAX2UnparsedEntityDecl: register the NDATA entity in the DTD.
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.UnparsedEntityDecl called while not in subset")
	}

	ent, _ := dtd.AddEntity(name, enum.ExternalGeneralUnparsedEntity, publicID, systemID, notation)

	// Build the full URI for unparsed entities by resolving the system ID
	// against the document's base URI (mirrors libxml2's xmlSAX2UnparsedEntityDecl).
	if ent != nil && ent.uri == "" && systemID != "" {
		base := ctx.baseURI
		if base != "" {
			resolved := BuildURI(systemID, base)
			if resolved != "" {
				ent.uri = resolved
			}
		}
		if ent.uri == "" {
			ent.uri = systemID
		}
	}

	return nil
}

func (t *TreeBuilder) Error(ctxif context.Context, err error) error {
	return nil
}

func (t *TreeBuilder) Warning(ctxif context.Context, err error) error {
	return nil
}
