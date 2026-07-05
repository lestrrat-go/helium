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
	// Snapshot the REAL (loaded) element and type components before the loops
	// below install any recovery placeholders for dangling refs, so the
	// non-imported-namespace check can tell a reference that resolves to a
	// genuine component from one that merely got a placeholder.
	realElems := make(map[QName]struct{}, len(c.schema.elements))
	for qn := range c.schema.elements {
		realElems[qn] = struct{}{}
	}
	realTypes := make(map[QName]struct{}, len(c.schema.types))
	for qn := range c.schema.types {
		realTypes[qn] = struct{}{}
	}
	for range 2 {
		for edecl, qn := range c.elemRefs {
			if edecl.Type != nil {
				continue
			}
			// First check if this is a reference to a global element.
			// Skip self-referencing elements (where the element name matches
			// the type name, e.g., <xs:element name="X" type="X"/>); these
			// should resolve against the type map instead.
			ge, ok := c.schema.elements[qn]
			if _, eligible := c.chameleonEligible[edecl]; !ok && edecl.IsRef && eligible {
				// A chameleon-eligible ref (unprefixed, no in-scope default
				// namespace) resolves to the schema's targetNamespace, but the
				// referenced global element may come from a no-targetNamespace
				// imported schema. Mirror the empty-namespace fallback used for
				// type and attribute-group references so such a valid reference
				// resolves. A prefixed ref or one bound by an in-scope default
				// namespace is NOT eligible, so a genuine unresolved reference
				// still reports rather than silently resolving to {}local.
				ge, ok = c.schema.elements[QName{Local: qn.Local}]
			}
			// A type= reference (edecl.IsRef==false) names the TYPE symbol space, so a
			// same-named TYPE takes precedence over a like-named GLOBAL ELEMENT:
			// <xs:element type="foo"> must adopt complexType foo, not a colliding
			// global element foo (W3C particlesL032). Only when NO such type exists
			// does helium leniently fall back to the element's type via the block
			// below. A ref= reference always uses the element symbol space.
			if ok && ge != edecl && !edecl.IsRef && c.typeExistsForRef(edecl, qn) {
				ok = false
			}
			if ok && ge != edecl {
				edecl.Type = ge.Type
				if edecl.IsRef {
					// Adopt the resolved global element's expanded name so the
					// content model matches instance children by the referenced
					// element's ACTUAL namespace. This corrects the empty-namespace
					// fallback above, where the ref's name was resolved to the
					// schema targetNamespace but the global lives in the absent
					// namespace of an imported no-targetNamespace schema. For an
					// ordinary ref the resolved name already equals edecl.Name, so
					// this is a no-op.
					edecl.Name = ge.Name
				}
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
				if len(edecl.substitutionGroupHeads()) == 0 {
					edecl.setSubstitutionGroupHeads(ge.substitutionGroupHeads())
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
					c.schemaError(ctx, schemaElemDeclErrorAttr(c.diagSourceOrRecorded(src.source), src.line, src.elemName, msg))
				}
				td = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				c.schema.types[qn] = td
			}
			edecl.Type = td
		}
	}

	// Reject an entry-document element @type/@ref into a namespace the entry
	// document did not directly import (src-resolve §3.3.2).
	c.checkNonImportedNamespaceRefs(ctx, realElems, realTypes)

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
			if c.recoveryBaseTypes == nil {
				c.recoveryBaseTypes = make(map[*TypeDef]bool)
			}
			c.recoveryBaseTypes[base] = true
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

	// Enforce src-ct.2 (Complex Type Definition Representation OK): a
	// <xs:simpleContent> derivation's base type must be of the right kind. Runs
	// after the base types above are resolved.
	c.checkSimpleContentBase(ctx)

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
	// Detect and CUT circular model-group references (a named xs:group that
	// references itself, directly or transitively) BEFORE the resolution below
	// shares group content slices — a self-referential group would otherwise make
	// the resolved content-model tree cyclic and overflow the stack in the
	// downstream walks (UPA, element-consistency, open content). The back-edge
	// placeholder is left empty so the tree stays acyclic; the schema is reported
	// invalid (circular groups are forbidden in both XSD 1.0 and 1.1).
	cutGroupRefs := c.checkCircularGroupRefs(ctx)
	// Collect every <xs:group ref="..."> that does NOT resolve to a globally
	// declared model group (a ref naming a component in the wrong symbol space —
	// a complexType or attributeGroup of the same name — or a name declared
	// nowhere). Reported after the loop, sorted for deterministic output, so the
	// invalid schema is rejected instead of silently compiling with empty content.
	type danglingGroupRef struct {
		source string
		line   int
		local  string
		qn     QName
	}
	var danglingGroups []danglingGroupRef
	for _, placeholder := range groupRefPlaceholders {
		if _, isCut := cutGroupRefs[placeholder]; isCut {
			continue
		}
		qn := c.groupRefs[placeholder]
		grp, ok := c.schema.groups[qn]
		if !ok && qn.NS != "" {
			// An unprefixed ref resolves to the schema's targetNamespace, but the
			// group may come from a chameleon / no-namespace imported schema. Mirror
			// the empty-namespace fallback used for element/type/attribute-group
			// references so such a valid reference is not flagged dangling.
			grp, ok = c.schema.groups[QName{Local: qn.Local}]
		}
		if !ok {
			if c.filename != "" {
				src := c.groupRefSources[placeholder]
				danglingGroups = append(danglingGroups, danglingGroupRef{
					source: c.diagSourceOrRecorded(src.source),
					line:   src.line,
					local:  src.local,
					qn:     qn,
				})
			}
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
			// XSD 1.1: a group reference nested directly inside an xs:all must resolve
			// to an 'all' model group; a referenced sequence/choice group is invalid.
			if c.version == Version11 {
				if src := c.groupRefSources[placeholder]; src.nested && src.parentCompositor == CompositorAll {
					file := c.diagSourceOrRecorded(src.source)
					c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
						"A reference within an 'all' model group must resolve to an 'all' model group."))
				}
			}
			continue
		}
		c.checkAllGroupRef(ctx, placeholder)
	}

	// Report unresolved model-group references (src-resolve / Model Group
	// Reference Representation OK): a ref to a component in the wrong symbol space
	// (a complexType or attributeGroup of the same name) or a name declared
	// nowhere. Version-independent — the resolution rule holds in both XSD 1.0 and
	// 1.1 (a permitted 1.1 circular ref still resolves to an existing group, so it
	// never reaches here). Sorted by (source, line, local) so the output is
	// independent of Go map iteration order.
	sort.Slice(danglingGroups, func(i, j int) bool {
		if danglingGroups[i].source != danglingGroups[j].source {
			return danglingGroups[i].source < danglingGroups[j].source
		}
		if danglingGroups[i].line != danglingGroups[j].line {
			return danglingGroups[i].line < danglingGroups[j].line
		}
		return danglingGroups[i].local < danglingGroups[j].local
	})
	for _, d := range danglingGroups {
		msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) model group definition.", d.qn.NS, d.qn.Local)
		c.schemaError(ctx, schemaParserErrorAttr(d.source, d.line, d.local, elemGroup, attrRef, msg))
	}

	// Reject an xs:attributeGroup ref that does not resolve to a globally-declared
	// attribute group (src-resolve / Attribute Group Definition Representation OK
	// 3): a ref naming a component in the wrong symbol space (a complexType or a
	// global attribute of the same name), a name that is declared nowhere, or an
	// empty/absent ref value. Reporting it here — before the flatten/expand walks,
	// which silently contribute nothing for a missing group — stops such an invalid
	// schema from compiling. Version-independent: the resolution rule holds in both
	// XSD 1.0 and 1.1 (a CIRCULAR ref, permitted in 1.1, still resolves to an
	// EXISTING group and so is unaffected).
	c.checkAttrGroupRefsResolve(ctx)

	// Reject an <xs:attribute> whose @type resolves to a complex type or whose @ref
	// resolves to a non-attribute component (src-resolve / §3.2.2). Both are
	// component-kind mis-resolutions the lexical QName check cannot catch, and both
	// are version-independent.
	c.checkAttributeResolution(ctx)

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
	//
	// au-props-correct.3 conflict diagnostics are collected here and emitted AFTER
	// the loop in a deterministic order. The copy steps below are per-use and
	// order-independent, but reporting inline while iterating the randomized
	// c.attrRefs map would surface multiple conflicting refs in nondeterministic
	// order; collect first, then sort by recorded source line/local (matching
	// checkAttrUseConstraints) before reporting.
	type attrRefConflict struct {
		au *AttrUse
		ga *AttrUse
		qn QName
	}
	var conflicts []attrRefConflict
	for au, qn := range c.attrRefs {
		ga, ok := c.schema.globalAttrs[qn]
		if !ok {
			// XSD 1.1: a ref to one of the four xsi: processor attributes resolves to
			// no user-declared global attribute, but the attribute has a FIXED
			// built-in type. Associate that built-in type (xsi:type→xs:QName,
			// xsi:nil→xs:boolean, xsi:noNamespaceSchemaLocation→xs:anyURI) so a
			// fixed/default constraint is validated for validity at compile time
			// (checkAttrUseConstraints) and compared in VALUE space at runtime
			// (fixedValueMatches) — instead of falling back to raw string equality
			// against a nil type. xsi:schemaLocation (a list of xs:anyURI) has no
			// scalar built-in, so its even-pair value validity stays with
			// validateDeclaredXsiAttrValue.
			if c.version == Version11 && qn.NS == lexicon.NamespaceXSI &&
				au.Type == nil && au.TypeName == (QName{}) {
				if bt, found := xsiProcessorAttrBuiltinType(qn.Local); found {
					au.TypeName = bt
				}
			}
			// XSD 1.0: a ref to one of the implicitly-available XML-namespace
			// attributes (xml:base/xml:lang/xml:space/xml:id) resolves to no
			// user-declared global attribute, but the attribute has a fixed built-in
			// type per the standard xml: namespace schema. Associate that type so a
			// DECLARED xml: use validates its value at instance time
			// (validateAttributes' declaredXML path) — e.g. xml:space="bogus" is
			// rejected — and a declared xml:id, now typed as xs:ID, participates in the
			// document-wide ID uniqueness/integrity pass. Scoped to Version10, where
			// xml: attributes are otherwise special-skipped and the declaredXML path
			// runs. XSD 1.1 is DELIBERATELY left byte-identical to origin: a 1.1
			// declared xml: ref stays UNTYPED (its value is not validated and xml:id ID
			// integrity is not applied) — declared XML-namespace-attribute value
			// validation in 1.1 is a deferred gap, out of scope for this 1.0-focused
			// path and avoided to prevent any 1.1 regression.
			if c.version != Version11 && qn.NS == lexicon.NamespaceXML &&
				au.Type == nil && au.TypeName == (QName{}) {
				if t := xmlNamespaceAttrType(qn.Local, c.schema); t != nil {
					au.Type = t
				}
			}
			continue
		}
		// A use="prohibited" ref corresponds to NO attribute-use component (XSD 1.0
		// §3.2.2): the attribute is REMOVED, so its (harmless) local fixed/default is
		// never compared with the referenced global's 'fixed' (au-props-correct.3
		// does not apply) and it inherits no value constraint. A prohibited ref needs
		// only its QName as an internal blocker — to forbid the attribute — so skip
		// it here. (The distinct compile-time rule that 'default' requires
		// use="optional" is enforced separately in checkAttributeUse and still
		// rejects a prohibited ref carrying a default.)
		if au.Prohibited {
			continue
		}
		// au-props-correct.3: if the referenced global declaration carries a
		// 'fixed' value constraint and this use declares its OWN value constraint,
		// the use's constraint must ALSO be 'fixed' and value-equal to the
		// declaration's. A local 'default', or a 'fixed' with a different value,
		// would let the use admit values the declaration pins, so it is rejected.
		// Enforced for EVERY referencing use (not only inside a restriction), so a
		// plain complexType with <xs:attribute ref="t:a" default="2"/> against a
		// fixed t:a is caught — the derivation-ok-restriction check only covers the
		// derived-vs-base relationship.
		if ga.Fixed != nil && (au.Default != nil || au.Fixed != nil) {
			conflicts = append(conflicts, attrRefConflict{au: au, ga: ga, qn: qn})
		}
		// A use inherits the declaration's value constraint ONLY when it has no
		// LOCAL value constraint of its own. A local 'default' must not be
		// overwritten by — nor silently absorb — the declaration's 'fixed': the
		// use's effective constraint stays its local 'default', so a derived use
		// like <xs:attribute ref="t:a" default="2"/> does NOT satisfy a base
		// 'fixed' constraint (au-props-correct.2 / derivation-ok-restriction).
		if au.Default == nil && au.Fixed == nil {
			au.Default = ga.Default
			au.DefaultNS = ga.DefaultNS
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
	// Sort key mirrors checkAttrRefFixedConflict's own source resolution (recorded
	// line/local, falling back to qn.Local), so diagnostics report in stable
	// document order regardless of map iteration order.
	conflictSortKey := func(au *AttrUse, qn QName) (int, string) {
		if src, ok := c.attrUseConstraintSources[au]; ok {
			return src.line, src.local
		}
		return 0, qn.Local
	}
	sort.Slice(conflicts, func(i, j int) bool {
		li, loci := conflictSortKey(conflicts[i].au, conflicts[i].qn)
		lj, locj := conflictSortKey(conflicts[j].au, conflicts[j].qn)
		if li != lj {
			return li < lj
		}
		return loci < locj
	})
	for _, cf := range conflicts {
		c.checkAttrRefFixedConflict(ctx, cf.au, cf.ga, cf.qn)
	}

	// Validate attribute default/fixed constraint values against the
	// attribute's declared simple type now that all type refs are resolved.
	// A retained-but-invalid constraint (e.g. an empty default="" on an
	// xs:integer attribute) is a schema error; catching it here avoids
	// injecting an invalid value into the instance during validation.
	c.checkAttrUseConstraints(ctx)

	// Validate element-declaration default/fixed constraint values against the
	// element's simple (content) type now that all type refs are resolved. §3.3.6
	// Element Default Valid is version-independent, so this runs in both XSD 1.0
	// and 1.1 (sources are recorded unconditionally).
	c.checkElementDeclConstraints(ctx)

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
		// XSD 1.1 open-content inheritance/merge across extension (and its mode-
		// tightening validity) is handled centrally by resolveOpenContent, AFTER the
		// content models are merged and the effective content type (incl. the per-
		// document <xs:defaultOpenContent>) is known.
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
		// §3.4.6.2 (Derivation Valid (Extension), cos-ct-extends 1.4.3.2.2.1): when both
		// the base and the derived type have complex content (element-only or mixed),
		// they must agree on mixedness — both mixed or both element-only. (An empty base
		// or derived particle is exempt: the extension may introduce content of either
		// flavor.) This is a version-INDEPENDENT XSD rule (enforced in both 1.0 and 1.1).
		{
			baseHasContent := td.BaseType.ContentType == ContentTypeElementOnly || td.BaseType.ContentType == ContentTypeMixed
			derivedHasContent := td.ContentType == ContentTypeElementOnly || td.ContentType == ContentTypeMixed
			baseMixed := td.BaseType.ContentType == ContentTypeMixed
			derivedMixed := td.ContentType == ContentTypeMixed
			if baseHasContent && derivedHasContent && baseMixed != derivedMixed {
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
		allExtErr := func() {
			if src, ok := c.typeDefSources[td]; ok && c.filename != "" {
				component := componentLocalComplexType
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component,
					"The 'all' model group needs to be the only child of the model group."))
			}
		}
		switch {
		case c.version == Version11 && baseMG != nil && derivedMG != nil &&
			(baseMG.Compositor == CompositorAll || derivedMG.Compositor == CompositorAll):
			// XSD 1.1 relaxes cos-all-limited: an xs:all may be extended by another
			// xs:all, merging into a SINGLE all group whose members are the union of
			// the base's and extension's. Both content models must be all groups (an
			// xs:sequence/xs:choice extending an all, or an all extending a
			// sequence/choice, remains invalid) with the SAME minOccurs (§3.4.2.2).
			if baseMG.Compositor != CompositorAll || derivedMG.Compositor != CompositorAll ||
				derivedMG.MaxOccurs == 0 || baseMG.MinOccurs != derivedMG.MinOccurs {
				allExtErr()
				continue
			}
			// bug 6202 (RESOLVED WONTFIX 2008-11-21): extending an empty *mixed*
			// base 'all' with another 'all' is invalid. A mixed base has a
			// non-absent content type even when its 'all' is empty, so §3.4.2.2
			// sequences the base and derived particles — producing an 'all' nested
			// in a sequence, which All Group Limited (§3.8.6.2) forbids. An empty
			// *non-mixed* base is genuinely empty content, so its extension by an
			// 'all' stays valid (the non-orthogonality the WG declined to fix).
			if !modelGroupHasContent(baseMG) && td.BaseType.ContentType == ContentTypeMixed {
				allExtErr()
				continue
			}
			merged := make([]*Particle, 0, len(baseMG.Particles)+len(derivedMG.Particles))
			merged = append(merged, baseMG.Particles...)
			merged = append(merged, derivedMG.Particles...)
			td.ContentModel = &ModelGroup{
				Compositor: CompositorAll,
				MinOccurs:  baseMG.MinOccurs,
				MaxOccurs:  1,
				Particles:  merged,
			}
		case baseMG != nil && derivedMG != nil && derivedMG.MaxOccurs != 0 && derivedMG.Compositor == CompositorAll && modelGroupHasContent(baseMG):
			// cos-all-limited.1.2 / §3.8.2 (XSD 1.0, or 1.1 extension of a non-all
			// base): an 'all' model group may only constitute the WHOLE content of a
			// type. Appending an 'all' onto a non-empty base would build a sequence
			// CONTAINING an 'all' group, which is forbidden.
			allExtErr()
			continue
		case baseMG != nil && derivedMG != nil:
			// Merge: create a sequence of base content + derived content.
			td.ContentModel = &ModelGroup{
				Compositor: CompositorSequence,
				MinOccurs:  1,
				MaxOccurs:  1,
				Particles: []*Particle{
					{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG},
					{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG},
				},
			}
		case baseMG != nil:
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
		// §3.4.6.4 (Derivation Valid (Restriction, Complex), cos-ct-restricts clause
		// 5.3.2): when both the base and the derived type have complex content, a
		// mixed derived content type requires the base to be mixed too (clause 5.3.2.1),
		// while an element-only derived type is always allowed (clause 5.3.2.2). Unlike
		// the SYMMETRIC extension rule (cos-ct-extends), restriction is ASYMMETRIC: a
		// mixed base MAY be restricted to element-only, but an element-only base may NOT
		// be restricted to mixed. So the only forbidden mixedness transition is
		// element-only base → mixed derived. Version-INDEPENDENT (§3.4.6.4 is not
		// version-specific), enforced in both XSD 1.0 and 1.1.
		{
			baseHasContent := td.BaseType.ContentType == ContentTypeElementOnly || td.BaseType.ContentType == ContentTypeMixed
			derivedHasContent := td.ContentType == ContentTypeElementOnly || td.ContentType == ContentTypeMixed
			baseMixed := td.BaseType.ContentType == ContentTypeMixed
			derivedMixed := td.ContentType == ContentTypeMixed
			if baseHasContent && derivedHasContent && derivedMixed && !baseMixed {
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

	// XSD 1.1: resolve each complex type's effective {open content} — fold in the
	// per-document <xs:defaultOpenContent>, inherit/merge across extension, and
	// check restriction-derivation validity. Runs after content models and content
	// types are finalized so the appliesToEmpty/empty-content decisions are correct.
	c.resolveOpenContent(ctx)

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

// checkSimpleContentBase enforces src-ct.2 (Complex Type Definition
// Representation OK, XSD §3.4.2): when a complex type uses <xs:simpleContent>,
// the base type resolved by the derivation's @base must have the right KIND and
// content. This is version-INDEPENDENT — the same constraint holds in XSD 1.0
// and 1.1 — so it runs in both. Called from resolveRefs after base types are
// resolved.
//
//	extension:   base must be a simple type, OR a complex type whose {content
//	             type} is a simple type. (clauses 2.1, 2.2)
//	restriction: base must be a complex type whose {content type} is a simple
//	             type, OR a complex type whose content is mixed with an emptiable
//	             particle AND the restriction carries a nested <xs:simpleType>.
//	             A simple-type base is invalid for a restriction — clause 2.2 is
//	             extension-only. (clauses 2.1, 2.3)
func (c *compiler) checkSimpleContentBase(ctx context.Context) {
	if c.filename == "" {
		return
	}
	tds := make([]*TypeDef, 0, len(c.typeRefs))
	for td := range c.typeRefs {
		if td.IsSimpleContent && td.BaseType != nil {
			tds = append(tds, td)
		}
	}
	sort.Slice(tds, func(i, j int) bool {
		si, sj := c.typeDefSources[tds[i]], c.typeDefSources[tds[j]]
		if si.line != sj.line {
			return si.line < sj.line
		}
		return si.ordinal < sj.ordinal
	})
	for _, td := range tds {
		base := td.BaseType
		if c.recoveryBaseTypes[base] {
			continue // base ref did not resolve; already reported
		}
		baseIsSimpleType := !base.IsComplex
		baseIsSimpleContent := base.IsComplex && base.ContentType == ContentTypeSimple
		var ok bool
		switch td.Derivation {
		case DerivationExtension:
			ok = baseIsSimpleType || baseIsSimpleContent
		case DerivationRestriction:
			switch {
			case baseIsSimpleContent:
				ok = true
			case base.IsComplex && base.ContentType == ContentTypeMixed &&
				contentModelEmptiable(base.ContentModel) && td.scHasSimpleTypeChild:
				ok = true
			case base.Name.NS == lexicon.NamespaceXSD && base.Name.Local == typeAnySimpleType:
				// xs:anySimpleType is the simple ur-type; a simpleContent restriction
				// may base on it and define a fresh simple content via a nested
				// <xs:simpleType>. (A non-narrowing restriction that leaves the content
				// as xs:anySimpleType is rejected separately by checkAnySimpleTypeUsage
				// in 1.1.)
				ok = true
			}
		}
		if ok {
			continue
		}
		src, srcOK := c.typeDefSources[td]
		if !srcOK {
			continue
		}
		component := componentLocalComplexType
		if !src.isLocal {
			component = "complex type '" + td.Name.Local + "'"
		}
		var msg string
		if td.Derivation == DerivationExtension {
			msg = "The base type of a 'simpleContent' extension must be a simple type or a complex type with simple content."
		} else {
			msg = "The base type of a 'simpleContent' restriction must be a complex type with simple content."
		}
		c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg))
	}
}

// contentModelEmptiable reports whether a complex type's content model can match
// the empty sequence (src-ct.2.3 "emptiable particle"). A nil model group (e.g.
// xs:anyType, or an attribute-only content type) is emptiable.
func contentModelEmptiable(mg *ModelGroup) bool {
	if mg == nil {
		return true
	}
	return particleEmptiable(&Particle{Term: mg, MinOccurs: mg.MinOccurs, MaxOccurs: mg.MaxOccurs})
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

// checkAttrGroupRefsResolve reports every <xs:attributeGroup ref="..."> — whether
// a nested child of a global attribute group (c.attrGroupRefChildren) or a member
// of a complex type / derivation body (c.attrGroupRefs) — whose ref does NOT
// resolve to a globally-declared attribute group (src-resolve). This covers a ref
// naming a component in a different symbol space (a complexType or a global
// attribute of the same name), a name declared nowhere, and an empty/absent ref
// value. Every valid reference (including a permitted 1.1 circular one, which
// resolves to an existing group) is left untouched.
//
// Diagnostics are collected and sorted by (source, line, local) so the output is
// independent of Go map iteration order.
func (c *compiler) checkAttrGroupRefsResolve(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type danglingRef struct {
		source    string
		line      int
		elemLocal string
		qn        QName
	}
	var dangling []danglingRef

	report := func(src, elemLocal string, line int, qn QName) {
		if _, ok := c.schema.attrGroups[qn]; ok {
			return
		}
		// An unprefixed ref resolves to the schema's targetNamespace, but the group
		// may come from an imported schema that has NO targetNamespace (a chameleon /
		// no-namespace import). Mirror the empty-namespace fallback used for element
		// and type references so such a valid reference is not flagged dangling.
		if qn.NS != "" {
			if _, ok := c.schema.attrGroups[QName{Local: qn.Local}]; ok {
				return
			}
		}
		dangling = append(dangling, danglingRef{
			source:    c.diagSourceOrRecorded(src),
			line:      line,
			elemLocal: elemLocal,
			qn:        qn,
		})
	}

	// Nested refs inside a global attribute group definition.
	for ownerQN, children := range c.attrGroupRefChildren {
		srcs := c.attrGroupRefSources[ownerQN]
		for i, refQN := range children {
			src := ""
			line := 0
			if i < len(srcs) {
				src = srcs[i].source
				line = srcs[i].line
			}
			report(src, elemAttributeGroup, line, refQN)
		}
	}

	// Refs on a complex type / derivation body. The implicit XSD 1.1
	// @defaultAttributes ref is resolved separately (checkSchemaDefaultAttributes),
	// so skip it here.
	for td, qns := range c.attrGroupRefs {
		srcs := c.attrGroupRefUseSources[td]
		for i, refQN := range qns {
			if i < len(srcs) && srcs[i].attr == attrDefaultAttributes {
				continue
			}
			src := ""
			line := 0
			elemLocal := elemAttributeGroup
			if i < len(srcs) {
				src = srcs[i].source
				line = srcs[i].line
				if srcs[i].elemLocal != "" {
					elemLocal = srcs[i].elemLocal
				}
			}
			report(src, elemLocal, line, refQN)
		}
	}

	sort.Slice(dangling, func(i, j int) bool {
		if dangling[i].source != dangling[j].source {
			return dangling[i].source < dangling[j].source
		}
		if dangling[i].line != dangling[j].line {
			return dangling[i].line < dangling[j].line
		}
		return dangling[i].elemLocal < dangling[j].elemLocal
	})
	for _, d := range dangling {
		msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) attribute group definition.", d.qn.NS, d.qn.Local)
		c.schemaError(ctx, schemaParserErrorAttr(d.source, d.line, d.elemLocal, elemAttributeGroup, attrRef, msg))
	}
}

// checkAttributeResolution enforces the two component-kind resolution rules on
// an <xs:attribute> (§3.2.2 / src-resolve), version-INDEPENDENT so it runs in
// BOTH XSD 1.0 and 1.1:
//
//   - the {type definition} named by @type must be a SIMPLE type — a @type that
//     resolves to a complexType (including the ur-type xs:anyType) is a schema
//     error (an attribute cannot have complex content); and
//   - a @ref must resolve to a globally-declared ATTRIBUTE — a ref naming a
//     component in a DIFFERENT symbol space (an attributeGroup, a complexType, a
//     global element) or a name declared nowhere is a schema error.
//
// Only actual mis-resolutions are reported: a @type that resolves to a simple
// type (built-in or user), an inline anonymous <xs:simpleType> (au.Type, no
// TypeName), and a @ref to a real global attribute all pass untouched. A @type
// that does not resolve at all is left to the existing (missing-type) handling —
// this check only ADDS the complex-type rejection. In XSD 1.1 a @ref to one of
// the four reserved xsi: processor attributes resolves to no user-declared global
// attribute but is legitimate, so it is exempt.
//
// Diagnostics are collected and sorted by (source, line, local) so the output is
// independent of Go map iteration order.
func (c *compiler) checkAttributeResolution(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type issue struct {
		source string
		line   int
		local  string
		qn     QName
		msg    string
	}
	var issues []issue

	// Part A: @type must resolve to a simple type. attrUseSources tracks every
	// non-ref attribute use (a ref use carries no explicit @type here — its
	// TypeName is copied from the referenced global, which is always simple).
	for au, src := range c.attrUseSources {
		if au.TypeName == (QName{}) {
			continue
		}
		td := c.resolveNamedType(au.TypeName)
		if td == nil || !td.IsComplex {
			continue
		}
		issues = append(issues, issue{
			source: c.diagSourceOrRecorded(src.source),
			line:   src.line,
			local:  src.local,
			qn:     au.TypeName,
			msg: fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) simple type definition.",
				au.TypeName.NS, au.TypeName.Local),
		})
	}

	// Part B: @ref must resolve to a global attribute declaration.
	for au, qn := range c.attrRefs {
		if _, ok := c.schema.globalAttrs[qn]; ok {
			continue
		}
		// An unprefixed ref resolves to the schema's targetNamespace, but the global
		// attribute may come from a no-targetNamespace (chameleon) imported schema;
		// mirror the empty-namespace fallback used for element/type/attributeGroup
		// references so a valid no-NS-import reference is not flagged.
		if qn.NS != "" {
			if _, ok := c.schema.globalAttrs[QName{Local: qn.Local}]; ok {
				continue
			}
		}
		// A ref into the reserved XSI namespace never resolves to a user-declared
		// global attribute (a schema may not declare an attribute there): the four
		// processor attributes are provided implicitly, and any other xsi: local name
		// is tolerated as a skipped special attribute (a required use of it is instead
		// left unsatisfied at instance validation). Exempt the whole namespace so
		// neither form is reported as an unresolvable ref.
		// The XSI namespace is reserved to the four processor attributes and the XML
		// namespace provides the built-in xml:lang/base/space/id attributes, which are
		// always implicitly available and never declared: a ref into either resolves to
		// no user-declared global attribute by design, so neither is reported.
		if qn.NS == lexicon.NamespaceXSI || qn.NS == lexicon.NamespaceXML {
			continue
		}
		// A ref that resolves to an EXISTING component of the WRONG symbol space (a
		// named type, a global element, or an attribute group) is always a src-resolve
		// error. A ref that resolves to NOTHING is left lenient ONLY when its namespace
		// was actually imported (`<xs:import namespace="...">`) but its schema document
		// could not be loaded — the namespace's declarations are then genuinely unknown,
		// so flagging the ref would over-reject valid schemas (matching libxml2). A ref
		// into a namespace that was NEVER imported — including the schema's own target
		// namespace and the absent namespace — is an error: referencing a component of a
		// non-imported namespace is illegal (src-resolve), and a self-reference should
		// have been declared. The built-in XML namespace is exempted above.
		if _, imported := c.importDeclaredNS[qn.NS]; imported && !c.qnameNamesNonAttribute(qn) {
			continue
		}
		src := c.attrRefSources[au]
		issues = append(issues, issue{
			source: c.diagSourceOrRecorded(src.source),
			line:   src.line,
			local:  src.local,
			qn:     qn,
			msg: fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) attribute declaration.",
				qn.NS, qn.Local),
		})
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].source != issues[j].source {
			return issues[i].source < issues[j].source
		}
		if issues[i].line != issues[j].line {
			return issues[i].line < issues[j].line
		}
		return issues[i].local < issues[j].local
	})
	for _, is := range issues {
		c.schemaError(ctx, schemaParserErrorAttr(is.source, is.line, elemAttribute, elemAttribute, is.local, is.msg))
	}
}

