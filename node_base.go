package helium

import (
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
	// drive letter is never mistaken for a scheme.
	if uripath.IsWindowsAbsolute(base) {
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
	return result
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
