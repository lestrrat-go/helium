package xsd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func (c *compiler) resolveRefs(ctx context.Context) {
	// Resolve element type references.
	// Two passes: the first pass resolves type-name refs and may leave
	// element-to-element refs with nil Type (because the target global element
	// hasn't had its own type resolved yet). The second pass picks those up.
	for range 2 {
		for edecl, qn := range c.elemRefs {
			if edecl.Type != nil {
				continue
			}
			// First check if this is a reference to a global element.
			// Skip self-referencing elements (where the element name matches
			// the type name, e.g., <xs:element name="X" type="X"/>); these
			// should resolve against the type map instead.
			if ge, ok := c.schema.elements[qn]; ok && ge != edecl {
				edecl.Type = ge.Type
				if edecl.Default == nil {
					edecl.Default = ge.Default
				}
				if edecl.Fixed == nil {
					edecl.Fixed = ge.Fixed
					edecl.FixedNS = ge.FixedNS
				}
				// Copy the referenced declaration's substitution-group
				// affiliation. A no-type substitution-group member leaves
				// edecl.Type nil here; without the affiliation, effectiveDeclType
				// cannot walk to the typed head, so xsi:nil lexical and
				// nilled-empty checks would be silently skipped for a direct
				// ref="member". The member's own Nillable (copied below) still
				// governs the nilled-element check.
				if edecl.SubstitutionGroup == (QName{}) {
					edecl.SubstitutionGroup = ge.SubstitutionGroup
				}
				edecl.Nillable = ge.Nillable
				edecl.Abstract = ge.Abstract
				if !edecl.BlockSet {
					edecl.Block = ge.Block
					edecl.BlockSet = ge.BlockSet
				}
				if !edecl.FinalSet {
					edecl.Final = ge.Final
					edecl.FinalSet = ge.FinalSet
				}
				continue
			}
			// For ref elements, report unresolved element declaration error.
			if edecl.IsRef {
				if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" {
					msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) element declaration.", qn.NS, qn.Local)
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, src.line, src.elemName, elemElement, attrRef, msg), helium.ErrorLevelFatal))
					c.errorCount++
				}
				edecl.Type = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				continue
			}
			td, ok := c.schema.types[qn]
			if !ok {
				if _, eligible := c.chameleonEligible[edecl]; eligible {
					// Try empty namespace as fallback — the type may come from an
					// imported schema with no targetNamespace (chameleon include).
					td, ok = c.schema.types[QName{Local: qn.Local, NS: ""}]
				}
			}
			if !ok {
				// Report the unresolved element type — whether an XSD built-in
				// that should exist or a missing user-defined type — before
				// installing a recovery placeholder, so an invalid schema cannot
				// silently compile and validate as if the type existed.
				if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" {
					msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaElemDeclErrorAttr(c.filename, src.line, src.elemName, attrType, msg), helium.ErrorLevelFatal))
					c.errorCount++
				}
				td = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				c.schema.types[qn] = td
			}
			edecl.Type = td
		}
	}

	// Resolve base type references.
	for td, qn := range c.typeRefs {
		base, ok := c.schema.types[qn]
		if !ok {
			if _, eligible := c.chameleonEligible[td]; eligible {
				// Try empty namespace as fallback — the type may come from an
				// imported schema with no targetNamespace (chameleon include).
				base, ok = c.schema.types[QName{Local: qn.Local, NS: ""}]
			}
		}
		if !ok {
			// Report the unresolved base type before installing a recovery
			// placeholder, so an invalid schema cannot silently compile.
			c.reportUnresolvedTypeRef(ctx, td, qn)
			base = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = base
		}
		td.BaseType = base
	}

	// Resolve list item type references.
	for td, qn := range c.itemTypeRefs {
		itemTD, ok := c.schema.types[qn]
		if !ok {
			if _, eligible := c.chameleonEligible[td]; eligible {
				// Try empty namespace as fallback — the item type may come from an
				// imported schema with no targetNamespace (chameleon include).
				itemTD, ok = c.schema.types[QName{Local: qn.Local, NS: ""}]
			}
		}
		if !ok {
			c.reportUnresolvedTypeRef(ctx, td, qn)
			itemTD = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = itemTD
		}
		td.ItemType = itemTD
	}

	// Resolve union member type references.
	for _, ref := range c.unionMemberRefs {
		memberTD, ok := c.schema.types[ref.name]
		if !ok && ref.chameleonEligible {
			// Try empty namespace as fallback — the member type may come from an
			// imported schema with no targetNamespace (chameleon include).
			memberTD, ok = c.schema.types[QName{Local: ref.name.Local, NS: ""}]
		}
		if !ok {
			c.reportUnresolvedTypeRef(ctx, ref.owner, ref.name)
			memberTD = &TypeDef{Name: ref.name, ContentType: ContentTypeSimple}
			c.schema.types[ref.name] = memberTD
		}
		ref.owner.MemberTypes = append(ref.owner.MemberTypes, memberTD)
	}

	// Propagate variety and item type through restriction derivation.
	for td := range c.typeRefs {
		if td.Derivation == DerivationRestriction && td.BaseType != nil {
			if td.Variety == TypeVarietyAtomic && td.BaseType.Variety == TypeVarietyList {
				td.Variety = TypeVarietyList
				td.ItemType = td.BaseType.ItemType
			}
		}
	}

	// Propagate variety and member types through restriction derivation of union types.
	for td := range c.typeRefs {
		if td.Derivation == DerivationRestriction && td.BaseType != nil {
			if td.Variety == TypeVarietyAtomic && resolveVariety(td.BaseType) == TypeVarietyUnion {
				td.Variety = TypeVarietyUnion
				if len(td.MemberTypes) == 0 {
					td.MemberTypes = resolveUnionMembers(td.BaseType)
				}
			}
		}
	}

	// Resolve group references — replace placeholder content with actual group content.
	//
	// Collect and sort by source line so the all-group reference constraint
	// diagnostics below are emitted in a deterministic order independent of Go
	// map iteration.
	groupRefPlaceholders := make([]*ModelGroup, 0, len(c.groupRefs))
	for placeholder := range c.groupRefs {
		groupRefPlaceholders = append(groupRefPlaceholders, placeholder)
	}
	sort.Slice(groupRefPlaceholders, func(i, j int) bool {
		return c.groupRefSources[groupRefPlaceholders[i]].line < c.groupRefSources[groupRefPlaceholders[j]].line
	})
	for _, placeholder := range groupRefPlaceholders {
		qn := c.groupRefs[placeholder]
		grp, ok := c.schema.groups[qn]
		if !ok {
			continue
		}
		// Copy the group's content into the placeholder.
		placeholder.Compositor = grp.Compositor
		placeholder.Particles = grp.Particles

		// Enforce the all-group reference constraints (XSD §3.8.6 cos-all-limited
		// / §3.8.2): a reference that resolves to an 'all' model group may only
		// appear as the entire content model of a complex type (never nested in
		// another model group), and its {max occurs} must be 1.
		if grp.Compositor != CompositorAll {
			continue
		}
		c.checkAllGroupRef(ctx, placeholder)
	}

	// Resolve attribute group references.
	for td, qns := range c.attrGroupRefs {
		for _, qn := range qns {
			if attrs, ok := c.schema.attrGroups[qn]; ok {
				td.Attributes = append(td.Attributes, attrs...)
			}
		}
	}

	// Reject duplicate attribute uses within a single type's own attribute set
	// (XSD 3.4.6 ct-props-correct.4 / 3.6.6 ag-props-correct.2). After
	// attribute-group expansion a type may carry two uses with the same expanded
	// name; the validation-time map would silently coalesce them, so catch the
	// collision here. This runs BEFORE base-type attribute merging so it only
	// inspects each type's OWN declared/expanded uses — duplicates between a base
	// type and its extension are reported separately during the merge below.
	c.checkDuplicateAttrUses(ctx)

	// Resolve attribute references: copy Default/Fixed/TypeName from global attr.
	for au, qn := range c.attrRefs {
		ga, ok := c.schema.globalAttrs[qn]
		if !ok {
			continue
		}
		if au.Default == nil {
			au.Default = ga.Default
		}
		if au.Fixed == nil {
			au.Fixed = ga.Fixed
			au.FixedNS = ga.FixedNS
		}
		if au.TypeName == (QName{}) {
			au.TypeName = ga.TypeName
		}
		if au.Type == nil {
			au.Type = ga.Type
		}
	}

	// Validate attribute default/fixed constraint values against the
	// attribute's declared simple type now that all type refs are resolved.
	// A retained-but-invalid constraint (e.g. an empty default="" on an
	// xs:integer attribute) is a schema error; catching it here avoids
	// injecting an invalid value into the instance during validation.
	c.checkAttrUseConstraints(ctx)

	// Topologically order extension types so each base type is merged before
	// the types that derive from it (the merge reads the base's finalized
	// content model and attributes).
	extensionTypes := make([]*TypeDef, 0, len(c.typeRefs))
	for td := range c.typeRefs {
		if td.Derivation != DerivationExtension || td.BaseType == nil {
			continue
		}
		extensionTypes = append(extensionTypes, td)
	}
	typeDepth := make(map[*TypeDef]int, len(extensionTypes))
	var depth func(td *TypeDef) int
	depth = func(td *TypeDef) int {
		if td == nil {
			return 0
		}
		if d, ok := typeDepth[td]; ok {
			return d
		}
		typeDepth[td] = 0 // cycle guard; XSD forbids cyclic extension but defend anyway
		d := 1 + depth(td.BaseType)
		typeDepth[td] = d
		return d
	}
	// Stable sort with source-line then QName tie-breaks so error messages
	// emitted during the merge (e.g. cos-ct-extends-1-1, duplicate attribute)
	// are deterministic among equal-depth types, matching the restriction loop
	// below. Line alone is insufficient — multiple types can share a line (e.g.
	// minified schemas), so fall back to QName before the randomized map order.
	sort.SliceStable(extensionTypes, func(i, j int) bool {
		ti, tj := extensionTypes[i], extensionTypes[j]
		di, dj := depth(ti), depth(tj)
		if di != dj {
			return di < dj
		}
		li, lj := c.typeDefSources[ti].line, c.typeDefSources[tj].line
		if li != lj {
			return li < lj
		}
		if ti.Name.NS != tj.Name.NS {
			return ti.Name.NS < tj.Name.NS
		}
		return ti.Name.Local < tj.Name.Local
	})

	// Merge content models for extension types. extensionTypes is already
	// filtered to extension types with a base, so no per-item guard is needed.
	for _, td := range extensionTypes {
		if td.ContentType == ContentTypeSimple {
			// simpleContent extension — inherit attributes and wildcard from base.
			if td.BaseType.Attributes != nil {
				td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
			}
			if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = td.BaseType.AnyAttribute
			} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute)
			}
			continue
		}
		// cos-ct-extends-1-1: complexContent extension requires the base type
		// to also have complex content (mixed or element-only), not simple content.
		// Only check when the derived type has element content (not empty/attribute-only).
		if td.BaseType.ContentType == ContentTypeSimple && (td.ContentType == ContentTypeElementOnly || td.ContentType == ContentTypeMixed) {
			if src, ok := c.typeDefSources[td]; ok && c.filename != "" {
				component := componentLocalComplexType
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
					"The content type of both, the type and its base type, must either 'mixed' or 'element-only'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
			continue
		}
		baseMG := td.BaseType.ContentModel
		derivedMG := td.ContentModel
		if baseMG != nil && derivedMG != nil {
			// Merge: create a sequence of base content + derived content.
			merged := &ModelGroup{
				Compositor: CompositorSequence,
				MinOccurs:  1,
				MaxOccurs:  1,
				Particles: []*Particle{
					{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG},
					{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG},
				},
			}
			td.ContentModel = merged
		} else if baseMG != nil {
			td.ContentModel = baseMG
		}
		// Inherit content type from base if not already set.
		if td.ContentType == ContentTypeEmpty && td.BaseType.ContentType != ContentTypeEmpty {
			td.ContentType = td.BaseType.ContentType
		}
		// Check for duplicate attributes before merging base type attributes.
		if td.BaseType.Attributes != nil && td.Attributes != nil && c.filename != "" {
			baseAttrNames := make(map[string]bool, len(td.BaseType.Attributes))
			for _, au := range td.BaseType.Attributes {
				baseAttrNames[au.Name.Local] = true
			}
			for _, au := range td.Attributes {
				if baseAttrNames[au.Name.Local] {
					if src, ok := c.typeDefSources[td]; ok {
						component := componentLocalComplexType
						if !src.isLocal {
							component = td.Name.Local
						}
						msg := fmt.Sprintf("Duplicate attribute use '%s'.", au.Name.Local)
						c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
						c.errorCount++
					}
				}
			}
		}
		// Inherit attributes from base.
		if td.BaseType.Attributes != nil {
			td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
		}
		// Inherit/union anyAttribute wildcards.
		if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
			td.AnyAttribute = td.BaseType.AnyAttribute
		} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
			td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute)
		}
	}

	// Check restriction attribute compatibility.
	// Collect and sort by source line for deterministic error ordering.
	var restrictionTypes []*TypeDef
	for td := range c.typeRefs {
		if td.Derivation != DerivationRestriction || td.BaseType == nil {
			continue
		}
		restrictionTypes = append(restrictionTypes, td)
	}
	sort.Slice(restrictionTypes, func(i, j int) bool {
		si := c.typeDefSources[restrictionTypes[i]]
		sj := c.typeDefSources[restrictionTypes[j]]
		return si.line < sj.line
	})
	for _, td := range restrictionTypes {
		c.checkRestrictionAttrs(ctx, td)
	}

	// Check UPA (Unique Particle Attribution) for all complex types with content models.
	// Only run UPA if there are no prior schema errors (libxml2 skips UPA when
	// the schema has structural parse errors).
	if c.filename != "" && c.errorCount == 0 {
		for td, src := range c.typeDefSources {
			if td.ContentModel != nil {
				c.checkUPA(ctx, td, src)
			}
		}
	}
}

