package xsd

import (
	"context"
	"slices"

	helium "github.com/lestrrat-go/helium"
)

const componentLocalComplexType = "local complex type"

// checkUPA checks that a content model is deterministic (Unique Particle Attribution).
// A non-deterministic model means that when an element arrives, there's ambiguity
// about which particle it belongs to.
func (c *compiler) checkUPA(ctx context.Context, td *TypeDef, src typeDefSource) {
	if td.ContentModel == nil {
		return
	}
	if !modelGroupIsDeterministic(td.ContentModel, c.schema) {
		component := componentLocalComplexType
		if !src.isLocal {
			component = td.Name.Local
		}
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component,
			"The content model is not determinist."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}

// modelGroupIsDeterministic checks if a content model satisfies UPA
// (Unique Particle Attribution, a.k.a. cos-nonambig).
//
// It builds a position automaton (Glushkov construction) over the content-model
// particle tree: every element/wildcard leaf particle becomes a numbered
// position, and the structure yields nullable/firstpos/lastpos/followpos. A
// model is ambiguous if, from any single state, two distinct positions are
// reachable that can both match the same element name (or via overlapping
// wildcards). The two competing states are firstpos(root) (the start state) and
// followpos(p) for every position p.
//
// This subsumes the older adjacent first/last heuristic: it catches
// non-adjacent ambiguity such as `a?, b?, a`, where skipping the optional first
// `a` makes the final `a` reachable from the same state as the first.
func modelGroupIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	a := newPositionAutomaton(schema)
	a.add(&Particle{MinOccurs: 1, MaxOccurs: 1, Term: mg})
	return a.isDeterministic()
}

// upaPosition is a single leaf particle (element or wildcard) in the position
// automaton. The entry captures everything needed to test name/wildcard overlap
// against another position.
type upaPosition struct {
	id    int
	entry firstSetEntry
}

// positionAutomaton accumulates nullable/firstpos/lastpos/followpos for a
// content-model particle tree using a bottom-up Glushkov construction.
type positionAutomaton struct {
	schema    *Schema
	positions []upaPosition
	followpos map[int][]int
}

func newPositionAutomaton(schema *Schema) *positionAutomaton {
	return &positionAutomaton{schema: schema, followpos: make(map[int][]int)}
}

// posInfo is the per-node result of the Glushkov walk.
type posInfo struct {
	nullable bool
	first    []int
	last     []int
}

// newPos allocates a fresh position for a leaf entry and returns it.
func (a *positionAutomaton) newPos(entry firstSetEntry) int {
	id := len(a.positions)
	a.positions = append(a.positions, upaPosition{id: id, entry: entry})
	return id
}

// add registers a top-level particle (the synthetic root particle wrapping the
// whole content model) and seeds the start-state choices (firstpos of the root)
// under the synthetic predecessor key -1.
func (a *positionAutomaton) add(p *Particle) posInfo {
	info := a.walkParticle(p)
	a.followpos[-1] = append(a.followpos[-1], info.first...)
	return info
}

// walkParticle computes nullable/firstpos/lastpos for a particle, accounting for
// its own minOccurs/maxOccurs repetition, and records the followpos edges that
// repetition introduces.
func (a *positionAutomaton) walkParticle(p *Particle) posInfo {
	// A maxOccurs="0" particle contributes nothing to the content model.
	if p.MaxOccurs == 0 {
		return posInfo{nullable: true}
	}

	base := a.walkTerm(p.Term)

	// Repetition (maxOccurs > 1 or unbounded) lets the particle's lastpos flow
	// back into its firstpos.
	if p.MaxOccurs == Unbounded || p.MaxOccurs > 1 {
		for _, l := range base.last {
			a.addFollow(l, base.first)
		}
	}

	// An optional particle (minOccurs == 0) is nullable.
	nullable := base.nullable || p.MinOccurs == 0
	return posInfo{nullable: nullable, first: base.first, last: base.last}
}

// walkTerm computes nullable/firstpos/lastpos for a particle term.
func (a *positionAutomaton) walkTerm(term ParticleTerm) posInfo {
	switch t := term.(type) {
	case *ElementDecl:
		// An element leaf and each of its substitution-group members is its own
		// position; any of them can match where the element is expected.
		var info posInfo
		ids := []int{a.newPos(firstSetEntry{qname: t.Name})}
		for _, member := range a.schema.substGroups[t.Name] {
			ids = append(ids, a.newPos(firstSetEntry{qname: member.Name}))
		}
		info.first = ids
		info.last = slices.Clone(ids)
		return info
	case *Wildcard:
		id := a.newPos(firstSetEntry{wildcard: t.Namespace, targetNS: t.TargetNS})
		return posInfo{first: []int{id}, last: []int{id}}
	case *ModelGroup:
		return a.walkModelGroup(t)
	}
	return posInfo{nullable: true}
}

