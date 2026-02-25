package xpointer

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

// CopyNode creates a deep copy of src, owned by targetDoc.
// Supports Element, Text, Comment, CDATASection, and ProcessingInstruction nodes.
func CopyNode(src helium.Node, targetDoc *helium.Document) (helium.Node, error) {
	return copyNode(src, targetDoc)
}

func copyNode(src helium.Node, doc *helium.Document) (helium.Node, error) {
	switch src.Type() {
	case helium.ElementNode:
		return copyElement(src.(*helium.Element), doc)
	case helium.TextNode:
		return doc.CreateText(copyBytes(src.Content()))
	case helium.CommentNode:
		return doc.CreateComment(copyBytes(src.Content()))
	case helium.CDATASectionNode:
		return doc.CreateCDATASection(copyBytes(src.Content()))
	case helium.ProcessingInstructionNode:
		return doc.CreatePI(src.Name(), string(src.Content()))
	default:
		return nil, fmt.Errorf("xpointer: cannot copy node of type %s", src.Type())
	}
}

func copyElement(src *helium.Element, doc *helium.Document) (*helium.Element, error) {
	elem, err := doc.CreateElement(src.LocalName())
	if err != nil {
		return nil, err
	}

	// Track which namespace prefixes have been declared on this element
	declaredPrefixes := make(map[string]bool)

	// Copy namespace declarations (nsDefs)
	if nc, ok := helium.Node(src).(helium.NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			if err := elem.SetNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
			declaredPrefixes[ns.Prefix()] = true
		}
	}

	// Copy the active namespace, adding a declaration if not already present.
	// This ensures the copied element is self-contained when placed in a
	// different tree context where the parent may not declare this namespace.
	if nsr, ok := helium.Node(src).(helium.Namespacer); ok {
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
		child, err := copyNode(c, doc)
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
