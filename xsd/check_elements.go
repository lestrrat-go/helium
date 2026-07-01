package xsd

import (
	"context"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// hasAttr checks whether an unqualified (no-namespace) schema attribute is
// physically present on the element. Unlike getAttr, this distinguishes absent
// from empty-string. A foreign-namespaced attribute that happens to share the
// local name (e.g. other:fixed) is not matched, since XSD schema attributes
// (name/type/fixed/default/minOccurs/...) are always unqualified.
func hasAttr(elem *helium.Element, name string) bool {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name && a.URI() == "" {
			return true
		}
	}
	return false
}

// isValidFinal checks if a value is valid for the 'final' attribute on elements.
func isValidFinal(v string) bool {
	if v == lexicon.ModeAll {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != attrValExtension && part != attrValRestriction {
			return false
		}
	}
	return true
}

// isValidBlock checks if a value is valid for the 'block' attribute.
func isValidBlock(v string) bool {
	if v == lexicon.ModeAll {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != attrValExtension && part != attrValRestriction && part != attrValSubstitution {
			return false
		}
	}
	return true
}

// isValidFinalDefault checks if a value is valid for the 'finalDefault' attribute on xs:schema.
// Accepts #all or space-separated list of extension|restriction|list|union.
func isValidFinalDefault(v string) bool {
	if v == lexicon.ModeAll {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != attrValExtension && part != attrValRestriction && part != attrValList && part != attrValUnion {
			return false
		}
	}
	return true
}

// splitSpace splits a string on XSD whitespace only (space, tab, CR, LF), not the
// wider Unicode set strings.Fields uses. It is the xsd-package splitter for
// XSD list-valued schema attributes (block/final, the wildcard namespace list,
// memberTypes), consistent with value.XSDFields: a token containing NBSP or other
// Unicode whitespace stays one token instead of being silently split.
func splitSpace(s string) []string {
	var parts []string
	start := -1
	for i := range len(s) {
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

// checkGlobalElement validates constraints on a global xs:element declaration.
func (c *compiler) checkGlobalElement(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	name := getAttr(elem, attrName)
	line := elem.Line()
	local := elem.LocalName()

	// name is required for global elements.
	if name == "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'name' is required but missing."))
	}

	// ref is not allowed at global level.
	if getAttr(elem, attrRef) != "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'ref' is not allowed."))
	}

	// minOccurs is not allowed at global level.
	if getAttr(elem, attrMinOccurs) != "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'minOccurs' is not allowed."))
	}

	// maxOccurs is not allowed at global level.
	if getAttr(elem, attrMaxOccurs) != "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'maxOccurs' is not allowed."))
	}

	// form is not allowed at global level.
	if getAttr(elem, attrForm) != "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'form' is not allowed."))
	}

	// Validate 'final' attribute value.
	if v := getAttr(elem, attrFinal); v != "" {
		if !isValidFinal(v) {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, elemElement, attrFinal,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction))'."))
		}
	}

	// Validate 'block' attribute value.
	if v := getAttr(elem, attrBlock); v != "" {
		if !isValidBlock(v) {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, elemElement, attrBlock,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}
	}

	// default and fixed are mutually exclusive.
	if hasAttr(elem, attrDefault) && hasAttr(elem, attrFixed) {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attributes 'default' and 'fixed' are mutually exclusive."))
	}

	// type and inline complexType/simpleType are mutually exclusive.
	if getAttr(elem, attrType) != "" {
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if isXSDElement(ce, "complexType") {
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <complexType> child are mutually exclusive."))
			}
			if isXSDElement(ce, "simpleType") {
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <simpleType> child are mutually exclusive."))
			}
		}
	}
}

// effectiveMinOccurs returns the effective minOccurs value for a particle given
// the raw minOccurs attribute lexical (empty when unset). An absent or invalid
// minOccurs defaults to 1, matching XSD's schema-default and libxml2's behavior
// when deciding whether a maxOccurs of 0 is a legal prohibited particle.
func effectiveMinOccurs(minOcc string) int {
	if minOcc == "" {
		return 1
	}
	n, ok := parseNonNegativeOccurs(minOcc, false)
	if !ok {
		return 1
	}
	return n
}

