package xsd

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// checkGlobalElement validates constraints on a global xs:element declaration.
func (c *compiler) checkGlobalElement(elem *helium.Element) {
	if c.filename == "" {
		return
	}
	name := getAttr(elem, "name")
	line := elem.Line()
	local := elem.LocalName()

	// name is required for global elements.
	if name == "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attribute 'name' is required but missing."))
	}

	// ref is not allowed at global level.
	if getAttr(elem, "ref") != "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attribute 'ref' is not allowed."))
	}

	// minOccurs is not allowed at global level.
	if getAttr(elem, "minOccurs") != "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attribute 'minOccurs' is not allowed."))
	}

	// maxOccurs is not allowed at global level.
	if getAttr(elem, "maxOccurs") != "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attribute 'maxOccurs' is not allowed."))
	}

	// form is not allowed at global level.
	if getAttr(elem, "form") != "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attribute 'form' is not allowed."))
	}

	// Validate 'final' attribute value.
	if v := getAttr(elem, "final"); v != "" {
		if !isValidFinal(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "final",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction))'."))
		}
	}

	// Validate 'block' attribute value.
	if v := getAttr(elem, "block"); v != "" {
		if !isValidBlock(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}
	}

	// default and fixed are mutually exclusive.
	if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
			"The attributes 'default' and 'fixed' are mutually exclusive."))
	}

	// type and inline complexType/simpleType are mutually exclusive.
	if getAttr(elem, "type") != "" {
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "complexType") {
				c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <complexType> child are mutually exclusive."))
			}
			if isXSDElement(ce, "simpleType") {
				c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <simpleType> child are mutually exclusive."))
			}
		}
	}
}

// checkLocalElement validates constraints on a local xs:element declaration.
func (c *compiler) checkLocalElement(elem *helium.Element) {
	if c.filename == "" {
		return
	}
	ref := getAttr(elem, "ref")
	name := getAttr(elem, "name")
	line := elem.Line()
	local := elem.LocalName()

	minOcc := getAttr(elem, "minOccurs")
	maxOcc := getAttr(elem, "maxOccurs")

	if ref != "" {
		// Matches libxml2 ordering for ref elements (src-element 2.2):
		// 1. maxOccurs >= 1, minOccurs > maxOccurs
		// 2. ref+name conflict
		// 3. First ref-restricted attribute (alphabetical)
		// 4. First child content error

		// maxOccurs must be >= 1.
		if maxOcc != "" && maxOcc != "unbounded" {
			maxVal := parseOccurs(maxOcc, 1)
			if maxVal < 1 {
				c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."))
			}
		}

		// minOccurs > maxOccurs check.
		if minOcc != "" && maxOcc != "" && maxOcc != "unbounded" {
			minVal := parseOccurs(minOcc, 1)
			maxVal := parseOccurs(maxOcc, 1)
			if minVal > maxVal {
				c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value must not be greater than the value of 'maxOccurs'."))
			}
		}

		// ref and name are mutually exclusive.
		if name != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
				"The attributes 'ref' and 'name' are mutually exclusive."))
		}

		// Report first ref-restricted attribute found (alphabetical order).
		notAllowedWithRef := []string{"abstract", "block", "default", "final", "fixed", "form", "nillable", "substitutionGroup", "type"}
		for _, attr := range notAllowedWithRef {
			if getAttr(elem, attr) != "" {
				c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", attr,
					"Only the attributes 'minOccurs', 'maxOccurs' and 'id' are allowed in addition to 'ref'."))
				break // only report first
			}
		}

		// First child not allowed with ref (except annotation).
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "complexType") || isXSDElement(ce, "simpleType") {
				c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The content is not valid. Expected is (annotation?)."))
				break // only report first
			}
		}
	} else if name != "" {
		// Named local element constraints.
		// Matches libxml2 ordering: maxOccurs, not-allowed attrs,
		// block/final value checks, default+fixed, type/content children.

		// maxOccurs must be >= 1.
		if maxOcc != "" && maxOcc != "unbounded" {
			maxVal := parseOccurs(maxOcc, 1)
			if maxVal < 1 {
				c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."))
			}
		}

		// Some attributes not allowed for local named elements.
		localNotAllowed := []string{"abstract", "substitutionGroup", "final"}
		for _, attr := range localNotAllowed {
			if getAttr(elem, attr) != "" {
				c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
					"The attribute '"+attr+"' is not allowed."))
			}
		}

		// Validate 'block' attribute value.
		if v := getAttr(elem, "block"); v != "" && !isValidBlock(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}

		// default and fixed mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// type and inline complexType/simpleType checks.
		hasType := getAttr(elem, "type") != ""
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "complexType") {
				if hasType {
					c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The attribute 'type' and the <complexType> child are mutually exclusive."))
				}
			} else if isXSDElement(ce, "simpleType") {
				if hasType {
					c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."))
				}
			}
		}
	}
}

