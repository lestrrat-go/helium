package xsd

// XSD 1.1 occurrence-counting subsumption of a base xs:all model group.
//
// In XSD 1.1 the members of an xs:all may carry occurrence ranges (minOccurs /
// maxOccurs > 1), and a derived content model restricting such a base no longer
// maps 1:1 to base members: several derived particles (e.g. two substitution
// group members of one base element, or one element repeated across an ordered
// sequence) may collectively restrict a single base member, and their COMBINED
// occurrence range must lie within that base member's range. recurseAll's
// distinct-mapping is therefore insufficient; allRestrictsByCounting computes,
// per base member, the summed occurrence contribution of the derived side and
// checks it against the base member's range. It handles a derived xs:all,
// xs:sequence, or xs:choice (the latter two restricting a base all per the
// RecurseUnordered / map rules), including nested choices/sequences, via a
// recursive per-name contribution walk. Wildcards are NOT handled here — those
// route to allRestrictsWithWildcards.

// allContribution is the occurrence contribution of a derived content model to a
// single element name: how few (min) and how many (max, Unbounded for unbounded)
// matching children it can emit, plus the contributing element declaration for
// the type-derivation check.
type allContribution struct {
	min  int
	max  int
	decl *ElementDecl
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

// particleContributions walks a derived particle and returns, per element
// expanded name, the occurrence range it can contribute. It returns ok=false if
// the particle contains a wildcard (handled by the wildcard-aware path instead).
func particleContributions(p *Particle) (map[QName]allContribution, bool) {
	switch t := p.Term.(type) {
	case *ElementDecl:
		return map[QName]allContribution{
			t.Name: {min: p.MinOccurs, max: p.MaxOccurs, decl: t},
		}, true
	case *Wildcard:
		return nil, false
	case *ModelGroup:
		childMaps := make([]map[QName]allContribution, 0, len(t.Particles))
		for _, cp := range t.Particles {
			cm, ok := particleContributions(cp)
			if !ok {
				return nil, false
			}
			childMaps = append(childMaps, cm)
		}
		var combined map[QName]allContribution
		if t.Compositor == CompositorChoice {
			combined = combineChoiceContributions(childMaps)
		} else {
			// sequence and all combine the same way: each member's content is
			// emitted independently, so contributions sum.
			combined = combineSeqContributions(childMaps)
		}
		// Scale by the group's own occurrence range.
		for name, c := range combined {
			combined[name] = allContribution{
				min:  occursMul(c.min, p.MinOccurs),
				max:  occursMul(c.max, p.MaxOccurs),
				decl: c.decl,
			}
		}
		return combined, true
	}
	return map[QName]allContribution{}, true
}

// combineSeqContributions sums per-name contributions across the members of a
// sequence/all group (an absent name contributes nothing).
func combineSeqContributions(childMaps []map[QName]allContribution) map[QName]allContribution {
	result := make(map[QName]allContribution)
	for _, cm := range childMaps {
		for name, c := range cm {
			prev := result[name]
			result[name] = allContribution{
				min:  occursAdd(prev.min, c.min),
				max:  occursAdd(prev.max, c.max),
				decl: pickDecl(prev.decl, c.decl),
			}
		}
	}
	return result
}

// combineChoiceContributions merges per-name contributions across the branches
// of a choice: at most one branch is selected per repetition, so a name's MAX is
// the largest branch max and its MIN is zero unless the name appears (with a
// positive min) in EVERY branch.
func combineChoiceContributions(childMaps []map[QName]allContribution) map[QName]allContribution {
	names := make(map[QName]struct{})
	for _, cm := range childMaps {
		for name := range cm {
			names[name] = struct{}{}
		}
	}
	result := make(map[QName]allContribution)
	for name := range names {
		maxOcc := 0
		minOcc := Unbounded // sentinel for "not yet set"; reduced to branch minima
		var decl *ElementDecl
		for _, cm := range childMaps {
			c, present := cm[name]
			branchMin := 0
			branchMax := 0
			if present {
				branchMin = c.min
				branchMax = c.max
				decl = pickDecl(decl, c.decl)
			}
			maxOcc = occursMax(maxOcc, branchMax)
			minOcc = occursMin(minOcc, branchMin)
		}
		if minOcc == Unbounded {
			minOcc = 0
		}
		result[name] = allContribution{min: minOcc, max: maxOcc, decl: decl}
	}
	return result
}

func pickDecl(a, b *ElementDecl) *ElementDecl {
	if a != nil {
		return a
	}
	return b
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

// findBaseAllMember returns the index of the base xs:all element member that a
// derived element NAME maps to: a direct expanded-name match, or a name that is
// a member of a base element's substitution group (XSD 1.1 lets a derived all
// restrict a base element to its substitution-group members). Returns -1 if no
// base member admits the name.
func findBaseAllMember(name QName, baseElems []*Particle, schema *Schema) int {
	for i, bp := range baseElems {
		if bd, ok := bp.Term.(*ElementDecl); ok && bd.Name == name {
			return i
		}
	}
	for i, bp := range baseElems {
		bd, ok := bp.Term.(*ElementDecl)
		if !ok {
			continue
		}
		for _, m := range schema.substGroups[bd.Name] {
			if m.Name == name {
				return i
			}
		}
	}
	return -1
}

// allRestrictsByCounting reports whether the derived particle is a valid XSD 1.1
// restriction of the base xs:all by occurrence counting. Each derived element
// name must map to a base member (by name or substitution group) whose type it
// is derived from; the summed contribution per base member must lie within that
// member's occurrence range; and every base member with no derived contribution
// must be emptiable.
func allRestrictsByCounting(derived *Particle, baseAll *ModelGroup, schema *Schema) bool {
	contrib, ok := particleContributions(derived)
	if !ok {
		return false
	}
	baseElems, hasWildcard := flattenBaseAllElements(baseAll)
	if hasWildcard {
		return false
	}

	sumMin := make([]int, len(baseElems))
	sumMax := make([]int, len(baseElems))
	for name, c := range contrib {
		bi := findBaseAllMember(name, baseElems, schema)
		if bi < 0 {
			return false
		}
		bd, _ := baseElems[bi].Term.(*ElementDecl)
		// Type derivation: the derived element's type must be the same as, or
		// derived from, the base member's type. When either is unresolved, accept
		// conservatively (matching elementRestrictsElement).
		if c.decl != nil && c.decl.Type != nil && bd != nil && bd.Type != nil {
			if !isDerivedFrom(c.decl.Type, bd.Type) {
				return false
			}
		}
		sumMin[bi] = occursAdd(sumMin[bi], c.min)
		sumMax[bi] = occursAdd(sumMax[bi], c.max)
	}

	for i, bp := range baseElems {
		// An unmapped base member (sum 0) is valid only if it is emptiable.
		if !occurrenceValidRestriction(sumMin[i], sumMax[i], bp.MinOccurs, bp.MaxOccurs) {
			return false
		}
	}
	return true
}
