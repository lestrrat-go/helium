package xsd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// matchSequence matches children[pos:] against a sequence model group.
// Returns (consumed, error). Does NOT check for leftover children.
//
// The greedy matching approach assumes UPA-compliant (deterministic) content
// models, which is enforced at compile time by checkUPA in check_upa.go.
//
// XSD 1.1 element-over-wildcard precedence: the choice case
// (matchChoice/tryMatchChoice) gives non-wildcard particles precedence over a
// wildcard, so a wildcard declared before an element cannot steal that element's
// child. The sequence case is handled here too via sequenceElementReservedLimit
// (Version11 only): before matching each particle, children a LATER element
// particle in the same sequence is responsible for are reserved, so a leading
// minOccurs=0 wildcard cannot greedily consume them. It is bounded lookahead (no
// backtracking) and a no-op for all-element sequences and for XSD 1.0.
func (vc *validationContext) matchSequence(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, pos int, edcScope *ModelGroup) (int, error) {
	startPos := pos

	tryOnce := func(p int) (int, error) {
		return vc.tryMatchSequenceOnce(ctx, mg, children, p)
	}

	hasWildcard := sequenceHasWildcard(mg)

	matchOnce := func(p int) (int, error) {
		cur := p
		var contentErr error
		for i, particle := range mg.Particles {
			limit := len(children)
			if vc.version == Version11 {
				limit = sequenceElementReservedLimit(mg.Particles, i, children, cur, vc.schema)
			}
			consumed, e := vc.matchParticle(ctx, parent, particle, children[:limit], cur, hasWildcard, edcScope)
			cur += consumed
			if e != nil {
				if consumed == 0 {
					// Structural error — stop processing subsequent particles.
					return cur - p, e
				}
				// Content error — continue but track error.
				contentErr = e
			}
		}
		return cur - p, contentErr
	}

	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	var contentErr error
	reps := 0
	for maxReps == Unbounded || reps < maxReps {
		// First try without side effects.
		tryCons, tryErr := tryOnce(pos)
		if tryErr != nil {
			if reps < minReps {
				// Must succeed — run with error reporting.
				consumed, e := matchOnce(pos)
				return pos - startPos + consumed, e
			}
			break
		}
		if tryCons == 0 {
			reps++
			if reps >= minReps {
				break
			}
			continue
		}
		// Actually run with error reporting (for nested validation).
		consumed, e := matchOnce(pos)
		if e != nil {
			contentErr = e
		}
		pos += consumed
		reps++
	}

	return pos - startPos, contentErr
}

func (vc *validationContext) tryMatchSequenceOnce(ctx context.Context, mg *ModelGroup, children []childElem, pos int) (int, error) {
	cur := pos
	for i, p := range mg.Particles {
		limit := len(children)
		if vc.version == Version11 {
			limit = sequenceElementReservedLimit(mg.Particles, i, children, cur, vc.schema)
		}
		consumed, err := vc.tryMatchParticle(ctx, p, children[:limit], cur)
		if err != nil {
			return 0, err
		}
		cur += consumed
	}
	return cur - pos, nil
}

// matchChoice matches children[pos:] against a choice model group.
// Returns (consumed, error). Does NOT check for leftover children.
func (vc *validationContext) matchChoice(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, pos int, edcScope *ModelGroup) (int, error) {
	startPos := pos

	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	var contentErr error

	matchAt := func(particle *Particle, p int) (int, bool) {
		consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
		if err != nil || consumed == 0 {
			return 0, false
		}
		// Now validate matched content with error reporting.
		actualConsumed, actualErr := vc.matchParticle(ctx, parent, particle, children, p, false, edcScope)
		if actualErr != nil {
			contentErr = actualErr
		}
		return actualConsumed, true
	}

	matchOnce := func(p int) (int, bool) {
		// First find a structurally matching particle that consumes a child.
		// In XSD 1.1, a branch that consumes the current child via an element
		// leaf takes precedence over one that would consume it via a wildcard
		// (element-over-wildcard precedence) regardless of declaration order or
		// nesting: a wildcard declared before an element — directly or wrapped in
		// a model group — must not steal a child the element declaration is
		// responsible for validating. XSD 1.0 uses pure declaration order.
		if vc.version == Version11 && p < len(children) {
			child := children[p]
			// Pass 1: branches that match the current child via an element leaf.
			// Element-over-wildcard precedence COMMITS: if any branch consumes the
			// current child via an element leaf as its first consuming term, the
			// choice MUST use an element-first branch for this child and MUST NOT
			// fall back to a wildcard branch — even if the chosen element branch
			// then fails structurally or by content. Otherwise a skip wildcard
			// would false-accept a child a typed element is responsible for.
			var elemFirst *Particle
			for _, particle := range mg.Particles {
				if !particleConsumesViaElement(particle, child, vc.schema) {
					continue
				}
				if consumed, ok := matchAt(particle, p); ok {
					return consumed, true
				}
				if elemFirst == nil {
					elemFirst = particle
				}
			}
			if elemFirst != nil {
				// No element-first branch matched fully. Surface the first
				// element-first branch's real failure (with error reporting)
				// instead of falling back to a wildcard branch.
				consumed, err := vc.matchParticle(ctx, parent, elemFirst, children, p, false, edcScope)
				if err != nil {
					contentErr = err
				}
				if consumed > 0 {
					return consumed, true
				}
				if contentErr == nil {
					contentErr = fmt.Errorf("element content does not match")
				}
				return 0, false
			}
			// Pass 2: no element-first branch for this child, so a wildcard branch
			// may consume it.
			for _, particle := range mg.Particles {
				if consumed, ok := matchAt(particle, p); ok {
					return consumed, true
				}
			}
		} else {
			for _, particle := range mg.Particles {
				if consumed, ok := matchAt(particle, p); ok {
					return consumed, true
				}
			}
		}
		// Try zero-length matches.
		for _, particle := range mg.Particles {
			consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
			if err == nil && consumed == 0 {
				return consumed, true
			}
		}
		return 0, false
	}

	reps := 0
	for maxReps == Unbounded || reps < maxReps {
		consumed, ok := matchOnce(pos)
		if !ok {
			break
		}
		reps++
		pos += consumed
		if consumed == 0 {
			// Zero-length match (e.g., optional element). If we still need more
			// reps to meet minReps, count them all at once since they'll all be
			// zero-length too.
			if reps < minReps {
				reps = minReps
			}
			break
		}
	}

	if reps < minReps {
		names := particleNames(mg.Particles, vc.schema)
		msg := formatExpected("Missing child element(s).", names)
		vc.reportValidityError(ctx, vc.filename, parent.Line(), elemDisplayName(parent), msg)
		return pos - startPos, fmt.Errorf("missing")
	}

	return pos - startPos, contentErr
}

// allMember is a flattened member of an xs:all model group used by matchAll. In
// XSD 1.1 an xs:all may carry element members with maxOccurs>1, element
// wildcards, and a nested all-group reference (flattened in here); each member
// is matched order-independently with per-member occurrence counting. In XSD 1.0
// the flat list is the particle list 1:1 (no wildcards, no nested all), each
// with max 1, so counting reduces to the previous boolean "seen" semantics.
type allMember struct {
	min, max int
	ed       *ElementDecl // non-nil for an element member
	wc       *Wildcard    // non-nil for a wildcard member
}

// flattenAllMembers builds the flat member list for an xs:all group. In XSD 1.1
// a wildcard particle becomes a wildcard member and a nested all-group (reached
// via an xs:group ref that resolved to an all group, occurrence 1/1) is
// flattened into the parent's members. In XSD 1.0 only element particles are
// recognized, matching the pre-feature behavior.
func flattenAllMembers(mg *ModelGroup, is11 bool) []allMember {
	var members []allMember
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			members = append(members, allMember{min: p.MinOccurs, max: p.MaxOccurs, ed: term})
		case *Wildcard:
			if is11 {
				members = append(members, allMember{min: p.MinOccurs, max: p.MaxOccurs, wc: term})
			}
		case *ModelGroup:
			if is11 && term.Compositor == CompositorAll && p.MinOccurs == 1 && p.MaxOccurs == 1 {
				members = append(members, flattenAllMembers(term, is11)...)
			}
		}
	}
	return members
}

