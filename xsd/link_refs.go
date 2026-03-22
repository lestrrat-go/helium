package xsd

import (
	"fmt"
	"sort"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func (c *compiler) resolveRefs() {
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
			if ge, ok := c.schema.elements[qn]; ok {
				edecl.Type = ge.Type
				if edecl.Default == nil {
					edecl.Default = ge.Default
				}
				if edecl.Fixed == nil {
					edecl.Fixed = ge.Fixed
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
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, src.line, src.elemName, elemElement, attrRef, msg), helium.ErrorLevelFatal))
					c.errorCount++
				}
				edecl.Type = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				continue
			}
			td, ok := c.schema.types[qn]
			if !ok {
				// Report unresolved type error for XSD built-in types that should exist.
				if qn.NS == lexicon.NamespaceXSD {
					if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" {
						msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
						c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaElemDeclErrorAttr(c.filename, src.line, src.elemName, attrType, msg), helium.ErrorLevelFatal))
						c.errorCount++
					}
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
		if !ok && qn.NS != "" {
			// Try empty namespace as fallback — the type may come from an
			// imported schema with no targetNamespace.
			base, ok = c.schema.types[QName{Local: qn.Local, NS: ""}]
		}
		if !ok {
			base = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = base
		}
		td.BaseType = base
	}

	// Resolve list item type references.
	for td, qn := range c.itemTypeRefs {
		itemTD, ok := c.schema.types[qn]
		if !ok {
			itemTD = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = itemTD
		}
		td.ItemType = itemTD
	}

	// Resolve union member type references.
	for _, ref := range c.unionMemberRefs {
		memberTD, ok := c.schema.types[ref.name]
		if !ok {
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
	for placeholder, qn := range c.groupRefs {
		grp, ok := c.schema.groups[qn]
		if !ok {
			continue
		}
		// Copy the group's content into the placeholder.
		placeholder.Compositor = grp.Compositor
		placeholder.Particles = grp.Particles
	}

	// Resolve attribute group references.
	for td, qns := range c.attrGroupRefs {
		for _, qn := range qns {
			if attrs, ok := c.schema.attrGroups[qn]; ok {
				td.Attributes = append(td.Attributes, attrs...)
			}
		}
	}

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
		}
		if au.TypeName == (QName{}) {
			au.TypeName = ga.TypeName
		}
	}

	// Merge content models for extension types.
	for td := range c.typeRefs {
		if td.Derivation != DerivationExtension || td.BaseType == nil {
			continue
		}
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
				component := "local complex type"
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
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
						component := "local complex type"
						if !src.isLocal {
							component = td.Name.Local
						}
						msg := fmt.Sprintf("Duplicate attribute use '%s'.", au.Name.Local)
						c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
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
		c.checkRestrictionAttrs(td)
	}

	// Check UPA (Unique Particle Attribution) for all complex types with content models.
	// Only run UPA if there are no prior schema errors (libxml2 skips UPA when
	// the schema has structural parse errors).
	if c.filename != "" && c.errorCount == 0 {
		for td, src := range c.typeDefSources {
			if td.ContentModel != nil {
				c.checkUPA(td, src)
			}
		}
	}
}

// checkRestrictionAttrs validates that a restriction-derived type's attributes
// are compatible with the base type's attribute uses.
func (c *compiler) checkRestrictionAttrs(td *TypeDef) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[td]
	if !hasSrc {
		return
	}

	component := "local complex type"
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg), helium.ErrorLevelFatal))
				c.errorCount++
			}
		} else if td.BaseType.AnyAttribute == nil {
			// No matching attribute and no wildcard in base.
			msg := fmt.Sprintf("Neither a matching attribute use, nor a matching wildcard exists in the base complex type definition %s.", baseQualified)
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType",
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
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// derivation-ok-restriction 4: Wildcard checks.
	if td.AnyAttribute != nil {
		// 4.1: Base must also have a wildcard.
		if td.BaseType.AnyAttribute == nil {
			msg := fmt.Sprintf("The complex type definition has an attribute wildcard, but the base complex type definition %s does not have one.", baseQualified)
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			// 4.2: Derived namespace must be subset of base namespace.
			if !wildcardNSSubset(td.AnyAttribute, td.BaseType.AnyAttribute) {
				msg := fmt.Sprintf("The attribute wildcard is not a valid subset of the wildcard in the base complex type definition %s.", baseQualified)
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, errLine, "complexType", errComponent, msg), helium.ErrorLevelFatal))
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
		for _, token := range strings.Fields(wc.Namespace) {
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
func (c *compiler) checkCircularSubstGroup(edecl *ElementDecl) {
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(errStr, helium.ErrorLevelFatal))
				c.errorCount++
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(errStr, helium.ErrorLevelFatal))
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
func (c *compiler) checkFinalOnTypes() {
	for _, td := range c.schema.types {
		src := c.typeDefSources[td]

		// Check base type final for extension/restriction derivation.
		if td.BaseType != nil && td.BaseType.Final != 0 {
			baseFinal := td.BaseType.Final
			if td.Derivation == DerivationExtension && baseFinal&FinalExtension != 0 {
				component := td.Name.Local
				if src.isLocal {
					component = "local complex type"
				}
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by extension is forbidden by the base type '"+td.BaseType.Name.Local+"'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
			if td.Derivation == DerivationRestriction && baseFinal&FinalRestriction != 0 {
				component := td.Name.Local
				if src.isLocal {
					component = "local complex type"
				}
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
					"Derivation by restriction is forbidden by the base type '"+td.BaseType.Name.Local+"'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// simpleType list: check if item type forbids list derivation.
		if td.Variety == TypeVarietyList && td.ItemType != nil && td.ItemType.Final&FinalList != 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
				"Derivation by list is forbidden by the item type '"+td.ItemType.Name.Local+"'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
		// simpleType union: check if any member type forbids union derivation.
		if td.Variety == TypeVarietyUnion {
			for _, member := range td.MemberTypes {
				if member.Final&FinalUnion != 0 {
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "simpleType", td.Name.Local,
						"Derivation by union is forbidden by the member type '"+member.Name.Local+"'."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
		}
	}
}

// checkFinalOnSubstGroups checks that substitution group members do not violate
// the head element's final constraint.
func (c *compiler) checkFinalOnSubstGroups() {
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
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaElemDeclError(c.filename, src.line, member.Name.Local,
						"The substitution group affiliation is forbidden by the head element's final value."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
			if head.Final&FinalRestriction != 0 && derivationUsesMethod(member.Type, head.Type, DerivationRestriction) {
				if src, ok := c.globalElemSources[member]; ok {
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaElemDeclError(c.filename, src.line, member.Name.Local,
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

// resolveQName resolves a prefixed name (like "xsd:string") to a QName
// using the namespace declarations in scope on the given element.
func (c *compiler) resolveQName(elem *helium.Element, ref string) QName {
	local := ref
	ns := c.schema.targetNamespace

	for i := 0; i < len(ref); i++ {
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