// qnameNamesNonAttribute reports whether qn names an existing schema component
// that is NOT a global attribute — a named type, a global element, or an
// attribute group. It is used by the @ref resolution check to distinguish a
// wrong-symbol-space reference (a src-resolve error) from one that resolves to
// nothing (left lenient). Only the exact {ns}local lookup is consulted (no
// chameleon empty-namespace fallback), so a reference that dangles into an
// unloaded namespace is never treated as wrong-kind.
func (c *compiler) qnameNamesNonAttribute(qn QName) bool {
	if _, ok := c.schema.types[qn]; ok {
		return true
	}
	if _, ok := c.schema.elements[qn]; ok {
		return true
	}
	if _, ok := c.schema.attrGroups[qn]; ok {
		return true
	}
	return false
}

// checkNonImportedNamespaceRefs enforces src-resolve §3.3.2: a reference to a
// component of a namespace that the referencing schema document did not DIRECTLY
// import is illegal, even when that namespace's components happen to be present
// in the assembly because ANOTHER document imported it (transitive import does
// not make a namespace referenceable — W3C Element_w3c/Schema_w3c elemZ006/Z007,
// schZ004/Z005). The check is version-INDEPENDENT.
//
// It is deliberately CONSERVATIVE to avoid over-rejecting valid multi-file
// schemas, and only fires for a reference that:
//   - originates in the ENTRY document (its recorded source is c.filename) — a
//     reference inside an imported/included sub-document is left alone (that
//     document has its own import context we do not re-check here);
//   - resolves to a REAL loaded component (in realElems/realTypes), not a
//     placeholder installed for a dangling ref (those keep the existing
//     "does not resolve" handling and its import-declared leniency);
//   - targets a namespace that genuinely requires an import — not the absent
//     namespace, not the XML/XSI built-in namespaces, and not the entry
//     document's own targetNamespace;
//   - targets a namespace that WAS import-declared somewhere in the assembly
//     (importDeclaredNS) yet was NOT directly imported by the entry document
//     (docImportedNS[c.filename]).
//
// From the entry document's perspective the QName does not resolve, so it
// reuses the same "does not resolve to a(n) …" diagnostic as a truly-dangling
// ref. Diagnostics are sorted by (source, line, local) for deterministic output.
func (c *compiler) checkNonImportedNamespaceRefs(ctx context.Context, realElems, realTypes map[QName]struct{}) {
	if c.filename == "" {
		return
	}

	type issue struct {
		line  int
		local string
		isRef bool
		src   elemRefSource
		qn    QName
	}
	var issues []issue

	for edecl, qn := range c.elemRefs {
		src, ok := c.elemRefSources[edecl]
		if !ok {
			continue
		}
		if c.diagSourceOrRecorded(src.source) != c.filename {
			continue
		}
		if qn.NS == "" || qn.NS == lexicon.NamespaceXML || qn.NS == lexicon.NamespaceXSI {
			continue
		}
		if qn.NS == c.schema.targetNamespace {
			continue
		}
		// Only a namespace that was import-declared somewhere but not directly by
		// this document is a violation; anything else is left to the existing
		// resolution handling.
		if _, declared := c.importDeclaredNS[qn.NS]; !declared {
			continue
		}
		if directImports, ok := c.docImportedNS[c.filename]; ok {
			if _, imported := directImports[qn.NS]; imported {
				continue
			}
		}
		var resolved bool
		if edecl.IsRef {
			_, resolved = realElems[qn]
		} else {
			_, resolved = realTypes[qn]
		}
		if !resolved {
			continue
		}
		issues = append(issues, issue{line: src.line, local: src.elemName, isRef: edecl.IsRef, src: src, qn: qn})
	}

	sort.Slice(issues, func(i, j int) bool {
		si := c.diagSourceOrRecorded(issues[i].src.source)
		sj := c.diagSourceOrRecorded(issues[j].src.source)
		if si != sj {
			return si < sj
		}
		if issues[i].line != issues[j].line {
			return issues[i].line < issues[j].line
		}
		return issues[i].local < issues[j].local
	})
	for _, is := range issues {
		source := c.diagSourceOrRecorded(is.src.source)
		if is.isRef {
			msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) element declaration.", is.qn.NS, is.qn.Local)
			c.schemaError(ctx, schemaParserErrorAttr(source, is.line, is.local, elemElement, attrRef, msg))
			continue
		}
		msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", is.qn.NS, is.qn.Local)
		c.schemaError(ctx, schemaElemDeclErrorAttr(source, is.line, is.local, msg))
	}
}

