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
// "not proven", keeping the greedy verdict). It uses the SAME per-child matching
// predicates as the greedy matcher (elemMatchesDeclOrSubst, wildcardAllowsExpandedName),
// so it models exactly the greedy content-model language and never accepts a
// non-member. `xs:all` groups are matched greedily (deterministic per-member
// counting, no occurrence backtracking), which is conservative: it can only fail
// to prove, never over-accept.
//
// It is used ONLY as a recovery step AFTER the greedy structural match fails to
// fully consume, so the common (greedy-complete) path is unchanged and pays only
// a pure structural pre-scan. When it accepts, validateContentModelChildren
// validates each child's content against its (UPA-unique) declaration.

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
// some occurrence partition. It is pure (no side effects, no diagnostics).
func (vc *validationContext) contentModelAccepts(ctx context.Context, mg *ModelGroup, children []childElem) bool {
	m := &btMemo{cache: make(map[reachKey][]int)}
	ends := vc.btReachGroup(ctx, m, mg, children)
	if m.capped {
		return false
	}
	return slices.Contains(ends, len(children))
}

func addUnique(dst []int, seen map[int]struct{}, e int) []int {
	if _, ok := seen[e]; ok {
		return dst
	}
	seen[e] = struct{}{}
	return append(dst, e)
}

// btReachParticle returns every end position reachable by matching particle p
// starting at pos, for every pos in an aggregate is handled by callers. Here it
// operates on a single start position embedded via the memo key.
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
	case *Wildcard:
		out = vc.btReachWild(p, term, children, pos)
	case *ModelGroup:
		// A group particle's occurrence lives on the ModelGroup term (matchParticle
		// reads term.MinOccurs/MaxOccurs, not p's), so delegate to the group reach.
		out = vc.btReachGroupAt(ctx, m, term, children, pos)
	}
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

// btReachWild mirrors btReachElem for a wildcard particle.
func (vc *validationContext) btReachWild(p *Particle, wc *Wildcard, children []childElem, pos int) []int {
	maxc := 0
	for pos+maxc < len(children) {
		ch := children[pos+maxc]
		if !wildcardAllowsExpandedName(wc, ch.name, ch.ns, vc.schema, false) {
			break
		}
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
// (compositor content) from pos, ignoring mg's own occurrence.
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
		// XSD 1.1 element-over-wildcard precedence, COMMIT-NO-FALLBACK (mirrors
		// tryMatchChoice/matchChoice): when ANY branch is an element-first consumer
		// for the current child, the choice MUST consume that child through an
		// element-first branch and may NOT fall back to a wildcard (or other
		// non-element-first) branch — even if the chosen branch then fails to fully
		// match. So only element-first branches contribute reachability for this
		// position; a wildcard branch that would re-admit the child is NOT eligible.
		// This prevents accepting an instance the greedy matcher (correctly) rejects
		// by committing to an element branch that later fails.
		var elemFirst []*Particle
		if vc.version == Version11 && pos < len(children) {
			child := children[pos]
			for _, part := range mg.Particles {
				if particleConsumesViaElement(part, child, vc.schema) {
					elemFirst = append(elemFirst, part)
				}
			}
		}
		branches := mg.Particles
		if len(elemFirst) > 0 {
			branches = elemFirst
		}
		out := make([]int, 0, len(branches))
		seen := make(map[int]struct{})
		for _, part := range branches {
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

// elemLeaf / wildLeaf hold a content-model leaf's already-asserted term, so the
// per-child validation pass avoids repeated type assertions.
type elemLeaf struct {
	ed *ElementDecl
}

type wildLeaf struct {
	wc *Wildcard
}

// validateContentModelChildren validates every child's content when
// contentModelAccepts has confirmed the children form a valid structural match.
// Because the content model is UPA-deterministic, each child's element name
// selects a unique element-declaration leaf (or, failing that, a wildcard leaf),
// independent of the occurrence partition; so each child is validated against the
// same declaration it would receive in any accepting partition. This visits each
// child exactly once, reusing the ordinary per-child validators so all content,
// type, xsi:type, assertion, and IDC-annotation side effects match the greedy
// path.
func (vc *validationContext) validateContentModelChildren(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem) error {
	var elemLeaves []elemLeaf
	var wildLeaves []wildLeaf
	collectContentLeaves(mg, &elemLeaves, &wildLeaves)

	var contentErr error
	for i := range children {
		child := children[i]
		single := children[i : i+1]
		if ed := elemLeafForChild(child, elemLeaves, vc.schema); ed != nil {
			one := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: ed}
			if _, err := vc.matchElementParticle(ctx, parent, one, ed, single, 0, len(wildLeaves) > 0); err != nil {
				contentErr = err
			}
			continue
		}
		if wc := wildLeafForChild(child, wildLeaves, vc.schema); wc != nil {
			one := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: wc}
			if _, err := vc.matchWildcardParticle(ctx, parent, one, wc, single, 0, mg); err != nil {
				contentErr = err
			}
			continue
		}
		// contentModelAccepts guaranteed placement, so this is unreachable; report
		// defensively rather than silently accept.
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, "This element is not expected.")
		contentErr = fmt.Errorf("unexpected element")
	}
	return contentErr
}

// collectContentLeaves gathers every element and wildcard leaf particle in the
// model group tree, in document order.
func collectContentLeaves(mg *ModelGroup, elemLeaves *[]elemLeaf, wildLeaves *[]wildLeaf) {
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			*elemLeaves = append(*elemLeaves, elemLeaf{ed: term})
		case *Wildcard:
			*wildLeaves = append(*wildLeaves, wildLeaf{wc: term})
		case *ModelGroup:
			collectContentLeaves(term, elemLeaves, wildLeaves)
		}
	}
}

func elemLeafForChild(child childElem, elemLeaves []elemLeaf, schema *Schema) *ElementDecl {
	for _, leaf := range elemLeaves {
		if elemMatchesDeclOrSubst(child, leaf.ed, schema) {
			return leaf.ed
		}
	}
	return nil
}

func wildLeafForChild(child childElem, wildLeaves []wildLeaf, schema *Schema) *Wildcard {
	for _, leaf := range wildLeaves {
		if wildcardAllowsExpandedName(leaf.wc, child.name, child.ns, schema, false) {
			return leaf.wc
		}
	}
	return nil
}
