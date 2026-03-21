package xsd

import (
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// checkUPA checks that a content model is deterministic (Unique Particle Attribution).
// A non-deterministic model means that when an element arrives, there's ambiguity
// about which particle it belongs to.
func (c *compiler) checkUPA(td *TypeDef, src typeDefSource) {
	if td.ContentModel == nil {
		return
	}
	if !modelGroupIsDeterministic(td.ContentModel, c.schema) {
		component := "local complex type"
		if !src.isLocal {
			component = td.Name.Local
		}
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
			"The content model is not determinist."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}

// modelGroupIsDeterministic checks if a content model satisfies UPA.
func modelGroupIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	switch mg.Compositor {
	case CompositorChoice:
		return choiceIsDeterministic(mg, schema)
	case CompositorSequence:
		return sequenceIsDeterministic(mg, schema)
	}
	return true
}

// choiceIsDeterministic checks that no two particles in a choice can match the same element.
func choiceIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	// Check each pair of particles for first-set overlap.
	for i := 0; i < len(mg.Particles); i++ {
		for j := i + 1; j < len(mg.Particles); j++ {
			if particleFirstSetsOverlap(mg.Particles[i], mg.Particles[j], schema) {
				return false
			}
		}
		// Recurse into nested model groups.
		if nested, ok := mg.Particles[i].Term.(*ModelGroup); ok {
			if !modelGroupIsDeterministic(nested, schema) {
				return false
			}
		}
	}
	return true
}

// sequenceIsDeterministic checks that adjacent particles in a sequence don't create ambiguity.
func sequenceIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	for i := 0; i < len(mg.Particles); i++ {
		p := mg.Particles[i]
		// Recurse into nested model groups.
		if nested, ok := p.Term.(*ModelGroup); ok {
			if !modelGroupIsDeterministic(nested, schema) {
				return false
			}
		}
		if i+1 < len(mg.Particles) {
			next := mg.Particles[i+1]
			if canRepeatOrEnd(p) {
				// When current particle can repeat or end, check two overlaps:
				// 1. Current's first-set vs next's first-set (can a new element
				//    start either a new repetition of current OR the next particle?)
				if particleFirstSetsOverlap(p, next, schema) {
					return false
				}
				// 2. Current's last-set vs next's first-set.
				if particleLastFirstOverlap(p, next, schema) {
					return false
				}
			}
		}
	}
	return true
}

// canRepeatOrEnd returns true if the particle could match more or stop matching,
// creating ambiguity about whether the next element belongs to this particle or the next.
func canRepeatOrEnd(p *Particle) bool {
	if p.MaxOccurs == Unbounded {
		return true
	}
	if p.MinOccurs < p.MaxOccurs {
		return true
	}
	// Check nested model groups with variable repetition.
	if mg, ok := p.Term.(*ModelGroup); ok {
		if mg.MinOccurs < mg.MaxOccurs || mg.MaxOccurs == Unbounded {
			return true
		}
	}
	return false
}

// particleFirstSetsOverlap checks if two particles can both match the same starting element.
func particleFirstSetsOverlap(p1, p2 *Particle, schema *Schema) bool {
	first1 := particleFirstSet(p1, schema)
	first2 := particleFirstSet(p2, schema)
	return firstSetsOverlap(first1, first2)
}

// particleLastFirstOverlap checks if the last elements matchable by p1 overlap
// with the first elements matchable by p2.
func particleLastFirstOverlap(p1, p2 *Particle, schema *Schema) bool {
	last1 := particleLastSet(p1, schema)
	first2 := particleFirstSet(p2, schema)
	return firstSetsOverlap(last1, first2)
}

// firstSetEntry represents an element or wildcard in a first/last set.
type firstSetEntry struct {
	qname    QName  // for elements
	wildcard string // for wildcards (namespace constraint)
	targetNS string // for wildcards
}

// particleFirstSet returns the set of elements/wildcards that can appear first.
func particleFirstSet(p *Particle, schema *Schema) []firstSetEntry {
	switch term := p.Term.(type) {
	case *ElementDecl:
		entries := make([]firstSetEntry, 1, 1+len(schema.substGroups[term.Name]))
		entries[0] = firstSetEntry{qname: term.Name}
		for _, member := range schema.substGroups[term.Name] {
			entries = append(entries, firstSetEntry{qname: member.Name})
		}
		return entries
	case *Wildcard:
		return []firstSetEntry{{wildcard: term.Namespace, targetNS: term.TargetNS}}
	case *ModelGroup:
		return modelGroupFirstSet(term, schema)
	}
	return nil
}

// particleLastSet returns the set of elements/wildcards that can appear last.
func particleLastSet(p *Particle, schema *Schema) []firstSetEntry {
	switch term := p.Term.(type) {
	case *ElementDecl:
		entries := make([]firstSetEntry, 1, 1+len(schema.substGroups[term.Name]))
		entries[0] = firstSetEntry{qname: term.Name}
		for _, member := range schema.substGroups[term.Name] {
			entries = append(entries, firstSetEntry{qname: member.Name})
		}
		return entries
	case *Wildcard:
		return []firstSetEntry{{wildcard: term.Namespace, targetNS: term.TargetNS}}
	case *ModelGroup:
		return modelGroupLastSet(term, schema)
	}
	return nil
}