// matchAll matches children[pos:] against an all model group.
// Returns (consumed, error). Does NOT check for leftover children. XSD 1.0 uses
// the legacy boolean matcher (byte-identical to before the xs:all-relaxation
// feature); XSD 1.1 uses the per-member occurrence-counting matcher.
func (vc *validationContext) matchAll(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, pos int, edcScope *ModelGroup) (int, error) {
	if vc.version == Version11 {
		return vc.matchAll11(ctx, parent, mg, children, pos, edcScope, false)
	}
	return vc.matchAll10(ctx, parent, mg, children, pos)
}

// matchAll10 is the XSD 1.0 xs:all matcher: each element particle may appear at
// most once (boolean seen[]), order-independent. It is a faithful copy of the
// pre-relaxation matcher so 1.0 behavior and diagnostics stay byte-identical. A
// wildcard particle (which the parser tolerates inside an xs:all even in 1.0) is
// NEVER matched here and a wildcard particle with minOccurs>0 is reported missing
// — exactly as the original did.
func (vc *validationContext) matchAll10(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, pos int) (int, error) {
	seen := make([]bool, len(mg.Particles))
	nameToIdx := make(map[QName]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name] = i
			// TRANSITIVE, block-filtered substitution closure (the pre-feature XSD
			// 1.0 matcher used this; a direct substGroups lookup misses h<-m1<-m2).
			for _, member := range substitutableMembersFor(ed, vc.schema) {
				nameToIdx[member.Name] = i
			}
		}
	}

	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[QName{Local: child.name, NS: child.ns}]
		if ok && !seen[idx] {
			if edecl, isElem := mg.Particles[idx].Term.(*ElementDecl); isElem {
				vc.recordElemDecl(child.elem, resolveSubstDecl(child, edecl, vc.schema))
			}
			seen[idx] = true
			consumed++
			continue
		}
		expected := unseenParticleNames(mg.Particles, seen, vc.schema)
		msg := "This element is not expected."
		if len(expected) > 0 {
			msg = formatExpected("This element is not expected.", expected)
		}
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, msg)
		return consumed, fmt.Errorf("unexpected element")
	}

	// Respect <all> group's minOccurs: if 0 and group is empty, skip required checks.
	if mg.MinOccurs == 0 && consumed == 0 {
		return 0, nil
	}

	hasRequired := false
	for i, p := range mg.Particles {
		// A wildcard particle is never matched in 1.0, so one with minOccurs>0 is
		// always missing.
		if _, isWC := p.Term.(*Wildcard); isWC {
			if p.MinOccurs > 0 {
				hasRequired = true
				break
			}
			continue
		}
		if !seen[i] && p.MinOccurs > 0 {
			hasRequired = true
			break
		}
	}
	if hasRequired {
		unseen := unseenParticleNames(mg.Particles, seen, vc.schema)
		msg := formatExpected("Missing child element(s).", unseen)
		vc.reportValidityError(ctx, vc.filename, parent.Line(), elemDisplayName(parent), msg)
		return consumed, fmt.Errorf("missing")
	}

	var contentErr error
	for i := range consumed {
		child := children[pos+i]
		idx, ok := nameToIdx[QName{Local: child.name, NS: child.ns}]
		if !ok {
			continue
		}
		edecl, isElem := mg.Particles[idx].Term.(*ElementDecl)
		if !isElem {
			continue
		}
		if err := vc.validateAllMatchedChild(ctx, child, edecl); err != nil {
			contentErr = err
		}
	}
	if contentErr != nil {
		return consumed, contentErr
	}
	return consumed, nil
}

// matchAll11 is the XSD 1.1 xs:all matcher: order-independent per-member
// occurrence counting over a flattened member list (element members with
// minOccurs/maxOccurs ranges, element wildcards, and flattened nested all-group
// members). A child is matched to an element member only if it is ADMISSIBLY
// substitutable for it (honoring block="substitution"/derivation-block via
// allMemberForChild); a declared element with remaining budget wins over a
// wildcard (weak-wildcard precedence).
//
// When lenientLeftover is true (the XSD 1.1 open-content suffix path), a child
// that matches no member is NOT reported as unexpected: matching simply STOPS and
// the consumed count is returned, so the caller can treat the remaining children
// as trailing open content. Required-member (minOccurs) and matched-child content
// checks still run.
func (vc *validationContext) matchAll11(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, pos int, edcScope *ModelGroup, lenientLeftover bool) (int, error) {
	members := flattenAllMembers(mg, true)
	counts := make([]int, len(members))

	// wcClaimed marks child positions (absolute index) consumed by a wildcard,
	// so the element-content validation pass below skips them.
	wcClaimed := make(map[int]bool)

	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		// A declared element wins over a wildcard (weak-wildcard precedence) while
		// it still has occurrence budget. Once exhausted, a further admissibly
		// matching child may instead be claimed by a wildcard member.
		idx := allMemberForChild(members, child, vc.schema)
		if idx >= 0 && (members[idx].max == Unbounded || counts[idx] < members[idx].max) {
			// Record the (possibly LOCAL) host declaration AS SOON AS this child is
			// matched — BEFORE any early return — so pass-2 IDC evaluation does not
			// fall back to a same-named GLOBAL declaration.
			vc.recordElemDecl(child.elem, resolveSubstDecl(child, members[idx].ed, vc.schema))
			counts[idx]++
			consumed++
			continue
		}
		if widx := vc.allWildcardMember(members, counts, child); widx >= 0 {
			wc := members[widx].wc
			if err := vc.validateWildcardChild(ctx, wc, child, edcScope); err != nil {
				return consumed, err
			}
			counts[widx]++
			wcClaimed[pos+consumed] = true
			consumed++
			continue
		}
		if lenientLeftover {
			// Open-content suffix path: stop at the first non-member child; the caller
			// validates the remaining children as trailing open content.
			break
		}
		expected := availableMemberNames(members, counts, vc.schema)
		msg := "This element is not expected."
		if len(expected) > 0 {
			msg = formatExpected("This element is not expected.", expected)
		}
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, msg)
		return consumed, fmt.Errorf("unexpected element")
	}

	// Respect <all> group's minOccurs: if 0 and group is empty, skip required checks.
	if mg.MinOccurs == 0 && consumed == 0 {
		return 0, nil
	}

	// Check for required missing members: each member's match count must reach its
	// minOccurs (which may be >1 in XSD 1.1).
	for i, m := range members {
		if counts[i] >= m.min {
			continue
		}
		unseen := underMinMemberNames(members, counts, vc.schema)
		msg := formatExpected("Missing child element(s).", unseen)
		vc.reportValidityError(ctx, vc.filename, parent.Line(), elemDisplayName(parent), msg)
		return consumed, fmt.Errorf("missing")
	}

	// Validate content of each matched child element.
	var contentErr error
	for i := range consumed {
		child := children[pos+i]
		if wcClaimed[pos+i] {
			continue // already validated inline by the wildcard path
		}
		idx := allMemberForChild(members, child, vc.schema)
		if idx < 0 || members[idx].ed == nil {
			continue
		}
		if err := vc.validateAllMatchedChild(ctx, child, members[idx].ed); err != nil {
			contentErr = err
		}
	}
	if contentErr != nil {
		return consumed, contentErr
	}

	return consumed, nil
}

// validateAllMatchedChild validates one element child matched to an element
// member of an xs:all (resolving substitution, type alternatives, xsi:type, the
// derivation-block check, xsi:nil, and the element content). Shared by the XSD
// 1.0 and 1.1 matchers so the per-child content validation is identical.
func (vc *validationContext) validateAllMatchedChild(ctx context.Context, child childElem, edecl *ElementDecl) error {
	actualDecl := resolveSubstDecl(child, edecl, vc.schema)
	declType := effectiveDeclType(actualDecl, vc.schema)
	declType = vc.applyTypeAlternatives(ctx, child.elem, actualDecl, declType)
	td, xsiErr := vc.resolveXsiType(ctx, child.elem, declType, vc.hasTypeTable(actualDecl))
	if xsiErr != nil {
		return xsiErr
	}
	// A blocked xsi:type derivation is a validity error (cvc-elt.4.3), enforced
	// here just like at matchElementParticle/root. The blocked set is the UNION of
	// the element declaration's block and the declared type's {prohibited
	// substitutions}.
	if td != declType && declType != nil && typeDerivationBlocked(td, declType, actualDecl.Block) {
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem),
			"The xsi:type definition is blocked by the element declaration.")
		return fmt.Errorf("blocked xsi:type")
	}
	if td != nil && td.Abstract {
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem), msgAbstractType)
		return fmt.Errorf("abstract type")
	}
	vc.annotateElement(ctx, child.elem, td, true)
	if td == nil {
		return nil
	}
	nilled, nilErr := vc.checkXsiNil(ctx, child.elem, actualDecl)
	if nilErr != nil {
		return nilErr
	}
	if nilled {
		return vc.validateNilledElement(ctx, child.elem, actualDecl, td)
	}
	return vc.validateElementContent(ctx, child.elem, actualDecl, td)
}

