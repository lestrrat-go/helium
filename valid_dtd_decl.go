package helium

import (
	"context"
	"slices"

	"github.com/lestrrat-go/helium/enum"
)

// dtdSubsets returns the internal and external subsets to validate, in order.
// Unlike docDTDs it does NOT honor standalone="yes": the declaration-consistency
// VCs apply to every declaration the DTD contains regardless of the standalone
// declaration (libxml2's xmlValidateDtdFinal scans both subsets unconditionally).
func dtdSubsets(doc *Document) []*DTD {
	var dtds []*DTD
	if doc.intSubset != nil {
		dtds = append(dtds, doc.intSubset)
	}
	if doc.extSubset != nil {
		dtds = append(dtds, doc.extSubset)
	}
	return dtds
}

// notationDeclared reports whether a notation named name is declared in either
// subset. Notation lookup is standalone-independent (libxml2:
// xmlValidateNotationUse scans intSubset then extSubset).
func notationDeclared(doc *Document, name string) bool {
	if doc.intSubset != nil {
		if _, ok := doc.intSubset.LookupNotation(name); ok {
			return true
		}
	}
	if doc.extSubset != nil {
		if _, ok := doc.extSubset.LookupNotation(name); ok {
			return true
		}
	}
	return false
}

// validateDTDDeclarations validates the DTD declarations themselves — as opposed
// to the instance tree — against the XML 1.0 validity constraints libxml2 checks
// in xmlValidateElementDecl / xmlValidateAttributeDecl / xmlValidateDtdFinal. It
// runs only under ValidateDTD(true), after the no-DTD guard in validateDocument,
// and reports:
//
//   - No Duplicate Types (§3.2.2): a Mixed content model may not name the same
//     element type twice.
//   - Attribute Default Legal (§3.3.2): the default value of an enumerated or
//     NOTATION attribute must be one of the declared tokens.
//   - ID Attribute Default (§3.3.1): an ID attribute's default must be #IMPLIED
//     or #REQUIRED (never a value or #FIXED).
//   - One ID per Element Type (§3.3.1): an element type may declare at most one
//     ID attribute.
//   - Notation Declared (§4.7): a notation named in a NOTATION attribute's
//     enumeration, or in an unparsed entity's NDATA clause, must be declared.
func validateDTDDeclarations(ctx context.Context, doc *Document, vctx *validCtx) {
	subsets := dtdSubsets(doc)

	for _, dtd := range subsets {
		for _, edecl := range dtd.elements {
			validateNoDuplicateTypes(ctx, edecl, vctx)
		}
		for _, adecl := range dtd.attributes {
			validateAttributeDeclLegal(ctx, adecl, vctx)
			validateNotationEnumDeclared(ctx, doc, adecl, vctx)
			validateNotationNotOnEmptyElement(ctx, doc, adecl, vctx)
		}
		for name, ent := range dtd.entities {
			validateUnparsedEntityNotation(ctx, doc, name, ent, vctx)
		}
	}

	validateOneIDPerElement(ctx, subsets, vctx)
}

// validateNoDuplicateTypes implements the No Duplicate Types VC (§3.2.2): a Mixed
// content model (#PCDATA|a|b|...)* must not name the same element type twice.
func validateNoDuplicateTypes(ctx context.Context, edecl *ElementDecl, vctx *validCtx) {
	if edecl.decltype != enum.MixedElementType {
		return
	}
	var leaves []*ElementContent
	collectMixedLeaves(edecl.content, &leaves)

	seen := make(map[string]struct{}, len(leaves))
	for _, leaf := range leaves {
		key := leaf.prefix + ":" + leaf.name
		if _, dup := seen[key]; dup {
			name := leaf.name
			if leaf.prefix != "" {
				name = leaf.prefix + ":" + leaf.name
			}
			vctx.addf(ctx, "element %s: definition has duplicate references of %s in mixed content", edecl.name, name)
			return
		}
		seen[key] = struct{}{}
	}
}

// collectMixedLeaves appends every named-element leaf of a mixed content tree to
// out, in document order.
func collectMixedLeaves(content *ElementContent, out *[]*ElementContent) {
	if content == nil {
		return
	}
	if content.ctype == ElementContentElement {
		*out = append(*out, content)
		return
	}
	collectMixedLeaves(content.c1, out)
	collectMixedLeaves(content.c2, out)
}