// checkAttributeUse validates constraints on an xs:attribute declaration.
func (c *compiler) checkAttributeUse(elem *helium.Element) {
	if c.filename == "" {
		return
	}
	ref := getAttr(elem, "ref")
	line := elem.Line()
	local := elem.LocalName()

	if ref != "" {
		// ref and name are mutually exclusive.
		if getAttr(elem, "name") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'name' is not allowed."))
		}

		// type not allowed with ref.
		if getAttr(elem, "type") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'type' is not allowed."))
		}

		// form not allowed with ref.
		if getAttr(elem, "form") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'form' is not allowed."))
		}

		// simpleType child not allowed with ref.
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "simpleType") {
				c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
					"The content is not valid. Expected is (annotation?)."))
			}
		}
	} else {
		// Attribute name must not be "xmlns".
		if getAttr(elem, "name") == "xmlns" {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "attribute", "name",
				"The value of the attribute must not match 'xmlns'."))
		}

		// Qualified attribute must not be in the XSI namespace.
		form := getAttr(elem, "form")
		if form == "qualified" || (form == "" && c.schema.attrFormQualified) {
			if c.schema.targetNamespace == "http://www.w3.org/2001/XMLSchema-instance" {
				c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
					"The target namespace must not match 'http://www.w3.org/2001/XMLSchema-instance'."))
			}
		}

		// default and fixed are mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// If default is present, use must be optional (or absent, which defaults to optional).
		if getAttr(elem, "default") != "" {
			use := getAttr(elem, "use")
			if use != "" && use != "optional" {
				c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "attribute",
					"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present."))
			}
		}

		// type and inline simpleType are mutually exclusive.
		if getAttr(elem, "type") != "" {
			for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
				if child.Type() != helium.ElementNode {
					continue
				}
				ce := child.(*helium.Element)
				if isXSDElement(ce, "simpleType") {
					c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"The attribute 'type' and the <simpleType> child are mutually exclusive."))
				}
			}
		}
	}
}

// checkAnnotation validates an xs:annotation element and its children.
func (c *compiler) checkAnnotation(elem *helium.Element) {
	if c.filename == "" {
		return
	}
	line := elem.Line()
	local := elem.LocalName()

	// Check for disallowed attributes on annotation (only id is allowed).
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		if attr.Prefix() != "" {
			continue // namespaced attributes are allowed
		}
		if name == "id" {
			continue
		}
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "annotation",
			"The attribute '"+name+"' is not allowed."))
	}

	// Check for invalid content (non-element children like text nodes).
	hasInvalidContent := false
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.TextNode {
			text := strings.TrimSpace(string(child.Content()))
			if text != "" {
				hasInvalidContent = true
				break
			}
		}
	}
	if hasInvalidContent {
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "annotation",
			"The content is not valid. Expected is (appinfo | documentation)*."))
	}

	// Check children (appinfo, documentation).
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		if isXSDElement(ce, "appinfo") {
			c.checkAppinfo(ce)
		} else if isXSDElement(ce, "documentation") {
			c.checkDocumentation(ce)
		}
	}
}

// checkAppinfo validates an xs:appinfo element.
func (c *compiler) checkAppinfo(elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()

	// Only "source" is allowed (no id).
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		if attr.Prefix() != "" {
			continue
		}
		if name == "source" {
			continue
		}
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "appinfo",
			"The attribute '"+name+"' is not allowed."))
	}
}

