package helium

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/uripath"
)

// NodeGetBase returns the effective base URI for a node by walking ancestors
// and resolving xml:base attributes. Returns empty string if no base URI found.
func NodeGetBase(doc *Document, n Node) string {
	if n == nil {
		return ""
	}

	// Collect xml:base values from the node up to the root.
	var bases []string
	for cur := n; cur != nil; cur = cur.Parent() {
		if elem, ok := AsNode[*Element](cur); ok {
			if val, ok := elem.GetAttributeNS("base", lexicon.NamespaceXML); ok && val != "" {
				bases = append(bases, val)
			}
		}
		// If an ancestor has an entity base URI, use it as the base
		// instead of continuing up to the document.
		if ebu := cur.baseDocNode().entityBaseURI; ebu != "" {
			// Resolve from outermost to innermost, starting from the
			// entity base URI instead of the document URL.
			base := ebu
			for _, v := range slices.Backward(bases) {
				base = BuildURI(v, base)
			}
			return base
		}
	}

	// Use the document's URL as the starting base, if available.
	var base string
	if doc != nil && doc.url != "" {
		base = doc.url
	}

	// Resolve from outermost ancestor inward (reverse order).
	for _, v := range slices.Backward(bases) {
		if base == "" {
			base = v
		} else {
			base = BuildURI(v, base)
		}
	}

	return base
}

// SetNodeBaseURI sets an explicit base URI on a node. This is used by
// xsl:copy to preserve the original element's base URI on the copy.
func SetNodeBaseURI(n Node, uri string) {
	n.baseDocNode().entityBaseURI = uri
}

// BuildURI resolves a relative system ID against a base URI.
//
// ARGUMENT ORDER: the reference comes FIRST, the base SECOND —
// BuildURI(reference, base). This is a byte-faithful port of libxml2's
// xmlBuildURI(URI, base) and keeps its (reference, base) order for parity, which
// is the OPPOSITE of Go's url.URL.ResolveReference and RFC 3986's resolve(base,
// ref). Passing the arguments in the conventional (base, reference) order returns
// a plausible-but-wrong result with no error. Callers who want the conventional
// order should use [ResolveURI] instead.
//
// For local file paths (no scheme or file: scheme), it joins with
// forward-slash (path) semantics. For other schemes, it uses
// url.ResolveReference.
//
// Native Windows paths (a drive-letter prefix such as "C:\\dir\\doc.xml" or a
// backslash/UNC path) are recognized and resolved with local-path semantics
// rather than being handed to url.Parse, which would otherwise read the drive
// letter "C" as a URI scheme and emit garbage like "c:///child.xml". POSIX
// behavior is unchanged: only inputs whose string shape is Windows-native take
// the native branch, so the shape is recognized — and tested — on any GOOS.
func BuildURI(systemID, base string) string {
	// An absolute systemID under either OS convention stands on its own.
	if uripath.IsWindowsAbsolute(systemID) {
		return systemID
	}

	// An absolute-URI systemID (one carrying a scheme, e.g. "http://host/p" or
	// "file:///x") stands on its own too, REGARDLESS of the base's shape. This
	// must be checked before the Windows-base branch below: when the base is a
	// native Windows path, that branch would otherwise treat the absolute URI as
	// a relative path segment and join it onto the base — collapsing "http://"
	// to "http:/" via path.Join. POSIX behavior is unchanged because the same
	// absoluteness is detected by url.Parse/IsAbs further down.
	if uripath.HasURIScheme(systemID) {
		return systemID
	}

	// When the base is a native Windows path, resolve the (relative) systemID
	// against it with local-path semantics, bypassing the URI machinery so the
	// drive letter is never mistaken for a scheme. This also covers a RELATIVE
	// Windows base (a backslash-bearing path with no URI scheme, e.g.
	// "..\dir\doc.xml" as produced by filepath.Join on Windows): url.Parse would
	// treat the backslashes as opaque, drop the directory via path.Dir, and
	// resolve "world.txt" to a bare "world.txt" that no longer points inside the
	// base directory. buildLocalPathURI normalizes with uripath.ToSlash first, so
	// the directory is preserved on every OS. A backslash in a POSIX filename is
	// pathological and not a real base shape, so gating on '\' is safe.
	if uripath.IsWindowsAbsolute(base) ||
		(!uripath.HasURIScheme(base) && strings.ContainsRune(base, '\\')) {
		return buildLocalPathURI(systemID, base)
	}

	u, err := url.Parse(systemID)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return systemID
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	if baseURL.Scheme != "" && baseURL.Scheme != "file" {
		return baseURL.ResolveReference(u).String()
	}

	if uripath.IsPOSIXAbsolute(systemID) {
		return systemID
	}
	basePath := baseURL.Path
	if basePath == "" {
		basePath = base
	}
	// Resolve with forward-slash (path) semantics, not filepath, so the
	// returned URI/path uses '/' on every OS. On Windows filepath.Dir/Join
	// would emit '\' here and corrupt a POSIX-shaped or file:-URI base (a
	// sibling "a.dtd" against "/dir/doc.xml" would become "\\dir\\a.dtd").
	dir := path.Dir(basePath)
	if strings.HasSuffix(basePath, "/") {
		dir = strings.TrimRight(basePath, "/")
	}
	result := path.Join(dir, systemID)
	if strings.HasSuffix(systemID, "/") && !strings.HasSuffix(result, "/") {
		result += "/"
	}
	// A "file:" base with a Windows drive letter (url.Parse("file:///D:/x").Path
	// == "/D:/x") yields a drive-rooted "/D:/..." result here, which is neither a
	// usable native path nor a "file:" URI: an fs.FS keyed on native paths can't
	// open "/D:/...". Re-attach the "file://" scheme so the value round-trips as
	// a proper file: URI ("file:///D:/...") that downstream file-URI-aware
	// loaders (normalizingFS, the entity resolver, catalogs) convert back to a
	// native path. A POSIX file: base ("file:///tmp/x" -> "/tmp/...") is left
	// untouched, so POSIX behavior is identical.
	if baseURL.Scheme == "file" && isDriveRootedPath(result) {
		return "file://" + result
	}
	return result
}