// allMemberForChild returns the index of the element member of an xs:all whose
// declaration the child is ADMISSIBLY substitutable for (a direct name match to a
// non-abstract element, or a substitution-group member not blocked by
// block="substitution"/a blocked derivation step) — using the same predicate as
// content-model element matching (elemMatchesDeclOrSubst). Returns -1 if none.
func allMemberForChild(members []allMember, child childElem, schema *Schema) int {
	for i, m := range members {
		if m.ed == nil {
			continue
		}
		if elemMatchesDeclOrSubst(child, m.ed, schema) {
			return i
		}
	}
	return -1
}

// allWildcardMember returns the index of a wildcard member in an xs:all group's
// flattened member list that admits child and still has occurrence budget
// (maxOccurs not reached), or -1 if none. Wildcards are tried in declaration
// order.
func (vc *validationContext) allWildcardMember(members []allMember, counts []int, child childElem) int {
	for i, m := range members {
		if m.wc == nil {
			continue
		}
		if m.max != Unbounded && counts[i] >= m.max {
			continue
		}
		if wildcardAllowsExpandedName(m.wc, child.name, child.ns, vc.schema, false) {
			return i
		}
	}
	return -1
}

// availableMemberNames lists the expected names of XSD 1.1 xs:all members that
// can still accept another child (remaining occurrence budget: unbounded, or
// count below maxOccurs), used for the unexpected-element "Expected is one of"
// hint. A member already at its maxOccurs is omitted (it cannot accept more).
func availableMemberNames(members []allMember, counts []int, schema *Schema) []string {
	var names []string
	for i, m := range members {
		if m.max != Unbounded && counts[i] >= m.max {
			continue
		}
		switch {
		case m.ed != nil:
			names = append(names, elementExpectedNamesWithSubst(m.ed, schema)...)
		case m.wc != nil:
			names = append(names, wildcardExpected(m.wc))
		}
	}
	return names
}

// underMinMemberNames lists the expected names of XSD 1.1 xs:all members that
// have not yet reached their minOccurs, used for the missing-element hint — so a
// member PRESENT but UNDER minOccurs (e.g. `c minOccurs="2"` with one `<c/>`) is
// still reported, and the expected list is never empty when a missing-required
// error fires.
func underMinMemberNames(members []allMember, counts []int, schema *Schema) []string {
	var names []string
	for i, m := range members {
		if counts[i] >= m.min {
			continue
		}
		switch {
		case m.ed != nil:
			names = append(names, elementExpectedNamesWithSubst(m.ed, schema)...)
		case m.wc != nil:
			names = append(names, wildcardExpected(m.wc))
		}
	}
	return names
}

// unseenParticleNames lists the expected names of xs:all PARTICLES not yet seen
// (XSD 1.0 matcher), used for the unexpected/missing diagnostics so the 1.0
// wording stays byte-identical to before the relaxation feature.
func unseenParticleNames(particles []*Particle, seen []bool, schema *Schema) []string {
	var names []string
	for i, p := range particles {
		if seen[i] {
			continue
		}
		switch term := p.Term.(type) {
		case *ElementDecl:
			names = append(names, elementExpectedNamesWithSubst(term, schema)...)
		case *Wildcard:
			names = append(names, wildcardExpected(term))
		}
	}
	return names
}

// validateContentModelTop validates children against a model group, checking
// that ALL children are consumed. This is the top-level entry point.
func (vc *validationContext) validateContentModelTop(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem) error {
	consumed, err := vc.matchContentModel(ctx, parent, mg, children)
	if err != nil {
		return err
	}

	// Check for unconsumed children.
	if consumed < len(children) {
		ce := children[consumed]
		vc.reportValidityError(ctx, vc.filename, ce.elem.Line(), ce.displayName, "This element is not expected.")
		return fmt.Errorf("unexpected element")
	}

	return nil
}

// matchContentModel matches children against the top-level model group from
// position 0 and returns how many were consumed (it does NOT report leftover
// children — callers that require full consumption do so themselves). Used by
// validateContentModelTop and by the XSD 1.1 open-content suffix path.
func (vc *validationContext) matchContentModel(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem) (int, error) {
	return vc.matchContentModelScoped(ctx, parent, mg, children, mg)
}

func (vc *validationContext) matchContentModelScoped(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem, edcScope *ModelGroup) (int, error) {
	switch mg.Compositor {
	case CompositorSequence:
		return vc.matchSequence(ctx, parent, mg, children, 0, edcScope)
	case CompositorChoice:
		return vc.matchChoice(ctx, parent, mg, children, 0, edcScope)
	case CompositorAll:
		return vc.matchAll(ctx, parent, mg, children, 0, edcScope)
	}
	return 0, nil
}

// matchParticle matches a particle against children[pos:], returning how many
// children were consumed. On failure, writes an error and returns an error.
func (vc *validationContext) matchParticle(ctx context.Context, parent *helium.Element, p *Particle, children []childElem, pos int, seqHasWildcard bool, edcScope *ModelGroup) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return vc.matchElementParticle(ctx, parent, p, term, children, pos, seqHasWildcard)
	case *ModelGroup:
		switch term.Compositor {
		case CompositorSequence:
			return vc.matchSequence(ctx, parent, term, children, pos, edcScope)
		case CompositorChoice:
			return vc.matchChoice(ctx, parent, term, children, pos, edcScope)
		case CompositorAll:
			return vc.matchAll(ctx, parent, term, children, pos, edcScope)
		}
	case *Wildcard:
		return vc.matchWildcardParticle(ctx, parent, p, term, children, pos, edcScope)
	}
	return 0, nil
}