// typeExistsForRef reports whether a type= reference qn resolves in the TYPE
// symbol space, mirroring the type-map resolution in the elemRefs loop (the exact
// qn, plus the chameleon empty-namespace fallback for a chameleon-eligible
// reference). It lets that loop prefer a same-named TYPE over a colliding global
// element for a type= reference, while still falling back to the element's type
// when no such type exists.
func (c *compiler) typeExistsForRef(edecl *ElementDecl, qn QName) bool {
	if _, ok := c.schema.types[qn]; ok {
		return true
	}
	if _, eligible := c.chameleonEligible[edecl]; eligible {
		if _, ok := c.schema.types[QName{Local: qn.Local, NS: ""}]; ok {
			return true
		}
	}
	return false
}

// resolveNamedType resolves a named type reference to its definition, mirroring
// the chameleon empty-namespace fallback used elsewhere: an unprefixed reference
// resolves to the schema targetNamespace, but the type may come from a
// no-targetNamespace imported schema.
func (c *compiler) resolveNamedType(qn QName) *TypeDef {
	if td, ok := c.schema.types[qn]; ok {
		return td
	}
	if qn.NS != "" {
		if td, ok := c.schema.types[QName{Local: qn.Local}]; ok {
			return td
		}
	}
	return nil
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
				// XSD 1.1 permits circular attribute group definitions (W3C bug
				// 15795 / attgD015): the cycle back-edge is still CUT so the
				// downstream flatten/expand walks terminate, but no diagnostic is
				// reported. XSD 1.0 rejects it (src-attribute_group.3),
				// byte-identical.
				if c.version != Version11 {
					c.reportCircularAttrGroupRefQName(ctx, child, edgeSrc)
				}
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
		// XSD 1.1 relaxes cos-all-limited: a reference to an 'all' model group may
		// be nested directly inside another xs:all (it is flattened into the parent
		// by matchAll), but it must occur exactly once (minOccurs = maxOccurs = 1).
		// A reference nested in an xs:sequence/xs:choice is still forbidden, as is
		// any nested all-group reference in XSD 1.0.
		if c.version == Version11 && src.parentCompositor == CompositorAll {
			if placeholder.MinOccurs == 1 && placeholder.MaxOccurs == 1 {
				return
			}
			c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
				"A reference to an 'all' model group nested in an 'all' model group must have minOccurs = maxOccurs = 1."))
			return
		}
		c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
			"A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups."))
		return
	}

	// Direct reference: cos-all-limited requires the referencing particle's {min
	// occurs} to be 0 or 1 (and {max occurs} to be 1). A minOccurs > 1 (e.g.
	// group ref minOccurs="2" to an all group) is a violation independent of
	// maxOccurs — with the default maxOccurs=1 it is also a min>max form the
	// generic occurrence validator does not flag for group refs.
	if placeholder.MinOccurs > 1 {
		c.schemaError(ctx, schemaParserError(file, src.line, src.local, elemGroup,
			"The particle's {min occurs} must be (0 | 1), since the reference resolves to an 'all' model group."))
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

// checkCircularGroupRefs detects circular references among named model group
// definitions (an xs:group that references itself, directly or transitively) and
// returns the set of back-edge group-ref placeholders to CUT — leave unresolved,
// so the resolved content-model tree stays acyclic. A circular group reference is
// forbidden in both XSD 1.0 and XSD 1.1 (a model group must not contain itself),
// so the cycle is reported as a schema error in both versions. The cut must
// happen before the group-ref resolution shares group content slices, otherwise a
// self-referential group makes the tree cyclic and the downstream Glushkov (UPA),
// element-consistency, and open-content walks overflow the stack.
//
// The reference graph is built by walking each named group's UNRESOLVED
// definition, treating a group-ref placeholder (a *ModelGroup registered in
// c.groupRefs) as a leaf edge to its target and recursing into inline
// sub-model-groups. A DFS then reports and cuts each back-edge, mirroring
// checkCircularAttrGroupRefs.
func (c *compiler) checkCircularGroupRefs(ctx context.Context) map[*ModelGroup]struct{} {
	cut := make(map[*ModelGroup]struct{})
	if c.filename == "" {
		return cut
	}

	// edge records a group-ref placeholder inside a named group's definition.
	type groupRefEdge struct {
		target      QName
		placeholder *ModelGroup
	}
	edges := make(map[QName][]groupRefEdge)

	var collect func(owner QName, mg *ModelGroup, seen map[*ModelGroup]struct{})
	collect = func(owner QName, mg *ModelGroup, seen map[*ModelGroup]struct{}) {
		if mg == nil {
			return
		}
		for _, p := range mg.Particles {
			sub, ok := p.Term.(*ModelGroup)
			if !ok {
				continue
			}
			if target, isRef := c.groupRefs[sub]; isRef {
				edges[owner] = append(edges[owner], groupRefEdge{target: target, placeholder: sub})
				continue
			}
			// Inline sub-model-group: recurse. The seen set guards against a shared
			// sub-tree being re-walked (group content slices are shared once resolved,
			// but this runs before resolution — the guard is defensive).
			if _, dup := seen[sub]; dup {
				continue
			}
			seen[sub] = struct{}{}
			collect(owner, sub, seen)
		}
	}

	roots := make([]QName, 0, len(c.schema.groups))
	for qn, mg := range c.schema.groups {
		roots = append(roots, qn)
		collect(qn, mg, map[*ModelGroup]struct{}{})
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].NS != roots[j].NS {
			return roots[i].NS < roots[j].NS
		}
		return roots[i].Local < roots[j].Local
	})

	// onStack is the current DFS recursion stack; done marks fully-explored groups
	// so a group reachable from two roots is not re-walked.
	onStack := make(map[QName]bool)
	done := make(map[QName]bool)
	var visit func(qn QName)
	visit = func(qn QName) {
		onStack[qn] = true
		for _, e := range edges[qn] {
			if onStack[e.target] {
				// Back-edge qn -> target closes a cycle through target. Cut this edge's
				// placeholder (leave it an empty model group) so resolution stays
				// acyclic, and report the circular reference once per back-edge.
				if _, already := cut[e.placeholder]; already {
					continue
				}
				cut[e.placeholder] = struct{}{}
				c.reportCircularGroupRef(ctx, e.target, e.placeholder)
				continue
			}
			if done[e.target] {
				continue
			}
			visit(e.target)
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
	return cut
}

// reportCircularGroupRef emits the circular-model-group-reference diagnostic,
// attributed to the back-edge placeholder's recorded xs:group ref source (the ref
// that closed the cycle), naming the group being circularly referenced.
func (c *compiler) reportCircularGroupRef(ctx context.Context, targetQN QName, placeholder *ModelGroup) {
	if c.filename == "" {
		return
	}
	src := c.groupRefSources[placeholder]
	file := c.diagSourceOrRecorded(src.source)
	local := src.local
	if local == "" {
		local = elemGroup
	}
	msg := fmt.Sprintf("Circular reference to the model group definition '%s' defined.", formatAttrQName(targetQN))
	c.schemaError(ctx, schemaParserError(file, src.line, local, elemGroup, msg))
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
		// A declared xsi:schemaLocation use has no scalar built-in type, so its
		// fixed/default LITERAL is validated against the list-of-xs:anyURI value
		// space directly (non-empty even pairs, each a valid xs:anyURI).
		if isDeclaredXsiSchemaLocationUse(it.au, c.version) {
			if err := validateXsiSchemaLocationValue(*val, c.version); err != nil {
				msg := fmt.Sprintf("The value '%s' is not a valid value of the xsi:schemaLocation type (a non-empty even list of xs:anyURI).", *val)
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, "attribute", it.src.local, msg))
			}
			continue
		}
		td := attrUseTypeDef(it.au, c.schema)
		if td == nil || td.ContentType != ContentTypeSimple {
			continue
		}
		// XSD 1.0 "Attribute Declaration Properties Correct" clause 3 (§3.2.6): if the
		// attribute's {type definition} is or is derived from xs:ID there must NOT be a
		// {value constraint} (default or fixed). builtinBaseLocal returns the first
		// XSD-namespace ancestor's local name, which is "ID" for xs:ID and any type
		// restricting it. XSD 1.1 removed this restriction (W3C bug 4077), so the check
		// is gated to Version10 and the 1.1 path stays byte-identical.
		if c.version == Version10 && builtinBaseLocal(td) == typeID {
			msg := "The attribute declaration is or is derived from ID and there must not be a value constraint."
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, "attribute", it.src.local, msg))
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