// checkDocumentation validates an xs:documentation element.
func (c *compiler) checkDocumentation(elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()

	// Only "source" and "xml:lang" are allowed (no id).
	// Check disallowed attributes first, then validate xml:lang value.
	var langValue string
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		prefix := attr.Prefix()
		if prefix != "" && prefix != "xml" {
			continue // other namespaced attributes are allowed
		}
		if prefix == "xml" && name == "lang" {
			langValue = string(attr.Content())
			continue
		}
		if name == "source" {
			continue
		}
		c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "documentation",
			"The attribute '"+name+"' is not allowed."))
	}

	// Validate xml:lang value after attribute checks.
	if langValue != "" && !languageRegex.MatchString(langValue) {
		c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "documentation",
			"{http://www.w3.org/XML/1998/namespace}lang",
			"'"+langValue+"' is not a valid value of the atomic type 'xs:language'."))
	}
}

// isValidFinal checks if a value is valid for the 'final' attribute.
func isValidFinal(v string) bool {
	if v == "#all" {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != "extension" && part != "restriction" {
			return false
		}
	}
	return true
}

// isValidBlock checks if a value is valid for the 'block' attribute.
func isValidBlock(v string) bool {
	if v == "#all" {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != "extension" && part != "restriction" && part != "substitution" {
			return false
		}
	}
	return true
}

// splitSpace splits a string on whitespace.
func splitSpace(s string) []string {
	var parts []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}

// checkUPA checks that a content model is deterministic (Unique Particle Attribution).
// A non-deterministic model means that when an element arrives, there's ambiguity
// about which particle it belongs to.
func (c *compiler) checkUPA(td *TypeDef, src typeDefSource) {
	if td.ContentModel == nil {
		return
	}
	if !modelGroupIsDeterministic(td.ContentModel, c.schema) {
		component := "local complex type"
		if !src.isLocal {
			component = td.Name.Local
		}
		c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component,
			"The content model is not determinist."))
	}
}

// modelGroupIsDeterministic checks if a content model satisfies UPA.
func modelGroupIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	switch mg.Compositor {
	case CompositorChoice:
		return choiceIsDeterministic(mg, schema)
	case CompositorSequence:
		return sequenceIsDeterministic(mg, schema)
	}
	return true
}

// choiceIsDeterministic checks that no two particles in a choice can match the same element.
func choiceIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	// Check each pair of particles for first-set overlap.
	for i := 0; i < len(mg.Particles); i++ {
		for j := i + 1; j < len(mg.Particles); j++ {
			if particleFirstSetsOverlap(mg.Particles[i], mg.Particles[j], schema) {
				return false
			}
		}
		// Recurse into nested model groups.
		if nested, ok := mg.Particles[i].Term.(*ModelGroup); ok {
			if !modelGroupIsDeterministic(nested, schema) {
				return false
			}
		}
	}
	return true
}

// sequenceIsDeterministic checks that adjacent particles in a sequence don't create ambiguity.
func sequenceIsDeterministic(mg *ModelGroup, schema *Schema) bool {
	for i := 0; i < len(mg.Particles); i++ {
		p := mg.Particles[i]
		// Recurse into nested model groups.
		if nested, ok := p.Term.(*ModelGroup); ok {
			if !modelGroupIsDeterministic(nested, schema) {
				return false
			}
		}
		if i+1 < len(mg.Particles) {
			next := mg.Particles[i+1]
			if canRepeatOrEnd(p) {
				// When current particle can repeat or end, check two overlaps:
				// 1. Current's first-set vs next's first-set (can a new element
				//    start either a new repetition of current OR the next particle?)
				if particleFirstSetsOverlap(p, next, schema) {
					return false
				}
				// 2. Current's last-set vs next's first-set.
				if particleLastFirstOverlap(p, next, schema) {
					return false
				}
			}
		}
	}
	return true
}

// canRepeatOrEnd returns true if the particle could match more or stop matching,
// creating ambiguity about whether the next element belongs to this particle or the next.
func canRepeatOrEnd(p *Particle) bool {
	if p.MaxOccurs == Unbounded {
		return true
	}
	if p.MinOccurs < p.MaxOccurs {
		return true
	}
	// Check nested model groups with variable repetition.
	if mg, ok := p.Term.(*ModelGroup); ok {
		if mg.MinOccurs < mg.MaxOccurs || mg.MaxOccurs == Unbounded {
			return true
		}
	}
	return false
}

// particleFirstSetsOverlap checks if two particles can both match the same starting element.
func particleFirstSetsOverlap(p1, p2 *Particle, schema *Schema) bool {
	first1 := particleFirstSet(p1, schema)
	first2 := particleFirstSet(p2, schema)
	return firstSetsOverlap(first1, first2)
}