// matchElementParticle matches an element particle.
func (vc *validationContext) matchElementParticle(ctx context.Context, parent *helium.Element, p *Particle, edecl *ElementDecl, children []childElem, pos int, seqHasWildcard bool) (int, error) {
	count := 0
	for pos+count < len(children) && elemMatchesDeclOrSubst(children[pos+count], edecl, vc.schema) {
		// Record each matched child's (possibly LOCAL) host declaration AS SOON
		// AS it is matched, BEFORE the count<MinOccurs early return below. A
		// partially-satisfied occurrence (e.g. minOccurs=2 with one child
		// present) still matched that child to this local declaration; without
		// recording it, pass-2 identity-constraint evaluation would fall back to
		// a same-named GLOBAL declaration and apply its IDCs spuriously.
		child := children[pos+count]
		vc.recordElemDecl(child.elem, resolveSubstDecl(child, edecl, vc.schema))
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}

	if count < p.MinOccurs {
		expectedNames := elementExpectedNamesWithSubst(edecl, vc.schema)
		if pos+count < len(children) {
			// There IS a child but it doesn't match — "This element is not expected."
			child := children[pos+count]
			msg := formatExpected("This element is not expected.", expectedNames)
			vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, msg)
		} else {
			// No more children at all — "Missing child element(s)."
			// When the sequence contains wildcards, suppress "Expected is" since the
			// expected set is ambiguous (wildcards could have consumed the elements).
			var msg string
			if seqHasWildcard {
				msg = "Missing child element(s)."
			} else {
				msg = formatExpected("Missing child element(s).", expectedNames)
			}
			vc.reportValidityError(ctx, vc.filename, parent.Line(), elemDisplayName(parent), msg)
		}
		return count, fmt.Errorf("missing")
	}

	// Validate each matched child element's own content model.
	// Continue after value/content errors so all errors are reported.
	// For substitution group members, use the member's declaration (type + default/fixed).
	// xsi:type overrides the declared type for polymorphism.
	var contentErr error
	for i := range count {
		child := children[pos+i]
		actualDecl := resolveSubstDecl(child, edecl, vc.schema)
		// The host declaration was already recorded during the initial match scan
		// above (before any early return). Nothing to record here.
		declType := effectiveDeclType(actualDecl, vc.schema)
		declType = vc.applyTypeAlternatives(ctx, child.elem, actualDecl, declType)
		td, xsiErr := vc.resolveXsiType(ctx, child.elem, declType, vc.hasTypeTable(actualDecl))
		if xsiErr != nil {
			contentErr = xsiErr
			continue
		}
		// Check block flags against xsi:type derivation (cvc-elt.4.3): the union of
		// the element declaration's block and the declared type's block.
		if td != declType && declType != nil && typeDerivationBlocked(td, declType, actualDecl.Block) {
			msg := "The xsi:type definition is blocked by the element declaration."
			vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem), msg)
			contentErr = fmt.Errorf("blocked xsi:type")
			continue
		}
		if td != nil && td.Abstract {
			msg := msgAbstractType
			vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem), msg)
			contentErr = fmt.Errorf("abstract type")
			continue
		}
		// Annotate child element with its type for pass-2 identity-constraint evaluation.
		vc.annotateElement(ctx, child.elem, td, true)
		if td != nil {
			nilled, nilErr := vc.checkXsiNil(ctx, child.elem, actualDecl)
			if nilErr != nil {
				contentErr = nilErr
			} else if nilled {
				if err := vc.validateNilledElement(ctx, child.elem, actualDecl, td); err != nil {
					contentErr = err
				}
			} else if err := vc.validateElementContent(ctx, child.elem, actualDecl, td); err != nil {
				contentErr = err
			}
		}
	}

	return count, contentErr
}

// resolveSubstDecl returns the actual element declaration for a child element.
// If the child matches the declaration directly, returns the original declaration.
// If the child is a substitution group member, returns the member's declaration.
func resolveSubstDecl(child childElem, edecl *ElementDecl, schema *Schema) *ElementDecl {
	if matchesDeclDirect(child, edecl) {
		return edecl
	}
	if schema != nil {
		// TRANSITIVE closure so a child matched via a multi-level substitution chain
		// (h<-m1<-m2) resolves to its actual member declaration, not the head.
		for _, member := range substitutableMembersFor(edecl, schema) {
			if matchesDeclDirect(child, member) {
				return member
			}
		}
	}
	return edecl
}

// effectiveDeclType returns the effective type definition for a declaration.
// It is actualDecl.Type when present; otherwise, for a no-type
// substitution-group member, it is the type inherited from the substitution
// head (walking the substitutionGroup chain until a typed head is found).
// Returns nil when no type can be resolved. The returned type is used to drive
// xsi:nil lexical validation and nilled-empty enforcement for no-type members;
// the member declaration itself is still used elsewhere so its own nillable
// flag is honored independently of the head.
func effectiveDeclType(decl *ElementDecl, schema *Schema) *TypeDef {
	if decl == nil {
		return nil
	}
	if decl.Type != nil {
		return decl.Type
	}
	if schema == nil {
		return nil
	}
	return inheritedTypeFromFirstSubstitutionHead(decl, func(qn QName) (*ElementDecl, bool) {
		return schema.LookupElement(qn.Local, qn.NS)
	})
}

func substitutableMembersFor(edecl *ElementDecl, schema *Schema) []*ElementDecl {
	if edecl == nil || schema == nil || edecl.Block&BlockSubstitution != 0 {
		return nil
	}
	// A substitution group is a property of the GLOBAL head element declaration
	// (or a ref to it), not of a LOCAL element particle that merely shares the
	// head's QName. Expand only when this declaration IS the registered global
	// head, or is a ref (which resolves to the global head); a distinct local
	// particle admits no substitution members even if it is named like a head.
	if !edecl.IsRef && schema.elements[edecl.Name] != edecl {
		return nil
	}
	type queuedMember struct {
		member *ElementDecl
		head   *ElementDecl
	}
	headType := effectiveDeclType(edecl, schema)
	queue := make([]queuedMember, 0, len(schema.substGroups[edecl.Name]))
	for _, member := range schema.substGroups[edecl.Name] {
		queue = append(queue, queuedMember{member: member, head: edecl})
	}
	seen := map[QName]struct{}{edecl.Name: {}}
	members := make([]*ElementDecl, 0, len(queue))
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		member := item.member
		if member == nil {
			continue
		}
		if _, ok := seen[member.Name]; ok {
			continue
		}
		memberType := effectiveDeclType(member, schema)
		curHeadType := effectiveDeclType(item.head, schema)
		// The head's EFFECTIVE {disallowed substitutions} unions the head element's
		// block with its declared TYPE's {prohibited substitutions} (Substitution
		// Group OK / cvc-elt.4.3), so a member reached by a derivation method the
		// intermediate OR original head's TYPE blocks is not admitted; the
		// substitution-specific walk also honors any INTERMEDIATE type's block on the
		// member's derivation chain.
		if substTypeDerivationBlocked(memberType, curHeadType, item.head.Block) {
			continue
		}
		if substTypeDerivationBlocked(memberType, headType, edecl.Block) {
			continue
		}
		seen[member.Name] = struct{}{}
		members = append(members, member)
		if member.Block&BlockSubstitution != 0 {
			continue
		}
		for _, child := range schema.substGroups[member.Name] {
			queue = append(queue, queuedMember{member: child, head: member})
		}
	}
	return members
}

// tryMatchParticle is like matchParticle but does not write errors.
func (vc *validationContext) tryMatchParticle(ctx context.Context, p *Particle, children []childElem, pos int) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return vc.tryMatchElementParticle(ctx, p, term, children, pos)
	case *ModelGroup:
		return vc.tryMatchModelGroup(ctx, term, children, pos)
	case *Wildcard:
		return vc.tryMatchWildcardParticle(ctx, p, term, children, pos)
	}
	return 0, nil
}

func (vc *validationContext) tryMatchElementParticle(_ context.Context, p *Particle, edecl *ElementDecl, children []childElem, pos int) (int, error) {
	count := 0
	for pos+count < len(children) && elemMatchesDeclOrSubst(children[pos+count], edecl, vc.schema) {
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}
	if count < p.MinOccurs {
		return 0, fmt.Errorf("insufficient")
	}
	return count, nil
}

func (vc *validationContext) tryMatchModelGroup(ctx context.Context, mg *ModelGroup, children []childElem, pos int) (int, error) {
	switch mg.Compositor {
	case CompositorSequence:
		return vc.tryMatchSequence(ctx, mg, children, pos)
	case CompositorChoice:
		return vc.tryMatchChoice(ctx, mg, children, pos)
	case CompositorAll:
		return vc.tryMatchAll(ctx, mg, children, pos)
	}
	return 0, fmt.Errorf("unknown compositor")
}

func (vc *validationContext) tryMatchSequence(ctx context.Context, mg *ModelGroup, children []childElem, pos int) (int, error) {
	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	scanOnce := func(p int) (int, error) {
		cur := p
		for i, particle := range mg.Particles {
			limit := len(children)
			if vc.version == Version11 {
				limit = sequenceElementReservedLimit(mg.Particles, i, children, cur, vc.schema)
			}
			consumed, err := vc.tryMatchParticle(ctx, particle, children[:limit], cur)
			if err != nil {
				return 0, err
			}
			cur += consumed
		}
		return cur - p, nil
	}

	cur := pos
	reps := 0
	for maxReps == Unbounded || reps < maxReps {
		consumed, err := scanOnce(cur)
		if err != nil {
			break
		}
		reps++
		cur += consumed
		if consumed == 0 {
			// Zero-length iteration: further iterations would also be
			// zero-length, so count remaining required reps at once.
			if reps < minReps {
				reps = minReps
			}
			break
		}
	}

	if reps < minReps {
		return 0, fmt.Errorf("insufficient sequence repetitions")
	}
	return cur - pos, nil
}