// checkElementDeclConstraints validates each element declaration's explicit
// default/fixed value against its declared simple (content) type (§3.3.6
// "Schema Component Constraint: Element Default Valid (Immediate)" — a
// version-independent XSD rule enforced in BOTH 1.0 and 1.1). It mirrors
// checkAttrUseConstraints: an invalid value (e.g. a decimal default of "XII", a
// boolean "Yes", or a list/union default that does not satisfy the type) is a
// schema error caught at compile time rather than silently injected into the
// instance.
//
// The type checked is the element's EFFECTIVE declared type (effectiveDeclType),
// so a no-type substitution-group member is validated against its inherited head
// type. Only an element whose type is a simple type, or a complex type with simple
// content, has a type-validated default; an element-only/mixed complex content
// default is character data not validated against a simple type, so it is skipped.
// A simpleContent default is validated against its FULL content chain (effective
// content type plus inherited base-content facets) via the shared
// validateSimpleContentValue, identical to the instance-value path.
func (c *compiler) checkElementDeclConstraints(ctx context.Context) {
	if c.filename == "" {
		return
	}
	type pending struct {
		decl *ElementDecl
		src  attrConstraintSource
	}
	items := make([]pending, 0, len(c.elemDeclConstraintSources))
	for decl, src := range c.elemDeclConstraintSources {
		items = append(items, pending{decl: decl, src: src})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].src.line != items[j].src.line {
			return items[i].src.line < items[j].src.line
		}
		return items[i].src.local < items[j].src.local
	})

	for _, it := range items {
		val := it.decl.Default
		if val == nil {
			val = it.decl.Fixed
		}
		if val == nil {
			continue
		}
		// Resolve the EFFECTIVE declared type: a no-type substitution-group member
		// inherits its head's type, so validating it.decl.Type directly would skip the
		// member (nil type) and miss an invalid inherited-type default/fixed.
		std := effectiveDeclType(it.decl, c.schema)
		if std == nil {
			continue
		}
		// XSD 1.0 §3.3.6 "Element Declaration Properties Correct" (the element analog of
		// au-props-correct.3 §3.2.6): if the element's {type definition} is or is derived
		// from xs:ID there must NOT be a {value constraint} (default or fixed).
		// builtinBaseLocal returns the first XSD-namespace ancestor's local name, which is
		// "ID" for xs:ID and any type restricting it. XSD 1.1 removed this restriction
		// (W3C bug 4077), so the check is gated to Version10 and the 1.1 path stays
		// byte-identical.
		if c.version == Version10 && builtinBaseLocal(std) == typeID {
			msg := "The element declaration is or is derived from ID and there must not be a value constraint."
			c.schemaError(ctx, schemaElemDeclError(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, msg))
			continue
		}
		// §3.3.6 "Element Default Valid (Immediate)" clause 2.1 (version-independent):
		// when a value constraint (default/fixed) is present and the type is a complex
		// type, its {content type} must be a simple type definition or mixed. An
		// element-only or empty complex content type carries no character-data value for
		// the constraint, so a default/fixed on such an element is a schema error rather
		// than silently ignored. Simple content is type-validated below; mixed content is
		// left to its own path (clause 2.2.2 emptiability is not checked here).
		if std.IsComplex && !std.IsSimpleContent {
			// A declaration carrying BOTH default and fixed is already a
			// mutually-exclusive representation error; the value constraint is ill-formed,
			// so do not additionally judge its type applicability (avoids a duplicate
			// diagnostic on an already-invalid declaration).
			bothConstraints := it.decl.Default != nil && it.decl.Fixed != nil
			if !bothConstraints && (std.ContentType == ContentTypeElementOnly || std.ContentType == ContentTypeEmpty) {
				kind := attrDefault
				if it.decl.Default == nil {
					kind = attrFixed
				}
				msg := fmt.Sprintf("The type of the element declaration must be a simple type or a complex type with mixed or simple content for a '%s' value constraint to be present.", kind)
				c.schemaError(ctx, schemaElemDeclError(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, msg))
			}
			continue
		}
		// Validate through the SAME shared simpleContent path the instance value uses
		// (validateSimpleContentValue: effective content type PLUS inherited base
		// content facets), version-/schema-aware so a 1.1 lexical form or an assertion
		// facet is handled exactly as for an attribute default/fixed. A plain simpleType
		// passes through that path unchanged (the nested-base walk is a no-op).
		vc := &validationContext{schema: c.schema, errorHandler: helium.NilErrorHandler{}, version: c.version}
		if err := vc.validateSimpleContentValue(ctx, *val, it.src.nsMap, std, it.src.local, it.src.line); err != nil {
			msg := fmt.Sprintf("The value '%s' is not a valid value of the atomic type '%s'.", *val, typeDisplayName(effectiveContentSimpleType(std)))
			c.schemaError(ctx, schemaElemDeclError(c.diagSourceOrRecorded(it.src.source), it.src.line, it.src.local, msg))
		}
	}
}

