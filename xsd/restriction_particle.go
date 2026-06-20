package xsd

import (
	"context"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkRestrictionParticles implements (a conservative subset of) the XSD 1.1
// derivation-ok-restriction constraint for the CONTENT MODEL of a complexContent
// restriction (§3.9.6, "Particle Valid (Restriction)"). It complements
// checkRestrictionAttrs (which only covers attribute uses) by verifying that the
// derived type's effective content model is a valid restriction of the base
// type's: each derived particle must map, order-preservingly, onto a base
// particle it restricts (name/type/occurrence), no derived particle may be added
// that the base does not allow, and any base particle not matched by the
// derivation must be emptiable.
//
// The check is intentionally CONSERVATIVE: whenever a sub-case cannot be decided
// with confidence (e.g. exotic wildcard interactions, substitution-group
// containment, or structural shapes the recursion does not model), it treats the
// derivation as VALID rather than risk a false rejection of a legitimate schema.
// Its job is to catch the clear violations — reordered particles, added/renamed
// particles, and widened occurrence ranges — without breaking the golden suite.
func (c *compiler) checkRestrictionParticles(ctx context.Context, td *TypeDef) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[td]
	if !hasSrc {
		return
	}
	base := td.BaseType
	if base == nil {
		return
	}

	// Restriction directly off the ur-type (xs:anyType) is unconstrained: any
	// content model is a valid restriction of xs:anyType, so accept regardless of
	// the derived/base model groups. This must be checked BEFORE the nil-model
	// handling below, otherwise an xs:anyType base (which carries no usable
	// content model) would fall into the baseMG==nil reject path.
	if base.Name.Local == typeAnyType && base.Name.NS == lexicon.NamespaceXSD {
		return
	}

	derivedMG := td.ContentModel
	baseMG := base.ContentModel

	// A derived restriction with NO content model restricts the base to empty
	// content. That is only a valid restriction when the base content model is
	// itself emptiable (can be satisfied by zero elements); restricting a base
	// that requires content down to empty content is a violation. When the base
	// also has no content model (empty/simple base), there is nothing to recurse
	// against and the content-type rules (cos-ct-extends) already cover it.
	if derivedMG == nil {
		if baseMG == nil {
			return
		}
		baseP := &Particle{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG}
		if particleEmptiable(baseP) {
			return
		}
		c.reportInvalidRestriction(ctx, td, base, src)
		return
	}
	// The derived type has a content model but the base does not (empty/simple
	// content base). Adding content where the base allows none is not a valid
	// restriction. (The xs:anyType base case is handled above.)
	if baseMG == nil {
		c.reportInvalidRestriction(ctx, td, base, src)
		return
	}

	// Wrap the top-level model groups as particles so the occurrence range of the
	// whole group participates in the check.
	derivedP := &Particle{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG}
	baseP := &Particle{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG}

	if particleValidRestriction(derivedP, baseP) {
		return
	}

	c.reportInvalidRestriction(ctx, td, base, src)
}

// reportInvalidRestriction emits the fatal derivation-ok-restriction diagnostic
// for a complexContent restriction whose content model does not validly restrict
// its base.
func (c *compiler) reportInvalidRestriction(ctx context.Context, td, base *TypeDef, src typeDefSource) {
	component := componentLocalComplexType
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}
	baseQualified := "'{" + base.Name.NS + "}" + base.Name.Local + "'"
	msg := "The content model is not a valid restriction of the content model of the base complex type definition " + baseQualified + "."
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg), helium.ErrorLevelFatal))
	c.errorCount++
}

