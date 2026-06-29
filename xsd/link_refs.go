package xsd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
					edecl.DefaultNS = ge.DefaultNS
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
				if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" && !c.deprecatedDatatypeQName(qn) {
					msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) element declaration.", qn.NS, qn.Local)
					c.schemaError(ctx, schemaParserErrorAttr(c.diagSourceOrRecorded(src.source), src.line, src.elemName, elemElement, attrRef, msg))
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
				if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" && !c.deprecatedDatatypeQName(qn) {
					msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
					c.schemaError(ctx, schemaElemDeclErrorAttr(c.diagSourceOrRecorded(src.source), src.line, src.elemName, attrType, msg))
				}
				td = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				c.schema.types[qn] = td
			}
			edecl.Type = td
		}
	}

	// Resolve XSD 1.1 conditional-type-assignment alternative @type references.
	c.resolveAltTypeRefs(ctx)

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

	// Detect and cut INDIRECT xs:attributeGroup reference cycles (e.g. h -> i,
	// i -> h) BEFORE any flattening or expansion. A circular attribute-group
	// reference is disallowed outside <redefine> (XSD §3.6.2 src-attribute_group.3),
	// just like the DIRECT self-reference caught at parse time in read_particles.go.
	// Reporting and cutting the back-edge here both surfaces the schema error and
	// keeps the cycle-guarded flatten/expand walks below from silently relying on a
	// recursion-stack guard that produced no diagnostic.
	c.checkCircularAttrGroupRefs(ctx)

	// Reject duplicate attribute uses inside a global attribute group definition
	// (ag-props-correct.2) BEFORE expanding the group into the types that
	// reference it. This both reports duplicates in attribute groups that NO type
	// references — which the per-type check below would never inspect — and
	// removes the duplicate use from the group so a referencing type does not
	// re-report the same collision (xmllint reports it once, at the group).
	c.checkAttrGroupDuplicates(ctx)
	c.checkSchemaDefaultAttributes(ctx)

	// Resolve attribute group references.
	//
	// An attribute group's effective attribute uses are NOT only the uses it
	// declares directly: an xs:attributeGroup may itself contain nested
	// xs:attributeGroup ref children, whose attribute uses are pulled in
	// transitively (XSD §3.6.2 / §3.4.2 — {attribute uses} is the union over
	// the group and all groups it references). c.schema.attrGroups[qn] holds
	// only the group's OWN direct uses; the nested refs live in
	// c.attrGroupRefChildren. Expand each referenced group recursively
	// (cycle-guarded) so the type's effective attributes include the
	// transitively-referenced uses — otherwise a required/defaulted/prohibited
	// attribute declared in a nested group is silently dropped.
	//
	// The expansion dedups WITHIN a single referenced group's transitive closure
	// (override semantics), but the result is appended to td.Attributes WITHOUT
	// further dedup, so a name that a type declares directly AND pulls in via a
	// group — or via two distinct groups — still surfaces as a duplicate use for
	// checkDuplicateAttrUses below (ct-props-correct.4), preserving the prior
	// behavior of appending raw group attributes.
	for td, qns := range c.attrGroupRefs {
		srcs := c.attrGroupRefUseSources[td]
		for i, qn := range qns {
			if _, ok := c.schema.attrGroups[qn]; !ok {
				continue
			}
			uses := c.expandAttrGroupUses(qn, map[QName]struct{}{})
			if i < len(srcs) && srcs[i].attr == attrDefaultAttributes {
				c.markDefaultAttrUses(td, uses)
			}
			td.Attributes = append(td.Attributes, uses...)
			// XSD 1.1: a referenced attribute group's xs:anyAttribute wildcard is
			// INTERSECTED into the type's effective attribute wildcard (XSD 3.4.2,
			// "complete wildcard"). Gated on Version11 so 1.0 (which drops group
			// wildcards) is unchanged.
			if c.version != Version11 {
				continue
			}
			if gw := c.attrGroupCompleteWildcard(qn, map[QName]struct{}{}); gw != nil {
				if td.AnyAttribute == nil {
					td.AnyAttribute = gw
				} else {
					td.AnyAttribute = intersectWildcards(td.AnyAttribute, gw)
				}
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
			au.DefaultNS = ga.DefaultNS
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
		// XSD 1.1 {inheritable}: a ref use without an explicit inheritable adopts
		// the referenced global declaration's value; an explicit one already won.
		if !au.InheritableSet {
			au.Inheritable = ga.Inheritable
			au.InheritableSet = ga.InheritableSet
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
			// simpleContent extension — check for base-vs-derived duplicate
			// attributes BEFORE inheriting the base attributes, then merge. In XSD
			// 1.1 attribute inheritance is deferred to finalizeEffectiveAttrs, which
			// is topological across BOTH extension and restriction derivations so an
			// extension of a restriction reads a FINALIZED base attribute set.
			if c.version != Version11 {
				c.checkExtensionAttrDuplicates(ctx, td)
				if td.BaseType.Attributes != nil {
					td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
				}
				if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
					td.AnyAttribute = td.BaseType.AnyAttribute
				} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
					td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute, c.version)
				}
			}
			continue
		}
		// cos-ct-extends-1-1: complexContent extension requires the base type
		// to also have complex content (mixed or element-only), not simple content.
		// (simpleContent extensions already continued above, so any type reaching
		// here is a complexContent derivation.) XSD 1.0 only flagged this when the
		// derived type had element content; XSD 1.1 flags the empty/attribute-only
		// case too (a complexContent extension of a simple base is always invalid).
		if td.BaseType.ContentType == ContentTypeSimple && (td.ContentType == ContentTypeElementOnly || td.ContentType == ContentTypeMixed || c.version == Version11) {
			if src, ok := c.typeDefSources[td]; ok && c.filename != "" {
				component := componentLocalComplexType
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component,
					"The content type of both, the type and its base type, must either 'mixed' or 'element-only'."))
			}
			continue
		}
		baseMG := td.BaseType.ContentModel
		derivedMG := td.ContentModel
		// cos-all-limited.1.2 / §3.8.2: an 'all' model group may only constitute
		// the WHOLE content of a type definition. When an extension appends an
		// 'all' group (directly, or via an xs:group ref that resolves to one) onto
		// a non-empty base content model, the merge below would build a sequence
		// CONTAINING an 'all' group, which is forbidden. The base-as-sole-content
		// and direct-group-ref paths are checked elsewhere; this catches the
		// extension-merge path, which they miss. libxml2 rejects this.
		if baseMG != nil && derivedMG != nil && derivedMG.MaxOccurs != 0 && derivedMG.Compositor == CompositorAll && modelGroupHasContent(baseMG) {
			if src, ok := c.typeDefSources[td]; ok && c.filename != "" {
				component := componentLocalComplexType
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component,
					"The 'all' model group needs to be the only child of the model group."))
			}
			continue
		}
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
		// Check for duplicate attributes before merging base type attributes, then
		// inherit. XSD 1.1 defers both to finalizeEffectiveAttrs (topological).
		if c.version != Version11 {
			c.checkExtensionAttrDuplicates(ctx, td)
			if td.BaseType.Attributes != nil {
				td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
			}
			if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = td.BaseType.AnyAttribute
			} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute, c.version)
			}
		}
	}

	// XSD 1.1: resolve ##definedSibling on element wildcards now that content
	// models (including expanded group refs) are fully built — BEFORE the
	// restriction-derivation checks below, which compare base/derived wildcards'
	// resolved SiblingNames.
	if c.version == Version11 {
		c.resolveDefinedSiblings()
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
		// In XSD 1.1 checkRestrictionAttrs is run inside finalizeEffectiveAttrs,
		// once the base's effective attributes are finalized; here only the content
		// model restriction check runs (it is independent of attribute finalization).
		if c.version != Version11 {
			c.checkRestrictionAttrs(ctx, td)
		}
		c.checkRestrictionParticles(ctx, td)
	}

	// XSD 1.1: finalize each derived complex type's effective {attribute uses}
	// TOPOLOGICALLY across BOTH extension and restriction derivations. A derivation
	// inherits each base attribute use it does not redeclare (§3.4.2); a required
	// base attribute that is not redeclared must stay required, so the merge — not
	// the relaxed checkRestrictionAttrs — is what keeps it enforced. The merge must
	// read a FINALIZED base attribute set, so an extension of a restriction (or vice
	// versa) inherits correctly regardless of source order; the extension and
	// restriction passes above DEFER all attribute work to here in 1.1. Gated to
	// 1.1 so XSD 1.0 stays byte-identical.
	if c.version == Version11 {
		derived := make([]*TypeDef, 0, len(extensionTypes)+len(restrictionTypes))
		derived = append(derived, extensionTypes...)
		derived = append(derived, restrictionTypes...)
		sort.SliceStable(derived, func(i, j int) bool {
			si, sj := c.typeDefSources[derived[i]], c.typeDefSources[derived[j]]
			if si.line != sj.line {
				return si.line < sj.line
			}
			return si.ordinal < sj.ordinal
		})
		merged := make(map[*TypeDef]bool)
		visiting := make(map[*TypeDef]bool)
		for _, td := range derived {
			c.finalizeEffectiveAttrs(ctx, td, merged, visiting)
		}
	}

	// Check UPA (Unique Particle Attribution) for all complex types with content models.
	// Only run UPA if there are no prior schema errors (libxml2 skips UPA when
	// the schema has structural parse errors).
	//
	// Iterate in a deterministic source order (line, then ordinal) rather than via
	// Go map ranging: checkUPA increments errorCount, and a stable order keeps both
	// which violation is reported first and the downstream errorCount-gated checks
	// (e.g. checkElementConsistent) independent of map iteration order.
	if c.filename != "" && c.errorCount == 0 {
		type upaTarget struct {
			td  *TypeDef
			src typeDefSource
		}
		targets := make([]upaTarget, 0, len(c.typeDefSources))
		for td, src := range c.typeDefSources {
			if td.ContentModel != nil {
				targets = append(targets, upaTarget{td: td, src: src})
			}
		}
		sort.Slice(targets, func(i, j int) bool {
			if targets[i].src.line != targets[j].src.line {
				return targets[i].src.line < targets[j].src.line
			}
			return targets[i].src.ordinal < targets[j].src.ordinal
		})
		for _, t := range targets {
			c.checkUPA(ctx, t.td, t.src)
		}
	}
}

