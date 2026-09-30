package xsd

import (
	"context"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
)

// A document parsed without entity substitution keeps each general entity
// reference as an EntityRef node whose expansion (the referenced entity's own
// children, owned by the DTD) holds the replacement's text and elements. XML
// Schema validates the tree entity substitution would build, so the validator
// reads an EntityRef child as its expansion spliced into the host at the
// reference: its elements are element children of the host in document order,
// and its character data is character data of the host (helium.CharacterData).
//
// An entity referenced more than once shares one set of element nodes across
// its references. Every reference is validated as its own occurrence, but the
// per-node PSVI maps (type annotations, nilled elements, is-id nodes) hold one
// entry per node, which the last occurrence validated writes.
//
// Diagnostics about an element inside an expansion report the line of the
// outermost reference that reached it (validationContext.entityLine), which is
// the line a SubstituteEntities(true) parse gives every node of that expansion.

// contentPiece is one item of an entity reference's expansion as entity
// substitution places it in the host: an element, or a run of character data.
type contentPiece struct {
	elem *helium.Element
	text []byte
}

// referencedEntity returns the entity ref expands to: its bound Entity child,
// or else the owning document's declaration of that name. It returns nil when
// the reference resolves to no declared entity.
func referencedEntity(ref helium.Node) *helium.Entity {
	if ent, ok := ref.FirstChild().(*helium.Entity); ok {
		return ent
	}
	doc := ref.OwnerDocument()
	if doc == nil {
		return nil
	}
	ent, ok := doc.GetEntity(ref.Name())
	if !ok {
		return nil
	}
	return ent
}

// appendExpansion appends the pieces of ref's expansion to dst in document
// order: each element of the entity's replacement, the text of each Text and
// CDATA node, and each nested reference expanded in place. Comments and PIs add
// nothing. An entity whose replacement was never parsed into nodes, or a
// predefined or unresolved reference, adds its character data as one piece.
// With elemsOnly set, only the elements are appended.
//
// active holds the entities being expanded on the current path, so a reference
// back to one of them (a cyclic entity graph, constructible only through the
// DOM API) adds nothing instead of looping.
func appendExpansion(dst []contentPiece, ref helium.Node, elemsOnly bool, active []*helium.Entity) []contentPiece {
	ent := referencedEntity(ref)
	if ent == nil || ent.EntityType() == enum.InternalPredefinedEntity || ent.FirstChild() == nil {
		if elemsOnly {
			return dst
		}
		if text := helium.CharacterData(ref); text != "" {
			dst = append(dst, contentPiece{text: []byte(text)})
		}
		return dst
	}
	if slices.Contains(active, ent) {
		return dst
	}
	active = append(active, ent)
	for child := range helium.Children(ent) {
		switch child.Type() {
		case helium.ElementNode:
			if elem, ok := helium.AsNode[*helium.Element](child); ok {
				dst = append(dst, contentPiece{elem: elem})
			}
		case helium.TextNode, helium.CDATASectionNode:
			if !elemsOnly {
				dst = append(dst, contentPiece{text: child.Content()})
			}
		case helium.EntityRefNode:
			dst = appendExpansion(dst, child, elemsOnly, active)
		}
	}
	return dst
}

// expansionHasElement reports whether ref's expansion holds an element.
func expansionHasElement(ref helium.Node) bool {
	return len(appendExpansion(nil, ref, true, nil)) > 0
}

// expansionLine returns the line to report for a node of the expansion of
// ref, a child of host: outer, the line of the outermost reference already in
// effect (validationContext.entityLine), or ref's own line when host lies
// outside every expansion. A reference without a line (one built through the
// DOM API) falls back to host's line.
func expansionLine(outer int, host *helium.Element, ref helium.Node) int {
	if outer != 0 {
		return outer
	}
	if line := ref.Line(); line != 0 {
		return line
	}
	return host.Line()
}

// enterEntityLine makes line the line reported for every diagnostic until the
// caller restores the returned previous value. A zero line keeps the current
// one, so entering a child that is not entity-borne changes nothing.
func (vc *validationContext) enterEntityLine(line int) int {
	prev := vc.entityLine
	if line != 0 {
		vc.entityLine = line
	}
	return prev
}

// restoreEntityLine restores the line enterEntityLine returned. It exists so a
// caller can restore it with defer.
func (vc *validationContext) restoreEntityLine(prev int) {
	vc.entityLine = prev
}

// elementOccurrence is one element occurrence visited by walkOccurrences.
// index numbers the occurrences in document order and parent is the index of
// the element holding this one (the host of the reference for an element
// inside an expansion), or -1 for an element child of the document. An element
// shared by two references is two occurrences.
type elementOccurrence struct {
	elem   *helium.Element
	index  int
	parent int
}

