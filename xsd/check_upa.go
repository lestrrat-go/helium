package xsd

import (
	"context"
	"slices"
)

const componentLocalComplexType = "local complex type"
const componentLocalSimpleType = "local simple type"

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
		c.schemaError(ctx, schemaComponentError(c.filename, src.line, "complexType", component,
			"The content model is not determinist."))
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

// upaMaxRequiredExpansion caps how many required occurrence copies a single
// range may expand into, to bound automaton size. XSD finite minOccurs is
// typically small; past this bound the required copies are summarized as TWO
// non-nullable body copies rather than the full chain. Two copies (not one) are
// kept because for determinism analysis `U{n}` with a non-nullable unit U and
// n>=2 is invariant in n: `U{2}` already realizes every inter-copy
// boundary-followpos overlap, so it correctly rejects `(a, a?){257}` while a
// single copy would drop the boundary and false-accept it. A required exact run
// is a deterministic chain regardless of length, so the summary copies are NEVER
// given a self-loop back-edge just because they were collapsed — a back-edge
// would manufacture ambiguity that the full chain does not have (e.g.
// `a{257}, a` is deterministic). The summary loops only when the original range
// genuinely repeats past the required count (unbounded maxOccurs, or a repeating
// optional remainder), handled separately below.
const upaMaxRequiredExpansion = 256

// walkParticle computes nullable/firstpos/lastpos for a particle, accounting for
// its own minOccurs/maxOccurs repetition, and records the followpos edges that
// repetition introduces.
func (a *positionAutomaton) walkParticle(p *Particle) posInfo {
	// A maxOccurs="0" particle contributes nothing to the content model.
	if p.MaxOccurs == 0 {
		return posInfo{nullable: true}
	}
	// A particle wrapping a model group carries the same occurrence range as the
	// group itself (the parser stores it on both). The group's own occurrence is
	// folded in by walkModelGroup, so applying it here too would double-count it
	// (e.g. expanding `(a|b){1,3}` into four copies instead of two).
	if _, ok := p.Term.(*ModelGroup); ok {
		return a.walkTerm(p.Term)
	}
	return a.applyOccurs(p.MinOccurs, p.MaxOccurs, func() posInfo {
		return a.walkTerm(p.Term)
	})
}

// applyOccurs folds a minOccurs/maxOccurs occurrence range over a body. The body
// closure walks the underlying term/group fresh on each call, allocating new
// automaton positions every time, so distinct occurrences become distinct
// automaton positions.
//
// When the body is NON-nullable, the required copies (minOccurs) are EXPANDED
// into a strict chain of distinct positions, and the optional remainder
// (maxOccurs-minOccurs, or unbounded) is modeled as a SINGLE additional body
// copy that self-loops only if it may repeat. Expanding the required chain keeps
// counted models such as `a{2}, a` deterministic — the loop-only model used
// previously collapsed the required count and falsely flagged them. The optional
// remainder stays a single copy, never expanded, because its iterations share
// the same term and are interchangeable under XSD's per-particle UPA; expanding
// them would falsely flag e.g. `<any maxOccurs="5"/>`.
//
// When the body is NULLABLE, expanding it is unsound: a skipped occurrence lets
// later copies' positions become reachable alongside earlier ones, manufacturing
// ambiguity that XSD's per-particle UPA does not see (e.g. `(a?, b?){1,3}`). In
// that case the whole range collapses to a single loop copy, matching the
// original construction.
func (a *positionAutomaton) applyOccurs(minOccurs, maxOccurs int, body func() posInfo) posInfo {
	// Build the first copy and inspect it. Probing nullability with a throwaway
	// walk would double-allocate positions, so this copy is always reused below.
	first := body()

	// Nullable body, or a single occurrence: fall back to the loop model
	// (self-loop iff the body may repeat). Expanding a nullable body is unsound —
	// a skipped occurrence makes later copies' positions reachable alongside
	// earlier ones, manufacturing ambiguity XSD's per-particle UPA never sees.
	if first.nullable || (minOccurs <= 1 && maxOccurs == 1) {
		if a.mayRepeat(minOccurs, maxOccurs) {
			for _, l := range first.last {
				a.addFollow(l, first.first)
			}
		}
		if minOccurs == 0 {
			first.nullable = true
		}
		return first
	}

	// Non-nullable body. Expand the required copies (minOccurs) into a strict
	// chain of distinct positions, then attach at most ONE optional remainder
	// copy. The chain keeps counted models such as `a{2}, a` deterministic; the
	// single optional copy avoids falsely flagging interchangeable repeated copies
	// (e.g. `<any maxOccurs="5"/>`).
	//
	// Past upaMaxRequiredExpansion the required chain is SUMMARIZED rather than
	// fully expanded, to bound automaton size. The summary must keep TWO copies,
	// not one: for determinism analysis `U{n}` with a non-nullable unit U and
	// n>=2 is invariant in n — every inter-copy boundary-followpos overlap that
	// can ever occur is already realized within `U{2}`; additional middle copies
	// only repeat the same boundary pattern. Collapsing to a single copy would
	// DROP that inter-copy boundary and false-accept models such as `(a, a?){257}`
	// (whose `(a, a?){2}` form is correctly rejected). Two copies preserve the
	// boundary while still bounding size.
	//
	// The summary copies carry NO back-edge: a required exact run is a
	// deterministic chain regardless of length, and the optional-tail shape below
	// is derived from the ORIGINAL range, never widened by the collapse, so a
	// finite exact count such as `a{257}` stays a non-looping required run
	// (`a{257}, a` keeps the deterministic `a{3}` shape). The body is guaranteed
	// non-nullable here (the nullable case returned via the loop model above), so
	// the n>=2 invariant holds.
	required := minOccurs
	if required > upaMaxRequiredExpansion {
		required = 2
	}

	optionalUnbounded := maxOccurs == Unbounded
	optionalCount := 0
	if !optionalUnbounded {
		optionalCount = maxOccurs - minOccurs
	}
	hasTail := optionalUnbounded || optionalCount >= 1
	// The optional remainder self-loops when it can occur more than once.
	tailLoops := optionalUnbounded || optionalCount >= 2

	// `first` is the first required copy when one is required; otherwise it serves
	// as the (single) optional remainder copy.
	if required == 0 {
		if !hasTail {
			// minOccurs==0 and maxOccurs==0 is handled by the caller; this is
			// defensive.
			first.nullable = true
			return first
		}
		if tailLoops {
			for _, l := range first.last {
				a.addFollow(l, first.first)
			}
		}
		first.nullable = true
		return first
	}

	acc := first
	for i := 1; i < required; i++ {
		acc = a.concat(acc, body())
	}

	if hasTail {
		tail := body()
		if tailLoops {
			for _, l := range tail.last {
				a.addFollow(l, tail.first)
			}
		}
		tail.nullable = true
		acc = a.concat(acc, tail)
	}

	return acc
}

