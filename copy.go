package helium

import (
	"fmt"
	"slices"

	"github.com/lestrrat-go/helium/enum"
)

// CopyNode creates a deep copy of src, owned by targetDoc.
// Supports Element, Text, Comment, CDATASection, PI, and EntityRef nodes.
func CopyNode(src Node, targetDoc *Document) (Node, error) {
	switch src.Type() {
	case DocumentNode:
		doc, ok := src.(*Document)
		if !ok {
			return nil, fmt.Errorf("helium: unexpected DocumentNode type %T", src)
		}
		return CopyDoc(doc)
	case ElementNode:
		return copyElement(src.(*Element), targetDoc)
	case TextNode:
		return targetDoc.CreateText(slices.Clone(src.Content()))
	case CommentNode:
		return targetDoc.CreateComment(slices.Clone(src.Content()))
	case CDATASectionNode:
		return targetDoc.CreateCDATASection(slices.Clone(src.Content()))
	case ProcessingInstructionNode:
		return targetDoc.CreatePI(src.Name(), string(src.Content()))
	case EntityRefNode:
		return targetDoc.CreateCharRef(src.Name())
	case NamespaceNode:
		// Namespace nodes are virtual; return a new wrapper with the same data.
		ns := NewNamespace(src.Name(), string(src.Content()))
		return NewNamespaceNodeWrapper(ns, nil), nil
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
			if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
			declaredPrefixes[ns.Prefix()] = true
		}
	}

	// Copy the active namespace, adding a declaration if not already present.
	if nsr, ok := Node(src).(Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if ns.URI() != "" && !declaredPrefixes[ns.Prefix()] {
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return nil, err
				}
				declaredPrefixes[ns.Prefix()] = true
			}
			if err := elem.SetActiveNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
		}
	}

	// Copy attributes, preserving namespace information
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			ns, nsErr := doc.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nil, nsErr
			}
			if err := elem.SetAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return nil, err
			}
		} else {
			if err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
				return nil, err
			}
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

// CopyDoc creates a deep copy of a document including all children.
func CopyDoc(src *Document) (*Document, error) {
	if src == nil {
		return nil, fmt.Errorf("helium: cannot copy nil document")
	}

	dst := NewDocument(src.version, src.encoding, src.standalone)

	// Deep-copy DTD (metadata + entities, elements, attributes, notations).
	if dtd := src.intSubset; dtd != nil {
		if err := copyDTD(dtd, dst); err != nil {
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

// CopyDTDInfo copies DTD information (entities, notations, element/attribute
// declarations) from src to dst. This preserves unparsed entity information
// when creating document copies via xsl:copy.
func CopyDTDInfo(src, dst *Document) {
	if src == nil || dst == nil {
		return
	}
	if dtd := src.intSubset; dtd != nil {
		_ = copyDTD(dtd, dst)
	}
}

// copyDTD deep-copies src into dst's internal subset, including all
// entities, parameter entities, element declarations, attribute
// declarations, and notation declarations.  Children are walked in
// order so that serialization of the copy matches the original.
func copyDTD(src *DTD, dst *Document) error {
	dstDTD, err := dst.CreateInternalSubset(src.name, src.externalID, src.systemID)
	if err != nil {
		return err
	}

	// Walk children in document order to preserve serialization ordering.
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Type() {
		case EntityNode:
			ent := c.(*Entity)
			cp := copyEntity(ent, dst)
			switch ent.entityType {
			case enum.InternalParameterEntity, enum.ExternalParameterEntity:
				dstDTD.pentities[ent.name] = cp
			default:
				dstDTD.entities[ent.name] = cp
			}
			_ = dstDTD.AddChild(cp)
		case ElementDeclNode:
			edecl := c.(*ElementDecl)
			cp := copyElementDecl(edecl, dst)
			dstDTD.elements[edecl.name+":"+edecl.prefix] = cp
			_ = dstDTD.AddChild(cp)
		case AttributeDeclNode:
			adecl := c.(*AttributeDecl)
			cp := copyAttributeDecl(adecl, dst)
			dstDTD.attributes[adecl.name+":"+adecl.prefix+":"+adecl.elem] = cp
			_ = dstDTD.AddChild(cp)
		case NotationNode:
			nota := c.(*Notation)
			cp := copyNotation(nota, dst)
			dstDTD.notations[nota.name] = cp
			_ = dstDTD.AddChild(cp)
		case CommentNode:
			cm, _ := dst.CreateComment(slices.Clone(c.Content()))
			_ = dstDTD.AddChild(cm)
		case ProcessingInstructionNode:
			pi, _ := dst.CreatePI(c.Name(), string(c.Content()))
			_ = dstDTD.AddChild(pi)
		}
	}

	return nil
}

func copyEntity(src *Entity, doc *Document) *Entity {
	e := newEntity(src.name, src.entityType, src.externalID, src.systemID, src.content, src.orig)
	e.uri = src.uri
	e.checked = src.checked
	e.expandedSize = src.expandedSize
	e.doc = doc
	return e
}

func copyElementDecl(src *ElementDecl, doc *Document) *ElementDecl {
	e := newElementDecl()
	e.name = src.name
	e.prefix = src.prefix
	e.decltype = src.decltype
	e.content = src.content.copyElementContent()
	e.doc = doc
	return e
}

func copyAttributeDecl(src *AttributeDecl, doc *Document) *AttributeDecl {
	a := newAttributeDecl()
	a.name = src.name
	a.prefix = src.prefix
	a.elem = src.elem
	a.atype = src.atype
	a.def = src.def
	a.defvalue = src.defvalue
	if src.tree != nil {
		a.tree = make(Enumeration, len(src.tree))
		copy(a.tree, src.tree)
	}
	a.doc = doc
	return a
}

func copyNotation(src *Notation, doc *Document) *Notation {
	n := &Notation{}
	n.etype = NotationNode
	n.name = src.name
	n.publicID = src.publicID
	n.systemID = src.systemID
	n.doc = doc
	return n
}
