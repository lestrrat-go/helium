package xsd

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// matchSequence matches children[pos:] against a sequence model group.
// Returns (consumed, error). Does NOT check for leftover children.
//
// The greedy matching approach assumes UPA-compliant (deterministic) content
// models, which is enforced at compile time by checkUPA in parse_check.go.
func matchSequence(parent *helium.Element, mg *ModelGroup, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	startPos := pos

	tryOnce := func(p int) (int, error) {
		return tryMatchSequenceOnce(mg, children, p, schema)
	}

	hasWildcard := sequenceHasWildcard(mg)

	matchOnce := func(p int) (int, error) {
		cur := p
		var contentErr error
		for _, particle := range mg.Particles {
			consumed, e := matchParticle(parent, particle, children, cur, schema, filename, out, hasWildcard)
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

	var contentErr error

	matchOnce := func(p int) (int, bool) {
		// First find a structurally matching particle.
		for _, particle := range mg.Particles {
			consumed, err := tryMatchParticle(particle, children, p, schema)
			if err == nil && consumed > 0 {
				// Now validate matched content with error reporting.
				actualConsumed, actualErr := matchParticle(parent, particle, children, p, schema, filename, out, false)
				if actualErr != nil {
					contentErr = actualErr
				}
				return actualConsumed, true
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
		names := particleNames(mg.Particles, schema)
		msg := formatExpected("Missing child element(s).", names)
		out.WriteString(validityError(filename, parent.Line(), elemDisplayName(parent), msg))
		return pos - startPos, fmt.Errorf("missing")
	}

	return pos - startPos, contentErr
}

// matchAll matches children[pos:] against an all model group.
// Returns (consumed, error). Does NOT check for leftover children.
func matchAll(parent *helium.Element, mg *ModelGroup, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	seen := make([]bool, len(mg.Particles))
	nameToIdx := make(map[QName]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name] = i
			// Also register substitution group members.
			for _, member := range schema.substGroups[ed.Name] {
				nameToIdx[member.Name] = i
			}
		}
	}

	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[QName{Local: child.name, NS: child.ns}]
		if !ok {
			// Try without namespace for unqualified declarations.
			idx, ok = nameToIdx[QName{Local: child.name}]
		}
		if !ok {
			// Unknown child in <all>.
			expected := unseenParticleNames(mg.Particles, seen, schema)
			msg := "This element is not expected."
			if len(expected) > 0 {
				msg = formatExpected("This element is not expected.", expected)
			}
			out.WriteString(validityError(filename, child.elem.Line(), child.displayName, msg))
			return consumed, fmt.Errorf("unexpected element")
		}
		if seen[idx] {
			// Duplicate — stop matching and report error.
			expected := unseenParticleNames(mg.Particles, seen, schema)
			msg := "This element is not expected."
			if len(expected) > 0 {
				msg = formatExpected("This element is not expected.", expected)
			}
			out.WriteString(validityError(filename, child.elem.Line(), child.displayName, msg))
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
		unseen := unseenParticleNames(mg.Particles, seen, schema)
		msg := formatExpected("Missing child element(s).", unseen)
		out.WriteString(validityError(filename, parent.Line(), elemDisplayName(parent), msg))
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
		out.WriteString(validityError(filename, ce.elem.Line(), ce.displayName, "This element is not expected."))
		return fmt.Errorf("unexpected element")
	}

	return nil
}

// matchParticle matches a particle against children[pos:], returning how many
// children were consumed. On failure, writes an error and returns an error.
func matchParticle(parent *helium.Element, p *Particle, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder, seqHasWildcard bool) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return matchElementParticle(parent, p, term, children, pos, schema, filename, out, seqHasWildcard)
	case *ModelGroup:
		switch term.Compositor {
		case CompositorSequence:
			return matchSequence(parent, term, children, pos, schema, filename, out)
		case CompositorChoice:
			return matchChoice(parent, term, children, pos, schema, filename, out)
		case CompositorAll:
			return matchAll(parent, term, children, pos, schema, filename, out)
		}
	case *Wildcard:
		return matchWildcardParticle(parent, p, term, children, pos, schema, filename, out)
	}
	return 0, nil
}

