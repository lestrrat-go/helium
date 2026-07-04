package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
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

// qnameCompanionUsable reports whether a QName-valued schema attribute (attr, e.g.
// @type/@ref) should fire the STRUCTURAL representation gate it participates in — a
// with-ref prohibition ("type not allowed"), an inline-type mutual-exclusion
// ("type and the <simpleType> child are mutually exclusive"), or the like. It
// returns true ONLY when attr is present with a value that COLLAPSES TO NON-EMPTY.
//
// A PRESENT-but-collapse-empty value — the literal "" OR a whitespace-only "   "
// (xs:QName has its whiteSpace facet fixed "collapse") — is instead an invalid
// (empty) xs:QName: the helper reports that ONE invalid-QName value diagnostic and
// returns false, so the structural secondary is suppressed. Because
// reportInvalidQNameValue dedups by (element, attribute, value), re-reporting a
// value already flagged at its QName store site (element/attribute @type, @base,
// …) collapses to a single diagnostic; where no store site validates the value
// (an @type alongside @ref is never routed), this helper is the sole reporter. So
// a present-empty and a whitespace-only companion stay SYMMETRIC — each yields
// exactly the one invalid-QName diagnostic with no spurious structural follow-on —
// while a genuinely-present non-empty value (valid, or a two-token/leading-colon
// malformed value that is out of scope here) still fires the structural gate.
func (c *compiler) qnameCompanionUsable(ctx context.Context, elem *helium.Element, attr string) bool {
	if !hasAttr(elem, attr) {
		return false
	}
	v := collapsedAttr(elem, attr)
	if v == "" {
		c.reportInvalidQNameValue(ctx, elem, attr, v)
		return false
	}
	return true
}