// expandAttrGroupUses returns the effective attribute uses contributed by the
// attribute group qn: its OWN direct uses (c.schema.attrGroups[qn]) plus, for
// each nested xs:attributeGroup ref child, that group's effective uses computed
// recursively. visited guards against reference cycles.
//
// Deduplication follows XSD attribute-group semantics: a use declared closer to
// the referencing group (the group's own use, or an earlier-referenced group)
// overrides one inherited from a more deeply / later referenced group, keyed by
// expanded attribute QName. A prohibited use removes the corresponding use from
// the effective set. The group's own uses take precedence over its nested refs.
func (c *compiler) expandAttrGroupUses(qn QName, visited map[QName]struct{}) []*AttrUse {
	if _, seen := visited[qn]; seen {
		return nil
	}
	visited[qn] = struct{}{}

	// The group's own direct uses come first (closest), then each nested ref's
	// expanded uses in declaration order.
	var uses []*AttrUse
	uses = append(uses, c.schema.attrGroups[qn]...)
	for _, refQN := range c.attrGroupRefChildren[qn] {
		uses = appendAttrUses(uses, c.expandAttrGroupUses(refQN, visited))
	}
	return uses
}

func (c *compiler) markDefaultAttrUses(td *TypeDef, uses []*AttrUse) {
	if td == nil || len(uses) == 0 {
		return
	}
	attrs := c.defaultAttrUses[td]
	if attrs == nil {
		attrs = make(map[QName]*AttrUse)
		c.defaultAttrUses[td] = attrs
	}
	for _, au := range uses {
		if au.Prohibited {
			continue
		}
		attrs[au.Name] = au
	}
}

