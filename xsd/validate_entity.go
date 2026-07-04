package xsd

import (
	"context"
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// validateEntities performs document-wide xs:ENTITY / xs:ENTITIES value-space
// validation (cvc-id / XSD §3.3.11): every xs:ENTITY value must name an UNPARSED
// general entity declared in the instance document's DTD (an external general
// unparsed entity, i.e. one declared with NDATA). Lexical validity (the value is
// an NCName) is already enforced in pass-1; this pass adds the value-space
// referential check that the named entity actually exists and is unparsed.
//
// It mirrors validateIDIDREF's structure (a document-wide walk over element
// simple content and attributes, with the same default/fixed substitution,
// nilled-element, child-element, and list/union decomposition rules) but needs no
// cross-element table: an ENTITY reference resolves against the DTD entity table
// (doc.GetEntity), which is fully known up front, so each token is checked in
// place. It is gated to XSD 1.1 mode by the caller, so XSD 1.0 stays
// byte-identical (helium does not enforce this datatype constraint in 1.0, and
// the libxml2-compat goldens depend on that).
func (vc *validationContext) validateEntities(ctx context.Context, doc *helium.Document) bool {
	valid := true

	if err := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		td := vc.elementTypeForID(elem)

		// Element simple content typed as ENTITY/ENTITIES (including via
		// list/union). The default/fixed substitution, nilled-element, and
		// child-element guards are identical to validateIDIDREF: an empty element
		// contributes its default/fixed value, a confirmed nilled element
		// contributes nothing, and an element with child elements is left to pass 1's
		// structural rejection (see validateIDIDREF for the full rationale).
		if td != nil && td.ContentType == ContentTypeSimple && entityFamilyType(td) && !hasChildElement(elem) {
			hostDecl := vc.idcHostDecl(elem)
			if hostDecl == nil || !hostDecl.Nillable || !isXsiNilTrue(elem) {
				raw := elemTextContent(elem)
				if raw == "" && hostDecl != nil {
					if hostDecl.Fixed != nil {
						raw = *hostDecl.Fixed
					} else if hostDecl.Default != nil {
						raw = *hostDecl.Default
					}
				}
				if !vc.checkEntityValue(ctx, doc, td, raw, elem, elem, "") {
					valid = false
				}
			}
		}

		// Attributes typed as ENTITY/ENTITIES (including via list/union). Default
		// attributes are already inserted into the live tree before this pass.
		for _, a := range elem.Attributes() {
			if vc.isSpecialAttr(a) {
				continue
			}
			atd := vc.attrTypeForID(a)
			if atd == nil || !entityFamilyType(atd) {
				continue
			}
			if !vc.checkEntityValue(ctx, doc, atd, a.Value(), a, elem, attrDisplayName(a)) {
				valid = false
			}
		}
		return nil
	})); err != nil {
		// A tree cycle (ErrWalkCycle) leaves the walk partial; the document
		// cannot be certified valid.
		valid = false
	}

	return valid
}

// entityFamilyType reports whether td involves xs:ENTITY or xs:ENTITIES anywhere
// in its variety structure, so the (more expensive) recursive decomposition is
// only run for types that can actually contribute ENTITY values. Mirrors
// idFamilyType.
func entityFamilyType(td *TypeDef) bool {
	if td == nil {
		return false
	}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return entityFamilyType(resolveItemType(td))
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), entityFamilyType)
	default:
		switch builtinBaseLocal(td) {
		case lexicon.TypeENTITY, lexicon.TypeENTITIES:
			return true
		}
		return false
	}
}

// checkEntityValue decomposes raw against td's variety, checking each atomic
// xs:ENTITY leaf against the document's declared unparsed entities. fieldNode
// supplies namespace context for union active-member resolution. List values are
// split and each item recursed; a union value is resolved to its active member.
// Mirrors collectIDFromValue. It returns false (and reports a validity error)
// when any leaf names an entity that is not a declared external unparsed entity.
func (vc *validationContext) checkEntityValue(ctx context.Context, doc *helium.Document, td *TypeDef, raw string, fieldNode helium.Node, elem *helium.Element, attr string) bool {
	switch resolveVariety(td) {
	case TypeVarietyList:
		item := resolveItemType(td)
		if item == nil {
			return true
		}
		valid := true
		for _, f := range value.XSDFields(raw) {
			if !vc.checkEntityValue(ctx, doc, item, f, fieldNode, elem, attr) {
				valid = false
			}
		}
		return valid
	case TypeVarietyUnion:
		m := unionActiveMember(ctx, raw, fieldNode, td)
		if m == nil {
			return true
		}
		return vc.checkEntityValue(ctx, doc, m, raw, fieldNode, elem, attr)
	default:
		switch builtinBaseLocal(td) {
		case lexicon.TypeENTITY:
			return vc.checkEntityToken(ctx, doc, normalizeWhiteSpace(raw, "collapse"), elem, attr)
		case lexicon.TypeENTITIES:
			// The built-in xs:ENTITIES is registered as a flat atomic placeholder
			// (no ItemType), so its multiple-token nature is handled here by name.
			valid := true
			for _, f := range value.XSDFields(raw) {
				if !vc.checkEntityToken(ctx, doc, f, elem, attr) {
					valid = false
				}
			}
			return valid
		}
		return true
	}
}

// checkEntityToken verifies that tok names an external general unparsed entity
// declared in the document's DTD (XSD §3.3.11 cvc-id). An empty token is ignored
// (pass 1 already diagnosed any lexical problem). On failure it reports a
// validity error and returns false.
func (vc *validationContext) checkEntityToken(ctx context.Context, doc *helium.Document, tok string, elem *helium.Element, attr string) bool {
	if tok == "" {
		return true
	}
	ent, found := doc.GetEntity(tok)
	if found && ent != nil && ent.EntityType() == enum.ExternalGeneralUnparsedEntity {
		return true
	}
	msg := fmt.Sprintf("There is no unparsed entity declared for the ENTITY value '%s'.", tok)
	if attr != "" {
		msg = fmt.Sprintf("There is no unparsed entity declared for the ENTITY value '%s' (attribute '%s').", tok, attr)
	}
	vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
	return false
}