// modelGroupFirstSet returns the first set for a model group.
func modelGroupFirstSet(mg *ModelGroup, schema *Schema) []firstSetEntry {
	switch mg.Compositor {
	case CompositorSequence:
		// First set = first set of first non-optional particle
		// (union with subsequent particles if earlier ones are optional).
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
			if p.MinOccurs > 0 {
				break // this particle is required, stop here
			}
		}
		return result
	case CompositorChoice:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
		}
		return result
	case CompositorAll:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
		}
		return result
	}
	return nil
}

// modelGroupLastSet returns the last set for a model group.
func modelGroupLastSet(mg *ModelGroup, schema *Schema) []firstSetEntry {
	switch mg.Compositor {
	case CompositorSequence:
		// Last set = last set of last non-optional particle
		// (union with previous particles if later ones are optional).
		var result []firstSetEntry
		for i := len(mg.Particles) - 1; i >= 0; i-- {
			p := mg.Particles[i]
			result = append(result, particleLastSet(p, schema)...)
			if p.MinOccurs > 0 {
				break
			}
		}
		return result
	case CompositorChoice:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleLastSet(p, schema)...)
		}
		return result
	}
	return nil
}

// firstSetsOverlap checks if two first-sets have any overlapping entries.
func firstSetsOverlap(a, b []firstSetEntry) bool {
	for _, ea := range a {
		for _, eb := range b {
			if entriesOverlap(ea, eb) {
				return true
			}
		}
	}
	return false
}

// entriesOverlap checks if two first-set entries can match the same element.
func entriesOverlap(a, b firstSetEntry) bool {
	// Two elements overlap if they have the same QName.
	if a.wildcard == "" && b.wildcard == "" {
		return a.qname == b.qname
	}
	// Element vs wildcard: check if the wildcard's namespace matches the element's namespace.
	if a.wildcard == "" && b.wildcard != "" {
		return wildcardMatchesNS(b.wildcard, b.targetNS, a.qname.NS)
	}
	if a.wildcard != "" && b.wildcard == "" {
		return wildcardMatchesNS(a.wildcard, a.targetNS, b.qname.NS)
	}
	// Two wildcards: check if their namespace constraints can both match the same namespace.
	return wildcardsOverlap(a, b)
}

// wildcardsOverlap checks if two wildcard entries can match the same namespace.
func wildcardsOverlap(a, b firstSetEntry) bool {
	aNS, aTarget := a.wildcard, a.targetNS
	bNS, bTarget := b.wildcard, b.targetNS

	// ##any overlaps with everything.
	if aNS == "##any" || bNS == "##any" {
		return true
	}

	// ##other matches any NS except targetNS and empty.
	// ##local matches only empty NS.
	// So ##other and ##local are always disjoint.
	if (aNS == "##other" && bNS == "##local") || (aNS == "##local" && bNS == "##other") {
		return false
	}

	// ##other vs ##other: overlap if there's any NS outside both targets
	// (practically always true unless both have the same target).
	if aNS == "##other" && bNS == "##other" {
		return true
	}

	// ##local vs ##local: obviously overlap.
	if aNS == "##local" && bNS == "##local" {
		return true
	}

	// For enumerated sets vs keywords, expand and check.
	aSet := upaWildcardNSSet(aNS, aTarget)
	bSet := upaWildcardNSSet(bNS, bTarget)

	// If either is nil, it means "any except target" — use conservative overlap.
	if aSet == nil || bSet == nil {
		return true
	}
	for ns := range aSet {
		if bSet[ns] {
			return true
		}
	}
	return false
}

// upaWildcardNSSet returns the explicit set of namespaces a wildcard can match,
// or nil if the wildcard matches an open-ended set (##other, ##any).
func upaWildcardNSSet(wcNS, targetNS string) map[string]bool {
	switch wcNS {
	case "##any", "##other", "##not-absent":
		return nil
	case "##local":
		return map[string]bool{"": true}
	default:
		result := make(map[string]bool)
		for _, part := range strings.Fields(wcNS) {
			switch part {
			case "##local":
				result[""] = true
			case "##targetNamespace":
				result[targetNS] = true
			default:
				result[part] = true
			}
		}
		return result
	}
}

// wildcardMatchesNS checks if a wildcard namespace constraint matches a given namespace.
func wildcardMatchesNS(wcNS, wcTargetNS, elemNS string) bool {
	switch wcNS {
	case "##any":
		return true
	case "##other":
		return elemNS != "" && elemNS != wcTargetNS
	case "##not-absent":
		return elemNS != ""
	default:
		for _, part := range strings.Fields(wcNS) {
			switch part {
			case "##local":
				if elemNS == "" {
					return true
				}
			case "##targetNamespace":
				if elemNS == wcTargetNS {
					return true
				}
			default:
				if elemNS == part {
					return true
				}
			}
		}
		return false
	}
}