// mayRepeat reports whether an occurrence range allows the body to appear more
// than once (and therefore needs a back-edge in the loop model).
func (a *positionAutomaton) mayRepeat(minOccurs, maxOccurs int) bool {
	return maxOccurs == Unbounded || maxOccurs > 1 || minOccurs > 1
}

// concat sequentially composes two segments (Glushkov concatenation), recording
// the followpos edges from the left segment's lastpos into the right segment's
// firstpos.
func (a *positionAutomaton) concat(left, right posInfo) posInfo {
	var out posInfo
	out.first = slices.Clone(left.first)
	if left.nullable {
		out.first = append(out.first, right.first...)
	}
	for _, l := range left.last {
		a.addFollow(l, right.first)
	}
	if right.nullable {
		out.last = append(slices.Clone(left.last), right.last...)
	} else {
		out.last = slices.Clone(right.last)
	}
	out.nullable = left.nullable && right.nullable
	return out
}

// walkTerm computes nullable/firstpos/lastpos for a particle term.
func (a *positionAutomaton) walkTerm(term ParticleTerm) posInfo {
	switch t := term.(type) {
	case *ElementDecl:
		// An element leaf and each of its substitution-group members is its own
		// position; any of them can match where the element is expected. ABSTRACT
		// members are NOT skipped here: XSD 1.1 (§3.8.6.4 / cos-nonambig) bases the
		// actual substitution group on DECLARATION membership, so an abstract member
		// still COMPETES for unique particle attribution even though it can never
		// appear in an instance — e.g. element `e` + a local `e1` plus an abstract
		// `e1 substitutionGroup="e"` IS a UPA violation in 1.1 (W3C wgData/sg/upa.xsd,
		// bug 4337). (Instance MATCHING, by contrast, does skip abstract members via
		// elemMatchesDeclOrSubst — that is a different question.)
		var info posInfo
		ids := []int{a.newPos(firstSetEntry{qname: t.Name})}
		// TRANSITIVE declaration-membership closure (abstract members included) — a
		// direct substGroups lookup misses a multi-level chain h<-m1<-m2 and would
		// miss a transitive UPA conflict.
		for _, member := range substitutableMembersFor(t, a.schema) {
			ids = append(ids, a.newPos(firstSetEntry{qname: member.Name}))
		}
		info.first = ids
		info.last = slices.Clone(ids)
		return info
	case *Wildcard:
		id := a.newPos(firstSetEntry{isWildcard: true, wildcard: t.Namespace, targetNS: t.TargetNS, wc: t})
		return posInfo{first: []int{id}, last: []int{id}}
	case *ModelGroup:
		return a.walkModelGroup(t)
	}
	return posInfo{nullable: true}
}