// validateAttributeDeclLegal implements the declaration-time attribute VCs:
//
//   - ID Attribute Default (§3.3.1): an ID attribute's declared default must be
//     #IMPLIED or #REQUIRED.
//   - Attribute Default Value Syntactically Correct (§3.3.2): a declared default
//     must satisfy the attribute's declared (tokenized) type.
//   - Attribute Default Legal (§3.3.2): an enumerated or NOTATION attribute's
//     default value must be one of the declared tokens.
func validateAttributeDeclLegal(ctx context.Context, adecl *AttributeDecl, vctx *validCtx) {
	if idAttrDefaultInvalid(adecl.atype, adecl.def) {
		vctx.addf(ctx, "element %s: ID attribute %s must be declared #IMPLIED or #REQUIRED", adecl.elem, adecl.name)
	}

	// The default-value checks run whenever a default VALUE is declared — a bare
	// default or a #FIXED value — not merely when the value is non-empty: a
	// literal empty default `""` is still a default (e.g. an empty IDREF/NMTOKEN
	// default is syntactically invalid). #IMPLIED/#REQUIRED carry no default
	// value, so presence is decided by the DefaultDecl kind, NOT by defvalue != ""
	// (helium collapses "no default" and an empty default to the same empty
	// string, and its parse-time syntactic check skips an empty default).
	if !attrHasDefaultValue(adecl.def) {
		return
	}

	// The default must satisfy the declared type for EVERY tokenized type
	// (ID/IDREF/IDREFS/ENTITY/ENTITIES/NMTOKEN/NMTOKENS); CDATA accepts anything.
	if err := validateAttributeValueInternal(nil, adecl.atype, adecl.defvalue); err != nil {
		vctx.addf(ctx, "element %s: default value %q for attribute %s is not valid: %s", adecl.elem, adecl.defvalue, adecl.name, err)
	}

	// Attribute Default Legal: an enumerated/NOTATION default must be one of the
	// declared tokens.
	if len(adecl.tree) == 0 {
		return
	}
	switch adecl.atype {
	case enum.AttrEnumeration, enum.AttrNotation:
		if !slices.Contains(adecl.tree, adecl.defvalue) {
			vctx.addf(ctx, "element %s: default value %q for attribute %s is not among the enumerated set", adecl.elem, adecl.defvalue, adecl.name)
		}
	}
}

// attrHasDefaultValue reports whether a DefaultDecl carries an actual default
// value (a bare default or #FIXED value) as opposed to #IMPLIED/#REQUIRED which
// carry none.
func attrHasDefaultValue(def enum.AttributeDefault) bool {
	return def == enum.AttrDefaultNone || def == enum.AttrDefaultFixed
}

// idAttrDefaultInvalid reports whether an ID attribute's DefaultDecl violates the
// ID Attribute Default VC (§3.3.1): an attribute of type ID must be declared
// #IMPLIED or #REQUIRED (never a bare default or #FIXED). Shared by the
// declaration-time validator (validateAttributeDeclLegal) and the public
// DTD.AddAttributeDecl input check so the two cannot drift.
func idAttrDefaultInvalid(atype enum.AttributeType, def enum.AttributeDefault) bool {
	return atype == enum.AttrID &&
		def != enum.AttrDefaultImplied &&
		def != enum.AttrDefaultRequired
}

// validateNotationEnumDeclared implements the Notation Declared VC (§4.7) for a
// NOTATION attribute: every notation name listed in the attribute's enumeration
// must be declared.
func validateNotationEnumDeclared(ctx context.Context, doc *Document, adecl *AttributeDecl, vctx *validCtx) {
	if adecl.atype != enum.AttrNotation {
		return
	}
	for _, nname := range adecl.tree {
		if !notationDeclared(doc, nname) {
			vctx.addf(ctx, "element %s: attribute %s enumerates undeclared notation %q", adecl.elem, adecl.name, nname)
		}
	}
}

// elementDeclForAttr looks up the declaration of the element that owns adecl,
// searching both subsets. It returns nil when the element is undeclared or only
// forward-referenced (UndefinedElementType).
func elementDeclForAttr(doc *Document, elemName string) *ElementDecl {
	for _, dtd := range dtdSubsets(doc) {
		if e, ok := dtd.GetElementDesc(elemName); ok && e.decltype != enum.UndefinedElementType {
			return e
		}
	}
	return nil
}

// validateNotationNotOnEmptyElement implements the No Notation on Empty Element
// VC (§3.3.1): an attribute of type NOTATION must not be declared on an element
// whose content type is EMPTY.
func validateNotationNotOnEmptyElement(ctx context.Context, doc *Document, adecl *AttributeDecl, vctx *validCtx) {
	if adecl.atype != enum.AttrNotation {
		return
	}
	edecl := elementDeclForAttr(doc, adecl.elem)
	if edecl != nil && edecl.decltype == enum.EmptyElementType {
		vctx.addf(ctx, "element %s: NOTATION attribute %s is not allowed on an EMPTY element", adecl.elem, adecl.name)
	}
}

// validateUnparsedEntityNotation implements the Notation Declared VC (§4.7) for
// an unparsed entity: the notation named in its NDATA clause must be declared.
// For an unparsed entity the notation name is stored as the entity content.
func validateUnparsedEntityNotation(ctx context.Context, doc *Document, name string, ent *Entity, vctx *validCtx) {
	if ent.EntityType() != enum.ExternalGeneralUnparsedEntity {
		return
	}
	notation := string(ent.Content())
	if notation != "" && !notationDeclared(doc, notation) {
		vctx.addf(ctx, "entity %s: NDATA notation %q is not declared", name, notation)
	}
}

// validateOneIDPerElement implements the One ID per Element Type VC (§3.3.1): an
// element type may declare at most one attribute of type ID. IDs declared in the
// internal and external subsets for the same element type are counted together.
func validateOneIDPerElement(ctx context.Context, subsets []*DTD, vctx *validCtx) {
	counts := make(map[string]int)
	for _, dtd := range subsets {
		for _, adecl := range dtd.attributes {
			if adecl.atype == enum.AttrID {
				counts[adecl.elem]++
			}
		}
	}
	for elem, n := range counts {
		if n > 1 {
			vctx.addf(ctx, "element %s has more than one ID attribute defined", elem)
		}
	}
}
