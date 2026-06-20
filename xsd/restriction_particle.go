package xsd

import (
	"context"

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

	derivedMG := td.ContentModel
	baseMG := base.ContentModel

	// Restricting to empty content (no derived model group) is always a valid
	// restriction, regardless of the base, provided the base content is
	// emptiable; that emptiability is enforced by the content-model validator at
	// instance time, so a missing derived model group is not flagged here.
	if derivedMG == nil {
		return
	}
	// If the base has no content model (e.g. the base is xs:anyType, empty, or
	// simple content), there is nothing to recurse against. A restriction of
	// xs:anyType to any content model is valid; an empty/simple base with a
	// derived element-content model is caught by other compile checks
	// (cos-ct-extends / content-type rules), so do not double-diagnose here.
	if baseMG == nil {
		return
	}
	// Restriction directly off the ur-type (xs:anyType) is unconstrained.
	if base.Name.Local == typeAnyType && base.Name.NS == lexicon.NamespaceXSD {
		return
	}

	// Wrap the top-level model groups as particles so the occurrence range of the
	// whole group participates in the check.
	derivedP := &Particle{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG}
	baseP := &Particle{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG}

	if particleValidRestriction(derivedP, baseP) {
		return
	}

	component := componentLocalComplexType
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}
	baseQualified := "'{" + base.Name.NS + "}" + base.Name.Local + "'"
	msg := "The content model is not a valid restriction of the content model of the base complex type definition " + baseQualified + "."
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
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
			// Recurse-As-If-Group: an element against a group. Conservatively
			// accept (do not attempt to prove containment in a group).
			return true
		}
	case *Wildcard:
		switch bt := b.Term.(type) {
		case *Wildcard:
			// NSSubset
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			return wildcardNSSubset(rt, bt)
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
			// NSRecurseCheckCardinality: every particle in the group must be a
			// valid restriction of the wildcard. Conservatively accept unless an
			// occurrence widening is obvious at the group level.
			return true
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
		return recurseOrdered(rg.Particles, bg.Particles)
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
	for bi := 0; bi < len(bParticles); bi++ {
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
		for _, child := range mg.Particles {
			if particleEmptiable(child) {
				return true
			}
		}
		return false
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

// wildcardAllowsName reports whether the wildcard's namespace constraint admits
// an element with the given expanded name. Used for the element-restricts-
// wildcard (NSCompat) case.
func wildcardAllowsName(wc *Wildcard, name QName) bool {
	switch wc.Namespace {
	case WildcardNSAny:
		return true
	case WildcardNSOther:
		return name.NS != wc.TargetNS
	case WildcardNSNotAbsent:
		return name.NS != ""
	default:
		for _, tok := range splitSpace(wc.Namespace) {
			switch tok {
			case WildcardNSLocal:
				if name.NS == "" {
					return true
				}
			case WildcardNSTargetNamespace:
				if name.NS == wc.TargetNS {
					return true
				}
			default:
				if name.NS == tok {
					return true
				}
			}
		}
		return false
	}
}