// walkModelGroup computes nullable/firstpos/lastpos for a model group,
// recording the followpos edges its compositor introduces and folding in the
// group's own minOccurs/maxOccurs occurrence range.
func (a *positionAutomaton) walkModelGroup(mg *ModelGroup) posInfo {
	// A prohibited (maxOccurs="0") model group emits nothing and is unreachable,
	// so it contributes NO positions or followpos edges to the automaton. This
	// mirrors walkParticle's maxOccurs==0 guard, which it bypasses when the group
	// is reached through walkParticle's model-group short-circuit: there the
	// occurrence range lives on the group, and the wrapping synthetic/parent
	// particle keeps maxOccurs>=1. Without this guard, a top-level prohibited
	// group's members leak their firstpos into the start state and can falsely
	// flag the (empty) content model — e.g. a `maxOccurs="0"` choice with two
	// same-name branches.
	if mg.MaxOccurs == 0 {
		return posInfo{nullable: true}
	}
	switch mg.Compositor {
	case CompositorChoice, CompositorSequence, CompositorAll:
		return a.applyOccurs(mg.MinOccurs, mg.MaxOccurs, func() posInfo {
			return a.walkCompositorBody(mg)
		})
	}
	return posInfo{nullable: true}
}

// walkCompositorBody computes nullable/firstpos/lastpos for one occurrence of a
// model group's compositor body, ignoring the group's own occurrence range.
func (a *positionAutomaton) walkCompositorBody(mg *ModelGroup) posInfo {
	if mg.Compositor == CompositorChoice {
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
		return info
	}

	if mg.Compositor == CompositorAll {
		return a.walkAllBody(mg)
	}

	// CompositorSequence. Sequentially concatenate each member, recording the
	// followpos edges that ordering introduces.
	info := posInfo{nullable: true}
	for _, p := range mg.Particles {
		info = a.concat(info, a.walkParticle(p))
	}
	if len(mg.Particles) == 0 {
		info.nullable = true
	}
	return info
}

// walkAllBody computes nullable/firstpos/lastpos for one occurrence of an
// xs:all compositor body. Unlike a sequence, xs:all is order-independent: every
// member is reachable regardless of which members have already been seen. So all
// member firstpos are competing from the start state, and after any member is
// consumed every OTHER member is still reachable. Modeling this faithfully
// (rather than as an ordered sequence) is what catches a duplicate same-name
// member — two members with the same element name overlap in the union of
// firstpos and fire the cos-nonambig (UPA) check.
func (a *positionAutomaton) walkAllBody(mg *ModelGroup) posInfo {
	if len(mg.Particles) == 0 {
		return posInfo{nullable: true}
	}

	members := make([]posInfo, 0, len(mg.Particles))
	for _, p := range mg.Particles {
		members = append(members, a.walkParticle(p))
	}

	var out posInfo
	out.nullable = true
	for _, m := range members {
		out.first = append(out.first, m.first...)
		out.last = append(out.last, m.last...)
		if !m.nullable {
			out.nullable = false
		}
	}

	// Mutual reachability: each member's lastpos may be followed by every other
	// member's firstpos (the other members can still appear in any order).
	for i, mi := range members {
		for j, mj := range members {
			if i == j {
				continue
			}
			for _, l := range mi.last {
				a.addFollow(l, mj.first)
			}
		}
	}

	return out
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
			version := Version10
			if a.schema != nil {
				version = a.schema.version
			}
			if entriesOverlap(a.positions[pi].entry, a.positions[pj].entry, version) {
				return false
			}
		}
	}
	return true
}

// firstSetEntry represents an element or wildcard in a first/last set. The
// isWildcard discriminator distinguishes the two cases EXPLICITLY: an empty
// wildcard string is NOT a reliable sentinel, because a wildcard with a
// present-but-empty namespace="" constraint (a degenerate list matching nothing,
// preserved by readWildcard) legitimately carries wildcard == "". Overloading
// the empty string to also mean "this is an element" would mis-treat such a
// wildcard as an element and false-reject deterministic models.
type firstSetEntry struct {
	qname      QName     // for elements
	isWildcard bool      // true iff this entry is a wildcard position
	wildcard   string    // for wildcards (namespace constraint)
	targetNS   string    // for wildcards
	wc         *Wildcard // for wildcards: the full term (carries XSD 1.1 notNamespace/notQName)
}