// ncnameCompanionUsable is the xs:NCName (@name) counterpart of
// qnameCompanionUsable: it reports whether the @name companion should fire its
// with-ref structural gate (a "ref and name are mutually exclusive" / "name is not
// allowed" prohibition). It returns true only when @name is present with a value
// that COLLAPSES TO NON-EMPTY. A PRESENT-but-collapse-empty @name (""/whitespace-
// only — xs:NCName also has whiteSpace fixed "collapse") is an invalid (empty)
// NCName: the helper reports that ONE invalid-NCName diagnostic (the same wording
// the primary declaration branch uses) and returns false, suppressing the
// structural secondary so a present-empty and a whitespace-only @name stay
// SYMMETRIC. A present non-empty @name (valid or malformed — the latter reported
// by the enclosing declaration's own NCName check) still fires the structural
// gate. xsdElem is the schema-for-schemas element name the diagnostic cites
// ("attribute"/"element"). The with-ref branch does not otherwise validate @name,
// so this helper is the sole reporter — no double diagnostic.
func (c *compiler) ncnameCompanionUsable(ctx context.Context, elem *helium.Element, xsdElem string) bool {
	if !hasAttr(elem, attrName) {
		return false
	}
	v := collapsedAttr(elem, attrName)
	if v == "" {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), xsdElem, attrName,
			"The value '"+v+"' is not a valid 'NCName'."))
		return false
	}
	return true
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

	// Closed attribute-vocabulary check (§3.3.2, version-INDEPENDENT).
	c.checkElementAttrVocabulary(ctx, elem)

	// Content-model ordering: annotation must be the first child (§3.3.2).
	c.checkElementContentOrder(ctx, elem)

	// name is required for global elements. Presence is detected on the RAW value
	// (an absent or literally-empty name), while the NCName test runs on the
	// COLLAPSED value (xs:element/@name is xs:NCName, whiteSpace fixed "collapse"):
	// a padded name="sub2-elem " is valid, a whitespace-only name collapses to
	// empty and is an invalid NCName, and an internal-whitespace name="a b" fails.
	if name == "" {
		c.schemaError(ctx, schemaParserError(c.filename, line, local, "element",
			"The attribute 'name' is required but missing."))
	} else if !xmlchar.IsValidNCName(normalizeWhiteSpace(name, "collapse")) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, elemElement, attrName,
			"The value '"+name+"' is not a valid 'NCName'."))
	}

	// ref is not allowed at global level. @ref is xs:QName: a PRESENT-but-collapse-
	// empty ref (""/whitespace-only) is instead an invalid (empty) QName, surfaced as
	// that one value diagnostic by qnameCompanionUsable (which returns false), so the
	// prohibition is suppressed and present-empty stays symmetric with whitespace-only;
	// a present non-empty ref still fires the prohibition.
	if c.qnameCompanionUsable(ctx, elem, attrRef) {
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

	// type and inline complexType/simpleType are mutually exclusive. @type is
	// xs:QName: a PRESENT-but-collapse-empty type is an invalid (empty) QName (its
	// one value diagnostic fires at the @type store site, re-reported deduped by
	// qnameCompanionUsable, which returns false), so the mutual-exclusion is
	// suppressed and present-empty stays symmetric with whitespace-only; a present
	// non-empty type still fires the mutual-exclusion.
	if c.qnameCompanionUsable(ctx, elem, attrType) {
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

// elementAttrAllowed reports whether name is a recognized unqualified attribute
// of an <xs:element> declaration in ANY of its forms — global (topLevelElement),
// local (localElement), or element reference — across XSD 1.0 and 1.1. Per
// §3.3.2 the schema-for-schemas gives xs:element a CLOSED attribute set with an
// `##other` anyAttribute admitting foreign-namespaced attributes; an unqualified
// attribute outside this vocabulary (e.g. `nullable`, a misspelling of
// `nillable`, or a random `foo`) is a schema-representation error. Attributes
// recognized but disallowed for the SPECIFIC form (ref/minOccurs/maxOccurs/form
// on a global, abstract/substitutionGroup/final on a local, …) are reported by
// the form-specific checks in checkGlobalElement/checkLocalElement, so this
// vocabulary predicate is the UNION of every form to avoid double-diagnosing
// those. targetNamespace and abstract/final/substitutionGroup are included even
// though a given form/version forbids them, since the goal here is only to reject
// attributes that belong to NO xs:element form at all.
func elementAttrAllowed(name string) bool {
	switch name {
	case "id", attrName, attrRef, attrType, attrMinOccurs, attrMaxOccurs,
		attrDefault, attrFixed, attrNillable, attrBlock, attrForm,
		attrSubstitutionGroup, attrFinal, attrAbstract, attrTargetNamespace:
		return true
	}
	return false
}

// checkElementAttrVocabulary reports every unqualified attribute on an
// <xs:element> declaration that is outside the closed §3.3.2 vocabulary.
// Foreign-namespaced attributes (vc:, xml:, any qualified name) are admitted by
// the schema-for-schemas `##other` anyAttribute, so only no-namespace attributes
// are checked. Version-INDEPENDENT.
func (c *compiler) checkElementAttrVocabulary(ctx context.Context, elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()
	src := c.diagSource()
	for _, attr := range elem.Attributes() {
		if attr.URI() != "" || elementAttrAllowed(attr.LocalName()) {
			continue
		}
		c.schemaError(ctx, schemaParserError(src, line, local, elemElement,
			"The attribute '"+attr.LocalName()+"' is not allowed."))
	}
}

// checkElementContentOrder enforces the §3.3.2 element content model ordering
// constraint that an <xs:annotation> must be the FIRST child of an <xs:element>
// declaration — it may not FOLLOW a recognized content child (<simpleType>,
// <complexType>, <alternative>, <unique>, <key>, <keyref>). This is
// version-INDEPENDENT: annotation is the leading term of the element content
// model in every XSD version ((annotation?, ((simpleType | complexType)?,
// alternative*, (unique | key | keyref)*))). Only recognized content children set
// the "saw a non-annotation" state, so a stray/foreign child that helium
// otherwise tolerates does not trigger the diagnostic. The at-most-one-annotation
// rule and the annotation's own content model are enforced separately by
// checkAnnotations.
func (c *compiler) checkElementContentOrder(ctx context.Context, elem *helium.Element) {
	sawContentChild := false
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if ce.URI() != lexicon.NamespaceXSD {
			continue
		}
		if isXSDElement(ce, elemAnnotation) {
			if sawContentChild {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "element",
					"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."))
			}
			continue
		}
		switch {
		case isXSDElement(ce, elemSimpleType), isXSDElement(ce, elemComplexType),
			isXSDElement(ce, elemAlternative), isXSDElement(ce, elemUnique),
			isXSDElement(ce, elemKey), isXSDElement(ce, elemKeyRef):
			sawContentChild = true
		}
	}
}

