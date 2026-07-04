package xsd

import (
	"context"
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// idRefOccurrence records a pending xs:IDREF token to be resolved against the
// document's collected ID values after the whole tree is walked.
type idRefOccurrence struct {
	value string
	elem  *helium.Element
	attr  string // attribute display name, or "" for element content
}

// idCollector accumulates ID values (and their owning element) and pending IDREF
// references during a document-wide xs:ID/xs:IDREF/xs:IDREFS validation pass.
type idCollector struct {
	// ids maps each collected xs:ID value to the element it identifies. In XSD 1.1
	// the SAME ID value may appear more than once as long as every occurrence
	// identifies the SAME element (e.g. two ID attributes of one element, or two
	// <id> children of one parent), so a repeat is a duplicate only when the owning
	// element differs; in XSD 1.0 any repeat is a duplicate (recordID gates the
	// relaxation on Version11).
	ids   map[string]helium.Node
	refs  []idRefOccurrence
	valid bool
}

// idOwner returns the element an ID value identifies. For an element-content ID
// (elementContent true) that is the element's PARENT (the element bearing the ID
// child); for an attribute ID it is the bearing element itself.
//
// An element-content ID on the DOCUMENT ROOT has no parent element, so it denotes
// NO element (XSD §3.3.4: an element-content ID identifies its parent). idOwner
// returns nil in that case and recordID skips it, so the value is never entered
// into the ID/IDREF table and any xs:IDREF to it dangles (W3C idIDREF
// s3_3_4ii26/ii27 — "ID on root does not denote any element").
func idOwner(elem *helium.Element, elementContent bool) helium.Node {
	if !elementContent {
		return elem
	}
	if parent, ok := elem.Parent().(*helium.Element); ok {
		return parent
	}
	return nil
}

// validateIDIDREF performs document-wide xs:ID / xs:IDREF / xs:IDREFS validation
// (XSD §3.3.4 cvc-id.1/cvc-id.2): every xs:ID value must be unique across the
// document (except across multiple ID attributes of one element), and every
// xs:IDREF token must match some xs:ID value.
//
// It runs in BOTH XSD 1.0 and 1.1 (cvc-id is version-independent). The 1.1
// relaxation allowing the same ID value to recur as long as it identifies the
// SAME element is applied in recordID under Version11 only; XSD 1.0 keeps the
// strict document-wide uniqueness (at most one ID per element). Values whose type
// is a list and/or union are decomposed to their atomic ID/IDREF leaves
// (mirroring canonicalValueKey), so e.g. a list of union(xs:ID, xs:integer)
// contributes each ID item.
func (vc *validationContext) validateIDIDREF(ctx context.Context, doc *helium.Document) bool {
	col := &idCollector{ids: make(map[string]helium.Node), valid: true}

	if err := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		td := vc.elementTypeForID(elem)

		// Element simple content typed as ID/IDREF (including via list/union). An
		// empty element with a default/fixed value constraint contributes that
		// value — an XSD 1.1 relaxation lets ID-typed elements carry a default
		// (e.g. two empty <id/> elements both defaulting to "p1" collide).
		//
		// The owner of an element-content ID — the element the ID identifies for
		// uniqueness purposes — is the PARENT element, not the typed element
		// itself. An xs:ID value identifies the element that BEARS it; an attribute
		// bears it on its owning element, and an element of type ID bears it on its
		// containing (parent) element. So two <id> siblings of one parent carrying
		// the same value (or an attribute ID and an <id> child of one element)
		// identify the SAME element and are not a duplicate, whereas the same value
		// reaching two different parents is (saxonData/Id id003, id004).
		// A CONFIRMED nilled element — one DECLARED nillable carrying
		// xsi:nil="true" — has NO element value, so its declared default/fixed must
		// NOT be substituted as an ID/IDREF (that would fabricate a duplicate ID or
		// a dangling IDREF and false-reject a valid document). Skip its
		// element-content collection. The check is by DECLARATION, not raw xsi:nil:
		// a processContents="lax" element with no declaration but a resolvable
		// xsi:type is NOT validly nilled (xsi:nil requires a nillable declaration) —
		// assessLaxElement validated its real content, so its xs:ID/xs:IDREF value
		// must still be collected. Attribute IDs always apply (handled below).
		// Only collect from genuinely-valid simple content. Simple content forbids
		// CHILD ELEMENTS; if the element has any, pass 1 already rejected it
		// structurally and there is no valid simple value here — `elemTextContent`
		// would ignore the children (and a default/fixed would be substituted for a
		// non-empty element), fabricating an ID/IDREF that never existed. Skipping
		// such elements avoids piling a spurious duplicate/dangling on top of the
		// real structural error.
		if td != nil && td.ContentType == ContentTypeSimple && idFamilyType(td) && !hasChildElement(elem) {
			hostDecl := vc.idcHostDecl(elem)
			if hostDecl == nil || !hostDecl.Nillable || !isXsiNilTrue(elem) {
				raw := elemTextContent(elem)
				// A default/fixed value is only the element's value when the content is
				// genuinely empty (no text, no children — children already excluded
				// above).
				if raw == "" && hostDecl != nil {
					if hostDecl.Fixed != nil {
						raw = *hostDecl.Fixed
					} else if hostDecl.Default != nil {
						raw = *hostDecl.Default
					}
				}
				vc.collectIDFromValue(ctx, col, td, raw, idOwner(elem, true), elem, elem, "")
			}
		}

		// Attributes typed as ID/IDREF (including via list/union). An attribute ID
		// is owned by its bearing element.
		idAttrCount := 0
		for _, a := range elem.Attributes() {
			if vc.isSpecialAttr(a) {
				continue
			}
			atd := vc.attrTypeForID(a)
			if atd == nil {
				continue
			}
			// XSD 1.0: an element may bear at most one attribute whose type is (or
			// derives from) xs:ID. XSD 1.1 removed this cap, so multiple ID attributes
			// are legal there — counted and enforced only under Version10.
			if builtinBaseLocal(atd) == "ID" {
				idAttrCount++
			}
			if !idFamilyType(atd) {
				continue
			}
			vc.collectIDFromValue(ctx, col, atd, a.Value(), elem, a, elem, attrDisplayName(a))
		}
		if vc.version == Version10 && idAttrCount > 1 {
			col.valid = false
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
				"An element may have at most one attribute of type ID.")
		}
		return nil
	})); err != nil {
		// A tree cycle (ErrWalkCycle) leaves the walk partial; the document
		// cannot be certified valid.
		col.valid = false
	}

	// Resolve all collected references now that every ID value is known.
	for _, r := range col.refs {
		if _, ok := col.ids[r.value]; ok {
			continue
		}
		col.valid = false
		msg := fmt.Sprintf("There is no ID/IDREF binding for the IDREF '%s'.", r.value)
		if r.attr != "" {
			msg = fmt.Sprintf("There is no ID/IDREF binding for the IDREF '%s' (attribute '%s').", r.value, r.attr)
		}
		vc.reportValidityError(ctx, vc.filename, r.elem.Line(), elemDisplayName(r.elem), msg)
	}
	return col.valid
}

