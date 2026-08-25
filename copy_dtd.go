package helium

import (
	"slices"

	"github.com/lestrrat-go/helium/enum"
)

// CopyDTDInfo copies DTD information (entities, notations, element/attribute
// declarations) from src's internal subset to dst. This preserves unparsed
// entity information when creating document copies via xsl:copy.
//
// It returns an error when the copy cannot be performed — most importantly when
// dst already has an internal subset (copyDTD calls CreateInternalSubset, which
// refuses to replace one). A nil src or dst is a no-op and returns nil.
func CopyDTDInfo(src, dst *Document) error {
	if src == nil || dst == nil {
		return nil
	}
	dtd := src.intSubset
	if dtd == nil {
		return nil
	}
	return copyDTD(dtd, dst)
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
	return copyDTDChildren(src, dstDTD, dst)
}

// CopyExtSubset deep-copies src's external DTD subset into dst, installing it as
// dst's external subset. The copy is fully independent: it owns its own *DTD and
// its own entity/element/attribute/notation declarations, so mutating dst's
// external subset (e.g. via AddNotation/AddEntity/AddElementDecl) never affects
// src's external subset, and vice versa.
//
// Unlike CopyDTDInfo (which copies the internal subset and links it into the
// document tree before the root element), the external subset is not a child of
// the document — it is referenced only via ExtSubset — so the copy is not added
// to dst's child list. If src has no external subset this is a no-op.
func CopyExtSubset(src, dst *Document) {
	if src == nil || dst == nil {
		return
	}
	srcDTD := src.extSubset
	if srcDTD == nil {
		return
	}

	dstDTD := newDTD()
	dstDTD.name = srcDTD.name
	dstDTD.externalID = srcDTD.externalID
	dstDTD.systemID = srcDTD.systemID
	dstDTD.doc = dst
	dstDTD.parent = dst

	_ = copyDTDChildren(srcDTD, dstDTD, dst)

	dst.extSubset = dstDTD
	// dstDTD claims dst as its parent but is deliberately NOT in dst's child
	// list, so dst now holds a node that can move its recorded lastChild off that
	// list. Record it: addSibling stops resolving an append point from
	// dst.lastChild (tailJumpTarget) and walks instead.
	dst.offChainChildClaim = true
}

// copyDTDChildren walks src's children in document order, copying each
// declaration as an independent node owned by dst and registering it both in
// dstDTD's lookup maps and as a child (so serialization round-trips
// identically). Shared by copyDTD (internal subset) and CopyExtSubset
// (external subset), which differ only in how dstDTD is allocated and where it
// is attached.
func copyDTDChildren(src, dstDTD *DTD, dst *Document) error {
	// Correspondence from each source attribute declaration to its copy, so the
	// copy's registration-order sequences can be rebuilt from the source's.
	attrCopies := make(map[*AttributeDecl]*AttributeDecl)

	// The DTD owns its declaration children, so Children's owned-boundary advance
	// equals a raw NextSibling walk here while adding a per-list seen guard, so a
	// corrupt (cyclic) declaration list terminates instead of spinning.
	for c := range Children(src) {
		switch c.Type() {
		case EntityNode:
			if ent, ok := AsNode[*Entity](c); ok {
				cp := copyEntity(ent, dst)
				switch ent.entityType {
				case enum.InternalParameterEntity, enum.ExternalParameterEntity:
					dstDTD.pentities[ent.name] = cp
				default:
					dstDTD.entities[ent.name] = cp
				}
				_ = dstDTD.AddChild(cp)
			}
		case ElementDeclNode:
			if edecl, ok := AsNode[*ElementDecl](c); ok {
				cp := copyElementDecl(edecl, dst)
				dstDTD.elements[edecl.name+":"+edecl.prefix] = cp
				_ = dstDTD.AddChild(cp)
			}
		case AttributeDeclNode:
			if adecl, ok := AsNode[*AttributeDecl](c); ok {
				cp := copyAttributeDecl(adecl, dst)
				// This is the one writer besides registerAttribute; it must keep every
				// container in sync since it bypasses registerAttribute (a direct map
				// write is needed here to overwrite a duplicate key, matching the
				// source's last-wins semantics for the attributes table). The
				// registration-order sequences are NOT filled here: the child list is
				// in serialization order, which a caller can reorder independently of
				// registration order, so they are rebuilt from the source's own
				// sequence once every copy exists (copyAttrDeclOrder below).
				dstDTD.attributes[attrDeclKey{local: adecl.name, prefix: adecl.prefix, elem: adecl.elem}] = cp
				attrCopies[adecl] = cp
				_ = dstDTD.AddChild(cp)
			}
		case NotationNode:
			if nota, ok := AsNode[*Notation](c); ok {
				cp := copyNotation(nota, dst)
				dstDTD.notations[nota.name] = cp
				_ = dstDTD.AddChild(cp)
			}
		case CommentNode:
			_ = dstDTD.AddChild(dst.CreateComment(slices.Clone(c.Content())))
		case ProcessingInstructionNode:
			_ = dstDTD.AddChild(dst.CreatePI(c.Name(), string(c.Content())))
		}
	}

	copyAttrDeclOrder(src, dstDTD, attrCopies)

	return nil
}

// copyAttrDeclOrder fills dstDTD's registration-order sequences (attrDecls and
// attrsByElem) by walking src's own attrDecls sequence and translating each
// source declaration through attrCopies. Order therefore comes from the source
// index, not from the DTD child list the copy walk followed: the two can differ,
// because relinking a declaration (AttributeDecl.AddSibling) moves it in the
// child list without re-registering it, and a declaration registered but never
// linked is absent from the child list altogether. A source declaration with no
// copy was not in the child list and so has no counterpart to record.
func copyAttrDeclOrder(src, dstDTD *DTD, attrCopies map[*AttributeDecl]*AttributeDecl) {
	for _, adecl := range src.attrDecls {
		cp, ok := attrCopies[adecl]
		if !ok {
			continue
		}
		dstDTD.attrDecls = append(dstDTD.attrDecls, cp)
		dstDTD.attrsByElem[adecl.elem] = append(dstDTD.attrsByElem[adecl.elem], cp)
	}
}

func copyEntity(src *Entity, doc *Document) *Entity {
	e := newEntity(src.name, src.entityType, src.externalID, src.systemID, src.content, src.orig)
	e.replacement = src.replacement
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