// checkDuplicateAttrUses reports duplicate attribute uses (by expanded QName)
// within a single complex type's own attribute set. Prohibited uses do not
// contribute an attribute use and are skipped. Types are processed in source
// line order for deterministic diagnostics.
func (c *compiler) checkDuplicateAttrUses(ctx context.Context) {
	if c.filename == "" {
		return
	}
	tds := make([]*TypeDef, 0, len(c.typeDefSources))
	for td := range c.typeDefSources {
		if len(td.Attributes) > 1 {
			tds = append(tds, td)
		}
	}
	sort.Slice(tds, func(i, j int) bool {
		return c.typeDefSources[tds[i]].line < c.typeDefSources[tds[j]].line
	})
	for _, td := range tds {
		seen := make(map[QName]bool, len(td.Attributes))
		reported := make(map[QName]bool)
		for _, au := range td.Attributes {
			if au.Prohibited {
				continue
			}
			if seen[au.Name] {
				if reported[au.Name] {
					continue
				}
				reported[au.Name] = true
				src := c.typeDefSources[td]
				component := componentLocalComplexType
				if !src.isLocal {
					component = td.Name.Local
				}
				msg := fmt.Sprintf("Duplicate attribute use '%s'.", au.Name.Local)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
				c.errorCount++
				continue
			}
			seen[au.Name] = true
		}
	}
}