// idFamilyType reports whether td involves xs:ID or xs:IDREF anywhere in its
// variety structure, so the (more expensive) recursive decomposition is only run
// for types that can actually contribute ID/IDREF values.
func idFamilyType(td *TypeDef) bool {
	if td == nil {
		return false
	}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return idFamilyType(resolveItemType(td))
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), idFamilyType)
	default:
		switch builtinBaseLocal(td) {
		case "ID", lexicon.TypeIDREF, lexicon.TypeIDREFS:
			return true
		}
		return false
	}
}

// collectIDFromValue decomposes raw against td's variety, recording xs:ID values
// and queuing xs:IDREF references at the atomic ID/IDREF leaves. fieldNode
// supplies namespace context for union active-member resolution. List values are
// split and each item recursed; a union value is resolved to its active member.
func (vc *validationContext) collectIDFromValue(ctx context.Context, col *idCollector, td *TypeDef, raw string, owner helium.Node, fieldNode helium.Node, elem *helium.Element, attr string) {
	switch resolveVariety(td) {
	case TypeVarietyList:
		item := resolveItemType(td)
		if item == nil {
			return
		}
		for _, f := range value.XSDFields(raw) {
			vc.collectIDFromValue(ctx, col, item, f, owner, fieldNode, elem, attr)
		}
	case TypeVarietyUnion:
		if m := unionActiveMember(ctx, raw, fieldNode, td); m != nil {
			vc.collectIDFromValue(ctx, col, m, raw, owner, fieldNode, elem, attr)
		}
	default:
		switch builtinBaseLocal(td) {
		case "ID":
			vc.recordID(ctx, col, normalizeWhiteSpace(raw, "collapse"), owner, elem, attr)
		case lexicon.TypeIDREF:
			vc.recordIDRef(col, normalizeWhiteSpace(raw, "collapse"), elem, attr)
		case lexicon.TypeIDREFS:
			// The built-in xs:IDREFS is registered as a flat atomic placeholder
			// (no ItemType), so its multiple-token nature is handled here by name.
			for _, f := range value.XSDFields(raw) {
				vc.recordIDRef(col, f, elem, attr)
			}
		}
	}
}