// checkAttrRefFixedConflict enforces au-props-correct.3 for an <xs:attribute
// ref> use whose referenced global declaration has a 'fixed' value constraint:
// the use's own value constraint must also be 'fixed' and value-equal. A local
// 'default' (no fixed) or a 'fixed' carrying a different value is rejected. Both
// constraint lexicals are typed by the global declaration's simple type, so the
// value comparison runs under that single type.
func (c *compiler) checkAttrRefFixedConflict(ctx context.Context, au, ga *AttrUse, qn QName) {
	if c.filename == "" {
		return
	}
	// A local fixed value-equal to the declaration's fixed is valid; nothing to
	// report. A local default (au.Fixed == nil) always conflicts.
	if au.Fixed != nil {
		gaTD := attrUseTypeDef(ga, c.schema)
		if fixedConstraintRestricts(ctx, *au.Fixed, *ga.Fixed, gaTD, gaTD, au.FixedNS, ga.FixedNS, c.schema, c.version) {
			return
		}
	}
	line := 0
	source := c.diagSource()
	local := qn.Local
	if src, ok := c.attrUseConstraintSources[au]; ok {
		line = src.line
		source = c.diagSourceOrRecorded(src.source)
		local = src.local
	}
	msg := fmt.Sprintf("The value constraint of the attribute use is inconsistent with the 'fixed' value constraint of the referenced attribute declaration '{%s}%s'.", qn.NS, qn.Local)
	c.schemaError(ctx, schemaParserErrorAttr(source, line, local, "attribute", local, msg))
}