// checkAllGroupRef enforces the constraints on an xs:group reference that
// resolves to an 'all' model group (XSD §3.8.6 cos-all-limited / §3.8.2):
//
//   - The reference may only appear as the entire content model of a complex
//     type, never nested inside another model group (xs:sequence / xs:choice /
//     xs:all). A nested reference is rejected.
//   - A direct (non-nested) reference's {max occurs} must be 1.
//
// The diagnostics mirror xmllint's wording and are attributed to the
// referencing xs:group element.
func (c *compiler) checkAllGroupRef(ctx context.Context, placeholder *ModelGroup) {
	if c.filename == "" {
		return
	}
	src, ok := c.groupRefSources[placeholder]
	if !ok {
		return
	}

	// A 0/0 occurrence is a prohibited particle that maps to no particle at all,
	// so the all-group constraints do not apply and the reference is valid
	// (xmllint accepts it). This applies to both direct and nested references.
	if placeholder.MinOccurs == 0 && placeholder.MaxOccurs == 0 {
		return
	}

	if src.nested {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, src.line, src.local, elemGroup,
			"A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups."), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}

	// Direct reference: {max occurs} must be 1. An absent maxOccurs defaults to
	// 1 and is fine; otherwise the lexical value must parse to exactly 1.
	if src.maxOccursRaw == "" {
		return
	}
	n, parsed := parseNonNegativeOccurs(src.maxOccursRaw, true)
	if parsed && n == 1 {
		return
	}
	// When the maxOccurs lexical value fails to parse, or it is a finite count
	// below 1 while minOccurs defaults to (or is explicitly) >= 1, the generic
	// occurrence validator already reports the lexical / ">= 1" diagnostic.
	// Emitting the all-specific "must be 1" error here would duplicate it, so
	// only flag an otherwise-valid occurrence range whose max != 1. An unbounded
	// maxOccurs is a valid lexical form that the generic validator accepts, so
	// it must still surface the all-specific error.
	if !parsed {
		return
	}
	if n != Unbounded && n < 1 && placeholder.MinOccurs >= 1 {
		return
	}
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, src.line, src.local, elemGroup,
		"The particle's {max occurs} must be 1, since the reference resolves to an 'all' model group."), helium.ErrorLevelFatal))
	c.errorCount++
}

