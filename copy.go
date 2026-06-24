package helium

import (
	"fmt"
	"slices"
)

// CopyNode creates a deep copy of src, owned by targetDoc.
// Supports Element, Text, Comment, CDATASection, PI, and EntityRef nodes.
func CopyNode(src Node, targetDoc *Document) (Node, error) {
	switch src.Type() {
	case DocumentNode:
		if doc, ok := AsNode[*Document](src); ok {
			return CopyDoc(doc)
		}
		return nil, fmt.Errorf("helium: unexpected DocumentNode type %T", src)
	case ElementNode:
		elem, ok := AsNode[*Element](src)
		if !ok {
			return nil, fmt.Errorf("helium: unexpected ElementNode type %T", src)
		}
		// The general copy path over-declares namespaces (load-bearing for
		// streaming's fixNamespacesAfterCopy) and links with the preflighted
		// AddChild, copying every node. See deepCopyOptions for the knobs.
		dc := &deepCopier{dst: targetDoc, opts: deepCopyOptions{overDeclareNS: true}}
		return dc.copyElement(elem, nil, nil)
	case TextNode:
		return targetDoc.CreateText(slices.Clone(src.Content())), nil
	case CommentNode:
		return targetDoc.CreateComment(slices.Clone(src.Content())), nil
	case CDATASectionNode:
		return targetDoc.CreateCDATASection(slices.Clone(src.Content())), nil
	case ProcessingInstructionNode:
		return targetDoc.CreatePI(src.Name(), string(src.Content())), nil
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

	// Copy all document children (the DTD was already handled) through the shared
	// core in over-declare / AddChild mode, reproducing the historical behavior.
	dc := &deepCopier{dst: dst, opts: deepCopyOptions{overDeclareNS: true}}
	if err := dc.copyChildren(src, dst, nil, nil); err != nil {
		return nil, err
	}

	return dst, nil
}
