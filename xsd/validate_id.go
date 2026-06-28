package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
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
	// ids maps each collected xs:ID value to the element it identifies. XSD 1.1
	// allows the SAME ID value to appear more than once as long as every
	// occurrence identifies the SAME element (e.g. two ID attributes of one
	// element, or two <id> children of one parent), so a repeat is a duplicate
	// only when the owning element differs.
	ids   map[string]helium.Node
	refs  []idRefOccurrence
	valid bool
}

// idOwner returns the element an ID value identifies. For an element-content ID
// (elementContent true) that is the element's PARENT (the element bearing the ID
// child); for an attribute ID it is the bearing element itself. A typed element
// with no element parent owns its own ID.
func idOwner(elem *helium.Element, elementContent bool) helium.Node {
	if !elementContent {
		return elem
	}
	if parent, ok := elem.Parent().(*helium.Element); ok {
		return parent
	}
	return elem
}

// validateIDIDREF performs document-wide xs:ID / xs:IDREF / xs:IDREFS validation
// (XSD §3.3.4 cvc-id.1/cvc-id.2): every xs:ID value must be unique across the
// document (except across multiple ID attributes of one element), and every
// xs:IDREF token must match some xs:ID value.
//
// It is gated to XSD 1.1 mode by the caller, so XSD 1.0 stays byte-identical
// (helium does not enforce these datatype constraints in 1.0, and the
// libxml2-compat goldens depend on that). The 1.1 relaxation allowing an element
// type to carry more than one xs:ID attribute is handled implicitly: every
// ID-typed attribute contributes to the same document-wide table with no
// per-element cap. Values whose type is a list and/or union are decomposed to
// their atomic ID/IDREF leaves (mirroring canonicalValueKey), so e.g. a list of
// union(xs:ID, xs:integer) contributes each ID item.
func (vc *validationContext) validateIDIDREF(ctx context.Context, doc *helium.Document) bool {
	col := &idCollector{ids: make(map[string]helium.Node), valid: true}

	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
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
		if td != nil && td.ContentType == ContentTypeSimple && idFamilyType(td) {
			raw := elemTextContent(elem)
			if raw == "" {
				if decl := vc.idcHostDecl(elem); decl != nil {
					if decl.Fixed != nil {
						raw = *decl.Fixed
					} else if decl.Default != nil {
						raw = *decl.Default
					}
				}
			}
			vc.collectIDFromValue(ctx, col, td, raw, idOwner(elem, true), elem, elem, "")
		}

		// Attributes typed as ID/IDREF (including via list/union). An attribute ID
		// is owned by its bearing element.
		for _, a := range elem.Attributes() {
			if isSpecialAttr(a) {
				continue
			}
			atd := vc.attrTypeForID(td, a)
			if atd == nil || !idFamilyType(atd) {
				continue
			}
			vc.collectIDFromValue(ctx, col, atd, a.Value(), elem, a, elem, attrDisplayName(a))
		}
		return nil
	}))

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
		for _, m := range resolveUnionMembers(td) {
			if idFamilyType(m) {
				return true
			}
		}
		return false
	default:
		switch builtinBaseLocal(td) {
		case "ID", "IDREF", "IDREFS":
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
		case "IDREF":
			vc.recordIDRef(col, normalizeWhiteSpace(raw, "collapse"), elem, attr)
		case "IDREFS":
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
	prev, seen := col.ids[tok]
	if seen {
		if prev == owner {
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
// purpose of ID/IDREF collection, preferring the actual type recorded during
// pass-1 content validation (which already accounts for xsi:type).
func (vc *validationContext) elementTypeForID(elem *helium.Element) *TypeDef {
	if td := vc.actualElemType[elem]; td != nil {
		return td
	}
	if decl := vc.actualElemDecl[elem]; decl != nil && decl.Type != nil {
		return decl.Type
	}
	if decl := lookupElemDecl(elem, vc.schema); decl != nil {
		return decl.Type
	}
	return nil
}

// attrTypeForID resolves the declared type of an instance attribute for ID/IDREF
// collection: the owning element's complex-type attribute uses first, then a
// matching global attribute declaration (which covers attributes admitted via an
// xs:anyAttribute wildcard).
func (vc *validationContext) attrTypeForID(elemType *TypeDef, a *helium.Attribute) *TypeDef {
	aqn := QName{Local: a.LocalName(), NS: a.URI()}
	if elemType != nil {
		if td := attrUseType(elemType, aqn, vc.schema); td != nil {
			return td
		}
	}
	if ga, ok := vc.schema.globalAttrs[aqn]; ok {
		return attrUseTypeDef(ga, vc.schema)
	}
	return nil
}