// validateAllOccurs validates the minOccurs/maxOccurs attributes of an xs:all
// compositor particle. Per XSD Part 1 §3.8.6 (cos-all-limited) the all
// compositor's minOccurs must be 0 or 1 and its maxOccurs must be 1. libxml2
// reports any other value (including non-integer lexicals such as "abc") with
// dedicated wording rather than the generic xs:nonNegativeInteger/xs:allNNI
// diagnostics, so the generic validateOccursAttrs is bypassed for xs:all.
func (c *compiler) validateAllOccurs(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	line := elem.Line()
	local := elem.LocalName()
	// Attribute to the declaring file (an included/imported schema when inside an
	// xs:include/xs:import/xs:redefine), not the top-level compiler filename.
	src := c.diagSource()

	// The all compositor's minOccurs lexical space allows leading zeros (e.g.
	// "01" parses to 1 and is accepted). libxml2 reports the all-specific
	// "(0 | 1)" wording for every value outside {0,1}, including lexically
	// invalid forms such as "-1", "abc", or "unbounded", so a failed parse and
	// an out-of-range parse are both reported with this diagnostic.
	if hasAttr(elem, attrMinOccurs) {
		v := getAttr(elem, attrMinOccurs)
		n, ok := parseNonNegativeOccurs(v, false)
		if !ok || (n != 0 && n != 1) {
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, elemAll, attrMinOccurs,
				"The value '"+v+"' is not valid. Expected is '(0 | 1)'."))
		}
	}

	// maxOccurs must parse to exactly 1; "1"/"01" are accepted, everything else
	// (including "unbounded" and lexically invalid forms) is reported with the
	// all-specific "Expected is '1'." wording. allowMax is true so "unbounded"
	// parses successfully and is then rejected for being != 1, matching libxml2.
	if hasAttr(elem, attrMaxOccurs) {
		v := getAttr(elem, attrMaxOccurs)
		n, ok := parseNonNegativeOccurs(v, true)
		if !ok || n != 1 {
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, elemAll, attrMaxOccurs,
				"The value '"+v+"' is not valid. Expected is '1'."))
		}
	}
}

// checkAllElementParticleOccurs validates the minOccurs/maxOccurs of an
// xs:element particle that appears directly inside an xs:all compositor. Per
// cos-all-limited each such particle may occur at most once, so both occurrence
// bounds must be 0 or 1. libxml2 reports a value outside that range with the
// dedicated "Invalid value for {min,max}Occurs (must be 0 or 1)." wording, after
// the generic occurs ordering that checkLocalElement already emitted.
func (c *compiler) checkAllElementParticleOccurs(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	// XSD 1.1 relaxes cos-all-limited: an element particle inside an xs:all may
	// have any minOccurs/maxOccurs (including maxOccurs>1 / unbounded). The
	// generic checkLocalElement still validates the lexical form and min<=max, so
	// only the all-specific "(must be 0 or 1)" restriction is dropped here.
	if c.version == Version11 {
		return
	}
	line := elem.Line()
	local := elem.LocalName()
	// Attribute to the declaring file (an included/imported schema when inside an
	// xs:include/xs:import/xs:redefine), not the top-level compiler filename.
	src := c.diagSource()

	// Child occurs lexical space allows leading zeros ("01" parses to 1 and is
	// accepted). The all-specific "(must be 0 or 1)" diagnostic only fires for a
	// lexically valid value outside {0,1}: a lexically invalid value (e.g. "-1",
	// "unbounded") already produced the generic nonNegativeInteger/allNNI error
	// in checkLocalElement, so suppress the all-specific message there to match
	// libxml2, which emits only the lexical error.
	if hasAttr(elem, attrMaxOccurs) {
		v := getAttr(elem, attrMaxOccurs)
		n, ok := parseNonNegativeOccurs(v, true)
		if ok && n != 0 && n != 1 {
			c.schemaError(ctx, schemaParserError(src, line, local, "element",
				"Invalid value for maxOccurs (must be 0 or 1)."))
		}
	}

	if hasAttr(elem, attrMinOccurs) {
		v := getAttr(elem, attrMinOccurs)
		n, ok := parseNonNegativeOccurs(v, false)
		if ok && n != 0 && n != 1 {
			c.schemaError(ctx, schemaParserError(src, line, local, "element",
				"Invalid value for minOccurs (must be 0 or 1)."))
		}
	}
}