// occurrenceVisitor is called by walkOccurrences for each element occurrence.
type occurrenceVisitor interface {
	visitOccurrence(ctx context.Context, occ elementOccurrence)
}

// occurrenceWalk is the state of one walkOccurrences run.
type occurrenceWalk struct {
	vc      *validationContext
	visitor occurrenceVisitor
	next    int
	// onPath holds the elements on the current descent path, so an element
	// reached again below itself (a corrupt child-pointer cycle) ends the walk.
	onPath map[*helium.Element]struct{}
	// entities holds the entities being expanded on the current path.
	entities []*helium.Entity
}

// walkOccurrences visits every element occurrence of doc's element trees in
// document order, the tree entity substitution would build: it descends into
// element children and splices in each entity reference's expansion, visiting
// the DTD and its declarations never. While it is inside an expansion,
// vc.entityLine holds the line of the outermost reference, so diagnostics the
// visitor reports point at the reference.
//
// It returns helium.ErrWalkCycle, after visiting what it reached, when the tree
// is not a tree: an element below itself, a sibling list that loops, or an
// entity whose expansion references itself.
func (vc *validationContext) walkOccurrences(ctx context.Context, doc *helium.Document, visitor occurrenceVisitor) error {
	w := &occurrenceWalk{vc: vc, visitor: visitor, onPath: make(map[*helium.Element]struct{})}
	return w.children(ctx, doc, nil, -1)
}

// children visits the element occurrences among parent's children. host is
// the element the children belong to after entity substitution (nil for the
// document), and hostIndex its occurrence index.
func (w *occurrenceWalk) children(ctx context.Context, parent helium.Node, host *helium.Element, hostIndex int) error {
	var guard nodeCycleGuard
	for child := parent.FirstChild(); child != nil; child = ownedNext(parent, child) {
		if guard.step(child) {
			return helium.ErrWalkCycle
		}
		switch child.Type() {
		case helium.ElementNode:
			elem, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if err := w.element(ctx, elem, hostIndex); err != nil {
				return err
			}
		case helium.EntityRefNode:
			if host == nil {
				continue
			}
			if err := w.expansion(ctx, child, host, hostIndex); err != nil {
				return err
			}
		}
	}
	return nil
}

// element visits the occurrence of elem, then the occurrences below it.
func (w *occurrenceWalk) element(ctx context.Context, elem *helium.Element, parentIndex int) error {
	if _, cyclic := w.onPath[elem]; cyclic {
		return helium.ErrWalkCycle
	}
	occ := elementOccurrence{elem: elem, index: w.next, parent: parentIndex}
	w.next++
	w.visitor.visitOccurrence(ctx, occ)
	w.onPath[elem] = struct{}{}
	err := w.children(ctx, elem, elem, occ.index)
	delete(w.onPath, elem)
	return err
}

// expansion visits the element occurrences of ref's expansion as children of
// host.
func (w *occurrenceWalk) expansion(ctx context.Context, ref helium.Node, host *helium.Element, hostIndex int) error {
	ent := referencedEntity(ref)
	if ent == nil || ent.FirstChild() == nil {
		return nil
	}
	if slices.Contains(w.entities, ent) {
		return helium.ErrWalkCycle
	}
	prevLine := w.vc.enterEntityLine(expansionLine(w.vc.entityLine, host, ref))
	w.entities = append(w.entities, ent)
	err := w.children(ctx, ent, host, hostIndex)
	w.entities = w.entities[:len(w.entities)-1]
	w.vc.entityLine = prevLine
	return err
}

// nodeCycleGuard detects a sibling list that loops back on itself with Brent's
// algorithm, without allocating: step reports true once the list is proven
// cyclic, within a small multiple of the loop length.
type nodeCycleGuard struct {
	tortoise helium.Node
	power    int
	lam      int
}

func (g *nodeCycleGuard) step(cur helium.Node) bool {
	if g.tortoise == nil {
		g.tortoise = cur
		g.power = 1
		return false
	}
	if g.tortoise == cur {
		return true
	}
	g.lam++
	if g.lam == g.power {
		g.power *= 2
		g.lam = 0
		g.tortoise = cur
	}
	return false
}

// idcDocument returns the document the identity-constraint pass walks. An
// xs:selector or xs:field path follows the child axis, which does not enter an
// entity reference, so when doc's DTD declares an entity holding elements the
// pass walks a copy of doc's element trees built as entity substitution would
// build them: each reference replaced by a copy of its expansion, every copied
// node of an expansion on the line of its outermost reference, and every PSVI
// record the pass reads (declaration, actual and assessed types, assessed
// attributes, skipped content) added for each copy under the copy's key. An
// element inside an entity referenced twice is then two elements, as it is
// after substitution. Otherwise, or when the copy cannot be built, it returns
// doc itself.
func (vc *validationContext) idcDocument(doc *helium.Document) *helium.Document {
	if !declaresElementEntity(doc.IntSubset()) && !declaresElementEntity(doc.ExtSubset()) {
		return doc
	}
	mirror := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	for root := range helium.ChildElements(doc) {
		copied, err := helium.CopyNode(root, mirror)
		if err != nil {
			return doc
		}
		ce, ok := helium.AsNode[*helium.Element](copied)
		if !ok {
			return doc
		}
		if err := mirror.AddChild(ce); err != nil {
			return doc
		}
		vc.mirrorElement(root, ce, 0, nil)
	}
	return mirror
}