func (c *compiler) defaultAttrUse(td *TypeDef, name QName) *AttrUse {
	if td == nil {
		return nil
	}
	return c.defaultAttrUses[td][name]
}

func (c *compiler) defaultAttrUseMatches(a, b *TypeDef, name QName) bool {
	au := c.defaultAttrUse(a, name)
	return au != nil && au == c.defaultAttrUse(b, name)
}

func (c *compiler) markDefaultAttrUse(td *TypeDef, au *AttrUse) {
	if td == nil || au == nil || au.Prohibited {
		return
	}
	attrs := c.defaultAttrUses[td]
	if attrs == nil {
		attrs = make(map[QName]*AttrUse)
		c.defaultAttrUses[td] = attrs
	}
	attrs[au.Name] = au
}

// attrGroupCompleteWildcard returns the XSD 1.1 "complete wildcard" of an
// attribute group: its OWN xs:anyAttribute (if any) INTERSECTED with the
// complete wildcards of every nested xs:attributeGroup ref child. visited guards
// against reference cycles. Returns nil if neither the group nor any referenced
// group declares a wildcard.
func (c *compiler) attrGroupCompleteWildcard(qn QName, visited map[QName]struct{}) *Wildcard {
	if _, seen := visited[qn]; seen {
		return nil
	}
	visited[qn] = struct{}{}

	result := c.attrGroupWildcards[qn]
	for _, refQN := range c.attrGroupRefChildren[qn] {
		nested := c.attrGroupCompleteWildcard(refQN, visited)
		if nested == nil {
			continue
		}
		if result == nil {
			result = nested
			continue
		}
		result = intersectWildcards(result, nested)
	}
	return result
}

// combineGroupWildcards intersects two possibly-nil attribute-group wildcards,
// returning nil when both are nil and the non-nil one when only one is present.
func combineGroupWildcards(a, b *Wildcard) *Wildcard {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return intersectWildcards(a, b)
}

// appendAttrUses merges the attribute uses in extra into dst applying
// attribute-group override semantics: a use already present in dst (by expanded
// QName) is kept and the incoming inherited use is discarded (closer wins). A
// prohibited incoming use whose name is not yet present is still appended so a
// later merge can observe the prohibition. The merge is order-preserving so the
// closest-declared use stays first.
func appendAttrUses(dst, extra []*AttrUse) []*AttrUse {
	seen := make(map[QName]struct{}, len(dst))
	for _, au := range dst {
		seen[au.Name] = struct{}{}
	}
	for _, au := range extra {
		if _, ok := seen[au.Name]; ok {
			continue
		}
		seen[au.Name] = struct{}{}
		dst = append(dst, au)
	}
	return dst
}

// checkExtensionAttrDuplicates reports an attribute use redeclared by a
// (simpleContent or complexContent) extension type that its base type already
// declares. Prohibited uses do not contribute an attribute use and are skipped
// on both sides (mirroring checkDuplicateAttrUses), so a prohibited use carried
// in via an attribute group does not falsely trigger a duplicate diagnostic.
// Must run BEFORE the base attributes are merged into td.Attributes.
func (c *compiler) checkExtensionAttrDuplicates(ctx context.Context, td *TypeDef) {
	if c.filename == "" || td.BaseType == nil {
		return
	}
	if td.BaseType.Attributes == nil || td.Attributes == nil {
		return
	}
	baseAttrNames := make(map[QName]bool, len(td.BaseType.Attributes))
	for _, au := range td.BaseType.Attributes {
		if au.Prohibited {
			continue
		}
		baseAttrNames[au.Name] = true
	}
	for _, au := range td.Attributes {
		if au.Prohibited {
			continue
		}
		if !baseAttrNames[au.Name] {
			continue
		}
		if c.defaultAttrUseMatches(td, td.BaseType, au.Name) {
			continue
		}
		src, ok := c.typeDefSources[td]
		if !ok {
			continue
		}
		component := componentLocalComplexType
		if !src.isLocal {
			component = td.Name.Local
		}
		msg := fmt.Sprintf("Duplicate attribute use '%s'.", au.Name.Local)
		c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg))
	}
}

// checkDuplicateAttrUses reports duplicate attribute uses (by expanded QName)
// within a single complex type's own attribute set. Prohibited uses do not
// contribute an attribute use and are skipped for the duplicate-error check.
//
// A prohibited use that shares its QName with a non-prohibited use in the same
// expanded set is, however, pointless: the prohibition cannot remove a use that
// the type itself declares. libxml2 strips such a prohibition and emits a schema
// parser WARNING attributed to the prohibited xs:attribute element, which the
// golden tests compare. We mirror that warning here.
//
// Types are processed in source line order for deterministic diagnostics.
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
		// Collect the QNames of every non-prohibited use so a pointless
		// prohibition (one whose QName already has a real use) can be detected.
		realUse := make(map[QName]struct{}, len(td.Attributes))
		for _, au := range td.Attributes {
			if au.Prohibited {
				continue
			}
			realUse[au.Name] = struct{}{}
		}

		seen := make(map[QName]bool, len(td.Attributes))
		reported := make(map[QName]bool)
		warnedProhib := make(map[QName]bool)
		for _, au := range td.Attributes {
			if au.Prohibited {
				if _, ok := realUse[au.Name]; !ok {
					continue
				}
				if warnedProhib[au.Name] {
					continue
				}
				warnedProhib[au.Name] = true
				c.warnPointlessProhibition(ctx, au)
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
				c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg))
				continue
			}
			seen[au.Name] = true
		}
	}
}

