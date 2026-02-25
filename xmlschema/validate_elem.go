package xmlschema

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// matchSequence matches children[pos:] against a sequence model group.
// Returns (consumed, error). Does NOT check for leftover children.
func matchSequence(parent *helium.Element, mg *ModelGroup, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	startPos := pos

	tryOnce := func(p int) (int, error) {
		return tryMatchSequenceOnce(mg, children, p, schema)
	}

	matchOnce := func(p int) (int, error) {
		cur := p
		for _, particle := range mg.Particles {
			consumed, e := matchParticle(parent, particle, children, cur, schema, filename, out)
			if e != nil {
				return cur - p, e
			}
			cur += consumed
		}
		return cur - p, nil
	}

	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	reps := 0
	for {
		if maxReps != Unbounded && reps >= maxReps {
			break
		}
		// First try without side effects.
		tryCons, tryErr := tryOnce(pos)
		if tryErr != nil {
			if reps < minReps {
				// Must succeed — run with error reporting.
				_, e := matchOnce(pos)
				return pos - startPos, e
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
			return pos - startPos + consumed, e
		}
		pos += consumed
		reps++
	}

	return pos - startPos, nil
}

func tryMatchSequenceOnce(mg *ModelGroup, children []childElem, pos int, schema *Schema) (int, error) {
	cur := pos
	for _, p := range mg.Particles {
		consumed, err := tryMatchParticle(p, children, cur, schema)
		if err != nil {
			return 0, err
		}
		cur += consumed
	}
	return cur - pos, nil
}

// matchChoice matches children[pos:] against a choice model group.
// Returns (consumed, error). Does NOT check for leftover children.
func matchChoice(parent *helium.Element, mg *ModelGroup, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	startPos := pos

	minReps := mg.MinOccurs
	maxReps := mg.MaxOccurs

	matchOnce := func(p int) (int, bool) {
		for _, particle := range mg.Particles {
			consumed, err := tryMatchParticle(particle, children, p, schema)
			if err == nil && consumed > 0 {
				return consumed, true
			}
		}
		// Try zero-length matches.
		for _, particle := range mg.Particles {
			consumed, err := tryMatchParticle(particle, children, p, schema)
			if err == nil && consumed == 0 {
				return consumed, true
			}
		}
		return 0, false
	}

	reps := 0
	for {
		if maxReps != Unbounded && reps >= maxReps {
			break
		}
		consumed, ok := matchOnce(pos)
		if !ok {
			break
		}
		reps++
		pos += consumed
		if consumed == 0 {
			break
		}
	}

	if reps < minReps {
		names := particleNames(mg.Particles)
		msg := formatExpected("Missing child element(s).", names)
		out.WriteString(validityError(filename, parent.Line(), parent.LocalName(), msg))
		return pos - startPos, fmt.Errorf("missing")
	}

	return pos - startPos, nil
}

// matchAll matches children[pos:] against an all model group.
// Returns (consumed, error). Does NOT check for leftover children.
func matchAll(parent *helium.Element, mg *ModelGroup, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	seen := make([]bool, len(mg.Particles))
	nameToIdx := make(map[string]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name.Local] = i
		}
	}

	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[child.name]
		if !ok {
			// Unknown child in <all>.
			var expected []string
			for i, p := range mg.Particles {
				if !seen[i] {
					if ed, ok2 := p.Term.(*ElementDecl); ok2 {
						expected = append(expected, ed.Name.Local)
					}
				}
			}
			msg := "This element is not expected."
			if len(expected) > 0 {
				msg = formatExpected("This element is not expected.", expected)
			}
			out.WriteString(validityError(filename, child.elem.Line(), child.name, msg))
			return consumed, fmt.Errorf("unexpected element")
		}
		if seen[idx] {
			// Duplicate — stop matching and report error.
			var expected []string
			for i, p := range mg.Particles {
				if !seen[i] {
					if ed, ok2 := p.Term.(*ElementDecl); ok2 {
						expected = append(expected, ed.Name.Local)
					}
				}
			}
			msg := "This element is not expected."
			if len(expected) > 0 {
				msg = formatExpected("This element is not expected.", expected)
			}
			out.WriteString(validityError(filename, child.elem.Line(), child.name, msg))
			return consumed, fmt.Errorf("duplicate")
		}
		seen[idx] = true
		consumed++
	}

	// Respect <all> group's minOccurs: if 0 and group is empty, skip required checks.
	if mg.MinOccurs == 0 && consumed == 0 {
		return 0, nil
	}

	// Check for required missing particles.
	hasRequired := false
	for i, p := range mg.Particles {
		if !seen[i] && p.MinOccurs > 0 {
			hasRequired = true
			break
		}
	}
	if hasRequired {
		var unseen []string
		for i, p := range mg.Particles {
			if !seen[i] {
				if ed, ok := p.Term.(*ElementDecl); ok {
					unseen = append(unseen, ed.Name.Local)
				}
			}
		}
		msg := formatExpected("Missing child element(s).", unseen)
		out.WriteString(validityError(filename, parent.Line(), parent.LocalName(), msg))
		return consumed, fmt.Errorf("missing")
	}

	return consumed, nil
}

