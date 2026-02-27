package helium

import "fmt"

// CopyNode creates a deep copy of src, owned by targetDoc.
// Supports Element, Text, Comment, CDATASection, PI, and EntityRef nodes.
func CopyNode(src Node, targetDoc *Document) (Node, error) {
	switch src.Type() {
	case ElementNode:
		return copyElement(src.(*Element), targetDoc)
	case TextNode:
		return targetDoc.CreateText(copyBytes(src.Content()))
	case CommentNode:
		return targetDoc.CreateComment(copyBytes(src.Content()))
	case CDATASectionNode:
		return targetDoc.CreateCDATASection(copyBytes(src.Content()))
	case ProcessingInstructionNode:
		return targetDoc.CreatePI(src.Name(), string(src.Content()))
	case EntityRefNode:
		return targetDoc.CreateCharRef(src.Name())
	default:
		return nil, fmt.Errorf("helium: cannot copy node of type %s", src.Type())
	}
}

func copyElement(src *Element, doc *Document) (*Element, error) {
	elem, err := doc.CreateElement(src.LocalName())
	if err != nil {
		return nil, err
	}

	// Track which namespace prefixes have been declared on this element
	declaredPrefixes := make(map[string]bool)

	// Copy namespace declarations (nsDefs)
	if nc, ok := Node(src).(NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			if err := elem.SetNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
			declaredPrefixes[ns.Prefix()] = true
		}
	}

	// Copy the active namespace, adding a declaration if not already present.
	if nsr, ok := Node(src).(Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if ns.Prefix() != "" && !declaredPrefixes[ns.Prefix()] {
				if err := elem.SetNamespace(ns.Prefix(), ns.URI()); err != nil {
					return nil, err
				}
				declaredPrefixes[ns.Prefix()] = true
			}
			if err := elem.SetNamespace(ns.Prefix(), ns.URI(), true); err != nil {
				return nil, err
			}
		}
	}

	// Copy attributes
	for _, a := range src.Attributes() {
		if err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
			return nil, err
		}
	}

	// Recursively copy children
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		child, err := CopyNode(c, doc)
		if err != nil {
			return nil, err
		}
		if err := elem.AddChild(child); err != nil {
			return nil, err
		}
	}

	return elem, nil
}

func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// CopyDoc creates a deep copy of a document including all children.
func CopyDoc(src *Document) (*Document, error) {
	if src == nil {
		return nil, fmt.Errorf("helium: cannot copy nil document")
	}

	dst := NewDocument(src.version, src.encoding, src.standalone)

	// Copy DTD metadata (name, externalID, systemID) if present.
	if dtd := src.intSubset; dtd != nil {
		if _, err := dst.CreateInternalSubset(dtd.name, dtd.externalID, dtd.systemID); err != nil {
			return nil, err
		}
	}

	// Copy all children (except the DTD which was already handled).
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == DTDNode {
			continue
		}
		child, err := CopyNode(c, dst)
		if err != nil {
			return nil, err
		}
		if err := dst.AddChild(child); err != nil {
			return nil, err
		}
	}

	return dst, nil
}
