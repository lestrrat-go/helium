package helium

import (
	"fmt"
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
	state, err := copyDTDDeclarations(src, dstDTD, dst)
	if err != nil {
		return err
	}
	return copyDTDReplacements(state, dst)
}

// CopyDTDSubsets deep-copies both DTD subsets from src into dst. It registers
// every declaration before copying entity replacement trees, so references can
// resolve across the internal/external subset boundary. A nil src or dst is a
// no-op. It returns an error when the internal subset cannot be installed or a
// replacement tree cannot be linked safely.
func CopyDTDSubsets(src, dst *Document) error {
	return copyDTDSubsets(src, dst, true)
}

func copyDTDSubsets(src, dst *Document, recordOffChainClaim bool) error {
	if src == nil || dst == nil {
		return nil
	}

	var states []dtdCopyState
	if src.intSubset != nil {
		dstDTD, err := dst.CreateInternalSubset(
			src.intSubset.name, src.intSubset.externalID, src.intSubset.systemID,
		)
		if err != nil {
			return err
		}
		state, err := copyDTDDeclarations(src.intSubset, dstDTD, dst)
		if err != nil {
			return err
		}
		states = append(states, state)
	}

	if src.extSubset != nil {
		dstDTD := newExternalSubsetCopy(src.extSubset, dst)
		state, err := copyDTDDeclarations(src.extSubset, dstDTD, dst)
		if err != nil {
			return err
		}
		dst.extSubset = dstDTD
		if recordOffChainClaim {
			dst.offChainChildClaim = true
		}
		states = append(states, state)
	}

	for _, state := range states {
		if err := copyDTDReplacements(state, dst); err != nil {
			return err
		}
	}

	return nil
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
	copyExtSubset(src, dst, true)
}

func copyExtSubset(src, dst *Document, recordOffChainClaim bool) {
	if src == nil || dst == nil {
		return
	}
	srcDTD := src.extSubset
	if srcDTD == nil {
		return
	}

	dstDTD := newExternalSubsetCopy(srcDTD, dst)
	state, err := copyDTDDeclarations(srcDTD, dstDTD, dst)
	if err != nil {
		return
	}

	dst.extSubset = dstDTD
	if recordOffChainClaim {
		dst.offChainChildClaim = true
	}
	if err := copyDTDReplacements(state, dst); err != nil {
		return
	}
	// dstDTD claims dst as its parent but is deliberately NOT in dst's child
	// list, so dst now holds a node that can move its recorded lastChild off that
	// list. Record it on dst, the parent that was handed the claimant: appends
	// onto dst itself stop resolving their point from dst.lastChild
	// (tailJumpTarget, resolveOwnedTail) and walk instead, while every element
	// dst owns keeps its own O(1) resolution.
}

func newExternalSubsetCopy(srcDTD *DTD, dst *Document) *DTD {
	dstDTD := newDTD()
	dstDTD.name = srcDTD.name
	dstDTD.externalID = srcDTD.externalID
	dstDTD.systemID = srcDTD.systemID
	dstDTD.doc = dst
	dstDTD.parent = dst
	return dstDTD
}

// dtdCopyState keeps the source-to-copy entity correspondence between the
// declaration and replacement-tree phases.
type dtdCopyState struct {
	src          *DTD
	entityCopies map[*Entity]*Entity
}

// copyDTDDeclarations walks src's children in document order, copying each
// declaration as an independent node owned by dst and registering it both in
// dstDTD's lookup maps and as a child (so serialization round-trips
// identically). Replacement trees are copied separately after all applicable
// subsets have completed this phase.
func copyDTDDeclarations(src, dstDTD *DTD, dst *Document) (dtdCopyState, error) {
	// Correspondence from each source attribute declaration to its copy, so the
	// copy's registration-order sequences can be rebuilt from the source's.
	attrCopies := make(map[*AttributeDecl]*AttributeDecl)
	entityCopies := make(map[*Entity]*Entity)

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
				entityCopies[ent] = cp
				if err := dstDTD.AddChild(cp); err != nil {
					return dtdCopyState{}, err
				}
			}
		case ElementDeclNode:
			if edecl, ok := AsNode[*ElementDecl](c); ok {
				cp := copyElementDecl(edecl, dst)
				dstDTD.elements[edecl.name+":"+edecl.prefix] = cp
				if err := dstDTD.AddChild(cp); err != nil {
					return dtdCopyState{}, err
				}
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
				if err := dstDTD.AddChild(cp); err != nil {
					return dtdCopyState{}, err
				}
			}
		case NotationNode:
			if nota, ok := AsNode[*Notation](c); ok {
				cp := copyNotation(nota, dst)
				dstDTD.notations[nota.name] = cp
				if err := dstDTD.AddChild(cp); err != nil {
					return dtdCopyState{}, err
				}
			}
		case CommentNode:
			if err := dstDTD.AddChild(dst.CreateComment(slices.Clone(c.Content()))); err != nil {
				return dtdCopyState{}, err
			}
		case ProcessingInstructionNode:
			if err := dstDTD.AddChild(dst.CreatePI(c.Name(), string(c.Content()))); err != nil {
				return dtdCopyState{}, err
			}
		}
	}

	copyAttrDeclOrder(src, dstDTD, attrCopies)
	return dtdCopyState{src: src, entityCopies: entityCopies}, nil
}