// particleValidRestriction reports whether the restriction particle r is a valid
// restriction of the base particle b. Returning true means "accepted" — and, per
// the conservative contract above, it also returns true for any case the
// recursion is not confident enough to reject.
func particleValidRestriction(r, b *Particle) bool {
	switch rt := r.Term.(type) {
	case *ElementDecl:
		switch bt := b.Term.(type) {
		case *ElementDecl:
			// NameAndTypeOK
			return elementRestrictsElement(r, rt, b, bt)
		case *Wildcard:
			// NSCompat: element restricts wildcard — the element's name must be
			// allowed by the wildcard and occurrence must be a valid restriction.
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			return wildcardAllowsName(bt, rt.Name)
		case *ModelGroup:
			// Recurse-As-If-Group (XSD §3.9.6): a derived element against a base
			// model group. Treat the element as a singleton group and map it through
			// the base group's compositor-specific children.
			return elementRestrictsGroup(r, b, bt)
		}
	case *Wildcard:
		switch bt := b.Term.(type) {
		case *Wildcard:
			// NSSubset: the derived wildcard restricts the base wildcard iff its
			// occurrence range is a subset, its namespace constraint is a subset, and
			// its processContents is at least as strong as the base's (strict > lax >
			// skip — a restriction may tighten but never weaken validation).
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			if processContentsStrength(rt.ProcessContents) < processContentsStrength(bt.ProcessContents) {
				return false
			}
			return wildcardConstraintSubset(rt, bt)
		case *ElementDecl:
			// A wildcard can never be a restriction of a single element. This is a
			// clear violation.
			return false
		case *ModelGroup:
			// NSRecurseCheckCardinality — conservatively accept.
			return true
		}
	case *ModelGroup:
		switch bt := b.Term.(type) {
		case *ModelGroup:
			return groupRestrictsGroup(r, rt, b, bt)
		case *Wildcard:
			// NSRecurseCheckCardinality (XSD §3.9.6): the derived group's effective
			// occurrence range must be within the base wildcard's range, and every
			// element/wildcard LEAF inside the derived group must be admitted by the
			// base wildcard (namespace) and within its cardinality.
			return groupRestrictsWildcard(r, rt, b, bt)
		case *ElementDecl:
			// A group against a single base element — conservatively accept.
			return true
		}
	}
	return true
}

// elementRestrictsElement checks the element-to-element (NameAndTypeOK) case:
// same expanded name, occurrence range subset, and the derived element's type is
// derived from (or equal to) the base element's type. nillable/fixed tightening
// is checked conservatively.
func elementRestrictsElement(r *Particle, rt *ElementDecl, b *Particle, bt *ElementDecl) bool {
	if rt.Name.Local != bt.Name.Local || rt.Name.NS != bt.Name.NS {
		return false
	}
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	// Type derivation: the derived element's type must be the same as, or derived
	// from, the base element's type. When either type is unresolved, accept
	// conservatively.
	if rt.Type != nil && bt.Type != nil {
		if !isDerivedFrom(rt.Type, bt.Type) {
			return false
		}
	}
	// A base element that is fixed forces the derived element to carry the same
	// fixed value; a base that is not nillable forbids the derived from becoming
	// nillable. These are tightening rules — only flag the clear widening cases.
	if !bt.Nillable && rt.Nillable {
		return false
	}
	if bt.Fixed != nil {
		if rt.Fixed == nil || *rt.Fixed != *bt.Fixed {
			return false
		}
	}
	return true
}

// groupRestrictsGroup handles the model-group cases (recurse / map-and-sum). It
// requires the derived group's occurrence range to be a valid restriction of the
// base's, then dispatches on compositor.
func groupRestrictsGroup(r *Particle, rg *ModelGroup, b *Particle, bg *ModelGroup) bool {
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	switch {
	case rg.Compositor == CompositorSequence && bg.Compositor == CompositorSequence:
		return recurseOrdered(rg.Particles, bg.Particles)
	case rg.Compositor == CompositorAll && bg.Compositor == CompositorAll:
		return recurseAll(rg.Particles, bg.Particles)
	case rg.Compositor == CompositorChoice && bg.Compositor == CompositorChoice:
		return recurseChoiceUnordered(rg.Particles, bg.Particles)
	case rg.Compositor == CompositorSequence && bg.Compositor == CompositorChoice:
		// MapAndSum-ish: each derived sequence member must match SOME base choice
		// alternative. Conservatively accept (the recurse below is order-based and
		// would false-reject legitimate map-and-sum shapes).
		return true
	default:
		// Mixed/unsupported compositor combination — conservatively accept.
		return true
	}
}

