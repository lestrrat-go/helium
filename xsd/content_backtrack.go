package xsd

import (
	"context"
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
)

// Bounded occurrence backtracking for content-model matching.
//
// The greedy matcher (matchSequence/matchChoice/matchElementParticle) commits to
// the largest consumption at each step, so an occurrence-partition-ambiguous but
// UPA-clean content model — a repeating outer compositor over a repeatable inner
// particle — can false-reject a VALID instance by starving a later required
// occurrence (e.g. `sequence minOccurs=2 (a{1,2}, b?)` over `a,a,b`, where the
// first pass greedily takes both `a`).
//
// contentModelAccepts proves, WITHOUT side effects, that children[0:] can be
// fully consumed by the model group under SOME occurrence partition. It computes
// the set of reachable end positions for each (particle|group, position) state,
// memoized so the state space is O(#particles * #children) rather than
// exponential, with a hard cap (btStateCap) after which it fails closed (returns
// "not proven", keeping the greedy verdict).
//
// WILDCARD-FREE PRECONDITION: the backtracker runs ONLY on content models with no
// xs:any particle at any nesting depth (modelGroupHasWildcard). A separate,
// looser automaton than the greedy matcher would otherwise have to re-implement
// every XSD 1.1 element-over-wildcard precedence rule (choice commit-no-fallback,
// sequence reservation, xs:all attribution) to avoid admitting an instance the
// greedy matcher correctly rejects. Gating out wildcards eliminates that entire
// class in one rule: with no wildcards, element-over-wildcard precedence can never
// arise, and a wildcard-free UPA-deterministic content model's reachability
// automaton accepts exactly its regular language. A model containing any wildcard
// is left to the greedy matcher, whose (precedence-aware) verdict already stands —
// fail-closed, since the backtracker only ever runs after greedy already failed.
//
// It uses the SAME per-child predicate as the greedy matcher
// (elemMatchesDeclOrSubst), so it models exactly the greedy content-model
// language and never accepts a non-member. `xs:all` groups are matched greedily
// (deterministic per-member counting, no occurrence backtracking), which is
// conservative: it can only fail to prove, never over-accept.
//
// When it accepts, validateContentModelChildren validates each child's content
// against its (UPA-unique) element declaration.

const btStateCap = 200000

type reachKey struct {
	p   *Particle
	g   *ModelGroup
	pos int
}

type btMemo struct {
	cache  map[reachKey][]int
	states int
	capped bool
}

// contentModelAccepts reports whether children can be fully consumed by mg under
// some occurrence partition. It is pure (no side effects, no diagnostics), and
// runs only on wildcard-free content models (see the wildcard-free precondition
// above); a model with any wildcard is not proven here (defers to greedy).
func (vc *validationContext) contentModelAccepts(ctx context.Context, mg *ModelGroup, children []childElem) bool {
	if modelGroupHasWildcard(mg) {
		return false
	}
	m := &btMemo{cache: make(map[reachKey][]int)}
	ends := vc.btReachGroup(ctx, m, mg, children)
	if m.capped {
		return false
	}
	return slices.Contains(ends, len(children))
}

// modelGroupHasWildcard reports whether the model group tree contains any xs:any
// wildcard particle at any nesting depth.
func modelGroupHasWildcard(mg *ModelGroup) bool {
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *Wildcard:
			return true
		case *ModelGroup:
			if modelGroupHasWildcard(term) {
				return true
			}
		}
	}
	return false
}

func addUnique(dst []int, seen map[int]struct{}, e int) []int {
	if _, ok := seen[e]; ok {
		return dst
	}
	seen[e] = struct{}{}
	return append(dst, e)
}