// copyDTDReplacements copies each entity's parsed replacement tree after every
// declaration needed by the operation has been registered. It validates the
// completed declaration graph with shared visit state before linking any fresh
// replacement roots, so both validation and linking are linear in graph size.
func copyDTDReplacements(state dtdCopyState, dst *Document) error {
	// EntityRef nodes share their declaration's parsed replacement subtree.
	// Copy it only after every declaration has been registered, so nested and
	// forward references resolve to destination-owned Entity nodes.
	dc := &deepCopier{dst: dst, opts: deepCopyOptions{overDeclareNS: true}}
	var replacements []dtdReplacementCopy
	for c := range Children(state.src) {
		ent, ok := AsNode[*Entity](c)
		if !ok {
			continue
		}
		dstEnt := state.entityCopies[ent]
		replacementCopy := dtdReplacementCopy{entity: dstEnt}
		for replacement := range Children(ent) {
			child, err := dc.copyNode(replacement, nil, nil, nil)
			if err != nil {
				return err
			}
			if child == nil {
				continue
			}
			replacementCopy.children = append(replacementCopy.children, child)
		}
		replacements = append(replacements, replacementCopy)
	}

	validator := newDTDReplacementGraphValidator(replacements)
	if validator.hasCycle() {
		return fmt.Errorf("%w: cannot copy a cyclic DTD entity replacement graph", ErrCyclicNode)
	}

	for _, replacement := range replacements {
		for _, child := range replacement.children {
			if err := appendCopiedChild(replacement.entity, child); err != nil {
				return err
			}
		}
	}

	return nil
}

type dtdReplacementCopy struct {
	entity   *Entity
	children []Node
}

const (
	dtdReplacementVisitActive uint8 = iota + 1
	dtdReplacementVisitDone
)

// dtdReplacementGraphValidator checks the prospective declaration graph before
// its detached replacement roots are linked. states is shared across every
// declaration root, so a chain reached from many entities is visited once.
type dtdReplacementGraphValidator struct {
	roots        []*Entity
	replacements map[*docnode][]Node
	states       map[*docnode]uint8
	visits       int
}

func newDTDReplacementGraphValidator(replacements []dtdReplacementCopy) *dtdReplacementGraphValidator {
	v := &dtdReplacementGraphValidator{
		roots:        make([]*Entity, 0, len(replacements)),
		replacements: make(map[*docnode][]Node, len(replacements)),
		states:       make(map[*docnode]uint8),
	}
	for _, replacement := range replacements {
		v.roots = append(v.roots, replacement.entity)
		v.replacements[replacement.entity.baseDocNode()] = replacement.children
	}
	return v
}

type dtdReplacementGraphFrame struct {
	node           Node
	entered        bool
	hasReplacement bool
	replacements   []Node
	replacementPos int
	nextChild      Node
	siblingGuard   siblingCycleGuard
}

// hasCycle reports whether any replacement child can reach an entity that is
// already active on the same child-pointer path. The explicit stack keeps deep
// declaration chains safe, while the shared done state bounds work to one visit
// per node across all entity roots.
func (v *dtdReplacementGraphValidator) hasCycle() bool {
	for _, entity := range v.roots {
		if v.states[entity.baseDocNode()] == dtdReplacementVisitDone {
			continue
		}
		if v.hasCycleFrom(entity) {
			return true
		}
	}
	return false
}

func (v *dtdReplacementGraphValidator) hasCycleFrom(root Node) bool {
	stack := []dtdReplacementGraphFrame{{node: root}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		dn := top.node.baseDocNode()
		if !top.entered {
			if v.states[dn] == dtdReplacementVisitActive {
				return true
			}
			if v.states[dn] == dtdReplacementVisitDone {
				stack = stack[:len(stack)-1]
				continue
			}
			v.states[dn] = dtdReplacementVisitActive
			v.visits++
			top.entered = true
			top.replacements, top.hasReplacement = v.replacements[dn]
			if !top.hasReplacement {
				top.nextChild = dn.firstChild
			}
		}

		child, cycle := top.takeChild(dn)
		if cycle {
			return true
		}
		if child == nil {
			v.states[dn] = dtdReplacementVisitDone
			stack = stack[:len(stack)-1]
			continue
		}
		cdn := child.baseDocNode()
		if v.states[cdn] == dtdReplacementVisitActive {
			return true
		}
		if v.states[cdn] == dtdReplacementVisitDone {
			continue
		}
		stack = append(stack, dtdReplacementGraphFrame{node: child})
	}
	return false
}

func (f *dtdReplacementGraphFrame) takeChild(owner *docnode) (Node, bool) {
	if f.hasReplacement {
		if f.replacementPos >= len(f.replacements) {
			return nil, false
		}
		child := f.replacements[f.replacementPos]
		f.replacementPos++
		return child, false
	}

	child := f.nextChild
	if child == nil {
		return nil, false
	}
	cdn := child.baseDocNode()
	if f.siblingGuard.step(cdn) {
		return nil, true
	}
	f.nextChild = nextOwnedSibling(owner, cdn)
	return child, false
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