// reportUnresolvedTypeRef reports a fatal schema parser error for a type
// reference (base type, list item type, or union member type) on owner that
// does not resolve to a type definition. The caller installs a recovery
// placeholder only after this records the error, so an invalid schema cannot
// silently compile and validate documents as if the missing type existed.
func (c *compiler) reportUnresolvedTypeRef(ctx context.Context, owner *TypeDef, qn QName) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[owner]
	if !hasSrc {
		return
	}
	component := owner.Name.Local
	if component == "" || src.isLocal {
		component = "local simple type"
	}
	msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "simpleType", component, msg), helium.ErrorLevelFatal))
	c.errorCount++
}

// checkAttrUseConstraints validates each attribute use's default/fixed value
// against its declared simple type. Reported errors are deterministic
// (ordered by source line then attribute name).
func (c *compiler) checkAttrUseConstraints(ctx context.Context) {
	if c.filename == "" {
		return
	}
	type pending struct {
		au  *AttrUse
		src attrConstraintSource
	}
	items := make([]pending, 0, len(c.attrUseConstraintSources))
	for au, src := range c.attrUseConstraintSources {
		items = append(items, pending{au: au, src: src})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].src.line != items[j].src.line {
			return items[i].src.line < items[j].src.line
		}
		return items[i].src.local < items[j].src.local
	})

	for _, it := range items {
		val := it.au.Default
		if val == nil {
			val = it.au.Fixed
		}
		if val == nil {
			continue
		}
		td := attrUseTypeDef(it.au, c.schema)
		if td == nil || td.ContentType != ContentTypeSimple {
			continue
		}
		if err := td.Validate(ctx, *val, it.src.nsMap); err != nil {
			msg := fmt.Sprintf("The value '%s' is not a valid value of the atomic type '%s'.", *val, typeDisplayName(td))
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, it.src.line, it.src.local, "attribute", it.src.local, msg), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
}