// validateContentModelTop validates children against a model group, checking
// that ALL children are consumed. This is the top-level entry point.
func validateContentModelTop(parent *helium.Element, mg *ModelGroup, children []childElem, schema *Schema, filename string, out *strings.Builder) error {
	var consumed int
	var err error

	switch mg.Compositor {
	case CompositorSequence:
		consumed, err = matchSequence(parent, mg, children, 0, schema, filename, out)
	case CompositorChoice:
		consumed, err = matchChoice(parent, mg, children, 0, schema, filename, out)
	case CompositorAll:
		consumed, err = matchAll(parent, mg, children, 0, schema, filename, out)
	}

	if err != nil {
		return err
	}

	// Check for unconsumed children.
	if consumed < len(children) {
		ce := children[consumed]
		out.WriteString(validityError(filename, ce.elem.Line(), ce.name, "This element is not expected."))
		return fmt.Errorf("unexpected element")
	}

	return nil
}

// matchParticle matches a particle against children[pos:], returning how many
// children were consumed. On failure, writes an error and returns an error.
func matchParticle(parent *helium.Element, p *Particle, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return matchElementParticle(parent, p, term, children, pos, schema, filename, out)
	case *ModelGroup:
		switch term.Compositor {
		case CompositorSequence:
			return matchSequence(parent, term, children, pos, schema, filename, out)
		case CompositorChoice:
			return matchChoice(parent, term, children, pos, schema, filename, out)
		case CompositorAll:
			return matchAll(parent, term, children, pos, schema, filename, out)
		}
	}
	return 0, nil
}

// matchElementParticle matches an element particle.
func matchElementParticle(parent *helium.Element, p *Particle, edecl *ElementDecl, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	count := 0
	for pos+count < len(children) && children[pos+count].name == edecl.Name.Local {
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}

	if count < p.MinOccurs {
		msg := formatExpected("Missing child element(s).", []string{edecl.Name.Local})
		out.WriteString(validityError(filename, parent.Line(), parent.LocalName(), msg))
		return count, fmt.Errorf("missing")
	}

	// Validate each matched child element's own content model.
	for i := 0; i < count; i++ {
		child := children[pos+i]
		if edecl.Type != nil {
			if err := validateElementContent(child.elem, edecl.Type, schema, filename, out); err != nil {
				return i, err
			}
		}
	}

	return count, nil
}

// tryMatchParticle is like matchParticle but does not write errors.
func tryMatchParticle(p *Particle, children []childElem, pos int, schema *Schema) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return tryMatchElementParticle(p, term, children, pos)
	case *ModelGroup:
		return tryMatchModelGroup(term, children, pos, schema)
	}
	return 0, nil
}

func tryMatchElementParticle(p *Particle, edecl *ElementDecl, children []childElem, pos int) (int, error) {
	count := 0
	for pos+count < len(children) && children[pos+count].name == edecl.Name.Local {
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

func tryMatchModelGroup(mg *ModelGroup, children []childElem, pos int, schema *Schema) (int, error) {
	switch mg.Compositor {
	case CompositorSequence:
		return tryMatchSequence(mg, children, pos, schema)
	case CompositorChoice:
		return tryMatchChoice(mg, children, pos, schema)
	case CompositorAll:
		return tryMatchAll(mg, children, pos, schema)
	}
	return 0, fmt.Errorf("unknown compositor")
}

func tryMatchSequence(mg *ModelGroup, children []childElem, pos int, schema *Schema) (int, error) {
	cur := pos
	for _, p := range mg.Particles {
		consumed, err := tryMatchParticle(p, children, cur, schema)
		if err != nil {
			return 0, err
		}
		cur += consumed
	}
	return cur - pos, nil
}

func tryMatchChoice(mg *ModelGroup, children []childElem, pos int, schema *Schema) (int, error) {
	for _, p := range mg.Particles {
		consumed, err := tryMatchParticle(p, children, pos, schema)
		if err == nil {
			return consumed, nil
		}
	}
	return 0, fmt.Errorf("no choice matched")
}

func tryMatchAll(mg *ModelGroup, children []childElem, pos int, schema *Schema) (int, error) {
	seen := make([]bool, len(mg.Particles))
	nameToIdx := make(map[string]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name.Local] = i
		}
	}
	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[child.name]
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

func formatExpected(prefix string, names []string) string {
	if len(names) == 1 {
		return fmt.Sprintf("%s Expected is ( %s ).", prefix, names[0])
	}
	return fmt.Sprintf("%s Expected is one of ( %s ).", prefix, strings.Join(names, ", "))
}

func particleNames(particles []*Particle) []string {
	var names []string
	for _, p := range particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			names = append(names, ed.Name.Local)
		}
	}
	return names
}
