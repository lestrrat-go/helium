package xsd

import "context"

// XSD 1.1 occurrence-counting subsumption of a base xs:all model group.
//
// In XSD 1.1 the members of an xs:all may carry occurrence ranges (minOccurs /
// maxOccurs > 1), and a derived content model restricting such a base no longer
// maps 1:1 to base members: several derived particles (e.g. two substitution
// group members of one base element, or one element repeated across an ordered
// sequence) may collectively restrict a single base member, and their COMBINED
// occurrence range must lie within that base member's range. recurseAll's
// distinct-mapping is therefore insufficient; allRestrictsByCounting computes,
// PER BASE MEMBER, the occurrence contribution of the derived side and checks it
// against the base member's range. It handles a derived xs:all, xs:sequence, or
// xs:choice (the latter two restricting a base all per the RecurseUnordered / map
// rules), including nested choices/sequences, via a recursive contribution walk
// that maps each derived element to a base member as it goes — so the branches of
// a choice correlate on the base member rather than on the derived element name
// (e.g. choice(A1{2},A2{2}) with A1,A2 in base member a's substitution group
// restricts base a{2}). Wildcards are NOT handled here — those route to
// allRestrictsWithWildcards.

// memberRange is an occurrence contribution to a single base xs:all member: how
// few (min) and how many (max, Unbounded for unbounded) matching children the
// derived side can emit for that member.
type memberRange struct {
	min int
	max int
}

// occursAdd adds two occurrence counts, propagating Unbounded (-1).
func occursAdd(a, b int) int {
	if a == Unbounded || b == Unbounded {
		return Unbounded
	}
	return a + b
}

// occursMul multiplies an occurrence count by a repetition factor, propagating
// Unbounded. Zero on either side yields zero (no contribution).
func occursMul(a, factor int) int {
	if a == 0 || factor == 0 {
		return 0
	}
	if a == Unbounded || factor == Unbounded {
		return Unbounded
	}
	return a * factor
}

// occursMax returns the larger of two occurrence counts (Unbounded dominates).
func occursMax(a, b int) int {
	if a == Unbounded || b == Unbounded {
		return Unbounded
	}
	if a > b {
		return a
	}
	return b
}

// occursMin returns the smaller of two occurrence counts (a bounded value is
// always smaller than Unbounded).
func occursMin(a, b int) int {
	if a == Unbounded {
		return b
	}
	if b == Unbounded {
		return a
	}
	if a < b {
		return a
	}
	return b
}

// memberContributions walks a derived particle and returns, per BASE xs:all
// member (indexed like baseElems), the occurrence range the derived side can
// contribute. It returns ok=false when the derived side contains an EMITTING
// wildcard (route to the wildcard-aware path), names an element no base member
// admits, or maps an element to a base member whose type it is not derived from.
// A non-emitting particle (prohibited, or an empty group) contributes nothing.
func memberContributions(ctx context.Context, p *Particle, baseElems []*Particle, schema *Schema, version Version) ([]memberRange, bool) {
	// A non-emitting particle (maxOccurs=0, or a group with no emitting content)
	// contributes nothing and must be checked BEFORE rejecting a wildcard term, so
	// a prohibited wildcard (e.g. <xs:any maxOccurs="0"/>) does not falsely abort.
	if particleEmitsNothing(p) {
		return make([]memberRange, len(baseElems)), true
	}
	switch t := p.Term.(type) {
	case *ElementDecl:
		bi, member := findBaseAllMember(t, baseElems, schema)
		if bi < 0 {
			return nil, false
		}
		// NameAndTypeOK against the ACTUAL constraining declaration (the base member
		// for a direct match, or the matched global substitution member otherwise) —
		// type derivation, nillable, and fixed value.
		if !derivedElemNameAndTypeOK(ctx, t, member, schema, version) {
			return nil, false
		}
		vec := make([]memberRange, len(baseElems))
		vec[bi] = memberRange{min: p.MinOccurs, max: p.MaxOccurs}
		return vec, true
	case *Wildcard:
		return nil, false
	case *ModelGroup:
		body, ok := groupBodyContributions(ctx, t, baseElems, schema, version)
		if !ok {
			return nil, false
		}
		// Scale by the group's OWN occurrence range (a nested group's occurrence
		// folds into its members' contributions).
		for i := range body {
			body[i] = memberRange{
				min: occursMul(body[i].min, p.MinOccurs),
				max: occursMul(body[i].max, p.MaxOccurs),
			}
		}
		return body, true
	}
	return make([]memberRange, len(baseElems)), true
}