// btReachParticle returns every end position reachable by matching particle p
// starting at pos. Results are memoized by (particle, pos).
func (vc *validationContext) btReachParticle(ctx context.Context, m *btMemo, p *Particle, children []childElem, pos int) []int {
	if m.capped {
		return nil
	}
	key := reachKey{p: p, pos: pos}
	if v, ok := m.cache[key]; ok {
		return v
	}
	m.states++
	if m.states > btStateCap {
		m.capped = true
		return nil
	}
	var out []int
	switch term := p.Term.(type) {
	case *ElementDecl:
		out = vc.btReachElem(p, term, children, pos)
	case *ModelGroup:
		// A group particle's occurrence lives on the ModelGroup term (matchParticle
		// reads term.MinOccurs/MaxOccurs, not p's), so delegate to the group reach.
		out = vc.btReachGroupAt(ctx, m, term, children, pos)
	}
	// A *Wildcard term cannot occur: contentModelAccepts gates out wildcard-bearing
	// models before any reach runs.
	m.cache[key] = out
	return out
}

// btReachElem returns pos+k for every occurrence count k in [MinOccurs, feasible]
// where the first k children from pos all match the element declaration.
func (vc *validationContext) btReachElem(p *Particle, edecl *ElementDecl, children []childElem, pos int) []int {
	maxc := 0
	for pos+maxc < len(children) && elemMatchesDeclOrSubst(children[pos+maxc], edecl, vc.schema) {
		maxc++
		if p.MaxOccurs != Unbounded && maxc >= p.MaxOccurs {
			break
		}
	}
	if maxc < p.MinOccurs {
		return nil
	}
	out := make([]int, 0, maxc-p.MinOccurs+1)
	for k := p.MinOccurs; k <= maxc; k++ {
		out = append(out, pos+k)
	}
	return out
}

// btReachGroup computes reachable end positions for the TOP model group starting
// at position 0 (applying mg's own occurrence).
func (vc *validationContext) btReachGroup(ctx context.Context, m *btMemo, mg *ModelGroup, children []childElem) []int {
	return vc.btReachGroupAt(ctx, m, mg, children, 0)
}

// btReachGroupAt returns reachable end positions for mg starting at pos, applying
// mg's compositor semantics and its own {MinOccurs, MaxOccurs} occurrence.
func (vc *validationContext) btReachGroupAt(ctx context.Context, m *btMemo, mg *ModelGroup, children []childElem, pos int) []int {
	if m.capped {
		return nil
	}
	key := reachKey{g: mg, pos: pos}
	if v, ok := m.cache[key]; ok {
		return v
	}
	m.states++
	if m.states > btStateCap {
		m.capped = true
		return nil
	}

	// An xs:all group is matched greedily (no occurrence backtracking) — its per-
	// member counting matcher already decides the single deterministic end. This
	// is conservative: it can fail to prove but never over-accepts.
	if mg.Compositor == CompositorAll {
		out := vc.btReachAll(ctx, mg, children, pos)
		m.cache[key] = out
		return out
	}

	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	reachable := make([]int, 0, 4)
	reachSeen := make(map[int]struct{})

	// Zero repetitions: valid iff MinOccurs is 0, or the body can make a
	// zero-length pass (so all required reps consume nothing).
	if minReps == 0 || slices.Contains(vc.btBodyReach(ctx, m, mg, children, pos), pos) {
		reachable = addUnique(reachable, reachSeen, pos)
	}

	frontier := []int{pos}
	reps := 0
	for maxReps == Unbounded || reps < maxReps {
		if m.capped {
			return nil
		}
		next := make([]int, 0, 4)
		nextSeen := make(map[int]struct{})
		for _, s := range frontier {
			for _, e := range vc.btBodyReach(ctx, m, mg, children, s) {
				if e == s {
					// Zero-length pass: does not advance, would loop forever. It is
					// accounted for via the emptiable-padding check below.
					continue
				}
				next = addUnique(next, nextSeen, e)
			}
		}
		if len(next) == 0 {
			break
		}
		reps++
		frontier = next
		for _, e := range frontier {
			// e is reachable after `reps` non-zero passes. It satisfies the
			// occurrence lower bound if reps already meets MinOccurs, or if the body
			// can pad the remaining reps with zero-length passes from e.
			if reps >= minReps || slices.Contains(vc.btBodyReach(ctx, m, mg, children, e), e) {
				reachable = addUnique(reachable, reachSeen, e)
			}
		}
		// Each non-zero pass advances by >=1 child, so reps cannot exceed the number
		// of children; the loop is bounded even for unbounded MaxOccurs.
		if reps > len(children) {
			break
		}
	}

	m.cache[key] = reachable
	return reachable
}