// builtinRestrictionParent maps each XSD builtin atomic type's local name to
// the local name of the builtin it is derived (by restriction) from, per the
// W3C XML Schema Part 2 type hierarchy. It is used to decide builtin-to-builtin
// restriction validity (e.g. xs:int restricts xs:integer), which the *TypeDef
// pointer chain cannot express because builtin types carry no BaseType links.
// Every atomic primitive (string, decimal, boolean, float, double, the
// date/time/g* family, duration, the binary types, anyURI, QName, NOTATION)
// is rooted at anySimpleType, which terminates the chain — so a cross-family
// pair is decided ("known") and REJECTED rather than treated as "unknown" and
// silently accepted. Only atomic types are listed here; the list builtins
// (IDREFS/ENTITIES/NMTOKENS) are handled separately by builtinDerivesFrom via
// builtinListItem so an atomic-vs-list pair is also decided rather than
// silently accepted.
var builtinRestrictionParent = map[string]string{
	// string family
	lexicon.TypeString:           typeAnySimpleType,
	lexicon.TypeNormalizedString: lexicon.TypeString,
	lexicon.TypeToken:            lexicon.TypeNormalizedString,
	typeLanguage:                 lexicon.TypeToken,
	typeName:                     lexicon.TypeToken,
	typeNMToken:                  lexicon.TypeToken,
	typeNCName:                   typeName,
	typeID:                       typeNCName,
	lexicon.TypeIDREF:            typeNCName,
	typeEntity:                   typeNCName,
	// decimal / integer family
	lexicon.TypeDecimal:            typeAnySimpleType,
	lexicon.TypeInteger:            lexicon.TypeDecimal,
	lexicon.TypeNonPositiveInteger: lexicon.TypeInteger,
	lexicon.TypeNegativeInteger:    lexicon.TypeNonPositiveInteger,
	lexicon.TypeLong:               lexicon.TypeInteger,
	lexicon.TypeInt:                lexicon.TypeLong,
	lexicon.TypeShort:              lexicon.TypeInt,
	lexicon.TypeByte:               lexicon.TypeShort,
	lexicon.TypeNonNegativeInteger: lexicon.TypeInteger,
	lexicon.TypeUnsignedLong:       lexicon.TypeNonNegativeInteger,
	lexicon.TypeUnsignedInt:        lexicon.TypeUnsignedLong,
	lexicon.TypeUnsignedShort:      lexicon.TypeUnsignedInt,
	lexicon.TypeUnsignedByte:       lexicon.TypeUnsignedShort,
	lexicon.TypePositiveInteger:    lexicon.TypeNonNegativeInteger,
	// remaining atomic primitives — each parented directly to anySimpleType.
	// Listing them (rather than leaving them "unknown") lets builtinDerivesFrom
	// REJECT an invalid builtin redeclaration whose derived type lives outside
	// the string/decimal families (e.g. base xs:int restricted by derived
	// xs:boolean), instead of returning "unknown" and silently accepting it.
	lexicon.TypeBoolean:    typeAnySimpleType,
	lexicon.TypeFloat:      typeAnySimpleType,
	lexicon.TypeDouble:     typeAnySimpleType,
	lexicon.TypeDuration:   typeAnySimpleType,
	lexicon.TypeDateTime:   typeAnySimpleType,
	lexicon.TypeTime:       typeAnySimpleType,
	lexicon.TypeDate:       typeAnySimpleType,
	lexicon.TypeGYearMonth: typeAnySimpleType,
	lexicon.TypeGYear:      typeAnySimpleType,
	lexicon.TypeGMonthDay:  typeAnySimpleType,
	lexicon.TypeGDay:       typeAnySimpleType,
	lexicon.TypeGMonth:     typeAnySimpleType,
	typeHexBinary:          typeAnySimpleType,
	typeBase64Binary:       typeAnySimpleType,
	lexicon.TypeAnyURI:     typeAnySimpleType,
	lexicon.TypeQName:      typeAnySimpleType,
	lexicon.TypeNotation:   typeAnySimpleType,
}

// builtinListItem maps each XSD builtin LIST type's local name to the local name
// of its item type, per XML Schema Part 2. These three list builtins carry no
// BaseType links (they are registered as bare names), so builtinDerivesFrom
// recognizes them explicitly to decide atomic-vs-list and list-vs-list
// derivation rather than treating them as "unknown".
var builtinListItem = map[string]string{
	typeIDRefs:   lexicon.TypeIDREF,
	typeEntities: typeEntity,
	typeNMTokens: typeNMToken,
}

// isBuiltinListName reports whether local is one of the XSD builtin list types.
func isBuiltinListName(local string) bool {
	_, ok := builtinListItem[local]
	return ok
}

// builtinDerivesFrom reports whether the builtin type named derived is the same
// as, or derived by restriction from, the builtin named base. The second bool
// is false ("unknown") when either name is not a recognized builtin (atomic or
// list), so callers can stay conservative on cases the table cannot decide.
func builtinDerivesFrom(derived, base string) (bool, bool) {
	// List builtins (IDREFS/ENTITIES/NMTOKENS) participate in derivation
	// decisions even though they carry no parent links: a list type is the same
	// as itself, validly derives from xs:anySimpleType (cos-st-derived-ok 2.2.3),
	// and is otherwise UNRELATED to every atomic type and to the other two list
	// types. Deciding these (rather than returning "unknown") rejects an invalid
	// widening such as xs:IDREFS "restricted" by xs:string.
	if isBuiltinListName(derived) || isBuiltinListName(base) {
		if derived == base {
			return true, true
		}
		if isBuiltinListName(derived) && base == typeAnySimpleType {
			return true, true
		}
		return false, true
	}
	if !isAtomicBuiltinName(derived) || !isAtomicBuiltinName(base) {
		return false, false
	}
	for cur := derived; ; {
		if cur == base {
			return true, true
		}
		parent, ok := builtinRestrictionParent[cur]
		if !ok {
			// Reached the anySimpleType root without matching base.
			return false, true
		}
		cur = parent
	}
}

// isAtomicBuiltinName reports whether local is a recognized atomic builtin in
// the restriction hierarchy (a map key, or the anySimpleType root).
func isAtomicBuiltinName(local string) bool {
	if local == typeAnySimpleType {
		return true
	}
	_, ok := builtinRestrictionParent[local]
	return ok
}

