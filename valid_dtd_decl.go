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

// validateAttributeDeclLegal implements two declaration-time attribute VCs:
//
//   - ID Attribute Default (§3.3.1): an ID attribute's declared default must be
//     #IMPLIED or #REQUIRED.
//   - Attribute Default Legal (§3.3.2): an enumerated or NOTATION attribute's
//     default value must be one of the declared tokens.
func validateAttributeDeclLegal(ctx context.Context, adecl *AttributeDecl, vctx *validCtx) {
	if adecl.atype == enum.AttrID &&
		adecl.def != enum.AttrDefaultImplied &&
		adecl.def != enum.AttrDefaultRequired {
		vctx.addf(ctx, "element %s: ID attribute %s must be declared #IMPLIED or #REQUIRED", adecl.elem, adecl.name)
	}

	if adecl.defvalue == "" || len(adecl.tree) == 0 {
		return
	}
	switch adecl.atype {
	case enum.AttrEnumeration, enum.AttrNotation:
		if !slices.Contains(adecl.tree, adecl.defvalue) {
			vctx.addf(ctx, "element %s: default value %q for attribute %s is not among the enumerated set", adecl.elem, adecl.defvalue, adecl.name)
		}
	}
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