// checkLocalElement validates constraints on a local xs:element declaration.
func (c *compiler) checkLocalElement(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	ref := getAttr(elem, attrRef)
	name := getAttr(elem, attrName)
	line := elem.Line()
	local := elem.LocalName()

	minOcc := getAttr(elem, attrMinOccurs)
	maxOcc := getAttr(elem, attrMaxOccurs)
	// Detect presence with hasAttr so an explicitly empty minOccurs="" /
	// maxOccurs="" is validated (and rejected as an invalid lexical) instead of
	// being treated as absent, matching xmllint.
	minPresent := hasAttr(elem, attrMinOccurs)
	maxPresent := hasAttr(elem, attrMaxOccurs)

	if ref != "" {
		// Matches libxml2 ordering for ref elements (src-element 2.2):
		// 1. maxOccurs >= 1, minOccurs > maxOccurs
		// 2. ref+name conflict
		// 3. First ref-restricted attribute (alphabetical)
		// 4. First child content error

		// maxOccurs must be a non-negative integer (or "unbounded"). A maxOccurs of
		// 0 is a legal prohibited particle when the effective minOccurs is also 0;
		// libxml2 only rejects maxOccurs<1 when the effective minOccurs is >= 1
		// (default minOccurs is 1), reporting the ">= 1" message on maxOccurs.
		if maxPresent && maxOcc != attrValUnbounded {
			maxVal, ok := parseNonNegativeOccurs(maxOcc, true)
			if !ok {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value '"+maxOcc+"' is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'."))
			} else if maxVal < 1 && effectiveMinOccurs(minOcc) >= 1 {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."))
			}
		}

		// minOccurs must be a non-negative integer.
		if minPresent {
			if _, ok := parseNonNegativeOccurs(minOcc, false); !ok {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value '"+minOcc+"' is not valid. Expected is 'xs:nonNegativeInteger'."))
			}
		}

		// minOccurs > maxOccurs check. Skip it when maxOccurs already failed the
		// >= 1 rule (maxVal < 1 with an effective minOccurs >= 1); libxml2 reports
		// only the maxOccurs error there.
		if minPresent && maxPresent && maxOcc != attrValUnbounded {
			minVal, minOK := parseNonNegativeOccurs(minOcc, false)
			maxVal, maxOK := parseNonNegativeOccurs(maxOcc, true)
			if minOK && maxOK && maxVal != Unbounded && (maxVal >= 1 || minVal < 1) && minVal > maxVal {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value must not be greater than the value of 'maxOccurs'."))
			}
		}

		// ref and name are mutually exclusive.
		if name != "" {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
				"The attributes 'ref' and 'name' are mutually exclusive."))
		}

		// Report first ref-restricted attribute found (alphabetical order).
		notAllowedWithRef := []string{attrAbstract, attrBlock, attrDefault, attrFinal, attrFixed, attrForm, attrNillable, attrSubstitutionGroup, attrType}
		if c.version == Version11 {
			notAllowedWithRef = []string{attrAbstract, attrBlock, attrDefault, attrFinal, attrFixed, attrForm, attrNillable, attrSubstitutionGroup, attrTargetNamespace, attrType}
		}
		for _, attr := range notAllowedWithRef {
			if hasAttr(elem, attr) {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, "element", attr,
					"Only the attributes 'minOccurs', 'maxOccurs' and 'id' are allowed in addition to 'ref'."))
				break // only report first
			}
		}

		// An element ref may only carry (annotation?). Any other XSD child — an
		// inline complexType/simpleType, an xs:alternative (XSD 1.1 CTA belongs to
		// the referenced GLOBAL declaration, not the ref), or any stray XSD element —
		// is invalid. The check is scoped to the XSD namespace because helium
		// consistently TOLERATES foreign-namespace element children across every
		// other schema component (complexType, global element, attribute, model
		// groups all silently ignore them via switch-on-isXSDElement with no default
		// rejection); rejecting them only here would be inconsistent. The diagnostic
		// is attributed to the declaring file (c.diagSource), so an included/redefined
		// schema's violation cites that file, not the top-level label.
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if ce.URI() == lexicon.NamespaceXSD && !isXSDElement(ce, elemAnnotation) {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "element",
					"The content is not valid. Expected is (annotation?)."))
				break // only report first
			}
		}
	} else if name != "" {
		// Named local element constraints.
		// Matches libxml2 ordering: maxOccurs, not-allowed attrs,
		// block/final value checks, default+fixed, type/content children.

		// maxOccurs must be a non-negative integer (or "unbounded"). A maxOccurs of
		// 0 is a legal prohibited particle when the effective minOccurs is also 0;
		// libxml2 only rejects maxOccurs<1 when the effective minOccurs is >= 1
		// (default minOccurs is 1), reporting the ">= 1" message on maxOccurs.
		if maxPresent && maxOcc != attrValUnbounded {
			maxVal, ok := parseNonNegativeOccurs(maxOcc, true)
			if !ok {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value '"+maxOcc+"' is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'."))
			} else if maxVal < 1 && effectiveMinOccurs(minOcc) >= 1 {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."))
			}
		}

		// minOccurs must be a non-negative integer.
		if minPresent {
			if _, ok := parseNonNegativeOccurs(minOcc, false); !ok {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value '"+minOcc+"' is not valid. Expected is 'xs:nonNegativeInteger'."))
			}
		}

		// minOccurs > maxOccurs check. Skip it when maxOccurs already failed the
		// >= 1 rule (maxVal < 1 with an effective minOccurs >= 1); libxml2 reports
		// only the maxOccurs error there.
		if minPresent && maxPresent && maxOcc != attrValUnbounded {
			minVal, minOK := parseNonNegativeOccurs(minOcc, false)
			maxVal, maxOK := parseNonNegativeOccurs(maxOcc, true)
			if minOK && maxOK && maxVal != Unbounded && (maxVal >= 1 || minVal < 1) && minVal > maxVal {
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value must not be greater than the value of 'maxOccurs'."))
			}
		}

		// Some attributes not allowed for local named elements.
		localNotAllowed := []string{attrAbstract, attrSubstitutionGroup, attrFinal}
		for _, attr := range localNotAllowed {
			if hasAttr(elem, attr) {
				c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
					"The attribute '"+attr+"' is not allowed."))
			}
		}
		c.checkLocalElementTargetNamespace(ctx, elem)

		// Validate 'block' attribute value.
		if v := getAttr(elem, attrBlock); v != "" && !isValidBlock(v) {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}

		// default and fixed mutually exclusive.
		if hasAttr(elem, attrDefault) && hasAttr(elem, attrFixed) {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// type and inline complexType/simpleType checks.
		hasType := getAttr(elem, attrType) != ""
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if isXSDElement(ce, elemComplexType) {
				if hasType {
					c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The attribute 'type' and the <complexType> child are mutually exclusive."))
				}
			} else if isXSDElement(ce, elemSimpleType) {
				if hasType {
					c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."))
				}
			}
		}
	} else if c.version == Version11 && hasAttr(elem, attrTargetNamespace) {
		c.checkLocalElementTargetNamespace(ctx, elem)
	}
}