// btBodyReach returns end positions reachable by ONE pass through mg's body
// (compositor content) from pos, ignoring mg's own occurrence. A sequence
// composes its particles' reach sets; a choice unions its branches'. Because the
// model is wildcard-free (contentModelAccepts gate), no element-over-wildcard
// precedence can arise, so a plain union faithfully models the choice language.
func (vc *validationContext) btBodyReach(ctx context.Context, m *btMemo, mg *ModelGroup, children []childElem, pos int) []int {
	switch mg.Compositor {
	case CompositorSequence:
		cur := []int{pos}
		for _, part := range mg.Particles {
			next := make([]int, 0, len(cur))
			seen := make(map[int]struct{})
			for _, s := range cur {
				for _, e := range vc.btReachParticle(ctx, m, part, children, s) {
					next = addUnique(next, seen, e)
				}
			}
			cur = next
			if len(cur) == 0 {
				break
			}
		}
		return cur
	case CompositorChoice:
		out := make([]int, 0, len(mg.Particles))
		seen := make(map[int]struct{})
		for _, part := range mg.Particles {
			for _, e := range vc.btReachParticle(ctx, m, part, children, pos) {
				out = addUnique(out, seen, e)
			}
		}
		return out
	}
	return nil
}

// btReachAll returns the deterministic greedy end position(s) for an xs:all group
// at pos (plus pos itself when the group is optional / emptiable).
func (vc *validationContext) btReachAll(ctx context.Context, mg *ModelGroup, children []childElem, pos int) []int {
	consumed, err := vc.tryMatchAll(ctx, mg, children, pos)
	if err != nil {
		return nil
	}
	if consumed == 0 {
		return []int{pos}
	}
	// The group's own optionality allows a zero-length alternative too.
	if mg.MinOccurs == 0 {
		return []int{pos, pos + consumed}
	}
	return []int{pos + consumed}
}

// validateContentModelChildren validates every child's content when
// contentModelAccepts has confirmed the (wildcard-free) children form a valid
// structural match. Because the content model is UPA-deterministic, each child's
// element name selects a unique element-declaration leaf, independent of the
// occurrence partition; so each child is validated against the same declaration
// it would receive in any accepting partition. This visits each child exactly
// once, reusing matchElementParticle so all content, type, xsi:type, assertion,
// and IDC-annotation side effects match the greedy path.
func (vc *validationContext) validateContentModelChildren(ctx context.Context, parent *helium.Element, children []childElem, elemLeaves []*ElementDecl) error {
	var contentErr error
	for i := range children {
		child := children[i]
		ed := elemLeafForChild(child, elemLeaves, vc.schema)
		if ed == nil {
			// contentModelAccepts guaranteed placement, so this is unreachable; report
			// defensively rather than silently accept.
			vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, "This element is not expected.")
			contentErr = fmt.Errorf("unexpected element")
			continue
		}
		one := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: ed}
		if _, err := vc.matchElementParticle(ctx, parent, one, ed, children[i:i+1], 0, false); err != nil {
			contentErr = err
		}
	}
	return contentErr
}

// collectElementLeaves gathers every element leaf declaration in the model group
// tree, in document order. The model is wildcard-free, so there are no wildcard
// leaves to collect.
func collectElementLeaves(mg *ModelGroup, leaves *[]*ElementDecl) {
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			*leaves = append(*leaves, term)
		case *ModelGroup:
			collectElementLeaves(term, leaves)
		}
	}
}

func elemLeafForChild(child childElem, elemLeaves []*ElementDecl, schema *Schema) *ElementDecl {
	for _, ed := range elemLeaves {
		if elemMatchesDeclOrSubst(child, ed, schema) {
			return ed
		}
	}
	return nil
}