// simpleTypeValidlyRestricts reports whether the derived simple type is a valid
// restriction of (same as, or derived by restriction from) the base simple
// type. It first consults the *TypeDef pointer chain (isDerivedFrom). When that
// fails it falls back to the builtin restriction hierarchy, but ONLY when the
// BASE is an actual XSD builtin — a user simple type that restricts a builtin
// must be derived from through the pointer chain, because widening it back to
// its builtin ancestor would drop the user-added facets. It is CONSERVATIVE: it
// returns true (valid) whenever derivation cannot be decided (unresolved types,
// list/union carriers, or a builtin pair the table does not cover), so it only
// ever rejects a clearly invalid restriction and never false-rejects a
// legitimate one.
func simpleTypeValidlyRestricts(derived, base *TypeDef) bool {
	if derived == nil || base == nil {
		return true
	}
	if isDerivedFrom(derived, base) {
		return true
	}
	// cos-st-derived-ok.2.2.4: a base that is a UNION admits a derived type that
	// is validly derived from (at least) ONE of its {member type definitions}.
	// This MUST be handled BEFORE the builtin-base early return below, because
	// builtinBaseLocal(base) is empty for a union (a union is not an atomic
	// builtin) and the early return would otherwise accept ANY derived type
	// unconditionally — wrongly accepting e.g. base union(xs:int xs:boolean)
	// redeclared as xs:date (date derives from NEITHER member, so the loop
	// rejects it). Members are walked transitively via the recursive call (a
	// member that is itself a union re-enters this branch, so an intervening
	// faceted member-union is rejected by its own facet gate below).
	//
	// XSD 1.0 SCOPE: cos-st-derived-ok (§3.14.6, Type Derivation OK Simple) has NO
	// "facets empty" condition on a union base — a type validly derived from any
	// member type is a valid restriction of the union, regardless of facets the
	// union carries. The "facets empty" gate is an XSD 1.1-only condition (§3.16.6.3
	// Type Derivation OK Simple), and this package targets XSD 1.0 (libxml2 parity),
	// so it is intentionally NOT enforced here.
	if resolveVariety(base) == TypeVarietyUnion {
		for _, member := range resolveUnionMembers(base) {
			if simpleTypeValidlyRestricts(derived, member) {
				return true
			}
		}
		return false
	}
	// cos-st-derived-ok.2.2: a base that is a LIST variety. isDerivedFrom already
	// failed above, so the derived type does NOT appear in the base list's
	// restriction chain (a real <xs:restriction base="theList"> sets the BaseType
	// pointer, which isDerivedFrom follows). A type that did not pass the pointer
	// chain is therefore NOT a valid restriction of the list: an unrelated list,
	// or a list with a different item type, admits values the base does not, and
	// xs:anySimpleType is the simple ur-type — a SUPERTYPE — so deriving the list
	// "down to" it would WIDEN to accept non-list values. A restriction can never
	// validly produce a supertype, so REJECT everything here.
	if resolveVariety(base) == TypeVarietyList {
		return false
	}
	// cos-st-derived-ok.2.2: the builtin LIST types (xs:IDREFS, xs:ENTITIES,
	// xs:NMTOKENS) are registered as bare atomic-variety names with no BaseType
	// link and no list marker, so resolveVariety reports Atomic and the list
	// branch above does not catch them. isDerivedFrom already failed, so the
	// derived type is not in the base list's restriction chain (a real
	// <xs:restriction base="xs:IDREFS"> sets the BaseType pointer). Decide here
	// rather than fall through to the db/bb shortcut, which returns "valid"
	// whenever the derived side has no builtin base name (db == "") — that is the
	// gap that let an unrelated user list (xs:list itemType="xs:string") stand in
	// for an xs:IDREFS base. An unrelated list or atomic is not a valid
	// restriction of a builtin list base, so REJECT.
	if base.Name.NS == lexicon.NamespaceXSD && isBuiltinListName(base.Name.Local) {
		return false
	}
	// A CONSTRUCTED derived list or union (resolveVariety List/Union) reaching this
	// point has already FAILED both the pointer-chain derivation (isDerivedFrom) and
	// the valid-union-base member shortcut above. A constructed list/union can only
	// be validly derived from xs:anySimpleType (the simple ur-type) or through a real
	// base-type chain — there is no other source. So accept ONLY when the base is the
	// actual xs:anySimpleType; otherwise REJECT. Without this, the db=="" "unknown =>
	// valid" fallback below would wrongly accept e.g. an atomic base xs:string
	// redeclared as a user xs:union or xs:list.
	if v := resolveVariety(derived); v == TypeVarietyList || v == TypeVarietyUnion {
		return base.Name.NS == lexicon.NamespaceXSD && base.Name.Local == typeAnySimpleType
	}
	db := builtinBaseLocal(derived)
	bb := builtinBaseLocal(base)
	if db == "" || bb == "" {
		return true
	}
	// The builtin restriction hierarchy may stand in for the missing builtin
	// BaseType links ONLY when the BASE type is an ACTUAL XSD builtin. Walking
	// the DERIVED side to its builtin ancestor is sound (a user restriction only
	// narrows), but treating a user simple type that RESTRICTS a builtin (e.g.
	// xs:int with maxInclusive="10") as that builtin would WIDEN the base back to
	// its ancestor and wrongly accept a derived type that drops the user-added
	// facets. When the base is a user-restricted (non-union) type, the only valid
	// derivation is through the pointer chain (isDerivedFrom, already checked
	// above) — so reject.
	if base.Name.NS != lexicon.NamespaceXSD {
		return false
	}
	ok, known := builtinDerivesFrom(db, bb)
	if !known {
		return true
	}
	return ok
}

// fixedConstraintRestricts reports whether a derived attribute use's 'fixed'
// value is value-equal to the base attribute use's 'fixed' value
// (derivation-ok-restriction.2.1.3). The two lexicals may be typed DIFFERENTLY
// when the restriction validly narrows the type (base xs:decimal fixed="1.0",
// derived xs:int fixed="1": equal values, but "1.0" is not a valid xs:int
// lexical), so each lexical must be compared under ITS OWN simple type. A
// same-type (or unresolved) fast path uses fixedValueMatches directly (so
// derived "01" still matches base "1" for xs:integer, and a nil type falls back
// to raw lexical equality); a cross-type pair is compared in its shared
// primitive value space via crossMemberValueEqual.
func fixedConstraintRestricts(ctx context.Context, derivedFixed, baseFixed string, derivedTD, baseTD *TypeDef, derivedNS, baseNS map[string]string, schema *Schema, version Version) bool {
	if derivedTD == nil || baseTD == nil || derivedTD == baseTD {
		return fixedValueMatches(ctx, derivedFixed, baseFixed, derivedTD, derivedNS, baseNS, schema, version)
	}
	// xs:anySimpleType — the simple ur-type — has NO primitive value-space family,
	// so crossMemberValueEqual (which needs a shared primitive family) can never
	// match against it and would false-reject. This arises when an untyped
	// attribute (whose effective type defaults to xs:anySimpleType) carries a
	// 'fixed' value and the other side narrows it to a real type. Per XSD 1.0 any
	// simple type validly derives from the ur-type, and a base 'fixed' value is
	// preserved iff the derived 'fixed' LITERAL is identical — there is no
	// narrower value space to compare in. So when either side's effective type is
	// the ur-type, accept on exact lexical equality.
	if isUrSimpleType(derivedTD) || isUrSimpleType(baseTD) {
		return derivedFixed == baseFixed
	}
	return crossMemberValueEqual(ctx, derivedFixed, baseFixed, derivedTD, baseTD, derivedNS, baseNS, schema, version)
}

// isUrSimpleType reports whether td is xs:anySimpleType, the simple ur-type. It
// has no primitive value-space family, so cross-member value comparison cannot
// route through it; callers fall back to lexical equality for the ur-type.
func isUrSimpleType(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && td.Name.Local == typeAnySimpleType
}