// matchElementParticle matches an element particle.
func matchElementParticle(parent *helium.Element, p *Particle, edecl *ElementDecl, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder, seqHasWildcard bool) (int, error) {
	count := 0
	for pos+count < len(children) && elemMatchesDeclOrSubst(children[pos+count], edecl, schema) {
		count++
		if p.MaxOccurs != Unbounded && count >= p.MaxOccurs {
			break
		}
	}

	if count < p.MinOccurs {
		expectedNames := elementExpectedNamesWithSubst(edecl, schema)
		if pos+count < len(children) {
			// There IS a child but it doesn't match — "This element is not expected."
			child := children[pos+count]
			msg := formatExpected("This element is not expected.", expectedNames)
			out.WriteString(validityError(filename, child.elem.Line(), child.displayName, msg))
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
			out.WriteString(validityError(filename, parent.Line(), elemDisplayName(parent), msg))
		}
		return count, fmt.Errorf("missing")
	}

	// Validate each matched child element's own content model.
	// Continue after value/content errors so all errors are reported.
	// For substitution group members, use the member's declaration (type + default/fixed).
	// xsi:type overrides the declared type for polymorphism.
	var contentErr error
	for i := 0; i < count; i++ {
		child := children[pos+i]
		actualDecl := resolveSubstDecl(child, edecl, schema)
		td := actualDecl.Type
		td, xsiErr := resolveXsiType(child.elem, td, schema, filename, out)
		if xsiErr != nil {
			contentErr = xsiErr
			continue
		}
		// Check block flags against xsi:type derivation.
		if td != actualDecl.Type && actualDecl.Type != nil && isDerivationBlocked(td, actualDecl.Type, actualDecl.Block) {
			msg := "The xsi:type definition is blocked by the element declaration."
			out.WriteString(validityError(filename, child.elem.Line(), elemDisplayName(child.elem), msg))
			contentErr = fmt.Errorf("blocked xsi:type")
			continue
		}
		if td != nil && td.Abstract {
			msg := "The type definition is abstract."
			out.WriteString(validityError(filename, child.elem.Line(), elemDisplayName(child.elem), msg))
			contentErr = fmt.Errorf("abstract type")
			continue
		}
		if td != nil {
			if hasXsiNil(child.elem) {
				if err := validateNilledElement(child.elem, actualDecl, td, schema, filename, out); err != nil {
					contentErr = err
				}
			} else if err := validateElementContent(child.elem, actualDecl, td, schema, filename, out); err != nil {
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
		for _, member := range schema.substGroups[edecl.Name] {
			if matchesDeclDirect(child, member) {
				return member
			}
		}
	}
	return edecl
}

// tryMatchParticle is like matchParticle but does not write errors.
func tryMatchParticle(p *Particle, children []childElem, pos int, schema *Schema) (int, error) {
	switch term := p.Term.(type) {
	case *ElementDecl:
		return tryMatchElementParticle(p, term, children, pos, schema)
	case *ModelGroup:
		return tryMatchModelGroup(term, children, pos, schema)
	case *Wildcard:
		return tryMatchWildcardParticle(p, term, children, pos, schema)
	}
	return 0, nil
}

func tryMatchElementParticle(p *Particle, edecl *ElementDecl, children []childElem, pos int, schema *Schema) (int, error) {
	count := 0
	for pos+count < len(children) && elemMatchesDeclOrSubst(children[pos+count], edecl, schema) {
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
	nameToIdx := make(map[QName]int, len(mg.Particles))
	for i, p := range mg.Particles {
		if ed, ok := p.Term.(*ElementDecl); ok {
			nameToIdx[ed.Name] = i
			for _, member := range schema.substGroups[ed.Name] {
				nameToIdx[member.Name] = i
			}
		}
	}
	consumed := 0
	for pos+consumed < len(children) {
		child := children[pos+consumed]
		idx, ok := nameToIdx[QName{Local: child.name, NS: child.ns}]
		if !ok {
			idx, ok = nameToIdx[QName{Local: child.name}]
		}
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

// matchWildcardParticle matches a wildcard particle against children.
func matchWildcardParticle(parent *helium.Element, p *Particle, wc *Wildcard, children []childElem, pos int, schema *Schema, filename string, out *strings.Builder) (int, error) {
	count := 0
	for pos+count < len(children) {
		child := children[pos+count]
		if !wildcardMatches(wc, child.elem.URI()) {
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
			out.WriteString(validityError(filename, children[pos].elem.Line(), children[pos].displayName, msg))
		} else {
			out.WriteString(validityError(filename, parent.Line(), elemDisplayName(parent), msg))
		}
		return count, fmt.Errorf("wildcard not matched")
	}

	// Validate matched elements per processContents.
	if wc.ProcessContents != ProcessSkip {
		var contentErr error
		for i := 0; i < count; i++ {
			child := children[pos+i]
			edecl := lookupElemDecl(child.elem, schema)
			if edecl == nil {
				if wc.ProcessContents == ProcessStrict {
					msg := "No matching global declaration available, but demanded by the strict wildcard."
					out.WriteString(validityError(filename, child.elem.Line(), child.displayName, msg))
					contentErr = fmt.Errorf("strict wildcard: no global element decl")
				}
				continue
			}
			td := edecl.Type
			td, xsiErr := resolveXsiType(child.elem, td, schema, filename, out)
			if xsiErr != nil {
				contentErr = xsiErr
				continue
			}
			if td != nil && td.Abstract {
				msg := "The type definition is abstract."
				out.WriteString(validityError(filename, child.elem.Line(), elemDisplayName(child.elem), msg))
				contentErr = fmt.Errorf("abstract type")
				continue
			}
			if td != nil {
				if hasXsiNil(child.elem) {
					if err := validateNilledElement(child.elem, edecl, td, schema, filename, out); err != nil {
						contentErr = err
					}
				} else if err := validateElementContent(child.elem, edecl, td, schema, filename, out); err != nil {
					contentErr = err
				}
			}
		}
		if contentErr != nil {
			return count, contentErr
		}
	}

	return count, nil
}

// tryMatchWildcardParticle is the try version (no error reporting).
func tryMatchWildcardParticle(p *Particle, wc *Wildcard, children []childElem, pos int, _ *Schema) (int, error) {
	count := 0
	for pos+count < len(children) {
		child := children[pos+count]
		if !wildcardMatches(wc, child.elem.URI()) {
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

// wildcardMatches checks if an element namespace matches a wildcard constraint.
func wildcardMatches(wc *Wildcard, elemNS string) bool {
	ns := wc.Namespace
	switch ns {
	case "##any":
		return true
	case "##other":
		// Matches any namespace other than the target namespace.
		// Also does not match absent namespace (no namespace).
		return elemNS != "" && elemNS != wc.TargetNS
	case "##not-absent":
		// Matches any namespace except absent (empty namespace).
		return elemNS != ""
	default:
		// Space-separated list that may include ##local, ##targetNamespace, and URIs.
		for _, part := range strings.Split(ns, " ") {
			switch part {
			case "##local":
				if elemNS == "" {
					return true
				}
			case "##targetNamespace":
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

// wildcardExpected formats the expected string for wildcard error messages.
func wildcardExpected(wc *Wildcard) string {
	switch wc.Namespace {
	case "##any":
		return "##any"
	case "##other":
		if wc.TargetNS != "" {
			return "##other{" + wc.TargetNS + "}*"
		}
		return "##other*"
	default:
		return wc.Namespace
	}
}

// elemMatchesDeclOrSubst checks if a child element matches a declaration
// directly or via substitution group. schema may be nil for basic matching.
func elemMatchesDeclOrSubst(child childElem, edecl *ElementDecl, schema *Schema) bool {
	if matchesDeclDirect(child, edecl) && !edecl.Abstract {
		return true
	}
	// Check substitution group members.
	if schema != nil {
		// If block="substitution", skip all substitution group members.
		if edecl.Block&BlockSubstitution != 0 {
			return false
		}
		for _, member := range schema.substGroups[edecl.Name] {
			if matchesDeclDirect(child, member) {
				// Check if the derivation chain from member's type to head's type
				// uses a blocked method.
				if isDerivationBlocked(member.Type, edecl.Type, edecl.Block) {
					continue
				}
				return true
			}
		}
	}
	return false
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
	return false
}

func matchesDeclDirect(child childElem, edecl *ElementDecl) bool {
	if child.name != edecl.Name.Local {
		return false
	}
	if edecl.Name.NS != "" {
		return child.ns == edecl.Name.NS
	}
	return true
}

// elementDisplayForExpected formats an element declaration name for error messages.
func elementDisplayForExpected(edecl *ElementDecl) string {
	if edecl.Name.NS != "" {
		return "{" + edecl.Name.NS + "}" + edecl.Name.Local
	}
	return edecl.Name.Local
}

// elementExpectedNamesWithSubst returns the list of expected element names
// for a declaration, including substitution group members.
// The head element is always listed first (even if abstract), followed by members.
func elementExpectedNamesWithSubst(edecl *ElementDecl, schema *Schema) []string {
	members := schema.substGroups[edecl.Name]
	if len(members) == 0 {
		return []string{elementDisplayForExpected(edecl)}
	}
	names := []string{elementDisplayForExpected(edecl)}
	for _, m := range members {
		names = append(names, elementDisplayForExpected(m))
	}
	return names
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