func (vc *validationContext) tryMatchChoice(ctx context.Context, mg *ModelGroup, children []childElem, pos int) (int, error) {
	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	scanOnce := func(p int) (int, bool) {
		// Prefer a branch that consumes at least one child, mirroring
		// matchChoice. An earlier optional branch can match zero-length, but a
		// later branch may be the one that actually consumes the current child;
		// returning the zero-length match first would leave that child stranded.
		//
		// In XSD 1.1, a branch that consumes the current child via an element
		// leaf takes precedence over one that would consume it via a wildcard
		// (element-over-wildcard precedence), regardless of declaration order or
		// nesting, so a wildcard declared before an element — directly or wrapped
		// in a model group — does not steal the element's child. XSD 1.0 uses
		// pure declaration order.
		if vc.version == Version11 && p < len(children) {
			child := children[p]
			// Pass 1: branches that match the current child via an element leaf.
			// Mirror matchChoice: element-over-wildcard precedence COMMITS, so if
			// any branch is element-first for this child the lookahead must reflect
			// that branch's success/failure and MUST NOT fall back to a wildcard
			// branch.
			hasElemFirst := false
			for _, particle := range mg.Particles {
				if !particleConsumesViaElement(particle, child, vc.schema) {
					continue
				}
				hasElemFirst = true
				consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
				if err == nil && consumed > 0 {
					return consumed, true
				}
			}
			if hasElemFirst {
				// An element-first branch is required for this child but none
				// matched; do not fall back to a wildcard branch.
				return 0, false
			}
			// Pass 2: no element-first branch for this child, so a wildcard branch
			// may consume it.
			for _, particle := range mg.Particles {
				consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
				if err == nil && consumed > 0 {
					return consumed, true
				}
			}
		} else {
			for _, particle := range mg.Particles {
				consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
				if err == nil && consumed > 0 {
					return consumed, true
				}
			}
		}
		// Fall back to a zero-length (optional) branch.
		for _, particle := range mg.Particles {
			consumed, err := vc.tryMatchParticle(ctx, particle, children, p)
			if err == nil && consumed == 0 {
				return consumed, true
			}
		}
		return 0, false
	}

	cur := pos
	reps := 0
	for maxReps == Unbounded || reps < maxReps {
		consumed, ok := scanOnce(cur)
		if !ok {
			break
		}
		reps++
		cur += consumed
		if consumed == 0 {
			// Zero-length iteration: further iterations would also be
			// zero-length, so count remaining required reps at once.
			if reps < minReps {
				reps = minReps
			}
			break
		}
	}

	if reps < minReps {
		return 0, fmt.Errorf("insufficient choice repetitions")
	}
	return cur - pos, nil
}

func (vc *validationContext) tryMatchAll(_ context.Context, mg *ModelGroup, children []childElem, pos int) (int, error) {
	if vc.version == Version11 {
		return vc.tryMatchAll11(mg, children, pos)
	}
	return vc.tryMatchAll10(mg, children, pos)
}

// tryMatchAll10 is the lookahead variant of the XSD 1.0 xs:all matcher (each
// element particle at most once; no wildcard matching), byte-identical to the
// pre-relaxation behavior.
func (vc *validationContext) tryMatchAll10(mg *ModelGroup, children []childElem, pos int) (int, error) {
	seen := make([]bool, len(mg.Particles))
	nameToIdx := make(map[QName]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name] = i
			// TRANSITIVE, block-filtered substitution closure (the pre-feature XSD
			// 1.0 matcher used this; a direct substGroups lookup misses h<-m1<-m2).
			for _, member := range substitutableMembersFor(ed, vc.schema) {
				nameToIdx[member.Name] = i
			}
		}
	}
	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[QName{Local: child.name, NS: child.ns}]
		if !ok {
			break
		}
		if seen[idx] {
			return 0, fmt.Errorf("duplicate")
		}
		seen[idx] = true
		consumed++
	}
	for i, p := range mg.Particles {
		if !seen[i] && p.MinOccurs > 0 {
			return 0, fmt.Errorf("missing required")
		}
	}
	return consumed, nil
}

// tryMatchAll11 is the lookahead variant of the XSD 1.1 occurrence-counting
// xs:all matcher (admissible substitution + wildcard members + counting).
func (vc *validationContext) tryMatchAll11(mg *ModelGroup, children []childElem, pos int) (int, error) {
	members := flattenAllMembers(mg, true)
	counts := make([]int, len(members))
	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx := allMemberForChild(members, child, vc.schema)
		if idx >= 0 && (members[idx].max == Unbounded || counts[idx] < members[idx].max) {
			counts[idx]++
			consumed++
			continue
		}
		if widx := vc.allWildcardMember(members, counts, child); widx >= 0 {
			counts[widx]++
			consumed++
			continue
		}
		if idx >= 0 {
			return 0, fmt.Errorf("duplicate")
		}
		break
	}
	for i, m := range members {
		if counts[i] < m.min {
			return 0, fmt.Errorf("missing required")
		}
	}
	return consumed, nil
}

// matchWildcardParticle matches a wildcard particle against children.
func (vc *validationContext) matchWildcardParticle(ctx context.Context, parent *helium.Element, p *Particle, wc *Wildcard, children []childElem, pos int, edcScope *ModelGroup) (int, error) {
	count := 0
	for pos+count < len(children) {
		child := children[pos+count]
		if !wildcardAllowsExpandedName(wc, child.elem.LocalName(), child.elem.URI(), vc.schema, false) {
			break
		}
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}

	if count < p.MinOccurs {
		msg := fmt.Sprintf("This element is not expected. Expected is ( %s ).", wildcardExpected(wc))
		if pos < len(children) {
			vc.reportValidityError(ctx, vc.filename, children[pos].elem.Line(), children[pos].displayName, msg)
		} else {
			vc.reportValidityError(ctx, vc.filename, parent.Line(), elemDisplayName(parent), msg)
		}
		return count, fmt.Errorf("wildcard not matched")
	}

	// Skipped content is NOT schema-assessed, so no validation runs. Still walk
	// the matched subtrees to RECORD actual types for any nested global IDC host:
	// pass-2 IDC field canonicalization needs the descendants' xsi:type actual
	// types, which would otherwise be missed under a skip wrapper.
	if wc.ProcessContents == ProcessSkip {
		for i := range count {
			vc.annotateSkipChildren(ctx, children[pos+i].elem)
		}
		return count, nil
	}

	// Validate matched elements per processContents (lax/strict). The per-child
	// logic — strict/lax assessment, CTA governing-type selection, the xsi:type
	// block check, and the lax/anyType descendant recursion — lives in
	// validateWildcardChild (shared with the idc lax-assessment path).
	var contentErr error
	for i := range count {
		if err := vc.validateWildcardChild(ctx, wc, children[pos+i], edcScope); err != nil {
			contentErr = err
		}
	}
	if contentErr != nil {
		return count, contentErr
	}

	return count, nil
}