// particleLastFirstOverlap checks if the last elements matchable by p1 overlap
// with the first elements matchable by p2.
func particleLastFirstOverlap(p1, p2 *Particle, schema *Schema) bool {
	last1 := particleLastSet(p1, schema)
	first2 := particleFirstSet(p2, schema)
	return firstSetsOverlap(last1, first2)
}

// firstSetEntry represents an element or wildcard in a first/last set.
type firstSetEntry struct {
	qname    QName  // for elements
	wildcard string // for wildcards (namespace constraint)
	targetNS string // for wildcards
}

// particleFirstSet returns the set of elements/wildcards that can appear first.
func particleFirstSet(p *Particle, schema *Schema) []firstSetEntry {
	switch term := p.Term.(type) {
	case *ElementDecl:
		entries := []firstSetEntry{{qname: term.Name}}
		for _, member := range schema.substGroups[term.Name] {
			entries = append(entries, firstSetEntry{qname: member.Name})
		}
		return entries
	case *Wildcard:
		return []firstSetEntry{{wildcard: term.Namespace, targetNS: term.TargetNS}}
	case *ModelGroup:
		return modelGroupFirstSet(term, schema)
	}
	return nil
}

// particleLastSet returns the set of elements/wildcards that can appear last.
func particleLastSet(p *Particle, schema *Schema) []firstSetEntry {
	switch term := p.Term.(type) {
	case *ElementDecl:
		entries := []firstSetEntry{{qname: term.Name}}
		for _, member := range schema.substGroups[term.Name] {
			entries = append(entries, firstSetEntry{qname: member.Name})
		}
		return entries
	case *Wildcard:
		return []firstSetEntry{{wildcard: term.Namespace, targetNS: term.TargetNS}}
	case *ModelGroup:
		return modelGroupLastSet(term, schema)
	}
	return nil
}

// modelGroupFirstSet returns the first set for a model group.
func modelGroupFirstSet(mg *ModelGroup, schema *Schema) []firstSetEntry {
	switch mg.Compositor {
	case CompositorSequence:
		// First set = first set of first non-optional particle
		// (union with subsequent particles if earlier ones are optional).
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
			if p.MinOccurs > 0 {
				break // this particle is required, stop here
			}
		}
		return result
	case CompositorChoice:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
		}
		return result
	case CompositorAll:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleFirstSet(p, schema)...)
		}
		return result
	}
	return nil
}

// modelGroupLastSet returns the last set for a model group.
func modelGroupLastSet(mg *ModelGroup, schema *Schema) []firstSetEntry {
	switch mg.Compositor {
	case CompositorSequence:
		// Last set = last set of last non-optional particle
		// (union with previous particles if later ones are optional).
		var result []firstSetEntry
		for i := len(mg.Particles) - 1; i >= 0; i-- {
			p := mg.Particles[i]
			result = append(result, particleLastSet(p, schema)...)
			if p.MinOccurs > 0 {
				break
			}
		}
		return result
	case CompositorChoice:
		var result []firstSetEntry
		for _, p := range mg.Particles {
			result = append(result, particleLastSet(p, schema)...)
		}
		return result
	}
	return nil
}

// firstSetsOverlap checks if two first-sets have any overlapping entries.
func firstSetsOverlap(a, b []firstSetEntry) bool {
	for _, ea := range a {
		for _, eb := range b {
			if entriesOverlap(ea, eb) {
				return true
			}
		}
	}
	return false
}

// entriesOverlap checks if two first-set entries can match the same element.
func entriesOverlap(a, b firstSetEntry) bool {
	// Two elements overlap if they have the same QName.
	if a.wildcard == "" && b.wildcard == "" {
		return a.qname == b.qname
	}
	// Element vs wildcard: check if the wildcard's namespace matches the element's namespace.
	if a.wildcard == "" && b.wildcard != "" {
		return wildcardMatchesNS(b.wildcard, b.targetNS, a.qname.NS)
	}
	if a.wildcard != "" && b.wildcard == "" {
		return wildcardMatchesNS(a.wildcard, a.targetNS, b.qname.NS)
	}
	// Two wildcards: check if their namespace constraints can both match the same namespace.
	return wildcardsOverlap(a, b)
}