// walkModelGroup computes nullable/firstpos/lastpos for a model group and
// records the followpos edges its compositor introduces.
func (a *positionAutomaton) walkModelGroup(mg *ModelGroup) posInfo {
	switch mg.Compositor {
	case CompositorChoice:
		var info posInfo
		for _, p := range mg.Particles {
			ci := a.walkParticle(p)
			info.first = append(info.first, ci.first...)
			info.last = append(info.last, ci.last...)
			if ci.nullable {
				info.nullable = true
			}
		}
		// An empty choice matches nothing; treat it as nullable for safety.
		if len(mg.Particles) == 0 {
			info.nullable = true
		}
		return a.applyGroupRepetition(mg, info)
	case CompositorSequence, CompositorAll:
		// xs:all is order-independent at validation time, but for UPA purposes a
		// sequence-style concatenation is a sound over-approximation: any false
		// edge it adds can only widen reachability, and xs:all members are
		// already constrained to be distinct elements by other checks.
		info := posInfo{nullable: true}
		for _, p := range mg.Particles {
			ci := a.walkParticle(p)
			// followpos: lastpos of everything matchable so far flows into the
			// firstpos of this particle.
			if info.nullable {
				info.first = append(info.first, ci.first...)
			}
			for _, l := range info.last {
				a.addFollow(l, ci.first)
			}
			if info.nullable {
				// prior segment could be empty: this particle may itself end the
				// segment, so its lastpos joins the running lastpos.
				info.last = append(info.last, ci.last...)
			} else {
				info.last = ci.last
			}
			info.nullable = info.nullable && ci.nullable
		}
		if len(mg.Particles) == 0 {
			info.nullable = true
		}
		return a.applyGroupRepetition(mg, info)
	}
	return posInfo{nullable: true}
}

// applyGroupRepetition folds a model group's own minOccurs/maxOccurs into the
// computed posInfo, mirroring walkParticle's repetition handling.
func (a *positionAutomaton) applyGroupRepetition(mg *ModelGroup, info posInfo) posInfo {
	if mg.MaxOccurs == Unbounded || mg.MaxOccurs > 1 {
		for _, l := range info.last {
			a.addFollow(l, info.first)
		}
	}
	if mg.MinOccurs == 0 {
		info.nullable = true
	}
	return info
}

// addFollow records that every position in `to` may follow `from`.
func (a *positionAutomaton) addFollow(from int, to []int) {
	if len(to) == 0 {
		return
	}
	a.followpos[from] = append(a.followpos[from], to...)
}

// isDeterministic returns true if no automaton state offers two distinct
// competing positions that overlap on the same element name (or wildcard).
func (a *positionAutomaton) isDeterministic() bool {
	// The start state's choices are firstpos(root). Re-derive it as followpos of
	// a synthetic predecessor: every position reachable initially is in
	// followpos[-1].
	root := a.followpos[-1]
	if !a.stateUnambiguous(root) {
		return false
	}
	for from := range a.positions {
		if !a.stateUnambiguous(a.followpos[from]) {
			return false
		}
	}
	return true
}

// stateUnambiguous returns true if no two distinct positions reachable from one
// state can match the same element.
func (a *positionAutomaton) stateUnambiguous(reachable []int) bool {
	for i := range reachable {
		for j := i + 1; j < len(reachable); j++ {
			pi, pj := reachable[i], reachable[j]
			if pi == pj {
				continue
			}
			if entriesOverlap(a.positions[pi].entry, a.positions[pj].entry) {
				return false
			}
		}
	}
	return true
}

// firstSetEntry represents an element or wildcard in a first/last set.
type firstSetEntry struct {
	qname    QName  // for elements
	wildcard string // for wildcards (namespace constraint)
	targetNS string // for wildcards
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
	if aNS == WildcardNSAny || bNS == WildcardNSAny {
		return true
	}

	// ##other matches any NS except targetNS and empty.
	// ##local matches only empty NS.
	// So ##other and ##local are always disjoint.
	if (aNS == WildcardNSOther && bNS == WildcardNSLocal) || (aNS == WildcardNSLocal && bNS == WildcardNSOther) {
		return false
	}

	// ##other vs ##other: overlap if there's any NS outside both targets
	// (practically always true unless both have the same target).
	if aNS == WildcardNSOther && bNS == WildcardNSOther {
		return true
	}

	// ##local vs ##local: obviously overlap.
	if aNS == WildcardNSLocal && bNS == WildcardNSLocal {
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
	case WildcardNSAny, WildcardNSOther, WildcardNSNotAbsent:
		return nil
	case WildcardNSLocal:
		return map[string]bool{"": true}
	default:
		result := make(map[string]bool)
		for _, part := range splitSpace(wcNS) {
			switch part {
			case WildcardNSLocal:
				result[""] = true
			case WildcardNSTargetNamespace:
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
	case WildcardNSAny:
		return true
	case WildcardNSOther:
		return elemNS != "" && elemNS != wcTargetNS
	case WildcardNSNotAbsent:
		return elemNS != ""
	default:
		for _, part := range splitSpace(wcNS) {
			switch part {
			case WildcardNSLocal:
				if elemNS == "" {
					return true
				}
			case WildcardNSTargetNamespace:
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