// validateWildcardChild validates a single element matched by a wildcard
// according to its processContents setting (skip = not assessed; lax = assess if
// a governing type can be found — a global declaration OR, with no declaration, a
// resolvable xsi:type; strict = a global declaration is required). It mirrors the
// per-child logic the run-based matchWildcardParticle applies, factored out so the
// xs:all matcher can reuse it.
func (vc *validationContext) validateWildcardChild(ctx context.Context, wc *Wildcard, child childElem, edcScope *ModelGroup) error {
	if wc.ProcessContents == ProcessSkip {
		vc.annotateSkipChildren(ctx, child.elem)
		return nil
	}
	edecl := lookupElemDecl(child.elem, vc.schema)
	if edecl == nil {
		if wc.ProcessContents == ProcessStrict {
			msg := "No matching global declaration available, but demanded by the strict wildcard."
			vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, msg)
			// Strict assessment FAILED (no declaration), so the element AND its whole
			// subtree are NOT schema-assessed — exactly like skip content. Walk it
			// with annotateSkipChildren (canonicalization-only: records pass-2
			// actualElemType with assessed=false, NEVER assessedElemType/actualAttrType
			// and reports no diagnostics), so pass 3 does NOT collect xs:ID/xs:IDREF
			// from this unassessed subtree. annotateAnyTypeChildren must NOT be used
			// here: it laxly ASSESSES globally-declared / xsi:typed descendants, which
			// would fabricate a duplicate-ID/dangling diagnostic on top of the real
			// strict-wildcard failure.
			vc.annotateSkipChildren(ctx, child.elem)
			return fmt.Errorf("strict wildcard: no global element decl")
		}
		// Lax with no global declaration: per XSD lax assessment, if a governing
		// type is found via xsi:type the element must be ·valid· against it and is
		// schema-assessed (so its xs:ID/xs:IDREF content participates in the
		// ID/IDREF pass); otherwise it is not assessed and only its subtree is
		// walked to record descendants' ACTUAL types for pass-2 IDC
		// canonicalization. assessLaxElement handles both cases and never lets
		// xsi:nil bypass type validation.
		if actual, ok := vc.resolveXsiTypeQuiet(child.elem); ok {
			if err := vc.validateWildcardElementConsistent(ctx, edcScope, child, actual); err != nil {
				return err
			}
		}
		return vc.assessLaxElement(ctx, child.elem)
	}
	declType := effectiveDeclType(edecl, vc.schema)
	declType = vc.applyTypeAlternatives(ctx, child.elem, edecl, declType)
	td, xsiErr := vc.resolveXsiType(ctx, child.elem, declType, vc.hasTypeTable(edecl))
	if xsiErr != nil {
		return xsiErr
	}
	// A blocked xsi:type derivation is a validity error (cvc-elt.4.3), enforced for
	// a strict wildcard-matched global element too. The blocked set unions the
	// element declaration's block with the declared type's {prohibited substitutions}.
	if td != declType && declType != nil && typeDerivationBlocked(td, declType, edecl.Block) {
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem),
			"The xsi:type definition is blocked by the element declaration.")
		return fmt.Errorf("blocked xsi:type")
	}
	if td != nil && td.Abstract {
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), elemDisplayName(child.elem), msgAbstractType)
		return fmt.Errorf("abstract type")
	}
	if err := vc.validateWildcardElementConsistent(ctx, edcScope, child, td); err != nil {
		return err
	}
	vc.annotateElement(ctx, child.elem, td, true)
	if td == nil {
		return nil
	}
	nilled, nilErr := vc.checkXsiNil(ctx, child.elem, edecl)
	if nilErr != nil {
		return nilErr
	}
	if nilled {
		return vc.validateNilledElement(ctx, child.elem, edecl, td)
	}
	return vc.validateElementContent(ctx, child.elem, edecl, td)
}

func (vc *validationContext) validateWildcardElementConsistent(ctx context.Context, mg *ModelGroup, child childElem, governing *TypeDef) error {
	if vc.version != Version11 || governing == nil {
		return nil
	}
	qn := QName{Local: child.name, NS: child.ns}
	for _, decl := range vc.edcLocalDecls(mg, qn) {
		localType := effectiveDeclType(decl, vc.schema)
		if localType == nil {
			continue
		}
		// The wildcard's governing type must be substitutable for the base-local
		// declared type AND not blocked by the EFFECTIVE {disallowed substitutions}
		// — the UNION of the element declaration's block and its declared TYPE's
		// {prohibited substitutions} (cvc-elt.4.3, via the shared helper) — so a
		// derivation the base-local's TYPE blocks cannot be re-admitted through the
		// wildcard.
		if isValidlySubstitutable(governing, localType) &&
			!typeDerivationBlocked(governing, localType, decl.Block) {
			continue
		}
		msg := fmt.Sprintf("The wildcard-matched element's governing type definition is not validly substitutable for the locally declared type definition of element '%s'.", child.displayName)
		vc.reportValidityError(ctx, vc.filename, child.elem.Line(), child.displayName, msg)
		return fmt.Errorf("wildcard element declaration inconsistent")
	}
	return nil
}

// edcLocalDecls gathers the same-named local element declarations that the
// dynamic EDC check must compare the wildcard's governing type against: those in
// the current content model (mg) AND those declared in the BASE type chain of the
// type whose content is being matched (vc.edcType). A restriction may drop a
// base's local element declaration while admitting it through a wildcard; the
// base declaration's type still constrains the element, so the wildcard's
// governing type must remain substitutable for it.
func (vc *validationContext) edcLocalDecls(mg *ModelGroup, qn QName) []*ElementDecl {
	decls := localElementDeclsByName(mg, qn)
	if vc.edcType == nil {
		return decls
	}
	seen := make(map[*TypeDef]struct{})
	for base := vc.edcType.BaseType; base != nil; base = base.BaseType {
		if _, ok := seen[base]; ok {
			break
		}
		seen[base] = struct{}{}
		if base.ContentModel == nil || base.ContentModel == mg {
			continue
		}
		decls = append(decls, localElementDeclsByName(base.ContentModel, qn)...)
	}
	return decls
}

func localElementDeclsByName(mg *ModelGroup, qn QName) []*ElementDecl {
	var decls []*ElementDecl
	visited := make(map[*ModelGroup]struct{})
	var walk func(*ModelGroup)
	walk = func(g *ModelGroup) {
		if g == nil || g.MaxOccurs == 0 {
			return
		}
		if _, ok := visited[g]; ok {
			return
		}
		visited[g] = struct{}{}
		for _, p := range g.Particles {
			if p.MaxOccurs == 0 {
				continue
			}
			switch term := p.Term.(type) {
			case *ElementDecl:
				if !term.IsRef && term.Name == qn {
					decls = append(decls, term)
				}
			case *ModelGroup:
				walk(term)
			}
		}
	}
	walk(mg)
	return decls
}

// tryMatchWildcardParticle is the try version (no error reporting).
func (vc *validationContext) tryMatchWildcardParticle(_ context.Context, p *Particle, wc *Wildcard, children []childElem, pos int) (int, error) {
	count := 0
	for pos+count < len(children) {
		child := children[pos+count]
		if !wildcardAllowsExpandedName(wc, child.elem.LocalName(), child.elem.URI(), vc.schema, false) {
			break
		}
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}

	if count < p.MinOccurs {
		return 0, fmt.Errorf("wildcard not matched")
	}

	return count, nil
}