// wildcardsOverlap checks if two wildcard entries can match the same namespace.
func wildcardsOverlap(a, b firstSetEntry) bool {
	aNS, aTarget := a.wildcard, a.targetNS
	bNS, bTarget := b.wildcard, b.targetNS

	// ##any overlaps with everything.
	if aNS == "##any" || bNS == "##any" {
		return true
	}

	// ##other matches any NS except targetNS and empty.
	// ##local matches only empty NS.
	// So ##other and ##local are always disjoint.
	if (aNS == "##other" && bNS == "##local") || (aNS == "##local" && bNS == "##other") {
		return false
	}

	// ##other vs ##other: overlap if there's any NS outside both targets
	// (practically always true unless both have the same target).
	if aNS == "##other" && bNS == "##other" {
		return true
	}

	// ##local vs ##local: obviously overlap.
	if aNS == "##local" && bNS == "##local" {
		return true
	}

	// For enumerated sets vs keywords, expand and check.
	aSet := upaWildcardNSSet(aNS, aTarget)
	bSet := upaWildcardNSSet(bNS, bTarget)

	// If either is nil, it means "any except target" — use conservative overlap.
	if aSet == nil || bSet == nil {
		return true
	}
	for ns := range aSet {
		if bSet[ns] {
			return true
		}
	}
	return false
}

// upaWildcardNSSet returns the explicit set of namespaces a wildcard can match,
// or nil if the wildcard matches an open-ended set (##other, ##any).
func upaWildcardNSSet(wcNS, targetNS string) map[string]bool {
	switch wcNS {
	case "##any", "##other", "##not-absent":
		return nil
	case "##local":
		return map[string]bool{"": true}
	default:
		result := make(map[string]bool)
		for _, part := range strings.Fields(wcNS) {
			switch part {
			case "##local":
				result[""] = true
			case "##targetNamespace":
				result[targetNS] = true
			default:
				result[part] = true
			}
		}
		return result
	}
}

// wildcardMatchesNS checks if a wildcard namespace constraint matches a given namespace.
func wildcardMatchesNS(wcNS, wcTargetNS, elemNS string) bool {
	switch wcNS {
	case "##any":
		return true
	case "##other":
		return elemNS != "" && elemNS != wcTargetNS
	case "##not-absent":
		return elemNS != ""
	default:
		for _, part := range strings.Fields(wcNS) {
			switch part {
			case "##local":
				if elemNS == "" {
					return true
				}
			case "##targetNamespace":
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

// compareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func compareDecimal(a, b string) int {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return -2
	}
	return ra.Cmp(rb)
}

// baseFacets returns the FacetSet from the nearest base type in the chain.
func baseFacets(td *TypeDef) *FacetSet {
	if td.BaseType == nil {
		return nil
	}
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		if cur.Facets != nil {
			return cur.Facets
		}
	}
	return nil
}

// checkFacetConsistency validates facet constraints for all named types.
// It checks same-type mutual exclusion, same-type consistency, and
// base-type restriction narrowing rules.
func (c *compiler) checkFacetConsistency() {
	if c.filename == "" {
		return
	}

	// Collect and sort types by name for deterministic error ordering.
	type facetEntry struct {
		qn QName
		td *TypeDef
	}
	var entries []facetEntry
	for qn, td := range c.schema.types {
		if td.Facets == nil {
			continue
		}
		if qn.NS == xsdNS {
			continue
		}
		entries = append(entries, facetEntry{qn: qn, td: td})
	}
	sort.Slice(entries, func(i, j int) bool {
		si, oki := c.typeDefSources[entries[i].td]
		sj, okj := c.typeDefSources[entries[j].td]
		if oki && okj {
			return si.line < sj.line
		}
		if oki != okj {
			return oki
		}
		return entries[i].qn.Local < entries[j].qn.Local
	})

	for _, entry := range entries {
		td := entry.td
		fs := td.Facets

		src, hasSrc := c.typeDefSources[td]
		component := td.Name.Local
		if component == "" {
			component = "local simple type"
		}
		line := 0
		if hasSrc {
			line = src.line
		}

		c.checkFacetMutualExclusion(fs, line, component)
		c.checkFacetSameTypeConsistency(fs, line, component)
		c.checkFacetBaseRestriction(td, fs, line, component)
	}
}

// checkFacetMutualExclusion checks that mutually exclusive facets are not
// both specified on the same type definition.
func (c *compiler) checkFacetMutualExclusion(fs *FacetSet, line int, component string) {
	if fs.Length != nil && (fs.MinLength != nil || fs.MaxLength != nil) {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'length' and either of 'minLength' or 'maxLength' to be specified on the same type definition."))
	}
	if fs.MaxInclusive != nil && fs.MaxExclusive != nil {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'maxInclusive' and 'maxExclusive' to be specified."))
	}
	if fs.MinInclusive != nil && fs.MinExclusive != nil {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'minInclusive' and 'minExclusive' to be specified."))
	}
}

// checkFacetSameTypeConsistency checks consistency of facets within the same type.
func (c *compiler) checkFacetSameTypeConsistency(fs *FacetSet, line int, component string) {
	if fs.MinLength != nil && fs.MaxLength != nil && *fs.MinLength > *fs.MaxLength {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'minLength' to be greater than the value of 'maxLength'."))
	}
	if fs.MinInclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxInclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'."))
		}
	}
	if fs.MinExclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxExclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxExclusive'."))
		}
	}
	if fs.FractionDigits != nil && fs.TotalDigits != nil && *fs.FractionDigits > *fs.TotalDigits {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'fractionDigits' to be greater than the value of 'totalDigits'."))
	}
	if fs.MinExclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxInclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxInclusive'."))
		}
	}
	if fs.MinInclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxExclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than or equal to the value of 'maxExclusive'."))
		}
	}
}