// attributeAttrAllowed reports whether name is a recognized unqualified attribute
// of an <xs:attribute> declaration in ANY of its forms — global
// (topLevelAttribute), local attribute use, or attribute reference — across XSD
// 1.0 and 1.1. Per §3.2.2 the schema-for-schemas gives xs:attribute a CLOSED
// attribute set with an `##other` anyAttribute admitting foreign-namespaced
// attributes; an unqualified attribute outside this vocabulary (e.g. a random
// `value`) is a schema-representation error. Attributes recognized but disallowed
// for the SPECIFIC form (use/form/ref on a global, …) are reported by the
// form-specific checks in checkAttributeUse, so this predicate is the UNION of
// every form. inheritable/targetNamespace are XSD 1.1 additions kept in the union
// so 1.1 schemas are not over-rejected.
func attributeAttrAllowed(name string) bool {
	switch name {
	case "id", attrName, attrRef, attrType, attrUse, attrDefault, attrFixed,
		attrForm, attrTargetNamespace, attrInheritable:
		return true
	}
	return false
}

// checkAttrVocabulary reports every unqualified attribute on an <xs:attribute>
// declaration that is outside the closed §3.2.2 vocabulary. Foreign-namespaced
// attributes are admitted by the schema-for-schemas `##other` anyAttribute, so
// only no-namespace attributes are checked. Version-INDEPENDENT.
func (c *compiler) checkAttrVocabulary(ctx context.Context, elem *helium.Element) {
	line := elem.Line()
	local := elem.LocalName()
	src := c.diagSource()
	for _, attr := range elem.Attributes() {
		if attr.URI() != "" || attributeAttrAllowed(attr.LocalName()) {
			continue
		}
		c.schemaError(ctx, schemaParserError(src, line, local, "attribute",
			"The attribute '"+attr.LocalName()+"' is not allowed."))
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

	// maxOccurs must parse to exactly 1 in XSD 1.0; "1"/"01" are accepted,
	// everything else (including "unbounded" and lexically invalid forms) is
	// reported with the all-specific "Expected is '1'." wording. allowMax is true
	// so "unbounded" parses successfully and is then rejected for being != 1,
	// matching libxml2. XSD 1.1 relaxes cos-all-limited to allow the all group's
	// own maxOccurs to be 0 or 1 (mgO001/mgO018), so 0 is also accepted there.
	if hasAttr(elem, attrMaxOccurs) {
		v := getAttr(elem, attrMaxOccurs)
		n, ok := parseNonNegativeOccurs(v, true)
		valid := ok && n == 1
		expected := "1"
		if c.version == Version11 {
			valid = ok && (n == 0 || n == 1)
			expected = "(0 | 1)"
		}
		if !valid {
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, elemAll, attrMaxOccurs,
				"The value '"+v+"' is not valid. Expected is '"+expected+"'."))
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
	// @ref is xs:QName (whiteSpace fixed "collapse"): dispatch on the COLLAPSED value
	// so a PRESENT-but-collapse-empty ref (""/whitespace-only) is NOT treated as a
	// present reference. It falls through to the named/else branch exactly like an
	// empty ref="", so the invalid (empty) QName — reported once at the @ref store
	// site (parseLocalElement) — does not also trigger the ref-branch's companion/
	// prohibition secondaries; present-empty and whitespace-only stay symmetric.
	ref := collapsedAttr(elem, attrRef)
	name := getAttr(elem, attrName)
	line := elem.Line()
	local := elem.LocalName()

	// Closed attribute-vocabulary check (§3.3.2, version-INDEPENDENT).
	c.checkElementAttrVocabulary(ctx, elem)

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

		// ref and name are mutually exclusive. @name is xs:NCName: a PRESENT-but-
		// collapse-empty name is instead an invalid (empty) NCName — ncnameCompanionUsable
		// reports that one value diagnostic and returns false, suppressing the
		// mutual-exclusion so present-empty stays symmetric with whitespace-only; a
		// present non-empty name still fires the mutual-exclusion.
		if c.ncnameCompanionUsable(ctx, elem, elemElement) {
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

		// Content-model ordering: annotation must be the first child (§3.3.2).
		c.checkElementContentOrder(ctx, elem)

		// The {name} of a local element declaration must be an NCName (XSD
		// Structures §3.3.2; xsd:element/@name is of type xs:NCName, whiteSpace
		// fixed "collapse"), exactly as for global declarations — the value is
		// collapsed before the NCName test.
		if !xmlchar.IsValidNCName(normalizeWhiteSpace(name, "collapse")) {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, elemElement, attrName,
				"The value '"+name+"' is not a valid 'NCName'."))
		}

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

		// type and inline complexType/simpleType checks. @type is xs:QName: a
		// PRESENT-but-collapse-empty type is an invalid (empty) QName (reported at the
		// store site, re-reported deduped here), so it does NOT count as a present type
		// for the mutual-exclusion — present-empty stays symmetric with whitespace-only;
		// a present non-empty type still fires the mutual-exclusion.
		hasType := c.qnameCompanionUsable(ctx, elem, attrType)
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
		// @base is an xs:QName: gate on PRESENCE and route the value through the QName
		// chokepoint so a PRESENT-but-empty base="" behaves like the whitespace-only
		// base="   " — both collapse to the invalidQName sentinel, whose invalid-QName
		// diagnostic already fired, so this secondary targetNamespace check does not
		// emit its own error (isInvalidQName → suppress).
		if !hasAttr(pe, attrBase) {
			return false
		}
		qn := c.resolveQName(ctx, pe, attrBase, getAttr(pe, attrBase))
		return isInvalidQName(qn) || qn.NS != lexicon.NamespaceXSD || qn.Local != typeAnyType
	}
	return false
}