// entriesOverlap checks if two first-set entries can match the same element.
//
// In XSD 1.1 wildcards are "weak": when an element particle and a wildcard
// compete for the same element name, the element declaration wins, so the pair
// is NOT a UPA (cos-nonambig) violation. In XSD 1.0 such a pair IS ambiguous.
// Element-vs-element and wildcard-vs-wildcard overlap are unchanged across
// versions.
//
// Note: this relaxation is the compile-time half of XSD 1.1 weak wildcards.
// The validation half — element-over-wildcard precedence — is enforced for the
// CHOICE case by matchChoice/tryMatchChoice (validate_elem.go), so a wildcard
// no longer steals a child attributable to a competing element declaration
// regardless of declaration order. The remaining gap is the SEQUENCE case (a
// minOccurs=0 wildcard preceding an element in a sequence), which the
// position-based sequence matcher does not yet override.
func entriesOverlap(a, b firstSetEntry, version Version) bool {
	// Two elements overlap if they have the same QName.
	if !a.isWildcard && !b.isWildcard {
		return a.qname == b.qname
	}
	// Element vs wildcard: in 1.1 the element wins (no conflict); in 1.0 they
	// overlap when the wildcard's namespace admits the element's namespace.
	if !a.isWildcard && b.isWildcard {
		if version == Version11 {
			return false
		}
		return entryWildcardMatchesNS(b, a.qname.NS)
	}
	if a.isWildcard && !b.isWildcard {
		if version == Version11 {
			return false
		}
		return entryWildcardMatchesNS(a, b.qname.NS)
	}
	// Two wildcards: check if their namespace constraints can both match the same namespace.
	return wildcardsOverlap(a, b)
}

// wildcardsOverlap reports whether two wildcards can both match some common
// element name — i.e. whether their namespace constraints INTERSECT. Two
// wildcards whose namespace sets are DISJOINT are deterministic when adjacent
// (after an element you can always tell which wildcard matched), so they must
// NOT be treated as a UPA overlap.
//
// A namespace constraint is one of two shapes (XSD 3.10.6, matching this
// package's ##other = not(targetNS) and not(absent) semantics — see
// wildcardMatchesNS): a finite SET of namespace URIs, or the NEGATION of a
// single namespace (##other negates the target namespace, ##not-absent negates
// the absent namespace, ##any negates nothing). The cases below cover every
// set/negation combination so a negation (open-ended) constraint no longer
// falls through to a blanket "always overlaps".
func wildcardsOverlap(a, b firstSetEntry) bool {
	// XSD 1.1: when either wildcard carries a notNamespace/notQName constraint
	// the string-based set/negation analysis below cannot represent it; decide
	// namespace intersection via the general constraint algebra.
	if (a.wc != nil && wildcardHas11Fields(a.wc)) || (b.wc != nil && wildcardHas11Fields(b.wc)) {
		return constraintsIntersect(wildcardConstraint(a.wc), wildcardConstraint(b.wc))
	}
	aSet := upaWildcardNSSet(a.wildcard, a.targetNS)
	bSet := upaWildcardNSSet(b.wildcard, b.targetNS)

	// Both are finite sets: overlap iff they share a member.
	if aSet != nil && bSet != nil {
		for ns := range aSet {
			if bSet[ns] {
				return true
			}
		}
		return false
	}

	// Both are negations: not(x) and not(y) always share something (any
	// namespace other than x and y, and there are infinitely many).
	if aSet == nil && bSet == nil {
		return true
	}

	// One negation, one finite set. They overlap iff the negation admits any
	// namespace in the set. wildcardMatchesNS already encodes this package's
	// negation semantics (##other = not(targetNS) and not(absent),
	// ##not-absent = not(absent), ##any = everything), so reuse it per member
	// rather than re-deriving the excluded namespaces here.
	neg := a
	set := bSet
	if aSet != nil {
		neg = b
		set = aSet
	}
	for ns := range set {
		if wildcardMatchesNS(neg.wildcard, neg.targetNS, ns) {
			return true
		}
	}
	return false
}

// upaWildcardNSSet returns the explicit set of namespaces a wildcard can match,
// or nil if the wildcard is a NEGATION constraint (##any, ##other,
// ##not-absent) whose membership is open-ended.
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

// entryWildcardMatchesNS reports whether a wildcard first-set entry admits a
// namespace, honoring an XSD 1.1 notNamespace constraint when the full *Wildcard
// is available (it always is for entries built by this package) and falling back
// to the string form otherwise.
func entryWildcardMatchesNS(e firstSetEntry, ns string) bool {
	if e.wc != nil {
		return wildcardMatches(e.wc, ns)
	}
	return wildcardMatchesNS(e.wildcard, e.targetNS, ns)
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