func (c *compiler) checkLocalElementTargetNamespace(ctx context.Context, elem *helium.Element) {
	if c.version != Version11 || !hasAttr(elem, attrTargetNamespace) {
		return
	}

	line := elem.Line()
	local := elem.LocalName()
	if getAttr(elem, attrName) == "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "element",
			"The attribute 'targetNamespace' requires a local element declaration with a 'name'."))
		return
	}
	if hasAttr(elem, attrForm) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "element",
			"The attributes 'targetNamespace' and 'form' are mutually exclusive."))
	}
	targetNS := getAttr(elem, attrTargetNamespace)
	sameSchemaTargetNS := c.schemaTargetNSSet && targetNS == c.schema.targetNamespace
	if !sameSchemaTargetNS && !c.localElementUnderNonAnyTypeRestriction(ctx, elem) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "element",
			"A local element declaration with 'targetNamespace' different from the schema target namespace, or with no schema target namespace, must appear in a restriction of a type other than xs:anyType."))
	}
}

func (c *compiler) localElementUnderNonAnyTypeRestriction(ctx context.Context, elem *helium.Element) bool {
	for parent := elem.Parent(); parent != nil; parent = parent.Parent() {
		pe, ok := helium.AsNode[*helium.Element](parent)
		if !ok {
			continue
		}
		if isXSDElement(pe, elemComplexType) {
			return false
		}
		if !isXSDElement(pe, elemRestriction) || !isContentDerivationRestriction(pe) {
			continue
		}
		base := getAttr(pe, attrBase)
		if base == "" {
			return false
		}
		qn := c.resolveQName(ctx, pe, base)
		return qn.NS != lexicon.NamespaceXSD || qn.Local != typeAnyType
	}
	return false
}