// checkAttributeUse validates constraints on an xs:attribute declaration.
func (c *compiler) checkAttributeUse(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	// xs:attribute/@ref is an xs:QName (whiteSpace fixed "collapse"): read it
	// collapsed so the whole function (presence dispatch, QName validity) sees the
	// same value the store/resolution does.
	ref := collapsedAttr(elem, attrRef)
	line := elem.Line()
	local := elem.LocalName()

	// Closed attribute-vocabulary check (§3.2.2, version-INDEPENDENT).
	c.checkAttrVocabulary(ctx, elem)

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

	// Schema Representation Constraint on the `use` attribute of <xs:attribute>
	// (version-INDEPENDENT — enforced in BOTH XSD 1.0 and 1.1). The schema for
	// schemas gives a top-level (global) attribute declaration the `topLevelAttribute`
	// type, which does NOT permit `use` (nor `form`/`ref`), so `use` on a global
	// attribute is a schema error. On a local attribute use `use` must be one of
	// the {optional, prohibited, required} enumeration; any other value (including
	// the empty string) is a schema error.
	if hasAttr(elem, attrUse) {
		switch {
		case isGlobalAttributeDecl(elem):
			c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
				"The attribute 'use' is not allowed."))
		default:
			switch getAttr(elem, attrUse) {
			case attrValOptional, attrValProhibited, attrValRequired:
			default:
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, "attribute", attrUse,
					"The value must be one of 'optional', 'prohibited', or 'required'."))
			}
		}
	}

	// Schema Representation Constraint on the `ref` attribute of <xs:attribute>
	// (version-INDEPENDENT — enforced in BOTH XSD 1.0 and 1.1, §3.2.2). A
	// top-level (global) attribute declaration is typed by the schema-for-schemas
	// `topLevelAttribute`, which omits `ref` (like `use`/`form`), so a `ref` on a
	// global attribute is a schema error. And `xs:attribute/@ref` is an xs:QName,
	// so a present value that is not a lexically valid QName (empty `ref=""`, a
	// leading-colon `:_`, a leading-digit `123`, …) is a schema-representation
	// error; without this an empty/malformed ref slips past prefix resolution and
	// silently resolves as an unprefixed reference.
	if hasAttr(elem, attrRef) {
		switch {
		case isGlobalAttributeDecl(elem):
			c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
				"The attribute 'ref' is not allowed."))
		default:
			// ref is already collapsed (collapsedAttr): reject a value still not a
			// valid QName after collapsing (empty, internal whitespace, a leading
			// colon, …); a padded ref=" p:a " is accepted.
			if !xmlchar.IsValidQName(ref) {
				c.reportInvalidQNameValue(ctx, elem, attrRef, ref)
			}
		}
	}

	if ref != "" {
		// name is not allowed alongside ref. @name is xs:NCName: a PRESENT-but-
		// collapse-empty name is instead an invalid (empty) NCName — ncnameCompanionUsable
		// reports that one value diagnostic and returns false, suppressing the
		// prohibition so present-empty stays symmetric with whitespace-only; a present
		// non-empty name still fires the prohibition.
		if c.ncnameCompanionUsable(ctx, elem, "attribute") {
			c.schemaError(ctx, schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'name' is not allowed."))
		}

		// type is not allowed alongside ref. @type is xs:QName: a PRESENT-but-collapse-
		// empty type is instead an invalid (empty) QName. The ref branch never routes
		// @type through a QName store site, so qnameCompanionUsable is the sole reporter
		// of that one value diagnostic (and returns false), suppressing the prohibition
		// so present-empty stays symmetric with whitespace-only; a present non-empty type
		// still fires the prohibition.
		if c.qnameCompanionUsable(ctx, elem, attrType) {
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
		// XSD Structures §3.2.2: the content of an <xs:attribute> that is NOT a
		// reference is (annotation?, simpleType?) — at most one annotation which
		// must be first, at most one simpleType, and no other element children.
		// Version-INDEPENDENT schema-representation rule (enforced in 1.0 and 1.1).
		c.checkAttributeChildren(ctx, elem)

		// xs:attribute/@name is xs:NCName (whiteSpace fixed "collapse"): read the
		// collapsed value — the same one parseAttributeUse/parseGlobalAttribute
		// registers — so the xmlns and NCName checks agree with storage. A padded
		// name="a " is valid; an internal-whitespace name="a b" fails NCName.
		cname := collapsedAttr(elem, attrName)
		if cname == lexicon.PrefixXMLNS {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, line, local, "attribute", "name",
				"The value of the attribute must not match 'xmlns'."))
		} else if hasAttr(elem, attrName) && !xmlchar.IsValidNCName(cname) {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, "attribute", "name",
				"The value '"+cname+"' is not a valid 'NCName'."))
		}

		// Schema Representation Constraint on the `form` attribute of
		// <xs:attribute> (version-INDEPENDENT — enforced in BOTH XSD 1.0 and 1.1).
		// The schema for schemas gives a top-level (global) attribute declaration
		// the `topLevelAttribute` type, which omits `form`, so `form` on a global
		// attribute is a schema error. On a local attribute declaration `form` must
		// be one of the {qualified, unqualified} `formChoice` enumeration; any other
		// value (including the empty string) is a schema error.
		if hasAttr(elem, attrForm) {
			switch {
			case isGlobalAttributeDecl(elem):
				c.schemaError(ctx, schemaParserError(c.diagSource(), line, local, "attribute",
					"The attribute 'form' is not allowed."))
			default:
				switch getAttr(elem, attrForm) {
				case attrValQualified, attrValUnqualified:
				default:
					c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), line, local, "attribute", attrForm,
						"The value must be one of 'qualified' or 'unqualified'."))
				}
			}
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

		// The `type` attribute of <xs:attribute> is an xs:QName (§3.2.2), so a
		// present value that is not a lexically valid QName (a leading-colon `:_`,
		// a leading-digit `123`, …) is a schema-representation error. Without this
		// such a value slips past prefix resolution and silently resolves as an
		// unprefixed reference. (Version-INDEPENDENT — enforced in 1.0 and 1.1. A
		// syntactically valid @type that resolves to a non-simple-type component is
		// a separate, resolution-time check.)
		if ct := collapsedAttr(elem, attrType); ct != "" {
			// xs:attribute/@type is an xs:QName (whiteSpace fixed "collapse"): validate
			// the COLLAPSED value so a padded type=" xs:string " is accepted, and reject
			// one still malformed after collapsing.
			if !xmlchar.IsValidQName(ct) {
				c.reportInvalidQNameValue(ctx, elem, attrType, ct)
			}
		}

		// type and inline simpleType are mutually exclusive. @type is xs:QName: a
		// PRESENT-but-collapse-empty type is an invalid (empty) QName (reported at the
		// store site, re-reported deduped here), so it does NOT trigger the
		// mutual-exclusion — present-empty stays symmetric with whitespace-only; a
		// present non-empty type still fires the mutual-exclusion.
		if c.qnameCompanionUsable(ctx, elem, attrType) {
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

// checkAttributeChildren enforces the schema-representation content model of a
// non-reference <xs:attribute> (XSD Structures §3.2.2): (annotation?, simpleType?).
// It reports (does not abort) an annotation that is not first or is duplicated, a
// second simpleType, and any other XSD-namespace element child (e.g. a nested
// xs:attribute/xs:element/xs:complexType). Foreign-namespace children are ignored.
// The type-vs-simpleType mutual-exclusion is enforced separately, so a simpleType
// child is accepted here as content-model-valid. Version-INDEPENDENT.
func (c *compiler) checkAttributeChildren(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	local := elem.LocalName()

	var annotationSeen bool
	var simpleTypeSeen bool
	var sawNonAnnotation bool
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemAnnotation):
			switch {
			case annotationSeen:
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), local, "attribute",
					"An attribute declaration must not have more than one annotation."))
			case sawNonAnnotation:
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), local, "attribute",
					"The annotation must appear before the simpleType."))
			default:
				annotationSeen = true
			}
		case isXSDElement(ce, elemSimpleType):
			sawNonAnnotation = true
			if simpleTypeSeen {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), local, "attribute",
					"An attribute declaration must not have more than one simpleType."))
			}
			simpleTypeSeen = true
		default:
			sawNonAnnotation = true
			if ce.URI() == lexicon.NamespaceXSD {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), local, "attribute",
					fmt.Sprintf("The element '%s' is not allowed as a child of an attribute declaration; expected one of annotation or simpleType.", ce.LocalName())))
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
	if !ok {
		return false
	}
	// A top-level <xs:attribute> is a child of <xs:schema>, OR — in XSD 1.1 — a
	// child of <xs:override>, which registers it (via parseGlobalAttribute) as a
	// wholesale replacement for a top-level attribute declaration. Both are GLOBAL
	// declarations typed by the schema-for-schemas topLevelAttribute (no `use`).
	// xs:override is 1.1-only, so this does not affect XSD 1.0.
	return isXSDElement(parent, elemSchema) || isXSDElement(parent, elemOverride)
}

func (c *compiler) localAttributeUnderNonAnyTypeRestriction(ctx context.Context, elem *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](elem.Parent())
	if !ok || !isXSDElement(parent, elemRestriction) || !isContentDerivationRestriction(parent) {
		return false
	}
	// @base is an xs:QName: gate on PRESENCE and route the value through the QName
	// chokepoint so a PRESENT-but-empty base="" behaves like the whitespace-only
	// base="   " — both collapse to the invalidQName sentinel, whose invalid-QName
	// diagnostic already fired, so this secondary targetNamespace check does not
	// emit its own error (isInvalidQName → suppress).
	if !hasAttr(parent, attrBase) {
		return false
	}
	qn := c.resolveQName(ctx, parent, attrBase, getAttr(parent, attrBase))
	return isInvalidQName(qn) || qn.NS != lexicon.NamespaceXSD || qn.Local != typeAnyType
}

func isContentDerivationRestriction(elem *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](elem.Parent())
	return ok && (isXSDElement(parent, elemSimpleContent) || isXSDElement(parent, elemComplexContent))
}