// wildcardMatches checks if an element namespace matches a wildcard's NAMESPACE
// constraint. In XSD 1.1 a wildcard may instead carry a @notNamespace negation
// (NotNamespace), which matches any namespace NOT in the excluded set; the two
// are mutually exclusive, so NotNamespace takes precedence when present. This is
// the namespace-only half of wildcard matching — the @notQName disallowed-name
// half is applied by callers that have the local name (wildcardExcludesName).
func wildcardMatches(wc *Wildcard, elemNS string) bool {
	if wc.NotNamespace != nil {
		return !slices.Contains(wc.NotNamespace, elemNS)
	}
	ns := wc.Namespace
	switch ns {
	case WildcardNSAny:
		return true
	case WildcardNSOther:
		// Matches any namespace other than the target namespace.
		// Also does not match absent namespace (no namespace).
		return elemNS != "" && elemNS != wc.TargetNS
	case WildcardNSNotAbsent:
		// Matches any namespace except absent (empty namespace).
		return elemNS != ""
	default:
		// Space-separated list that may include ##local, ##targetNamespace, and URIs.
		for _, part := range splitSpace(ns) {
			switch part {
			case WildcardNSLocal:
				if elemNS == "" {
					return true
				}
			case WildcardNSTargetNamespace:
				if elemNS == wc.TargetNS {
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

// wildcardExcludesName reports whether the wildcard's XSD 1.1 @notQName
// disallowed-name set excludes the given expanded name. isAttr selects the
// global declaration table consulted for ##defined (attributes vs elements).
// A name excluded here is NOT matched by the wildcard even though its namespace
// is admitted. In XSD 1.0 (no notQName fields) this always returns false.
func wildcardExcludesName(wc *Wildcard, local, ns string) bool {
	for _, qn := range wc.NotQName {
		if qn.Local == local && qn.NS == ns {
			return true
		}
	}
	for _, qn := range wc.SiblingNames {
		if qn.Local == local && qn.NS == ns {
			return true
		}
	}
	return false
}

// wildcardAllowsExpandedName is the full XSD 1.1 "Wildcard allows Expanded Name"
// test: the namespace constraint admits ns AND the @notQName disallowed-name set
// (including ##defined / ##definedSibling) does not exclude {local, ns}. isAttr
// selects the global table for ##defined.
func wildcardAllowsExpandedName(wc *Wildcard, local, ns string, schema *Schema, isAttr bool) bool {
	if !wildcardMatches(wc, ns) {
		return false
	}
	if wildcardExcludesName(wc, local, ns) {
		return false
	}
	if wc.NotQNameDefined && schema != nil {
		qn := QName{Local: local, NS: ns}
		if isAttr {
			if _, ok := schema.globalAttrs[qn]; ok {
				return false
			}
		} else if _, ok := schema.elements[qn]; ok {
			return false
		}
	}
	return true
}

// wildcardExpected formats the expected string for wildcard error messages.
func wildcardExpected(wc *Wildcard) string {
	switch wc.Namespace {
	case WildcardNSAny:
		return WildcardNSAny
	case WildcardNSOther:
		if wc.TargetNS != "" {
			return WildcardNSOther + "{" + wc.TargetNS + "}*"
		}
		return WildcardNSOther + "*"
	default:
		return wc.Namespace
	}
}

// instanceSubstMembers returns the INSTANCE-ADMISSIBLE substitution-group closure
// of a head: the transitive declaration-membership closure (substitutableMembersFor,
// which is already block/derivation-filtered and traverses THROUGH abstract
// intermediates) with ABSTRACT members removed. An abstract element can never
// appear in an instance, but a concrete descendant behind an abstract intermediate
// (h <- abstract m1 <- concrete m2) is still reached. Used by runtime matching
// (elemMatchesDeclOrSubst) and restriction subsumption (findBaseAllMember). It
// builds ON TOP of #877's substitutableMembersFor rather than re-walking substGroups.
func instanceSubstMembers(head *ElementDecl, schema *Schema) []*ElementDecl {
	all := substitutableMembersFor(head, schema)
	concrete := all[:0] // substitutableMembersFor returns a fresh slice; reuse it
	for _, m := range all {
		if !m.Abstract {
			concrete = append(concrete, m)
		}
	}
	return concrete
}

// elemMatchesDeclOrSubst checks if a child element matches a declaration
// directly or via substitution group. schema may be nil for basic matching.
func elemMatchesDeclOrSubst(child childElem, edecl *ElementDecl, schema *Schema) bool {
	if matchesDeclDirect(child, edecl) && !edecl.Abstract {
		return true
	}
	// Check the TRANSITIVE instance-admissible substitution-group closure
	// (abstract members excluded — they can never appear in an instance).
	if schema != nil {
		for _, member := range instanceSubstMembers(edecl, schema) {
			if matchesDeclDirect(child, member) {
				return true
			}
		}
	}
	return false
}

// prohibitedSubstitutions returns the type's {prohibited substitutions} (block),
// or the empty set when the type is nil. cvc-elt.4.3 unions this with the element
// declaration's {disallowed substitutions} when checking an xsi:type derivation.
func (td *TypeDef) prohibitedSubstitutions() BlockFlags {
	if td == nil {
		return 0
	}
	return td.Block
}

// typeDerivationBlocked reports whether deriving `derived` from `base` uses a
// method forbidden by the EFFECTIVE {disallowed substitutions}: the UNION of the
// supplied element-declaration block `elemBlock` and the BASE TYPE's {prohibited
// substitutions}, masked to the derivation bits (extension/restriction). Both
// cvc-elt.4.3 (xsi:type / CTA) and Substitution Group OK use this union, so a
// derivation blocked by the base TYPE's @block is caught even when the referring
// element declaration carries no block. Nil-safe throughout (prohibitedSubstitutions
// and isDerivationBlocked both handle nil), so it doubles as the single shared
// derivation-block gate at every substitutability site.
func typeDerivationBlocked(derived, base *TypeDef, elemBlock BlockFlags) bool {
	const derivBits = BlockExtension | BlockRestriction
	return isDerivationBlocked(derived, base, (elemBlock|base.prohibitedSubstitutions())&derivBits)
}

// isDerivationBlocked walks the BaseType chain from derived to base and returns
// true if any step uses a derivation method blocked by the given BlockFlags.
func isDerivationBlocked(derived, base *TypeDef, blocked BlockFlags) bool {
	if derived == nil || base == nil || blocked == 0 {
		return false
	}
	td := derived
	for td != nil && td != base {
		if td.Derivation == DerivationExtension && blocked&BlockExtension != 0 {
			return true
		}
		if td.Derivation == DerivationRestriction && blocked&BlockRestriction != 0 {
			return true
		}
		td = td.BaseType
	}
	// A declared union can be substituted by one of its member types even though the
	// member's BaseType chain does not point back to the union. Treat that path as a
	// restriction for element block enforcement; otherwise block="restriction" on a
	// union-typed element would be bypassed by xsi:type naming a member.
	if td != base && blocked&BlockRestriction != 0 && resolveVariety(base) == TypeVarietyUnion {
		for _, member := range resolveUnionMembers(base) {
			if isXsiTypeDerivedFromDeclared(derived, member) {
				return true
			}
		}
	}
	// The BaseType pointer chain is NOT linked for built-in simple types, so it can
	// bottom out (td == nil) before reaching a built-in base. ALL built-in
	// simple-type derivation is by RESTRICTION, so when base is a built-in simple
	// type that derived's effective built-in base is a STRICT subtype of, a blocked
	// restriction derivation must be rejected — e.g. xsi:type="xs:int" over a
	// declared xs:integer with block="restriction" (or block="#all"). Without this
	// the block is bypassed because isDerivedFrom-style pointer walking can't chain
	// xs:int → xs:integer.
	if td != base && blocked&BlockRestriction != 0 && isBuiltinSimpleType(base) {
		db := builtinBaseLocal(derived)
		if db != base.Name.Local && builtinSimpleDerivedFrom(db, base.Name.Local) {
			return true
		}
	}
	// Reaching xs:anyType is ALWAYS a restriction step (§3.4.2): a simple type
	// restricts xs:anySimpleType which restricts xs:anyType, and a complex type
	// with no explicit <xs:complexContent>/<xs:simpleContent> derivation has an
	// IMPLICIT {base type definition} = xs:anyType with {derivation method} =
	// restriction. So on an xs:anyType-typed element a block="restriction"/"#all"
	// must reject such an xsi:type (cvc-elt.4.3). The loop exits with td != base
	// (i.e. td == nil) precisely when the BaseType pointer chain bottoms out WITHOUT
	// reaching base by pointer — that is the implicit-restriction-to-anyType case,
	// covering a simple xsi:type, a complex type with a direct model group, AND an
	// explicit extension chain that bottoms out at such an implicit-base complex
	// type (its suffix down to anyType still contains the implicit restriction step).
	// An explicit <xs:extension base="xs:anyType"> instead reaches base BY POINTER
	// (td == base), so it is correctly NOT blocked here — its only derivation step
	// is the extension. Gated on blocked&BlockRestriction, so a block="extension"-
	// only element still accepts these (the restriction step is not in its block).
	if td != base && blocked&BlockRestriction != 0 && isAnyType(base) {
		return true
	}
	return false
}

// substTypeDerivationBlocked reports whether admitting `derived` as a substitution
// for a head declaration whose effective type is `base` is blocked. It layers the
// substitution-group-specific rule on top of typeDerivationBlocked (which handles
// the head element's block unioned with the BASE type's {prohibited substitutions}):
// per Substitution Group OK (Transitive) §3.3.6.3, the {prohibited substitutions} of
// every INTERMEDIATE type definition in the derivation chain also block a step using
// the forbidden method. So a chain Base <- Mid(block="extension") <- Leaf is not
// substitutable for a head of type Base even though neither Base nor the head element
// blocks extension — Mid does. Only the substitution-group path considers
// intermediate-type blocks; xsi:type and CTA use typeDerivationBlocked directly (they
// key on the selected/base type's block, not every intermediate's).
func substTypeDerivationBlocked(derived, base *TypeDef, elemBlock BlockFlags) bool {
	if derived == nil || base == nil {
		return false
	}
	if typeDerivationBlocked(derived, base, elemBlock) {
		return true
	}
	// For every INTERMEDIATE base type `mid` strictly between derived and base, the
	// substitution is blocked if the FULL derivation from derived DOWN TO mid uses any
	// method mid's {prohibited substitutions} forbids — not merely the single child
	// step into mid. A blocked method may occur ANYWHERE in the derived..mid suffix:
	// e.g. Base <- Mid(block="extension") <- R(restriction) <- Leaf(extension) is
	// blocked because Leaf..Mid includes Leaf's extension step even though the direct
	// step into Mid (R's) is a restriction. isDerivationBlocked walks the derived..mid
	// suffix and returns true iff the given mask intersects the set of methods used
	// along it, giving exactly that whole-suffix semantics. The final `base` is already
	// covered by typeDerivationBlocked's base-type union above.
	const derivBits = BlockExtension | BlockRestriction
	for mid := derived.BaseType; mid != nil && mid != base; mid = mid.BaseType {
		if isDerivationBlocked(derived, mid, mid.prohibitedSubstitutions()&derivBits) {
			return true
		}
	}
	return false
}

func matchesDeclDirect(child childElem, edecl *ElementDecl) bool {
	if child.name != edecl.Name.Local {
		return false
	}
	return child.ns == edecl.Name.NS
}

// elementDisplayForExpected formats an element declaration name for error messages.
func elementDisplayForExpected(edecl *ElementDecl) string {
	if edecl.Name.NS != "" {
		return helium.ClarkName(edecl.Name.NS, edecl.Name.Local)
	}
	return edecl.Name.Local
}

// elementExpectedNamesWithSubst returns the list of expected element names
// for a declaration, including substitution group members.
// The head element is always listed first (even if abstract), followed by members.
func elementExpectedNamesWithSubst(edecl *ElementDecl, schema *Schema) []string {
	members := substitutableMembersFor(edecl, schema)
	if len(members) == 0 {
		return []string{elementDisplayForExpected(edecl)}
	}
	names := []string{elementDisplayForExpected(edecl)}
	for _, m := range members {
		names = append(names, elementDisplayForExpected(m))
	}
	return names
}

// consumerKind classifies how a particle would consume a given child as its
// FIRST consuming term: through an element leaf, through a wildcard leaf, or not
// at all.
type consumerKind int

const (
	consumerNone consumerKind = iota
	consumerElement
	consumerWildcard
)

// particleConsumesViaElement reports whether the particle would consume the
// given child through a non-wildcard (element) leaf AS ITS FIRST CONSUMING TERM.
// It is path-aware: it respects compositor order, occurrences, and emptiable
// prefixes, so a leading wildcard (direct or nested) that would consume the
// child first makes the particle NOT an element-first-consumer — even when a
// later element leaf inside the same model group also matches the child's name.
//
// Used for XSD 1.1 element-over-wildcard precedence: when selecting a choice
// branch for the current child, branches that consume it via an element leaf as
// first consumer are preferred over branches that would consume it via a
// wildcard, so a skip wildcard (direct or nested as a leading term) cannot steal
// a child a typed element declaration is responsible for validating. This is
// bounded first-consumer determination, not full backtracking. Side-effect free.
func particleConsumesViaElement(p *Particle, child childElem, schema *Schema) bool {
	return particleFirstConsumerKind(p, child, schema) == consumerElement
}

// sequenceElementReservedLimit implements XSD 1.1 element-over-wildcard
// precedence for the SEQUENCE case. It returns an end index into children that
// the particle at index i may see while matching, so a LEADING minOccurs=0
// wildcard (direct, or nested as the particle's first consuming term) does not
// greedily consume a child that a LATER element particle in the same sequence is
// responsible for validating. Without this, an effective content model like
// sequence( any{0,unbounded} lax, state, currency, zip ) — produced by extending
// a base whose content is just a lax unbounded wildcard — has the wildcard eat
// the required named elements, falsely reporting "Missing child element(s)".
//
// The reservation fires ONLY when particle i would consume the child via a
// wildcard, NOT via an element leaf: element-vs-element matching stays in
// declaration order (the wildcard is the only thing element precedence
// displaces). It returns len(children) when nothing is reserved, so an
// all-element sequence is unaffected (byte-identical). Bounded lookahead, no
// backtracking; mirrors the matchChoice/particleConsumesViaElement approach.
// Callers MUST gate on Version11. Side-effect free.
//
// REACHABILITY: a later element particle P(j) only competes for children[k] if
// it is reachable from the current position with the wildcard yielding the
// child — i.e. every intervening particle P(i+1..j-1) is emptiable. A REQUIRED
// (non-emptiable) intervening particle must itself consume content here, so it
// blocks P(j) from the child; that child is then NOT reserved (the wildcard or
// the required particle handles it in declaration order). Without this gating an
// optional element after a required one would steal a child the required element
// (or the wildcard) must keep — an over-rejection.
func sequenceElementReservedLimit(particles []*Particle, i int, children []childElem, cur int, schema *Schema) int {
	for k := cur; k < len(children); k++ {
		child := children[k]
		reservedForLater := false
		for j := i + 1; j < len(particles); j++ {
			if particleConsumesViaElement(particles[j], child, schema) {
				reservedForLater = true
				break
			}
			// P(j) does not consume this child via an element. If it is not
			// emptiable it is required here and blocks any later particle from
			// reaching this child, so stop looking past it.
			if !particleEmptiable(particles[j]) {
				break
			}
		}
		if !reservedForLater {
			continue
		}
		// Element-vs-element: particle i keeps the child by declaration order, so
		// only a child particle i would take via a WILDCARD is reserved.
		if particleConsumesViaElement(particles[i], child, schema) {
			continue
		}
		return k
	}
	return len(children)
}

// particleFirstConsumerKind classifies how the particle would consume child as
// its first consuming term.
func particleFirstConsumerKind(p *Particle, child childElem, schema *Schema) consumerKind {
	switch term := p.Term.(type) {
	case *ElementDecl:
		if p.MaxOccurs == 0 {
			return consumerNone
		}
		if elemMatchesDeclOrSubst(child, term, schema) {
			return consumerElement
		}
		return consumerNone
	case *Wildcard:
		if p.MaxOccurs == 0 {
			return consumerNone
		}
		if wildcardAllowsExpandedName(term, child.name, child.ns, schema, false) {
			return consumerWildcard
		}
		return consumerNone
	case *ModelGroup:
		if term.MaxOccurs == 0 {
			return consumerNone
		}
		return groupFirstConsumerKind(term, child, schema)
	}
	return consumerNone
}

// groupFirstConsumerKind classifies how a model group would consume child as its
// first consuming term, respecting compositor order and emptiable prefixes.
func groupFirstConsumerKind(mg *ModelGroup, child childElem, schema *Schema) consumerKind {
	switch mg.Compositor {
	case CompositorSequence:
		// Walk in order: the first member that can consume child decides; a
		// non-matching but emptiable prefix member is skipped; a non-matching,
		// non-emptiable member means the group cannot reach child at all.
		for _, p := range mg.Particles {
			kind := particleFirstConsumerKind(p, child, schema)
			if kind != consumerNone {
				return kind
			}
			if !particleEmptiable(p) {
				return consumerNone
			}
		}
		return consumerNone
	default: // choice, all
		// Element-first wins if ANY member is element-first; otherwise fall back
		// to wildcard-first if some member consumes child via a wildcard.
		result := consumerNone
		for _, p := range mg.Particles {
			switch particleFirstConsumerKind(p, child, schema) {
			case consumerElement:
				return consumerElement
			case consumerWildcard:
				result = consumerWildcard
			}
		}
		return result
	}
}

// sequenceHasWildcard returns true if any particle in the model group is a wildcard.
func sequenceHasWildcard(mg *ModelGroup) bool {
	for _, p := range mg.Particles {
		if _, ok := p.Term.(*Wildcard); ok {
			return true
		}
	}
	return false
}

func formatExpected(prefix string, names []string) string {
	if len(names) == 1 {
		return fmt.Sprintf("%s Expected is ( %s ).", prefix, names[0])
	}
	return fmt.Sprintf("%s Expected is one of ( %s ).", prefix, strings.Join(names, ", "))
}

func particleNames(particles []*Particle, schema *Schema) []string {
	var names []string
	for _, p := range particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			names = append(names, elementExpectedNamesWithSubst(term, schema)...)
		case *Wildcard:
			names = append(names, wildcardExpected(term))
		}
	}
	return names
}