// checkAttributeUse validates constraints on an xs:attribute declaration.
func (c *compiler) checkAttributeUse(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	ref := getAttr(elem, attrRef)
	line := elem.Line()
	local := elem.LocalName()

	// XSD 1.1 Schema Representation Constraint (Attribute Declaration
	// Representation OK): a prohibited attribute use must not carry a value
	// constraint. A prohibited attribute can never appear, so a fixed value is
	// meaningless. XSD 1.0 tolerates this (the schemas are valid there), so the
	// check is gated on Version11 to keep 1.0 byte-identical. A `default` on a
	// prohibited use is already rejected in both versions by the "default requires
	// use=optional" check below, so only `fixed` needs handling here.
	if c.version == Version11 && getAttr(elem, attrUse) == attrValProhibited && hasAttr(elem, attrFixed) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
			"The attribute 'fixed' is not allowed when the value of the attribute 'use' is 'prohibited'."))
	}

	if ref != "" {
		// ref and name are mutually exclusive.
		if getAttr(elem, attrName) != "" {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'name' is not allowed."))
		}

		// type not allowed with ref.
		if getAttr(elem, attrType) != "" {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'type' is not allowed."))
		}

		// default and fixed are mutually exclusive.
		if hasAttr(elem, attrDefault) && hasAttr(elem, attrFixed) {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// If default is present, use must be optional (or absent, which
		// defaults to optional). default/fixed are incompatible with
		// use="prohibited"; a prohibited use can never carry a value.
		if hasAttr(elem, attrDefault) {
			use := getAttr(elem, attrUse)
			if use != "" && use != attrValOptional {
				c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
					"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present."))
			}
		}

		// form not allowed with ref.
		if getAttr(elem, attrForm) != "" {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'form' is not allowed."))
		}

		if c.version == Version11 && hasAttr(elem, attrTargetNamespace) {
			c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
				"The attribute 'targetNamespace' is not allowed with a referenced attribute declaration."))
		}

		// simpleType child not allowed with ref.
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if isXSDElement(ce, elemSimpleType) {
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
					"The content is not valid. Expected is (annotation?)."))
			}
		}
	} else {
		// Attribute name must not be "xmlns".
		if getAttr(elem, attrName) == lexicon.PrefixXMLNS {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "attribute", "name",
				"The value of the attribute must not match 'xmlns'."))
		}

		c.checkLocalAttributeTargetNamespace(ctx, elem)

		// Qualified attribute must not be in the XSI namespace.
		form := getAttr(elem, attrForm)
		if form == "qualified" || (form == "" && c.schema.attrFormQualified) {
			if c.schema.targetNamespace == lexicon.NamespaceXSI {
				c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
					"The target namespace must not match '"+lexicon.NamespaceXSI+"'."))
			}
		}

		// default and fixed are mutually exclusive.
		if hasAttr(elem, attrDefault) && hasAttr(elem, attrFixed) {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// If default is present, use must be optional (or absent, which defaults to optional).
		if hasAttr(elem, attrDefault) {
			use := getAttr(elem, attrUse)
			if use != "" && use != attrValOptional {
				c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
					"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present."))
			}
		}

		// type and inline simpleType are mutually exclusive.
		if getAttr(elem, attrType) != "" {
			for child := range helium.Children(elem) {
				if child.Type() != helium.ElementNode {
					continue
				}
				ce, ok := helium.AsNode[*helium.Element](child)
				if !ok {
					continue
				}
				if isXSDElement(ce, elemSimpleType) {
					c.schemaError(ctx, schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"The attribute 'type' and the <simpleType> child are mutually exclusive."))
				}
			}
		}
	}
}