// checkFacetBaseRestriction checks that facet values properly narrow (not widen)
// the base type's facets.
func (c *compiler) checkFacetBaseRestriction(td *TypeDef, fs *FacetSet, line int, component string) {
	base := baseFacets(td)
	if base == nil {
		return
	}

	// Length facets.
	if fs.MinLength != nil && base.MinLength != nil && *fs.MinLength < *base.MinLength {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'minLength' value '%d' is less than the 'minLength' value of the base type '%d'.", *fs.MinLength, *base.MinLength)))
	}
	if fs.MaxLength != nil && base.MaxLength != nil && *fs.MaxLength > *base.MaxLength {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'maxLength' value '%d' is greater than the 'maxLength' value of the base type '%d'.", *fs.MaxLength, *base.MaxLength)))
	}
	if fs.Length != nil && base.Length != nil && *fs.Length != *base.Length {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'length' value '%d' does not match the 'length' value of the base type '%d'.", *fs.Length, *base.Length)))
	}

	// Digit facets.
	if fs.TotalDigits != nil && base.TotalDigits != nil && *fs.TotalDigits > *base.TotalDigits {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'totalDigits' value '%d' is greater than the 'totalDigits' value of the base type '%d'.", *fs.TotalDigits, *base.TotalDigits)))
	}
	if fs.FractionDigits != nil && base.FractionDigits != nil && *fs.FractionDigits > *base.FractionDigits {
		c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'fractionDigits' value '%d' is greater than the 'fractionDigits' value of the base type '%d'.", *fs.FractionDigits, *base.FractionDigits)))
	}

	// Inclusive/exclusive boundary facets vs base.
	if fs.MaxInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxInclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxInclusive)))
		}
	}
	if fs.MaxInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxExclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxExclusive)))
		}
	}
	if fs.MaxInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinInclusive) < 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinInclusive)))
		}
	}
	if fs.MaxInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinExclusive) <= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinExclusive)))
		}
	}
	if fs.MaxExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxExclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxExclusive)))
		}
	}
	if fs.MaxExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxInclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxInclusive)))
		}
	}
	if fs.MaxExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinInclusive) <= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinInclusive)))
		}
	}
	if fs.MaxExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinExclusive) <= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinExclusive)))
		}
	}
	if fs.MinInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinInclusive) < 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinInclusive)))
		}
	}
	if fs.MinInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinExclusive) <= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinExclusive)))
		}
	}
	if fs.MinInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxInclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxInclusive)))
		}
	}
	if fs.MinInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxExclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxExclusive)))
		}
	}
	if fs.MinExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinExclusive) < 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinExclusive)))
		}
	}
	if fs.MinExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinInclusive) < 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinInclusive)))
		}
	}
	if fs.MinExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxInclusive) > 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxInclusive)))
		}
	}
	if fs.MinExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxExclusive) >= 0 {
			c.schemaErrors.WriteString(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxExclusive)))
		}
	}
}