// recurseOrdered implements the order-preserving "Recurse" mapping for
// sequence→sequence (and, used conservatively, choice→choice). Each base
// particle is consumed left-to-right: it either restricts the next derived
// particle (advancing both) or is skipped only if it is emptiable. Every derived
// particle must be consumed, and any trailing base particles must be emptiable.
func recurseOrdered(rParticles, bParticles []*Particle) bool {
	ri := 0
	for bi := range bParticles {
		bp := bParticles[bi]
		if ri < len(rParticles) && particleValidRestriction(rParticles[ri], bp) {
			ri++
			continue
		}
		// This base particle is not matched by the current derived particle. It is
		// only allowed to be unmatched if it is emptiable (can occur zero times).
		if !particleEmptiable(bp) {
			return false
		}
	}
	// Every derived particle must have been consumed by some base particle.
	return ri == len(rParticles)
}

// recurseChoiceUnordered implements the choice→choice mapping (RecurseLax,
// XSD §3.9.6): order is NOT significant in a choice, so each derived alternative
// need only be a valid restriction of SOME base alternative, and base
// alternatives may be dropped freely (narrowing a choice to fewer branches is a
// valid restriction). A derived alternative with no matching base alternative is
// a clear violation (it admits content the base choice does not).
func recurseChoiceUnordered(rParticles, bParticles []*Particle) bool {
	for _, rp := range rParticles {
		matched := false
		for _, bp := range bParticles {
			if particleValidRestriction(rp, bp) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// recurseAll handles all→all: every derived particle must restrict a DISTINCT
// base particle (order is irrelevant in an all group), and every base particle
// not matched must be emptiable.
func recurseAll(rParticles, bParticles []*Particle) bool {
	used := make([]bool, len(bParticles))
	for _, rp := range rParticles {
		matched := false
		for bi, bp := range bParticles {
			if used[bi] {
				continue
			}
			if particleValidRestriction(rp, bp) {
				used[bi] = true
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for bi, bp := range bParticles {
		if used[bi] {
			continue
		}
		if !particleEmptiable(bp) {
			return false
		}
	}
	return true
}

// elementRestrictsGroup implements Recurse-As-If-Group (XSD §3.9.6): a derived
// ELEMENT particle restricting a base MODEL GROUP particle. The derived element
// is treated as a singleton group and mapped through the base group's children
// according to the base compositor:
//   - sequence/all: the element must restrict EXACTLY ONE base child, and every
//     OTHER base child must be emptiable;
//   - choice: the element must restrict SOME base alternative.
//
// The element particle's occurrence range must also be a valid restriction of
// the base group particle's range. When the recursion cannot decide a sub-case
// with confidence it stays conservative (accepts) rather than risk a false
// rejection.
func elementRestrictsGroup(r *Particle, b *Particle, bg *ModelGroup) bool {
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	switch bg.Compositor {
	case CompositorChoice:
		// The element must restrict SOME alternative of the base choice.
		return slices.ContainsFunc(bg.Particles, func(bp *Particle) bool {
			return particleValidRestriction(r, bp)
		})
	case CompositorSequence, CompositorAll:
		// The element must restrict exactly one base child; every unmatched base
		// child must be emptiable.
		matched := false
		for _, bp := range bg.Particles {
			if !matched && particleValidRestriction(r, bp) {
				matched = true
				continue
			}
			if !particleEmptiable(bp) {
				return false
			}
		}
		return matched
	default:
		// Unknown compositor — stay conservative.
		return true
	}
}

// groupRestrictsWildcard implements NSRecurseCheckCardinality (XSD §3.9.6): a
// derived MODEL GROUP particle restricting a base WILDCARD particle. The derived
// group's EFFECTIVE occurrence range (the product of the nesting occurrence
// ranges down to each leaf) must be within the base wildcard particle's range,
// and every element/wildcard LEAF reachable inside the derived group must be
// admitted by the base wildcard's namespace constraint. When a sub-case cannot
// be decided with confidence (e.g. a nested group whose effective range is
// genuinely undecidable) it stays conservative (accepts).
func groupRestrictsWildcard(r *Particle, rg *ModelGroup, b *Particle, bw *Wildcard) bool {
	// The whole derived group particle's occurrence range must be within the base
	// wildcard particle's range.
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	// Each leaf inside the derived group must be admitted by the wildcard, and its
	// effective occurrence (folding in the enclosing group ranges) must be within
	// the base wildcard particle's range.
	return groupLeavesWithinWildcard(rg, r.MinOccurs, r.MaxOccurs, b)
}

// groupLeavesWithinWildcard walks the derived group's particles, threading the
// accumulated effective occurrence range (encMin/encMax = the product of the
// enclosing groups' occurrence ranges), and checks every element/wildcard leaf
// against the base wildcard particle bw: the leaf's namespace must be admitted
// and its effective occurrence range (enclosing × the leaf's own) must be a
// valid restriction of the base wildcard particle's range. Returns false on the
// first clear violation.
func groupLeavesWithinWildcard(rg *ModelGroup, encMin, encMax int, bw *Particle) bool {
	for _, p := range rg.Particles {
		leafMin := mulOccurs(encMin, p.MinOccurs)
		leafMax := mulOccurs(encMax, p.MaxOccurs)
		switch t := p.Term.(type) {
		case *ElementDecl:
			wc, ok := bw.Term.(*Wildcard)
			if !ok {
				return true
			}
			if !wildcardAllowsName(wc, t.Name) {
				return false
			}
			if !occurrenceValidRestriction(leafMin, leafMax, bw.MinOccurs, bw.MaxOccurs) {
				return false
			}
		case *Wildcard:
			bwc, ok := bw.Term.(*Wildcard)
			if !ok {
				return true
			}
			// A derived wildcard leaf must be a namespace subset of the base
			// wildcard, with at-least-as-strong processContents and within range.
			if processContentsStrength(t.ProcessContents) < processContentsStrength(bwc.ProcessContents) {
				return false
			}
			if !wildcardConstraintSubset(t, bwc) {
				return false
			}
			if !occurrenceValidRestriction(leafMin, leafMax, bw.MinOccurs, bw.MaxOccurs) {
				return false
			}
		case *ModelGroup:
			if !groupLeavesWithinWildcard(t, leafMin, leafMax, bw) {
				return false
			}
		}
	}
	return true
}

// mulOccurs multiplies two occurrence bounds, treating -1 as unbounded
// (unbounded × anything non-zero = unbounded; anything × 0 = 0).
func mulOccurs(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	if a == -1 || b == -1 {
		return -1
	}
	return a * b
}

// occurrenceValidRestriction reports whether the occurrence range [rMin,rMax] is
// a valid restriction of [bMin,bMax]: rMin >= bMin and rMax <= bMax (with -1
// meaning unbounded for the maxima).
func occurrenceValidRestriction(rMin, rMax, bMin, bMax int) bool {
	if rMin < bMin {
		return false
	}
	if bMax == -1 {
		return true // base unbounded — any derived max is within range
	}
	if rMax == -1 {
		return false // derived unbounded but base bounded — widening
	}
	return rMax <= bMax
}

// particleEmptiable reports whether a particle can be satisfied by zero
// occurrences of its content (minOccurs 0, or a group whose minimum effective
// content is empty).
func particleEmptiable(p *Particle) bool {
	if p.MinOccurs == 0 {
		return true
	}
	mg, ok := p.Term.(*ModelGroup)
	if !ok {
		return false
	}
	switch mg.Compositor {
	case CompositorChoice:
		// A choice is emptiable if ANY branch is emptiable.
		return slices.ContainsFunc(mg.Particles, particleEmptiable)
	default:
		// sequence/all: emptiable only if EVERY member is emptiable.
		for _, child := range mg.Particles {
			if !particleEmptiable(child) {
				return false
			}
		}
		return true
	}
}

// wildcardConstraintSubset reports whether the namespace constraint of sub is a
// subset of that of super, per XSD §3.10.6 ("Wildcard Subset"). It handles the
// negation constraints (##any/##other/##not-absent) that the union-oriented
// wildcardNSSubset helper folds into empty sets, so a derived ##other is not
// wrongly treated as a subset of a finite set. The model used:
//   - ##any           = the universe (matches every namespace incl. absent)
//   - ##other         = universe minus {targetNS, absent}
//   - ##not-absent    = universe minus {absent}
//   - finite set      = the listed URIs (##local→absent, ##targetNamespace→TNS)
//
// sub ⊆ super when every namespace sub admits is also admitted by super.
func wildcardConstraintSubset(sub, super *Wildcard) bool {
	// super = ##any admits everything.
	if super.Namespace == WildcardNSAny {
		return true
	}
	// super is not ##any, so sub = ##any cannot be a subset.
	if sub.Namespace == WildcardNSAny {
		return false
	}

	subNeg := sub.Namespace == WildcardNSOther || sub.Namespace == WildcardNSNotAbsent
	superNeg := super.Namespace == WildcardNSOther || super.Namespace == WildcardNSNotAbsent

	// Both negations: sub ⊆ super iff super's excluded set ⊆ sub's excluded set
	// (excluding more shrinks the admitted set). ##not-absent excludes only
	// {absent}; ##other excludes {absent, targetNS}.
	if subNeg && superNeg {
		superExcl := wildcardExcludedSet(super)
		subExcl := wildcardExcludedSet(sub)
		for ns := range superExcl {
			if !subExcl[ns] {
				return false
			}
		}
		return true
	}

	// sub is a negation but super is a finite set: a negation admits infinitely
	// many namespaces, a finite set admits finitely many, so it cannot be a
	// subset.
	if subNeg {
		return false
	}

	// sub is a finite set, super is a negation: every URI sub admits must be
	// admitted by super, i.e. not in super's excluded set.
	if superNeg {
		superExcl := wildcardExcludedSet(super)
		for ns := range wildcardNSSet(sub) {
			if superExcl[ns] {
				return false
			}
		}
		return true
	}

	// Both finite sets: ordinary subset.
	superSet := wildcardNSSet(super)
	for ns := range wildcardNSSet(sub) {
		if !superSet[ns] {
			return false
		}
	}
	return true
}

// wildcardExcludedSet returns the namespaces a negation wildcard
// (##other/##not-absent) does NOT admit. The empty namespace is represented by
// "".
func wildcardExcludedSet(wc *Wildcard) map[string]bool {
	switch wc.Namespace {
	case WildcardNSNotAbsent:
		return map[string]bool{"": true}
	case WildcardNSOther:
		return map[string]bool{"": true, wc.TargetNS: true}
	default:
		return map[string]bool{}
	}
}

// wildcardAllowsName reports whether the wildcard's namespace constraint admits
// an element with the given expanded name. Used for the element-restricts-
// wildcard (NSCompat) case. It delegates to the validator's wildcardMatches so
// the restriction check and instance validation share one definition of which
// names a wildcard admits — in particular ##other excludes the ABSENT namespace
// (an element with no namespace is not matched by ##other), which the validator
// enforces and the restriction check must mirror to avoid false-accepting an
// empty-namespace element as a restriction of an ##other wildcard.
func wildcardAllowsName(wc *Wildcard, name QName) bool {
	return wildcardMatches(wc, name.NS)
}