func (c *compiler) checkLocalAttributeTargetNamespace(ctx context.Context, elem *helium.Element) {
	if c.version != Version11 || !hasAttr(elem, attrTargetNamespace) || isGlobalAttributeDecl(elem) {
		return
	}

	line := elem.Line()
	local := elem.LocalName()
	if getAttr(elem, attrName) == "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
			"The attribute 'targetNamespace' requires a local attribute declaration with a 'name'."))
	}
	if hasAttr(elem, attrForm) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
			"The attributes 'targetNamespace' and 'form' are mutually exclusive."))
	}
	targetNS := getAttr(elem, attrTargetNamespace)
	if targetNS == lexicon.NamespaceXSI {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, "attribute", attrTargetNamespace,
			"An attribute declaration must not be in the XSI namespace."))
	}
	if targetNS != c.schema.targetNamespace && !c.localAttributeUnderNonAnyTypeRestriction(ctx, elem) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
			"A local attribute declaration with 'targetNamespace' different from the schema target namespace must appear in a restriction of a type other than xs:anyType."))
	}
}

func isGlobalAttributeDecl(elem *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](elem.Parent())
	return ok && isXSDElement(parent, elemSchema)
}

func (c *compiler) localAttributeUnderNonAnyTypeRestriction(ctx context.Context, elem *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](elem.Parent())
	if !ok || !isXSDElement(parent, elemRestriction) || !isContentDerivationRestriction(parent) {
		return false
	}
	base := getAttr(parent, attrBase)
	if base == "" {
		return false
	}
	qn := c.resolveQName(ctx, parent, base)
	return qn.NS != lexicon.NamespaceXSD || qn.Local != typeAnyType
}

func isContentDerivationRestriction(elem *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](elem.Parent())
	return ok && (isXSDElement(parent, elemSimpleContent) || isXSDElement(parent, elemComplexContent))
}

// checkAnnotation validates an xs:annotation element and its children.
func (c *compiler) checkAnnotation(ctx context.Context, elem *helium.Element) {
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
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "annotation",
			"The attribute '"+name+"' is not allowed."))
	}

	// Check for invalid content (non-element children like text nodes).
	hasInvalidContent := false
	for child := range helium.Children(elem) {
		if child.Type() == helium.TextNode {
			text := strings.TrimSpace(string(child.Content()))
			if text != "" {
				hasInvalidContent = true
				break
			}
		}
	}
	if hasInvalidContent {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "annotation",
			"The content is not valid. Expected is (appinfo | documentation)*."))
	}

	// Check children (appinfo, documentation).
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemAppinfo) {
			c.checkAppinfo(ctx, ce)
		} else if isXSDElement(ce, elemDocumentation) {
			c.checkDocumentation(ctx, ce)
		}
	}
}

// checkAppinfo validates an xs:appinfo element.
func (c *compiler) checkAppinfo(ctx context.Context, elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()

	// Only "source" is allowed (no id).
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		if attr.Prefix() != "" {
			continue
		}
		if name == attrSource {
			continue
		}
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "appinfo",
			"The attribute '"+name+"' is not allowed."))
	}
}

// checkDocumentation validates an xs:documentation element.
func (c *compiler) checkDocumentation(ctx context.Context, elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()

	// Only "source" and "xml:lang" are allowed (no id).
	// Check disallowed attributes first, then validate xml:lang value.
	var langValue string
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		prefix := attr.Prefix()
		if prefix != "" && prefix != lexicon.PrefixXML {
			continue // other namespaced attributes are allowed
		}
		if prefix == lexicon.PrefixXML && name == lexicon.AttrLang {
			langValue = string(attr.Content())
			continue
		}
		if name == attrSource {
			continue
		}
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "documentation",
			"The attribute '"+name+"' is not allowed."))
	}

	// Validate xml:lang value after attribute checks.
	if langValue != "" && !languageRegex.MatchString(langValue) {
		c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "documentation",
			helium.ClarkName(lexicon.NamespaceXML, lexicon.AttrLang),
			"'"+langValue+"' is not a valid value of the atomic type 'xs:language'."))
	}
}
