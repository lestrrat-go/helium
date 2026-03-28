package helium

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
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
		if cur.Type() == ElementNode {
			elem := cur.(*Element) //nolint:forcetypeassert
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
			for i := len(bases) - 1; i >= 0; i-- {
				base = BuildURI(bases[i], base)
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
	for i := len(bases) - 1; i >= 0; i-- {
		if base == "" {
			base = bases[i]
		} else {
			base = BuildURI(bases[i], base)
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
// For local file paths (no scheme or file: scheme), it uses filepath.Join.
// For other schemes, it uses url.ResolveReference.
func BuildURI(systemID, base string) string {
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

	if filepath.IsAbs(systemID) {
		return systemID
	}
	basePath := baseURL.Path
	if basePath == "" {
		basePath = base
	}
	dir := filepath.Dir(basePath)
	if strings.HasSuffix(basePath, "/") {
		dir = strings.TrimRight(basePath, "/")
	}
	result := filepath.Join(dir, systemID)
	if strings.HasSuffix(systemID, "/") && !strings.HasSuffix(result, "/") {
		result += "/"
	}
	return result
}