// ResolveURI resolves a relative reference against a base URI, using the
// conventional (base, reference) argument order matching Go's
// url.URL.ResolveReference and RFC 3986 resolve(base, ref). It wraps the
// libxml2-parity primitive [BuildURI] (which takes its arguments in the reverse,
// (reference, base) order) and returns an error when the reference cannot be
// resolved against the base.
func ResolveURI(base, ref string) (string, error) {
	resolved := BuildURI(ref, base)
	if resolved == "" {
		return "", fmt.Errorf(`failed to resolve reference %q against base %q`, ref, base)
	}
	return resolved, nil
}

// isDriveRootedPath reports whether p has the shape "/X:/..." — a leading slash
// followed by a Windows drive letter and colon. This is the malformed form that
// url.Parse produces for the path component of a "file:///X:/..." URI. The check
// is purely string-based, so it is GOOS-independent (and testable on POSIX).
func isDriveRootedPath(p string) bool {
	return len(p) >= 3 && p[0] == '/' && uripath.IsWindowsDriveLetter(p[1]) && p[2] == ':'
}

// buildLocalPathURI resolves a relative systemID against a native Windows base
// path. It works entirely in forward-slash space (so it is deterministic and
// testable on any GOOS): it normalizes both inputs with uripath.ToSlash (an
// unconditional backslash→slash conversion, unlike filepath.ToSlash which is a
// no-op on POSIX), strips the base's last path segment, and joins with path.Join
// (slash semantics). The forward-slash result is a valid path on Windows (the
// Win32 file APIs accept "/") and avoids the drive-letter-as-scheme corruption
// that url.Parse would produce. systemID is already known not to be
// Windows-absolute.
func buildLocalPathURI(systemID, base string) string {
	slashBase := uripath.ToSlash(base)
	slashRef := uripath.ToSlash(systemID)

	dir := slashBase
	if idx := strings.LastIndexByte(slashBase, '/'); idx >= 0 {
		// Keep everything up to (not including) the last slash. For a
		// drive-only base like "C:" with no slash, dir stays the whole base.
		dir = slashBase[:idx]
	}

	result := path.Join(dir, slashRef)
	// path.Join cleans the result, which collapses a leading "//" (UNC) to a
	// single "/". Restore the UNC double-slash when the base was a UNC path.
	if strings.HasPrefix(slashBase, "//") && !strings.HasPrefix(result, "//") {
		result = "/" + result
	}
	if strings.HasSuffix(slashRef, "/") && !strings.HasSuffix(result, "/") {
		result += "/"
	}
	return result
}