// groupBodyContributions computes the per-base-member contribution of a model
// group's BODY (its members combined per compositor) WITHOUT folding the group's
// OWN occurrence range — the caller scales it (a nested group via
// memberContributions, the root derived group not at all, since
// groupRestrictsGroup already checks the root occurrence range separately, so
// folding it again would double-count and reject e.g. an identical restriction of
// an optional `xs:all minOccurs="0"`).
func groupBodyContributions(ctx context.Context, mg *ModelGroup, baseElems []*Particle, schema *Schema, version Version) ([]memberRange, bool) {
	childVecs := make([][]memberRange, 0, len(mg.Particles))
	for _, cp := range mg.Particles {
		cv, ok := memberContributions(ctx, cp, baseElems, schema, version)
		if !ok {
			return nil, false
		}
		childVecs = append(childVecs, cv)
	}
	if mg.Compositor == CompositorChoice {
		return combineChoiceVecs(childVecs, len(baseElems)), true
	}
	// sequence and all combine the same way: each member's content is emitted
	// independently, so contributions sum.
	return combineSeqVecs(childVecs, len(baseElems)), true
}

// derivedElemNameAndTypeOK checks the NameAndTypeOK constraints (XSD §3.9.6) of a
// derived element against the actual constraining base declaration, IGNORING
// occurrence (which is accounted separately by summing): type derivation, no
// nillable widening, and fixed-value equality. Mirrors elementRestrictsElement
// minus the name/occurrence checks.
func derivedElemNameAndTypeOK(ctx context.Context, rt, constraining *ElementDecl, schema *Schema, version Version) bool {
	// Compare EFFECTIVE types, not the raw ElementDecl.Type pointers: a derived or
	// constraining declaration with no explicit @type (e.g. a substitution-group
	// member that inherits its head's type) has a nil Type, which would otherwise
	// SKIP the derivation check and false-accept. effectiveDeclType resolves the
	// type through the substitution-group chain.
	rtType := effectiveDeclType(rt, schema)
	conType := effectiveDeclType(constraining, schema)
	// The derived element's type must validly RESTRICT the constraining base
	// member's type — the SAME NameAndTypeOK gate the direct element:element path
	// (elementRestrictsElement) applies: built-in-aware derived-from, no EXTENSION
	// step (clause 3.2.5.2), and no base-type-@block-forbidden derivation
	// (cvc-elt.4.3). Without this, an xs:all element retyped to an extension-derived
	// type — rejected as an xs:sequence restriction — would compile.
	if !elementTypeValidlyRestricts(rtType, conType, version) {
		return false
	}
	if !constraining.Nillable && rt.Nillable {
		return false
	}
	if constraining.Fixed != nil {
		if rt.Fixed == nil {
			return false
		}
		if !fixedValueMatches(ctx, *rt.Fixed, *constraining.Fixed, rtType, rt.FixedNS, constraining.FixedNS, schema, version) {
			return false
		}
	}
	return true
}

// combineSeqVecs sums, per base member, the contributions of a sequence/all
// group's members (each member's content is emitted).
func combineSeqVecs(childVecs [][]memberRange, n int) []memberRange {
	out := make([]memberRange, n)
	for _, cv := range childVecs {
		for i := range n {
			out[i] = memberRange{min: occursAdd(out[i].min, cv[i].min), max: occursAdd(out[i].max, cv[i].max)}
		}
	}
	return out
}

// combineChoiceVecs merges, per base member, the contributions of a choice's
// branches: at most one branch is selected per repetition, so a member's MAX is
// the largest branch max and its MIN is the smallest branch min (a branch that
// does not produce the member contributes min 0, so the choice guarantees the
// member only if EVERY branch produces it). Correlating on the base member (not
// the derived element name) keeps alternative substitution-group members of one
// base member from being summed.
func combineChoiceVecs(childVecs [][]memberRange, n int) []memberRange {
	out := make([]memberRange, n)
	for i := range n {
		maxOcc := 0
		minOcc := Unbounded // sentinel for "no branch seen yet"
		for _, cv := range childVecs {
			maxOcc = occursMax(maxOcc, cv[i].max)
			minOcc = occursMin(minOcc, cv[i].min)
		}
		if minOcc == Unbounded {
			minOcc = 0
		}
		out[i] = memberRange{min: minOcc, max: maxOcc}
	}
	return out
}

