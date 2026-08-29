package helium

import (
	"fmt"
	"slices"
	"strings"
)

// CopyNode creates a deep copy of src, owned by targetDoc. A nil targetDoc
// creates a standalone copy.
// Supports Element, Text, Comment, CDATASection, PI, and EntityRef nodes.
//
// A nil or typed-nil src (e.g. the *Element that Document.DocumentElement
// returns for a rootless document) returns ErrNilNode instead of panicking.
func CopyNode(src Node, targetDoc *Document) (Node, error) {
	if isNilNode(src) {
		return nil, ErrNilNode
	}
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
		// streaming's fixNamespacesAfterCopy) and uses the copier-only direct
		// linker. See deepCopyOptions for the knobs.
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
		return copyEntityReference(src, targetDoc, nil)
	case NamespaceNode:
		// Namespace nodes are virtual; return a new wrapper with the same data.
		ns := NewNamespace(src.Name(), string(src.Content()))
		return NewNamespaceNodeWrapper(ns, nil), nil
	default:
		return nil, fmt.Errorf("helium: cannot copy node of type %s", src.Type())
	}
}

// copyEntityReference creates a fresh reference owned by targetDoc. When the
// enclosing copy includes the source reference's actual declaration,
// entityCopies binds the reference to that declaration's copy. A document copy
// keeps an unbound source reference unbound even when the copied DTD has a
// declaration with the same name. Other callers retain the name-based targetDoc
// lookup used by CopyNode. Numeric character references always remain bare. No
// path aliases the source DTD or invents a destination declaration.
func copyEntityReference(
	src Node,
	targetDoc *Document,
	entityCopies map[*Entity]*Entity,
) (*EntityRef, error) {
	name := src.Name()
	if strings.HasPrefix(name, "#") {
		return targetDoc.CreateCharRef(name)
	}
	if targetDoc == nil {
		return targetDoc.CreateCharRef(name)
	}
	if srcEntity, ok := AsNode[*Entity](src.FirstChild()); ok {
		if dstEntity := entityCopies[srcEntity]; dstEntity != nil {
			ref, err := targetDoc.CreateCharRef(name)
			if err != nil {
				return nil, err
			}
			bindEntityReference(ref, dstEntity)
			return ref, nil
		}
	}
	if entityCopies != nil {
		return targetDoc.CreateCharRef(name)
	}
	return targetDoc.CreateReference(name)
}

// CopyDoc creates a complete deep copy of a document: its children, its internal
// AND external DTD subsets, and the document-level state a caller reasonably
// relies on (the XML-declaration version/encoding/standalone, the base URI, the
// property flags, the ID-skip state, and the interned ID table). The copy is
// fully independent — the external subset is deep-copied, each bound entity
// reference binds to the copy of its actual source declaration, and the ID table
// is rebuilt against the copy's own elements. No mutable map or DTD is aliased
// between src and the result. Mutating one document never affects the other.
func CopyDoc(src *Document) (*Document, error) {
	if src == nil {
		return nil, fmt.Errorf("helium: cannot copy nil document")
	}

	dst := NewDocument(src.version, src.encoding, src.standalone)

	// Document-level state the copy stands in for the source on. url backs base-URI
	// resolution; properties record how the source was produced (DocWellFormed,
	// DocHTML, …); idsSkip preserves id()/GetElementByID semantics (a SkipIDs source
	// must yield a copy that resolves NO ids).
	dst.url = src.url
	dst.properties = src.properties
	dst.idsSkip = src.idsSkip

	// Deep-copy both DTD subsets before document content so EntityRef copies can
	// bind to the copy of their actual source declaration. The external subset's
	// off-chain parent claim is recorded after the child list is built to retain
	// O(1) document appends.
	entityCopies, err := copyDTDSubsets(src, dst, false)
	if err != nil {
		return nil, err
	}

	// Copy all document children (the DTD was already handled) through the shared
	// core in over-declare mode, reproducing the historical behavior.
	// onCopy records the source->copy element correspondence so the ID table can be
	// rebuilt against the copy's own elements, aliasing nothing in the source map.
	// Build this fresh child list before the DTD-subset copy records its off-chain
	// claim on dst; otherwise every append must walk the growing list to recover
	// its tail.
	var onCopy func(src, cp Node)
	var corr elemCorrespondence
	if len(src.ids) > 0 {
		corr.m = make(map[*Element]*Element, len(src.ids))
		onCopy = corr.record
	}
	dc := &deepCopier{
		dst: dst,
		opts: deepCopyOptions{
			overDeclareNS: true,
			onCopy:        onCopy,
			entityCopies:  entityCopies,
		},
	}
	if err := dc.copyChildren(src, dst, nil, nil); err != nil {
		return nil, err
	}

	if dst.extSubset != nil {
		dst.offChainChildClaim = true
	}

	// Rebuild the interned ID table by translating each source entry's element
	// through the source->copy correspondence, so id()/GetElementByID on the copy
	// resolves to the copy's own elements at O(1). Sharing src.ids would alias the
	// source's elements into the copy.
	for id, srcElem := range src.ids {
		if cp := corr.m[srcElem]; cp != nil {
			dst.RegisterID(id, cp)
		}
	}

	return dst, nil
}

// elemCorrespondence records the source->copy element mapping produced during a
// deep copy, backing CopyDoc's ID-table rebuild. Its record method is used as the
// deepCopier onCopy hook.
type elemCorrespondence struct {
	m map[*Element]*Element
}

func (e *elemCorrespondence) record(src, cp Node) {
	srcElem, ok := AsNode[*Element](src)
	if !ok {
		return
	}
	if cpElem, ok := AsNode[*Element](cp); ok {
		e.m[srcElem] = cpElem
	}
}