// warnPointlessProhibition emits the libxml2-compatible schema parser WARNING for
// a prohibited attribute use whose QName already names a real (non-prohibited)
// use in the same type definition. The diagnostic is attributed to the
// prohibited xs:attribute element at its recorded source line/file (covering
// xs:include/xs:import cases). If no source was recorded, the warning falls back
// to the compiler's own filename and line 0.
func (c *compiler) warnPointlessProhibition(ctx context.Context, au *AttrUse) {
	file := c.filename
	line := 0
	if src, ok := c.attrUseSources[au]; ok {
		line = src.line
		file = c.diagSourceOrRecorded(src.source)
	}
	msg := fmt.Sprintf("Skipping pointless attribute use prohibition '%s', since a corresponding attribute use exists already in the type definition.", formatAttrQName(au.Name))
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(file, line, "attribute", "attribute", msg), helium.ErrorLevelWarning))
}

// formatAttrQName renders an attribute QName the way libxml2's
// xmlSchemaFormatQName does: "{ns}local" when a namespace is present, otherwise
// the bare local name (no braces).
func formatAttrQName(qn QName) string {
	if qn.NS == "" {
		return qn.Local
	}
	return fmt.Sprintf("{%s}%s", qn.NS, qn.Local)
}

// checkAttrGroupDuplicates reports duplicate attribute uses (by expanded QName)
// within a single GLOBAL attribute group definition (ag-props-correct.2) and
// strips the later duplicate from the stored group. It must run BEFORE the
// attribute groups are expanded into the types that reference them, so that:
//
//   - a group that NO type references — which the per-type checkDuplicateAttrUses
//     never inspects — still has its internal duplicates rejected, and
//   - a referencing type does not re-report the same collision (xmllint reports
//     an attribute group's internal duplicate once, attributed to the group).
//
// Prohibited uses do not contribute an attribute use and are skipped. Groups are
// processed in source line order for deterministic diagnostics. The diagnostic
// matches xmllint's wording, attributed to the xs:attributeGroup element.
func (c *compiler) checkAttrGroupDuplicates(ctx context.Context) {
	if c.filename == "" {
		return
	}
	qns := make([]QName, 0, len(c.attrGroupSources))
	for qn := range c.attrGroupSources {
		// A group needs inspection when it has more than one own attribute use OR
		// when it pulls in attribute uses through nested xs:attributeGroup ref
		// children, either of which can produce a duplicate (ag-props-correct.2).
		if len(c.schema.attrGroups[qn]) > 1 || len(c.attrGroupRefChildren[qn]) > 0 {
			qns = append(qns, qn)
		}
	}
	sort.Slice(qns, func(i, j int) bool {
		return c.attrGroupSources[qns[i]].line < c.attrGroupSources[qns[j]].line
	})
	for _, qn := range qns {
		attrs := c.schema.attrGroups[qn]
		seen := make(map[QName]bool, len(attrs))
		reported := make(map[QName]bool)
		deduped := attrs[:0]
		for _, au := range attrs {
			if au.Prohibited {
				deduped = append(deduped, au)
				continue
			}
			if !seen[au.Name] {
				seen[au.Name] = true
				deduped = append(deduped, au)
				continue
			}
			if reported[au.Name] {
				continue
			}
			reported[au.Name] = true
			c.reportAttrGroupDuplicate(ctx, qn, au.Name)
		}
		c.schema.attrGroups[qn] = deduped

		// After deduping the group's OWN attribute uses, flatten the attribute uses
		// brought in through nested xs:attributeGroup ref children (recursively,
		// cycle-guarded) and report any name that collides with a use already
		// present in the group or another referenced group.
		visited := map[QName]bool{qn: true}
		for _, refQN := range c.attrGroupRefChildren[qn] {
			c.flattenAttrGroupRefDuplicates(ctx, qn, refQN, seen, reported, visited)
		}
	}
}

// flattenAttrGroupRefDuplicates walks a nested attribute-group reference,
// recording each (non-prohibited) attribute use it contributes into seen and
// reporting — once, attributed to the owning group ownerQN — any name already
// present. visited is a RECURSION STACK (the groups currently on the path being
// expanded), not a global "seen ever" set: a group is added on entry and removed
// on exit, so a true reference CYCLE is still cut, but two SIBLING refs to the
// same group are each expanded — so a name contributed by both siblings surfaces
// as a duplicate (g -> h, h with h carrying attribute x is a duplicate use of x,
// which xmllint rejects). The walk descends into the referenced group's own
// nested refs as well.
func (c *compiler) flattenAttrGroupRefDuplicates(ctx context.Context, ownerQN, refQN QName, seen, reported map[QName]bool, visited map[QName]bool) {
	if visited[refQN] {
		return
	}
	visited[refQN] = true
	defer delete(visited, refQN)
	// A name repeated WITHIN this referenced group is that group's own internal
	// duplicate (reported when the group is processed as owner), not a collision to
	// attribute to ownerQN. Track this group's local names so each distinct name is
	// merged into seen once and only a cross-group collision is reported here.
	local := make(map[QName]bool)
	for _, au := range c.schema.attrGroups[refQN] {
		if au.Prohibited {
			continue
		}
		if local[au.Name] {
			continue
		}
		local[au.Name] = true
		if !seen[au.Name] {
			seen[au.Name] = true
			continue
		}
		if reported[au.Name] {
			continue
		}
		reported[au.Name] = true
		c.reportAttrGroupDuplicate(ctx, ownerQN, au.Name)
	}
	for _, nextQN := range c.attrGroupRefChildren[refQN] {
		c.flattenAttrGroupRefDuplicates(ctx, ownerQN, nextQN, seen, reported, visited)
	}
}

// reportCircularAttrGroupRef emits the src-attribute_group.3 circular-reference
// diagnostic for a self-referential <xs:attributeGroup ref="..."> that resolves
// to the group being defined (groupQN). A circular attribute-group reference is
// disallowed outside <redefine>; libxml2 reports it against the referencing
// <attributeGroup> element and cuts the reference. The diagnostic is attributed
// to the ref element's source line (ce) via diagSource so an included schema is
// cited correctly.
func (c *compiler) reportCircularAttrGroupRef(ctx context.Context, ce *helium.Element, groupQN QName) {
	if c.filename == "" {
		return
	}
	msg := fmt.Sprintf("Circular reference to the attribute group '%s' defined.", formatAttrQName(groupQN))
	c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "attributeGroup", msg))
}