// flattenAllParticles expands a particle list, recursively inlining the members
// of any nested 1/1 xs:all group (an xs:group ref that resolved to an all group),
// so element and wildcard members reached through a nested all are accounted for
// alongside the direct members. Non-all groups and groups with a non-1/1
// occurrence are left intact.
func flattenAllParticles(particles []*Particle) []*Particle {
	var out []*Particle
	for _, p := range particles {
		if mg, ok := p.Term.(*ModelGroup); ok &&
			mg.Compositor == CompositorAll && p.MinOccurs == 1 && p.MaxOccurs == 1 {
			out = append(out, flattenAllParticles(mg.Particles)...)
			continue
		}
		out = append(out, p)
	}
	return out
}

// flattenBaseAllElements returns the element particles of a base xs:all,
// flattening a nested 1/1 all-group (an xs:group ref that resolved to an all).
// hasWildcard reports whether any wildcard member was found, signalling the
// caller to use the wildcard-aware subsumption path instead.
func flattenBaseAllElements(mg *ModelGroup) ([]*Particle, bool) {
	var elems []*Particle
	hasWildcard := false
	for _, p := range mg.Particles {
		switch t := p.Term.(type) {
		case *ElementDecl:
			elems = append(elems, p)
		case *Wildcard:
			hasWildcard = true
		case *ModelGroup:
			if t.Compositor == CompositorAll && p.MinOccurs == 1 && p.MaxOccurs == 1 {
				sub, wc := flattenBaseAllElements(t)
				elems = append(elems, sub...)
				hasWildcard = hasWildcard || wc
			} else {
				hasWildcard = true
			}
		}
	}
	return elems, hasWildcard
}

// findBaseAllMember returns the index of the base xs:all element member a derived
// element maps to AND the actual CONSTRAINING declaration to validate against: a
// direct expanded-name match to a NON-ABSTRACT base member (an abstract base head
// admits no direct instance, only its substitutes, so a direct match to it is
// rejected — the constraining decl is the base member itself), or membership in a
// base element's substitution group (XSD 1.1 lets a derived all restrict a base
// element to its substitution-group members — the constraining decl is the matched
// GLOBAL member, so the derived element is checked against the member's own
// type/fixed/nillable, not the head's), honoring block="substitution", a blocked
// derivation step, and skipping abstract members (never instance-valid). Returns
// (-1, nil) if none admits it.
func findBaseAllMember(derived *ElementDecl, baseElems []*Particle, schema *Schema) (int, *ElementDecl) {
	for i, bp := range baseElems {
		if bd, ok := bp.Term.(*ElementDecl); ok && bd.Name == derived.Name && !bd.Abstract {
			return i, bd
		}
	}
	for i, bp := range baseElems {
		bd, ok := bp.Term.(*ElementDecl)
		if !ok {
			continue
		}
		// Use the TRANSITIVE instance-admissible closure so subsumption agrees with
		// the runtime matcher on which members can substitute for the head — incl.
		// multi-level chains (h<-m1<-m2) a direct substGroups lookup would miss.
		for _, m := range instanceSubstMembers(bd, schema) {
			if m.Name == derived.Name {
				return i, m
			}
		}
	}
	return -1, nil
}

// allRestrictsByCounting reports whether the derived particle is a valid XSD 1.1
// restriction of the base xs:all by occurrence counting. Each derived element
// must map to a base member (by name or substitution group) whose type it is
// derived from; the combined contribution per base member must lie within that
// member's occurrence range; and every base member with no derived contribution
// must be emptiable.
func allRestrictsByCounting(ctx context.Context, derived *Particle, baseAll *ModelGroup, schema *Schema, version Version) bool {
	baseElems, hasWildcard := flattenBaseAllElements(baseAll)
	if hasWildcard {
		return false
	}
	// Compute the derived side's per-base-member contribution. For a derived GROUP
	// use its BODY contribution (NOT scaled by the root group's own occurrence —
	// groupRestrictsGroup already checked that range separately, so folding it here
	// would double-count); a bare element/particle uses memberContributions.
	var vec []memberRange
	var ok bool
	if mg, isGroup := derived.Term.(*ModelGroup); isGroup {
		vec, ok = groupBodyContributions(ctx, mg, baseElems, schema, version)
	} else {
		vec, ok = memberContributions(ctx, derived, baseElems, schema, version)
	}
	if !ok {
		return false
	}
	for i, bp := range baseElems {
		// An unmapped base member (range 0,0) is valid only if it is emptiable.
		if !occurrenceValidRestriction(vec[i].min, vec[i].max, bp.MinOccurs, bp.MaxOccurs) {
			return false
		}
	}
	return true
}