// declaresElementEntity reports whether dtd declares a general entity whose
// parsed replacement holds an element. Every element reached through an
// entity reference is a child of such an entity.
func declaresElementEntity(dtd *helium.DTD) bool {
	if dtd == nil {
		return false
	}
	var scan elementEntityScan
	dtd.ForEachEntity(scan.check)
	return scan.found
}

// elementEntityScan is the ForEachEntity callback of declaresElementEntity.
type elementEntityScan struct {
	found bool
}

func (s *elementEntityScan) check(_ string, ent *helium.Entity) {
	if s.found {
		return
	}
	for child := range helium.Children(ent) {
		if child.Type() == helium.ElementNode {
			s.found = true
			return
		}
	}
}

// mirrorElement completes copied, the idcDocument copy of the live element
// live: it adds live's PSVI records for copied and its attributes, and walks
// the two child lists in parallel (helium.CopyNode keeps child types and
// order), replacing each copied entity reference with a copy of the live
// reference's expansion. line is the outermost reference line for an element
// inside an expansion, set on the copy, or 0. active holds the entities being
// expanded on the current path.
func (vc *validationContext) mirrorElement(live, copied *helium.Element, line int, active []*helium.Entity) {
	if line != 0 {
		copied.SetLine(line)
	}
	vc.mirrorPSVI(live, copied)
	oc := childNodes(live)
	cc := childNodes(copied)
	for i := range min(len(oc), len(cc)) {
		switch oc[i].Type() {
		case helium.ElementNode:
			oe, ok1 := helium.AsNode[*helium.Element](oc[i])
			ce, ok2 := helium.AsNode[*helium.Element](cc[i])
			if ok1 && ok2 {
				vc.mirrorElement(oe, ce, line, active)
			}
		case helium.EntityRefNode:
			ref, ok := cc[i].(helium.MutableNode)
			if !ok {
				continue
			}
			nodes := vc.mirrorExpansion(live, oc[i], copied.OwnerDocument(), line, active)
			if len(nodes) == 0 {
				helium.UnlinkNode(ref)
				continue
			}
			_ = ref.Replace(nodes...)
		}
	}
}

// mirrorExpansion builds in doc the nodes that replace the copy of the live
// entity reference ref, a child of host: its character data and a completed
// copy of each of its elements (mirrorElement).
func (vc *validationContext) mirrorExpansion(host *helium.Element, ref helium.Node, doc *helium.Document, line int, active []*helium.Entity) []helium.Node {
	ent := referencedEntity(ref)
	if ent != nil && slices.Contains(active, ent) {
		return nil
	}
	if ent != nil {
		active = append(active, ent)
	}
	line = expansionLine(line, host, ref)
	var nodes []helium.Node
	for _, p := range appendExpansion(nil, ref, false, nil) {
		if p.elem == nil {
			nodes = append(nodes, doc.CreateText(p.text))
			continue
		}
		copied, err := helium.CopyNode(p.elem, doc)
		if err != nil {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](copied)
		if !ok {
			continue
		}
		vc.mirrorElement(p.elem, ce, line, active)
		nodes = append(nodes, ce)
	}
	return nodes
}

// mirrorPSVI adds the PSVI records the identity-constraint pass reads for the
// live element live, and for its attributes, under their copies.
func (vc *validationContext) mirrorPSVI(live, copied *helium.Element) {
	if td, ok := vc.actualElemType[live]; ok {
		vc.actualElemType[copied] = td
	}
	if decl, ok := vc.actualElemDecl[live]; ok {
		vc.actualElemDecl[copied] = decl
	}
	if td, ok := vc.assessedElemType[live]; ok {
		vc.assessedElemType[copied] = td
	}
	if vc.skipContentNodes != nil {
		if _, ok := vc.skipContentNodes[live]; ok {
			vc.skipContentNodes[copied] = struct{}{}
		}
	}
	for la := range helium.Attributes(live) {
		if _, ok := vc.assessedAttrs[la]; !ok {
			continue
		}
		for ca := range helium.Attributes(copied) {
			if ca.LocalName() == la.LocalName() && ca.URI() == la.URI() {
				vc.assessedAttrs[ca] = struct{}{}
				break
			}
		}
	}
}