// checkCircularAttrGroupRefs detects INDIRECT xs:attributeGroup reference cycles
// over the c.attrGroupRefChildren adjacency (e.g. h -> i, i -> h, or the 3-node
// h -> i -> j -> h) and reports each as a circular reference
// (src-attribute_group.3), matching the DIRECT self-reference caught at parse
// time in read_particles.go. Direct self-references never reach here — they are
// reported and dropped during parsing — so this catches only multi-node cycles.
//
// The back-edge that closes a cycle is CUT (removed from the adjacency) so the
// downstream cycle-guarded flatten (flattenAttrGroupRefDuplicates) and expand
// (expandAttrGroupUses) walks no longer rely on their recursion-stack guard to
// silently truncate the cycle, and so a circular schema can never compile as if
// it were valid. Cutting also guarantees no duplicate-attribute false positive
// is introduced by the cycle.
//
// Groups and their ref children are visited in a deterministic order (sorted
// QName, declaration order within a group) so the reported cycle and any cut
// edge are independent of Go map iteration order.
func (c *compiler) checkCircularAttrGroupRefs(ctx context.Context) {
	if c.filename == "" {
		return
	}

	roots := make([]QName, 0, len(c.attrGroupRefChildren))
	for qn := range c.attrGroupRefChildren {
		roots = append(roots, qn)
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].NS != roots[j].NS {
			return roots[i].NS < roots[j].NS
		}
		return roots[i].Local < roots[j].Local
	})

	// onStack is the current DFS recursion stack; done marks fully-explored nodes
	// so a shared subtree reachable from two roots is not re-walked.
	onStack := make(map[QName]bool)
	done := make(map[QName]bool)

	var visit func(qn QName)
	visit = func(qn QName) {
		onStack[qn] = true
		// Re-read the slice each iteration: a cut splices out the back-edge in place.
		for i := 0; i < len(c.attrGroupRefChildren[qn]); i++ {
			child := c.attrGroupRefChildren[qn][i]
			if onStack[child] {
				// Back-edge qn -> child closes a cycle through child. Report it as a
				// circular reference to child, attributed to THIS back-edge ref
				// element's recorded source (index-aligned with attrGroupRefChildren),
				// then cut the edge so the flatten/expand walks below terminate
				// without a diagnostic-less truncation.
				edgeSrc := c.attrGroupRefSourceAt(qn, i)
				c.reportCircularAttrGroupRefQName(ctx, child, edgeSrc)
				children := c.attrGroupRefChildren[qn]
				c.attrGroupRefChildren[qn] = append(children[:i], children[i+1:]...)
				if srcs := c.attrGroupRefSources[qn]; i < len(srcs) {
					c.attrGroupRefSources[qn] = append(srcs[:i], srcs[i+1:]...)
				}
				i--
				continue
			}
			if done[child] {
				continue
			}
			visit(child)
		}
		onStack[qn] = false
		done[qn] = true
	}

	for _, qn := range roots {
		if done[qn] {
			continue
		}
		visit(qn)
	}
}

// attrGroupRefSourceAt returns the per-edge source recorded for the i-th nested
// xs:attributeGroup ref child of ownerQN (index-aligned with
// c.attrGroupRefChildren[ownerQN]), falling back to the owning group's
// declaration source if the parallel slice is short (which should not happen).
func (c *compiler) attrGroupRefSourceAt(ownerQN QName, i int) attrGroupSource {
	if srcs := c.attrGroupRefSources[ownerQN]; i < len(srcs) {
		return srcs[i]
	}
	return c.attrGroupSources[ownerQN]
}

// reportCircularAttrGroupRefQName emits the src-attribute_group.3 circular-
// reference diagnostic for an INDIRECT attribute-group cycle. The diagnostic
// names the group being circularly referenced (targetQN) and is attributed to
// the BACK-EDGE <xs:attributeGroup ref="..."> element's recorded source
// (edgeSrc) — the ref that actually closed the cycle — so the reported line is
// the ref line, matching the direct-self-reference path's attribution and
// pointing at the right file when the cycle spans included/redefined schemas.
func (c *compiler) reportCircularAttrGroupRefQName(ctx context.Context, targetQN QName, edgeSrc attrGroupSource) {
	if c.filename == "" {
		return
	}
	msg := fmt.Sprintf("Circular reference to the attribute group '%s' defined.", formatAttrQName(targetQN))
	c.schemaError(ctx, schemaParserError(c.diagSourceOrRecorded(edgeSrc.source), edgeSrc.line, "attributeGroup", "attributeGroup", msg))
}

// reportAttrGroupDuplicate emits the ag-props-correct.2 duplicate-attribute-use
// diagnostic for name, attributed to the attribute group ownerQN's source.
func (c *compiler) reportAttrGroupDuplicate(ctx context.Context, ownerQN, name QName) {
	src := c.attrGroupSources[ownerQN]
	msg := fmt.Sprintf("Duplicate attribute use '%s'.", name.Local)
	c.schemaError(ctx, schemaParserError(c.diagSourceOrRecorded(src.source), src.line, "attributeGroup", "attributeGroup", msg))
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

	// Attribute to the declaring file recorded at parse time (an included/imported
	// schema when the ref was parsed inside an xs:include/xs:import/xs:redefine),
	// not the top-level compiler filename. c.includeFile has been restored by the
	// time this deferred check runs, so the recorded source is used.
	file := c.diagSourceOrRecorded(src.source)
	if src.nested {
		c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
			"A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups."))
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
	c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
		"The particle's {max occurs} must be 1, since the reference resolves to an 'all' model group."))
}