// checkRestrictionAttrs validates that a restriction-derived type's attributes
// are compatible with the base type's attribute uses.
func (c *compiler) checkRestrictionAttrs(ctx context.Context, td *TypeDef) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[td]
	if !hasSrc {
		return
	}

	component := componentLocalComplexType
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}

	baseTypeName := td.BaseType.Name.Local
	baseTypeNS := td.BaseType.Name.NS
	baseQualified := fmt.Sprintf("'{%s}%s'", baseTypeNS, baseTypeName)

	// Build map of base type's non-prohibited attributes.
	baseAttrs := make(map[string]*AttrUse, len(td.BaseType.Attributes))
	for _, au := range td.BaseType.Attributes {
		if !au.Prohibited {
			baseAttrs[au.Name.Local] = au
		}
	}

	// Check each derived non-prohibited attribute against the base.
	for _, au := range td.Attributes {
		if au.Prohibited {
			continue
		}
		baseAU, found := baseAttrs[au.Name.Local]
		if found {
			// Check use consistency: optional cannot restrict required.
			if baseAU.Required && !au.Required {
				msg := fmt.Sprintf("The 'optional' attribute use is inconsistent with the corresponding 'required' attribute use of the base complex type definition %s.", baseQualified)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg), helium.ErrorLevelFatal))
				c.errorCount++
			}
		} else if td.BaseType.AnyAttribute == nil {
			// No matching attribute and no wildcard in base.
			msg := fmt.Sprintf("Neither a matching attribute use, nor a matching wildcard exists in the base complex type definition %s.", baseQualified)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType",
				component+", attribute use '"+au.Name.Local+"'", msg), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// Check that all required base attributes have a matching non-prohibited derived attribute.
	derivedAttrs := make(map[string]*AttrUse, len(td.Attributes))
	for _, au := range td.Attributes {
		derivedAttrs[au.Name.Local] = au
	}
	for _, baseAU := range td.BaseType.Attributes {
		if !baseAU.Required {
			continue
		}
		derived, found := derivedAttrs[baseAU.Name.Local]
		if !found || derived.Prohibited {
			msg := fmt.Sprintf("A matching attribute use for the 'required' attribute use '%s' of the base complex type definition %s is missing.", baseAU.Name.Local, baseQualified)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// derivation-ok-restriction 4: Wildcard checks.
	if td.AnyAttribute != nil {
		// 4.1: Base must also have a wildcard.
		if td.BaseType.AnyAttribute == nil {
			msg := fmt.Sprintf("The complex type definition has an attribute wildcard, but the base complex type definition %s does not have one.", baseQualified)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			// 4.2: Derived namespace must be subset of base namespace.
			if !wildcardNSSubset(td.AnyAttribute, td.BaseType.AnyAttribute) {
				msg := fmt.Sprintf("The attribute wildcard is not a valid subset of the wildcard in the base complex type definition %s.", baseQualified)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
				c.errorCount++
			}
			// 4.3: Derived processContents must be >= base strength (strict > lax > skip).
			// libxml2 attributes this error to the base type's source location.
			if processContentsStrength(td.AnyAttribute.ProcessContents) < processContentsStrength(td.BaseType.AnyAttribute.ProcessContents) {
				errLine := src.line
				errComponent := component
				if baseSrc, ok := c.typeDefSources[td.BaseType]; ok {
					errLine = baseSrc.line
					if !baseSrc.isLocal {
						errComponent = "complex type '" + td.BaseType.Name.Local + "'"
					}
				}
				msg := fmt.Sprintf("The {process contents} of the attribute wildcard is weaker than the one in the base complex type definition %s.", baseQualified)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, errLine, "complexType", errComponent, msg), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}
	}
}

// wildcardNSSubset checks whether the namespace constraint of sub is a subset
// of the namespace constraint of super, per XSD §3.10.6.
func wildcardNSSubset(sub, super *Wildcard) bool {
	// ##any is a superset of everything.
	if super.Namespace == WildcardNSAny {
		return true
	}
	// If sub is ##any but super is not, sub is not a subset.
	if sub.Namespace == WildcardNSAny {
		return false
	}
	// Both are specific namespace sets — sub must be contained in super.
	subSet := wildcardNSSet(sub)
	superSet := wildcardNSSet(super)
	for ns := range subSet {
		if !superSet[ns] {
			return false
		}
	}
	return true
}

// wildcardNSSet expands a wildcard's namespace constraint into a set of URIs.
func wildcardNSSet(wc *Wildcard) map[string]bool {
	s := make(map[string]bool)
	switch wc.Namespace {
	case WildcardNSAny:
		// Matches everything — not representable as a finite set.
	case WildcardNSOther:
		// Everything except target namespace and absent (empty) — not finite.
		// For subset checking, treat as "not targetNS".
	case WildcardNSLocal:
		s[""] = true
	case WildcardNSTargetNamespace:
		s[wc.TargetNS] = true
	default:
		// Space-separated list of URIs, possibly including ##local and ##targetNamespace.
		for _, token := range splitSpace(wc.Namespace) {
			switch token {
			case WildcardNSLocal:
				s[""] = true
			case WildcardNSTargetNamespace:
				s[wc.TargetNS] = true
			default:
				s[token] = true
			}
		}
	}
	return s
}

// wildcardUnion computes the union of two attribute wildcards.
// Per XSD 1.0 spec section 3.10.6: Attribute Wildcard Union.
//
// Namespace constraints are classified as:
//   - "any"       → matches everything
//   - "not(ns)"   → ##other: matches everything except ns and absent
//   - "not(absent)" → matches everything except absent (empty namespace)
//   - "set"       → finite set of namespace URIs (empty string = absent)
func wildcardUnion(w1, w2 *Wildcard) *Wildcard {
	pc := w1.ProcessContents
	tns := w1.TargetNS

	// Case 2: If either is ##any, result is ##any.
	if w1.Namespace == WildcardNSAny || w2.Namespace == WildcardNSAny {
		return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
	}

	w1IsNeg := w1.Namespace == WildcardNSOther || w1.Namespace == WildcardNSNotAbsent
	w2IsNeg := w2.Namespace == WildcardNSOther || w2.Namespace == WildcardNSNotAbsent

	// Case 1: Both are the same value.
	if w1.Namespace == w2.Namespace && w1.TargetNS == w2.TargetNS {
		return &Wildcard{Namespace: w1.Namespace, ProcessContents: pc, TargetNS: tns}
	}

	// Case 3: Both are sets (neither is a negation or ##any).
	if !w1IsNeg && !w2IsNeg {
		set := wildcardNSSet(w1)
		for ns := range wildcardNSSet(w2) {
			set[ns] = true
		}
		return wildcardFromSet(set, pc, tns)
	}

	// Case 4: Both are negations.
	if w1IsNeg && w2IsNeg {
		w1NegNS := wildcardNegatedNS(w1)
		w2NegNS := wildcardNegatedNS(w2)
		if w1NegNS == w2NegNS {
			// Same negated value → same result.
			return &Wildcard{Namespace: w1.Namespace, ProcessContents: pc, TargetNS: tns}
		}
		// Different negated values → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}

	// Cases 5 & 6: One is a negation, the other is a set.
	var neg, set *Wildcard
	if w1IsNeg {
		neg, set = w1, w2
	} else {
		neg, set = w2, w1
	}

	negNS := wildcardNegatedNS(neg)
	s := wildcardNSSet(set)
	hasAbsent := s[""]
	hasNegated := negNS != "" && s[negNS]

	if negNS == "" {
		// Case 6: neg is not(absent).
		if hasAbsent {
			// 6.1: Set includes absent → any.
			return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
		}
		// 6.2: Set doesn't include absent → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}

	// Case 5: neg is not(ns).
	if hasNegated && hasAbsent {
		// 5.1: Set includes both negated ns and absent → any.
		return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
	}
	if hasNegated && !hasAbsent {
		// 5.2: Set includes negated ns but not absent → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}
	if !hasNegated && !hasAbsent {
		// 5.4: Set includes neither → the negation.
		return &Wildcard{Namespace: neg.Namespace, ProcessContents: pc, TargetNS: neg.TargetNS}
	}
	// 5.3: Set includes absent but not negated ns → not expressible.
	// Fall back to ##any (permissive).
	return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
}

// wildcardNegatedNS returns the namespace being negated.
// For ##other, it's the target namespace. For ##not-absent, it's "".
func wildcardNegatedNS(wc *Wildcard) string {
	if wc.Namespace == WildcardNSNotAbsent {
		return ""
	}
	// ##other negates the target namespace.
	return wc.TargetNS
}

// wildcardFromSet builds a Wildcard from a namespace set.
func wildcardFromSet(s map[string]bool, pc ProcessContentsKind, tns string) *Wildcard {
	var parts []string
	for ns := range s {
		if ns == "" {
			parts = append(parts, WildcardNSLocal)
		} else {
			parts = append(parts, ns)
		}
	}
	sort.Strings(parts)
	return &Wildcard{
		Namespace:       strings.Join(parts, " "),
		ProcessContents: pc,
		TargetNS:        tns,
	}
}

// processContentsStrength returns the strength of a processContents value.
// strict(2) > lax(1) > skip(0).
func processContentsStrength(pc ProcessContentsKind) int {
	switch pc {
	case ProcessStrict:
		return 2
	case ProcessLax:
		return 1
	default:
		return 0
	}
}

// checkCircularSubstGroup detects if an element's substitution group chain
// leads back to itself. Only reports an error if the element itself is part
// of the cycle (not if it just points to a cyclic element).
func (c *compiler) checkCircularSubstGroup(ctx context.Context, edecl *ElementDecl) {
	visited := map[QName]bool{}
	current := edecl.SubstitutionGroup
	for current != (QName{}) {
		if current == edecl.Name {
			// Cycle leads back to this element.
			// libxml2 reports this error twice.
			if src, ok := c.globalElemSources[edecl]; ok {
				msg := fmt.Sprintf("The element declaration '%s' defines a circular substitution group to element declaration '%s'.",
					edecl.Name.Local, current.Local)
				errStr := schemaElemDeclError(c.filename, src.line, edecl.Name.Local, msg)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(errStr, helium.ErrorLevelFatal))
				c.errorCount++
				c.errorHandler.Handle(ctx, helium.NewLeveledError(errStr, helium.ErrorLevelFatal))
				c.errorCount++
			}
			return
		}
		if visited[current] {
			// Hit a cycle that doesn't include this element.
			return
		}
		visited[current] = true
		head, ok := c.schema.elements[current]
		if !ok {
			return
		}
		current = head.SubstitutionGroup
	}
}

// checkFinalOnTypes checks that no type derivation violates the base type's final constraint.
func (c *compiler) checkFinalOnTypes(ctx context.Context) {
	for _, td := range c.schema.types {
		src := c.typeDefSources[td]

		// Check base type final for extension/restriction derivation.
		if td.BaseType != nil && td.BaseType.Final != 0 {
			baseFinal := td.BaseType.Final
			if td.Derivation == DerivationExtension && baseFinal&FinalExtension != 0 {
				component := td.Name.Local
				if src.isLocal {
					component = componentLocalComplexType
				}
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by extension is forbidden by the base type '"+td.BaseType.Name.Local+"'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
			if td.Derivation == DerivationRestriction && baseFinal&FinalRestriction != 0 {
				component := td.Name.Local
				if src.isLocal {
					component = componentLocalComplexType
				}
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by restriction is forbidden by the base type '"+td.BaseType.Name.Local+"'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// simpleType list: check if item type forbids list derivation.
		if td.Variety == TypeVarietyList && td.ItemType != nil && td.ItemType.Final&FinalList != 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
				"Derivation by list is forbidden by the item type '"+td.ItemType.Name.Local+"'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
		// simpleType union: check if any member type forbids union derivation.
		if td.Variety == TypeVarietyUnion {
			for _, member := range td.MemberTypes {
				if member.Final&FinalUnion != 0 {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
						"Derivation by union is forbidden by the member type '"+member.Name.Local+"'."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
		}
	}
}

// checkFinalOnSubstGroups checks that substitution group members do not violate
// the head element's final constraint.
func (c *compiler) checkFinalOnSubstGroups(ctx context.Context) {
	for headQN, members := range c.schema.substGroups {
		head, ok := c.schema.elements[headQN]
		if !ok {
			continue
		}
		if head.Final == 0 {
			continue
		}
		for _, member := range members {
			if head.Final&FinalExtension != 0 && derivationUsesMethod(member.Type, head.Type, DerivationExtension) {
				if src, ok := c.globalElemSources[member]; ok {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
			if head.Final&FinalRestriction != 0 && derivationUsesMethod(member.Type, head.Type, DerivationRestriction) {
				if src, ok := c.globalElemSources[member]; ok {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
		}
	}
}

// derivationUsesMethod walks the BaseType chain from derived to base and
// returns true if any step in the chain uses the given derivation method.
func derivationUsesMethod(derived, base *TypeDef, method DerivationKind) bool {
	if derived == nil || base == nil {
		return false
	}
	td := derived
	for td != nil && td != base {
		if td.Derivation == method {
			return true
		}
		td = td.BaseType
	}
	return false
}

// refChameleonEligible reports whether the lexical type ref at the given
// element is eligible for the no-targetNamespace ({}) chameleon fallback. The
// fallback exists for chameleon includes: an imported no-targetNamespace schema
// contributes its unqualified types as if they belonged to the including
// schema's target namespace, so a ref that resolved to {targetNamespace}name
// may instead bind to the imported {}name.
//
// Eligibility is tracked from the LEXICAL ref and fires ONLY when the ref was
// BOTH (a) unprefixed (no "prefix:" in the lexical QName) AND (b) had no
// in-scope default namespace (no xmlns="..." covering it). In every other case
// the ref is qualified: a prefixed ref (m:t) binds to its prefix's namespace,
// and an unprefixed ref under a default namespace (xmlns="urn:main" -> t binds
// to urn:main) binds to that namespace. Such qualified refs must NOT mask to
// {}; if they do not resolve in their bound namespace, an unresolved error is
// reported. The eligibility bit is recorded at the ref collection site (where
// the lexical form and in-scope namespaces are available) via
// markChameleonEligible / unionMemberRef.chameleonEligible.
func refChameleonEligible(elem *helium.Element, ref string) bool {
	if strings.ContainsRune(ref, ':') {
		return false
	}
	// A default namespace in scope (xmlns="...") qualifies the unprefixed ref.
	return lookupNS(elem, "") == ""
}

// markChameleonEligible records that the ref owned by owner (an *ElementDecl or
// *TypeDef) is eligible for the no-targetNamespace ({}) fallback, based on the
// lexical ref at elem. Call at the collection site.
func (c *compiler) markChameleonEligible(owner any, elem *helium.Element, ref string) {
	if refChameleonEligible(elem, ref) {
		c.chameleonEligible[owner] = struct{}{}
	}
}

// resolveQName resolves a prefixed name (like "xsd:string") to a QName
// using the namespace declarations in scope on the given element.
func (c *compiler) resolveQName(_ context.Context, elem *helium.Element, ref string) QName {
	local := ref
	ns := c.schema.targetNamespace

	for i := range len(ref) {
		if ref[i] == ':' {
			prefix := ref[:i]
			local = ref[i+1:]
			ns = lookupNS(elem, prefix)
			return QName{Local: local, NS: ns}
		}
	}

	// Unprefixed name: use the default namespace from the element context.
	// This is critical for inline schemas where the default namespace is
	// the XSD namespace — unprefixed type refs like "string" must resolve
	// to xs:string, not to {targetNamespace}string.
	if defNS := lookupNS(elem, ""); defNS != "" {
		ns = defNS
	}

	return QName{Local: local, NS: ns}
}
