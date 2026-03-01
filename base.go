package helium

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
			elem := cur.(*Element)
			if val, ok := elem.GetAttributeNS("base", XMLNamespace); ok && val != "" {
				bases = append(bases, val)
			}
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