// modelGroupHasContent reports whether mg carries any actual content particle
// (an element, wildcard, or nested non-empty group). A nil group, a group whose
// own occurrence is prohibited (maxOccurs=0), or a group that wraps only empty
// or prohibited sub-particles, has no content. Used to decide whether an
// extension base content model is "effectively non-empty" before merging. A
// prohibited particle (minOccurs=0 maxOccurs=0) maps to no particle at all, so
// it must not be counted as content.
func modelGroupHasContent(mg *ModelGroup) bool {
	if mg == nil || mg.MaxOccurs == 0 {
		return false
	}
	for _, p := range mg.Particles {
		if p.MaxOccurs == 0 {
			continue
		}
		switch term := p.Term.(type) {
		case *ModelGroup:
			if modelGroupHasContent(term) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// reportUnresolvedTypeRef reports a fatal schema parser error for a type
// reference (base type, list item type, or union member type) on owner that
// does not resolve to a type definition. The caller installs a recovery
// placeholder only after this records the error, so an invalid schema cannot
// silently compile and validate documents as if the missing type existed.
func (c *compiler) reportUnresolvedTypeRef(ctx context.Context, owner *TypeDef, qn QName) {
	if c.deprecatedDatatypeQName(qn) {
		return
	}
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[owner]
	if !hasSrc {
		return
	}
	// Component label and the reporting element kind follow the owner type's
	// actual element kind (complexType vs simpleType), captured at parse time,
	// rather than assuming a simpleType. A local complexType base ref that does
	// not resolve must report "element complexType" / "local complex type".
	elemKind := src.elemKind
	if elemKind == "" {
		elemKind = elemSimpleType
	}
	component := owner.Name.Local
	if component == "" || src.isLocal {
		if elemKind == elemComplexType {
			component = componentLocalComplexType
		} else {
			component = componentLocalSimpleType
		}
	}
	msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
	c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, elemKind, component, msg))
}

func (c *compiler) checkSchemaDefaultAttributes(ctx context.Context) {
	for _, ref := range c.schemaDefaultAttrRefs {
		if _, ok := c.schema.attrGroups[ref.qn]; ok {
			continue
		}
		c.reportUnresolvedAttrGroupRef(ctx, ref.qn, ref.src)
	}
}

func (c *compiler) reportUnresolvedAttrGroupRef(ctx context.Context, qn QName, src attrGroupRefUseSource) {
	if c.filename == "" {
		return
	}
	elemLocal := src.elemLocal
	if elemLocal == "" {
		elemLocal = elemAttributeGroup
	}
	attr := src.attr
	if attr == "" {
		attr = attrRef
	}
	msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) attribute group definition.", qn.NS, qn.Local)
	c.schemaError(ctx, schemaParserErrorAttr(c.diagSourceOrRecorded(src.source), src.line, elemLocal, elemLocal, attr, msg))
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
		// Validate through the version-aware path so a 1.1 schema accepts 1.1-only
		// lexical forms (e.g. "+INF") in attribute default/fixed constraints. The
		// version-less (*TypeDef).Validate would build a Version10 context. schema is
		// supplied because validateValue may evaluate an xs:assertion facet whose
		// schema-aware cast (`castable as t:T`) needs the schema declarations — a nil
		// schema would make that cast fail closed and reject a valid default/fixed.
		vc := &validationContext{schema: c.schema, errorHandler: helium.NilErrorHandler{}, version: c.version}
		if err := validateValue(ctx, *val, it.src.nsMap, td, "", "", 0, vc); err != nil {
			msg := fmt.Sprintf("The value '%s' is not a valid value of the atomic type '%s'.", *val, typeDisplayName(td))
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, "attribute", it.src.local, msg))
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

	// Attribute the diagnostics to the file that actually declared this derived
	// type: for an included/imported/redefined type, src.line refers to the nested
	// schema, so c.filename (the parent) would mis-cite the location. Mirror the
	// restriction-particle check (checkRestrictionParticles).
	file := c.diagSourceOrRecorded(src.source)

	baseTypeName := td.BaseType.Name.Local
	baseTypeNS := td.BaseType.Name.NS
	baseQualified := fmt.Sprintf("'{%s}%s'", baseTypeNS, baseTypeName)

	// Build map of base type's non-prohibited attributes, keyed by the full
	// QName so an unqualified derived attribute does not collide with a
	// namespaced base attribute that shares its local name.
	baseAttrs := make(map[QName]*AttrUse, len(td.BaseType.Attributes))
	for _, au := range td.BaseType.Attributes {
		if !au.Prohibited {
			baseAttrs[au.Name] = au
		}
	}

	// Check each derived non-prohibited attribute against the base.
	for _, au := range td.Attributes {
		if au.Prohibited {
			continue
		}
		baseAU, found := baseAttrs[au.Name]
		if found {
			// Check use consistency: optional cannot restrict required.
			if baseAU.Required && !au.Required {
				msg := fmt.Sprintf("The 'optional' attribute use is inconsistent with the corresponding 'required' attribute use of the base complex type definition %s.", baseQualified)
				c.schemaError(ctx, schemaComponentError(file, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}
			// XSD 1.1 derivation-ok-restriction: a restricting attribute use must
			// keep the base use's {inheritable} (true→false and false→true both fail).
			if c.version == Version11 && au.Inheritable != baseAU.Inheritable {
				msg := fmt.Sprintf("The 'inheritable' property of the attribute use '%s' is inconsistent with the corresponding attribute use of the base complex type definition %s.", au.Name.Local, baseQualified)
				c.schemaError(ctx, schemaComponentError(file, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}
		} else if td.BaseType.AnyAttribute == nil || !wildcardAllowsExpandedName(td.BaseType.AnyAttribute, au.Name.Local, au.Name.NS, c.schema, true) {
			// No matching attribute, and no base wildcard that ADMITS this derived
			// attribute's expanded name — the full test honors the base wildcard's
			// notNamespace/notQName/##defined, not just its namespace constraint, so
			// a derived attribute the base wildcard excludes by name is rejected.
			msg := fmt.Sprintf("Neither a matching attribute use, nor a matching wildcard exists in the base complex type definition %s.", baseQualified)
			c.schemaError(ctx, schemaComponentError(file, src.line, "complexType",
				component+", attribute use '"+au.Name.Local+"'", msg))
		}
	}

	// Check that all required base attributes have a matching non-prohibited derived attribute.
	derivedAttrs := make(map[QName]*AttrUse, len(td.Attributes))
	for _, au := range td.Attributes {
		derivedAttrs[au.Name] = au
	}
	for _, baseAU := range td.BaseType.Attributes {
		if !baseAU.Required {
			continue
		}
		derived, found := derivedAttrs[baseAU.Name]
		// XSD restriction inherits a base attribute use that the derived type does
		// not redeclare (§3.4.2.2): an absent derived declaration is not "missing",
		// it carries the base use forward. In XSD 1.1 mode that inheritance is
		// honored, so only an explicit prohibition of a required base attribute is
		// an error. XSD 1.0 mode keeps its historical behavior byte-identical.
		if c.version == Version11 {
			if found && derived.Prohibited {
				msg := fmt.Sprintf("A matching attribute use for the 'required' attribute use '%s' of the base complex type definition %s is missing.", baseAU.Name.Local, baseQualified)
				c.schemaError(ctx, schemaComponentError(file, src.line, "complexType", component, msg))
			}
			continue
		}
		if !found || derived.Prohibited {
			msg := fmt.Sprintf("A matching attribute use for the 'required' attribute use '%s' of the base complex type definition %s is missing.", baseAU.Name.Local, baseQualified)
			c.schemaError(ctx, schemaComponentError(file, src.line, "complexType", component, msg))
		}
	}

	// derivation-ok-restriction 4: Wildcard checks.
	if td.AnyAttribute != nil {
		// 4.1: Base must also have a wildcard.
		if td.BaseType.AnyAttribute == nil {
			msg := fmt.Sprintf("The complex type definition has an attribute wildcard, but the base complex type definition %s does not have one.", baseQualified)
			c.schemaError(ctx, schemaComponentError(file, src.line, "complexType", component, msg))
		} else {
			// 4.2: Derived namespace must be subset of base namespace.
			if !wildcardConstraintSubset(td.AnyAttribute, td.BaseType.AnyAttribute, c.schema, true) {
				msg := fmt.Sprintf("The attribute wildcard is not a valid subset of the wildcard in the base complex type definition %s.", baseQualified)
				c.schemaError(ctx, schemaComponentError(file, src.line, "complexType", component, msg))
			}
			// 4.3: Derived processContents must be >= base strength (strict > lax > skip).
			// libxml2 attributes this error to the base type's source location.
			if processContentsStrength(td.AnyAttribute.ProcessContents) < processContentsStrength(td.BaseType.AnyAttribute.ProcessContents) {
				errLine := src.line
				errComponent := component
				errFile := file
				if baseSrc, ok := c.typeDefSources[td.BaseType]; ok {
					errLine = baseSrc.line
					// This error is attributed to the BASE type's location, so cite the
					// base type's declaring file too (it may live in a different
					// included/imported document than the derived type).
					errFile = c.diagSourceOrRecorded(baseSrc.source)
					if !baseSrc.isLocal {
						errComponent = "complex type '" + td.BaseType.Name.Local + "'"
					}
				}
				msg := fmt.Sprintf("The {process contents} of the attribute wildcard is weaker than the one in the base complex type definition %s.", baseQualified)
				c.schemaError(ctx, schemaComponentError(errFile, errLine, "complexType", errComponent, msg))
			}
		}
	}
}

// finalizeEffectiveAttrs computes the effective {attribute uses} of a derived
// complex type (XSD 1.1) TOPOLOGICALLY across BOTH extension and restriction
// derivations: the base is finalized FIRST (recursively, regardless of its
// derivation kind), then the derivation attribute check runs against td's OWN
// declarations and the now-finalized base, then td inherits every base use it
// does not redeclare. This makes the result independent of source order and, in
// particular, lets an extension of a restriction (or a restriction of an
// extension, at any depth) read a complete base attribute set — the round-3 merge
// only recursed through restriction bases and so dropped attributes that a base
// restriction had itself inherited.
//
// A derived declaration (including use="prohibited") overrides the base use of
// the same expanded QName; validation handles prohibited uses (rejecting a
// present instance attribute), so they are kept in the merged set. An attribute
// wildcard is inherited when the derived type declares none; an extension also
// UNIONS its own wildcard with the base's. merged memoizes completed types (each
// finalizes once); visiting guards a cyclic base chain (an invalid schema
// reported by the circular-type check) from infinite recursion.
func (c *compiler) finalizeEffectiveAttrs(ctx context.Context, td *TypeDef, merged, visiting map[*TypeDef]bool) {
	if td == nil || merged[td] {
		return
	}
	if visiting[td] {
		return // cyclic base chain; the circular-type check reports the error
	}
	visiting[td] = true
	base := td.BaseType
	if base != nil && base.Derivation != DerivationNone {
		c.finalizeEffectiveAttrs(ctx, base, merged, visiting)
	}
	delete(visiting, td)
	merged[td] = true

	if base == nil || td.Derivation == DerivationNone {
		return
	}

	// The base now carries its FINAL effective attribute set. Run the derivation
	// attribute check with td's OWN declarations against that finalized base BEFORE
	// inheriting (so the check sees the real derived-vs-base comparison and the
	// extension duplicate detection sees inherited base attrs), then merge.
	switch td.Derivation {
	case DerivationRestriction:
		c.checkRestrictionAttrs(ctx, td)
	case DerivationExtension:
		c.checkExtensionAttrDuplicates(ctx, td)
	}

	if len(base.Attributes) > 0 {
		derivedByName := make(map[QName]struct{}, len(td.Attributes))
		for _, au := range td.Attributes {
			derivedByName[au.Name] = struct{}{}
		}
		for _, bau := range base.Attributes {
			if _, redeclared := derivedByName[bau.Name]; redeclared {
				continue
			}
			td.Attributes = append(td.Attributes, bau)
			c.markDefaultAttrUse(td, c.defaultAttrUse(base, bau.Name))
		}
	}
	// anyAttribute: a restriction inherits the base wildcard when it declares none;
	// an extension additionally unions its own wildcard with the base's.
	switch {
	case td.AnyAttribute == nil:
		td.AnyAttribute = base.AnyAttribute
	case td.Derivation == DerivationExtension && base.AnyAttribute != nil:
		td.AnyAttribute = wildcardUnion(base.AnyAttribute, td.AnyAttribute, c.version)
	}
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
func wildcardUnion(w1, w2 *Wildcard, version Version) *Wildcard {
	// Route to the general constraint algebra for EVERY XSD 1.1 union, not just
	// those whose operands carry a notNamespace/notQName field. The 1.0 case
	// analysis below keys processContents on w1 and APPROXIMATES some namespace
	// unions (e.g. ##other|##local as ##any), both wrong for 1.1: the extension
	// union must take the DERIVED (w2) processContents (XSD 3.4.2), and an xs:all
	// restriction's base-wildcard union must compute the EXACT namespace set so a
	// target-namespace element is not falsely admitted. The 1.0 path is kept
	// STRICTLY for Version10 so existing goldens stay byte-identical; the
	// 1.1-field guard remains as a defensive fallback.
	if version == Version11 || wildcardHas11Fields(w1) || wildcardHas11Fields(w2) {
		return unionWildcards11(w1, w2)
	}

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
				c.schemaError(ctx, errStr)
				c.schemaError(ctx, errStr)
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
				c.schemaError(ctx, schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by extension is forbidden by the base type '"+td.BaseType.Name.Local+"'."))
			}
			if td.Derivation == DerivationRestriction && baseFinal&FinalRestriction != 0 {
				component := td.Name.Local
				if src.isLocal {
					component = componentLocalComplexType
				}
				c.schemaError(ctx, schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by restriction is forbidden by the base type '"+td.BaseType.Name.Local+"'."))
			}
		}

		// simpleType list: check if item type forbids list derivation.
		if td.Variety == TypeVarietyList && td.ItemType != nil && td.ItemType.Final&FinalList != 0 {
			c.schemaError(ctx, schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
				"Derivation by list is forbidden by the item type '"+td.ItemType.Name.Local+"'."))
		}
		// simpleType union: check if any member type forbids union derivation.
		if td.Variety == TypeVarietyUnion {
			for _, member := range td.MemberTypes {
				if member.Final&FinalUnion != 0 {
					c.schemaError(ctx, schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
						"Derivation by union is forbidden by the member type '"+member.Name.Local+"'."))
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
					c.schemaError(ctx, schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."))
				}
			}
			if head.Final&FinalRestriction != 0 && derivationUsesMethod(member.Type, head.Type, DerivationRestriction) {
				if src, ok := c.globalElemSources[member]; ok {
					c.schemaError(ctx, schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."))
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
func (c *compiler) resolveQName(ctx context.Context, elem *helium.Element, ref string) QName {
	local := ref
	ns := c.schema.targetNamespace

	for i := range len(ref) {
		if ref[i] == ':' {
			prefix := ref[:i]
			local = ref[i+1:]
			ns = lookupNS(elem, prefix)
			// A prefixed QName whose prefix is not bound in scope must be a fatal
			// schema error (src-resolve): otherwise it silently maps to the empty
			// namespace, letting an invalid schema compile and an unbound-prefix
			// typo resolve to an unrelated no-namespace declaration. lookupNS
			// always returns the XML namespace for the predeclared "xml" prefix,
			// so that case is never flagged here.
			if ns == "" && prefix != "" {
				c.reportUnboundQNamePrefix(ctx, elem, ref, prefix)
			}
			c.rejectDeprecatedDatatypeNamespace(ctx, elem, ref, ns)
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

	c.rejectDeprecatedDatatypeNamespace(ctx, elem, ref, ns)
	return QName{Local: local, NS: ns}
}

// rejectDeprecatedDatatypeNamespace reports the XSD 1.1 rule that schema QName
// references must not use the old XML Schema datatypes namespace.
func (c *compiler) rejectDeprecatedDatatypeNamespace(ctx context.Context, elem *helium.Element, ref, ns string) bool {
	if c.version != Version11 || ns != lexicon.NamespaceXSDDatatypes {
		return false
	}
	msg := fmt.Sprintf("The namespace '%s' used by QName value '%s' has been deprecated; use '%s' for XML Schema built-in datatypes.", ns, ref, lexicon.NamespaceXSD)
	c.schemaError(ctx,
		schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), "QName value", msg))
	return true
}

func (c *compiler) deprecatedDatatypeQName(qn QName) bool {
	return c.version == Version11 && qn.NS == lexicon.NamespaceXSDDatatypes
}

// reportUnboundQNamePrefix emits a fatal schema-compilation error for a prefixed
// QName-valued attribute (e.g. @type, @ref, @base, @itemType) whose prefix is not
// bound in scope. Mirrors the wording used for an unbound xs:keyref/@refer prefix.
func (c *compiler) reportUnboundQNamePrefix(ctx context.Context, elem *helium.Element, ref, prefix string) {
	if c.filename == "" {
		return
	}
	msg := fmt.Sprintf("The QName value '%s' uses the namespace prefix '%s', which is not bound to a namespace.", ref, prefix)
	c.schemaError(ctx,
		schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), "QName value", msg))
}

// reportInvalidQNameValue emits a fatal schema-compilation error for a
// QName-valued attribute whose value is not a lexically valid xs:QName (e.g. a
// leading colon like ":u"). Without this such a value would slip past the
// prefix-resolution path (strings.Cut yields an empty prefix that bypasses the
// unbound-prefix check) and resolve as an unprefixed reference.
func (c *compiler) reportInvalidQNameValue(ctx context.Context, elem *helium.Element, ref string) {
	if c.filename == "" {
		return
	}
	msg := fmt.Sprintf("The QName value '%s' is not a valid QName.", ref)
	c.schemaError(ctx,
		schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), "QName value", msg))
}