// effectiveAttrWildcard returns the {attribute wildcard} of a complex type for
// the restriction-derivation check. In XSD 1.1 the group-ref wildcards are
// already merged into td.AnyAttribute at link time, so this returns it unchanged.
// In XSD 1.0 group wildcards are NOT merged into a type's {attribute wildcard} at
// validation (byte-identical); this re-derives one on demand ONLY to recognize a
// base whose attribute wildcard comes SOLELY through a referenced attribute group
// (td.AnyAttribute nil): it fills in the first group-ref complete wildcard so
// derivation-ok-restriction 4.1 sees a wildcard and 4.2/4.3 have one to compare.
//
// A DIRECT td.AnyAttribute is returned as-is and is NEVER narrowed by intersecting
// group-ref wildcards: 1.0 instance validation uses only the direct wildcard, so
// narrowing it here would newly reject a restriction whose derived wildcard is a
// subset of the direct base wildcard but not of the (narrower) intersection —
// breaking 1.0 byte-identical behavior. The first group-ref wildcard suffices to
// discharge 4.1 for the transitive-only case; the full multi-group intersection is
// not modeled here.
func (c *compiler) effectiveAttrWildcard(td *TypeDef) *Wildcard {
	if td == nil {
		return nil
	}
	w := td.AnyAttribute
	if c.version == Version11 || w != nil {
		return w
	}
	for _, qn := range c.attrGroupRefs[td] {
		if _, ok := c.schema.attrGroups[qn]; !ok {
			continue
		}
		if gw := c.attrGroupCompleteWildcard(qn, map[QName]struct{}{}); gw != nil {
			return gw
		}
	}
	return nil
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

	// Attribute the diagnostic to the schema that DECLARES this type: src.line is
	// the line within that schema, so the filename must come from src.source
	// (an included/imported/redefined file), not the top-level c.filename. Mirrors
	// the restriction-particle check (checkRestrictionParticles).
	source := c.diagSourceOrRecorded(src.source)

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

	// The base's EFFECTIVE {attribute wildcard} — its directly-declared
	// xs:anyAttribute PLUS any wildcard contributed transitively through a
	// referenced attribute group. In XSD 1.0 td.BaseType.AnyAttribute alone omits
	// the group-ref wildcard, so a valid restriction against such a base would be
	// falsely rejected without this.
	baseWildcard := c.effectiveAttrWildcard(td.BaseType)

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
				c.schemaError(ctx, schemaComponentError(source, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}
			// XSD 1.1 derivation-ok-restriction: a restricting attribute use must
			// keep the base use's {inheritable} (true→false and false→true both fail).
			if c.version == Version11 && au.Inheritable != baseAU.Inheritable {
				msg := fmt.Sprintf("The 'inheritable' property of the attribute use '%s' is inconsistent with the corresponding attribute use of the base complex type definition %s.", au.Name.Local, baseQualified)
				c.schemaError(ctx, schemaComponentError(source, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}

			// derivation-ok-restriction.2.1.2: the derived attribute's type must
			// be the same as, or derived by restriction from, the base
			// attribute's type. Attribute types are simple types, so any
			// derivation is by restriction; isDerivedFrom captures the chain.
			// When either type is unresolved, accept conservatively (mirrors the
			// element-to-element restriction check). An ABSENT attribute type is
			// xs:anySimpleType (XSD §3.2.2.1), so attrUseEffectiveTypeDef defaults
			// to the ur-type: an untyped derived attribute restricting a narrower
			// base (e.g. xs:int) is the ur-type widening the base and is rejected.
			derivedTD := attrUseEffectiveTypeDef(au, c.schema)
			baseTD := attrUseEffectiveTypeDef(baseAU, c.schema)
			if derivedTD != nil && baseTD != nil && !simpleTypeValidlyRestricts(derivedTD, baseTD) {
				msg := fmt.Sprintf("The type definition of the attribute use is not a valid restriction of the corresponding attribute use's type definition of the base complex type definition %s.", baseQualified)
				c.schemaError(ctx, schemaComponentError(source, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}

			// derivation-ok-restriction.2.1.3: a base 'fixed' value constraint
			// forces the derived attribute to carry a value-space-equal 'fixed'
			// value (a default, or no constraint, would admit values the base
			// pins). Each lexical is compared under ITS OWN type so a valid
			// narrowing across types is not false-rejected (base xs:decimal
			// fixed="1.0" narrowed by derived xs:int fixed="1": equal values, but
			// "1.0" is not a valid xs:int lexical). fixedConstraintRestricts uses a
			// same-type fast path (so base "1" accepts derived "01" for xs:integer)
			// and falls back to the cross-type value-equality helper otherwise.
			if baseAU.Fixed != nil {
				if au.Fixed == nil || !fixedConstraintRestricts(ctx, *au.Fixed, *baseAU.Fixed, derivedTD, baseTD, au.FixedNS, baseAU.FixedNS, c.schema, c.version) {
					msg := fmt.Sprintf("The effective value constraint of the attribute use is inconsistent with the 'fixed' value constraint of the corresponding attribute use of the base complex type definition %s.", baseQualified)
					c.schemaError(ctx, schemaComponentError(source, src.line, "complexType",
						component+", attribute use '"+au.Name.Local+"'", msg))
				}
			}
		} else if baseWildcard == nil || !wildcardAllowsExpandedName(baseWildcard, au.Name.Local, au.Name.NS, c.schema, true) {
			// No matching attribute, and no base wildcard that ADMITS this derived
			// attribute's expanded name — the full test honors the base wildcard's
			// notNamespace/notQName/##defined, not just its namespace constraint, so
			// a derived attribute the base wildcard excludes by name is rejected.
			msg := fmt.Sprintf("Neither a matching attribute use, nor a matching wildcard exists in the base complex type definition %s.", baseQualified)
			c.schemaError(ctx, schemaComponentError(source, src.line, "complexType",
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
				c.schemaError(ctx, schemaComponentError(source, src.line, "complexType", component, msg))
			}
			continue
		}
		if !found || derived.Prohibited {
			msg := fmt.Sprintf("A matching attribute use for the 'required' attribute use '%s' of the base complex type definition %s is missing.", baseAU.Name.Local, baseQualified)
			c.schemaError(ctx, schemaComponentError(source, src.line, "complexType", component, msg))
		}
	}

	// derivation-ok-restriction 4: Wildcard checks. The base side uses its
	// EFFECTIVE {attribute wildcard} (baseWildcard) so a wildcard the base holds
	// only transitively through a referenced attribute group still satisfies 4.1
	// and is compared for 4.2/4.3.
	if td.AnyAttribute != nil {
		// 4.1: Base must also have a wildcard.
		if baseWildcard == nil {
			msg := fmt.Sprintf("The complex type definition has an attribute wildcard, but the base complex type definition %s does not have one.", baseQualified)
			c.schemaError(ctx, schemaComponentError(source, src.line, "complexType", component, msg))
		} else {
			// 4.2: Derived namespace must be subset of base namespace.
			if !wildcardConstraintSubset(td.AnyAttribute, baseWildcard, c.schema, true) {
				msg := fmt.Sprintf("The attribute wildcard is not a valid subset of the wildcard in the base complex type definition %s.", baseQualified)
				c.schemaError(ctx, schemaComponentError(source, src.line, "complexType", component, msg))
			}
			// 4.3: Derived processContents must be >= base strength (strict > lax > skip).
			// libxml2 attributes this error to the base type's source location.
			if processContentsStrength(td.AnyAttribute.ProcessContents) < processContentsStrength(baseWildcard.ProcessContents) {
				errLine := src.line
				errComponent := component
				errFile := source
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
	// DFS over ALL heads (XSD 1.1 multiple-head substitution): a cycle may close
	// through any one of an element's heads, so following only the first head
	// would miss cycles that run through a later head.
	visited := map[QName]bool{}
	for _, current := range edecl.substitutionGroupHeads() {
		if c.substGroupChainContains(edecl.Name, current, visited) {
			// Cycle leads back to this element.
			// libxml2 reports this error twice.
			if src, ok := c.globalElemSources[edecl]; ok {
				msg := fmt.Sprintf("The element declaration '%s' defines a circular substitution group to element declaration '%s'.",
					edecl.Name.Local, edecl.Name.Local)
				errStr := schemaElemDeclError(c.filename, src.line, edecl.Name.Local, msg)
				c.schemaError(ctx, errStr)
				c.schemaError(ctx, errStr)
			}
			return
		}
	}
}

func (c *compiler) substGroupChainContains(target, current QName, visited map[QName]bool) bool {
	if current == (QName{}) {
		return false
	}
	if current == target {
		return true
	}
	if visited[current] {
		// Hit a cycle that doesn't include this element.
		return false
	}
	visited[current] = true
	head, ok := c.schema.elements[current]
	if !ok {
		return false
	}
	for _, next := range head.substitutionGroupHeads() {
		if c.substGroupChainContains(target, next, visited) {
			return true
		}
	}
	return false
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
		headType := c.resolveDeclaredType(head)
		for _, member := range members {
			memberType := c.resolveDeclaredType(member)
			if head.Final&FinalExtension != 0 && derivationUsesMethod(memberType, headType, DerivationExtension) {
				if src, ok := c.globalElemSources[member]; ok {
					c.schemaError(ctx, schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."))
				}
			}
			if head.Final&FinalRestriction != 0 && derivationUsesMethod(memberType, headType, DerivationRestriction) {
				if src, ok := c.globalElemSources[member]; ok {
					c.schemaError(ctx, schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."))
				}
			}
		}
	}
}

func (c *compiler) checkSubstGroupAffiliations(ctx context.Context) {
	for headQN, members := range c.schema.substGroups {
		head, ok := c.schema.elements[headQN]
		if !ok {
			continue
		}
		headType := c.resolveDeclaredType(head)
		for _, member := range members {
			memberType := c.resolveDeclaredType(member)
			if isXsiTypeDerivedFromDeclared(memberType, headType) {
				continue
			}
			if src, ok := c.globalElemSources[member]; ok {
				msg := fmt.Sprintf("The substitution group affiliation to '%s' is not validly substitutable for the head element's type definition.", head.Name.Local)
				c.schemaError(ctx, schemaElemDeclError(c.filename, src.line, member.Name.Local, msg))
			}
		}
	}
}

// derivationUsesMethod reports whether derived reaches base through the given
// derivation method. It follows explicit BaseType links and the extra simple-type
// derivation paths that are not pointer-linked: built-in narrowing and an
// unfaceted union head substitutable through one of its members.
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
	if td == base {
		return false
	}
	if method == DerivationRestriction && resolveVariety(base) == TypeVarietyUnion {
		for _, member := range resolveUnionMembers(base) {
			if isXsiTypeDerivedFromDeclared(derived, member) {
				return true
			}
		}
	}
	if method == DerivationRestriction && isBuiltinSimpleType(base) {
		db := builtinBaseLocal(derived)
		if db != base.Name.Local && builtinSimpleDerivedFrom(db, base.Name.Local) {
			return true
		}
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