// recordID registers an xs:ID value, flagging a duplicate when the same value is
// already bound to a DIFFERENT owning element.
func (vc *validationContext) recordID(ctx context.Context, col *idCollector, tok string, owner helium.Node, elem *helium.Element, attr string) {
	if tok == "" {
		return
	}
	// A nil owner means the ID denotes no element (an element-content ID on the
	// document root). Such an ID is not entered into the table, so any IDREF to it
	// dangles. Skip it without recording.
	if owner == nil {
		return
	}
	prev, seen := col.ids[tok]
	if seen {
		// XSD 1.1 relaxation: the same ID value may recur as long as every
		// occurrence identifies the SAME element (e.g. two ID attributes of one
		// element, or two <id> children of one parent). XSD 1.0 has no such
		// relaxation — an ID value must be unique across the whole document (at
		// most one ID per element), so a repeat is always a duplicate.
		if vc.version == Version11 && prev == owner {
			return
		}
		col.valid = false
		msg := fmt.Sprintf("Duplicate key-sequence; the ID value '%s' is already defined elsewhere in the document.", tok)
		if attr != "" {
			msg = fmt.Sprintf("Duplicate key-sequence; the ID value '%s' (attribute '%s') is already defined elsewhere in the document.", tok, attr)
		}
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return
	}
	col.ids[tok] = owner
}

// recordIDRef queues an xs:IDREF token for resolution after the full document
// walk completes.
func (vc *validationContext) recordIDRef(col *idCollector, tok string, elem *helium.Element, attr string) {
	if tok == "" {
		return
	}
	col.refs = append(col.refs, idRefOccurrence{value: tok, elem: elem, attr: attr})
}

// elementTypeForID resolves the effective type of an instance element for the
// purpose of ID/IDREF collection, using assessedElemType as the SOLE source: the
// type recorded at a genuine pass-1 ASSESSMENT site (the validation root, a
// content-model particle match whose content was actually validated, an
// xs:anyType/lax child WITH a global declaration, or assessLaxElement), all
// post-xsi:type. It deliberately consults NEITHER actualElemType — also populated
// for processContents="skip"/lax-no-declaration subtrees purely for pass-2 IDC
// canonicalization — NOR actualElemDecl, which recordElemDecl writes as soon as a
// child MATCHES a particle, BEFORE content validation (so a particle that fails
// early, e.g. an unsatisfied minOccurs, would otherwise leave a matched-but-
// UNASSESSED child wrongly classified as ID/IDREF and report a spurious
// duplicate/dangling). It also never falls back to a global element-declaration
// lookup, for the same reason. The host declaration (default/fixed/nillable
// metadata) is consulted by the caller only AFTER this returns a non-nil
// (assessed) type.
func (vc *validationContext) elementTypeForID(elem *helium.Element) *TypeDef {
	return vc.assessedElemType[elem]
}

// attrTypeForID resolves the declared type of an instance attribute for ID/IDREF
// collection, using ONLY the type recorded for an attribute that was actually
// schema-assessed during pass-1 (explicit use, or strict/lax wildcard with a
// matching global declaration, or an inserted default/fixed). An attribute
// admitted by a processContents="skip" wildcard is absent from this map and so
// is never treated as xs:ID/xs:IDREF.
func (vc *validationContext) attrTypeForID(a *helium.Attribute) *TypeDef {
	return vc.actualAttrType[a]
}

// isXsiNilTrue reports whether elem carries xsi:nil with a true value ("true" or
// "1", after whitespace collapse), mirroring checkXsiNil's true-detection without
// reporting any error (an invalid xsi:nil lexical was already diagnosed in pass-1
// and is treated here as not-nilled). A nilled element has no element value, so
// the ID/IDREF pass must not substitute its default/fixed as element content.
func isXsiNilTrue(elem *helium.Element) bool {
	for _, a := range elem.Attributes() {
		if a.URI() != lexicon.NamespaceXSI || a.LocalName() != attrNil {
			continue
		}
		switch normalizeWhiteSpace(a.Value(), "collapse") {
		case "true", "1":
			return true
		}
		return false
	}
	return false
}

// hasChildElement reports whether elem has any child ELEMENT node. Simple content
// forbids child elements; when one is present, pass 1 already rejected the element
// structurally, so the ID/IDREF pass must not treat its (children-ignoring) text
// or a substituted default/fixed as a valid simple ID/IDREF value.
func hasChildElement(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			return true
		}
	}
	return false
}
