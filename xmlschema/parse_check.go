package xmlschema

import (
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
		// maxOccurs must be >= 1.
		if maxOcc != "" && maxOcc != "unbounded" {
			maxVal := parseOccurs(maxOcc, 1)
			if maxVal < 1 {
				c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."))
			}
		}

		// ref and name are mutually exclusive.
		if name != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
				"The attributes 'ref' and 'name' are mutually exclusive."))
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

		// When ref is present, only minOccurs, maxOccurs, and id are allowed.
		notAllowedWithRef := []string{"abstract", "block", "default", "final", "fixed", "form", "nillable", "substitutionGroup", "type"}
		for _, attr := range notAllowedWithRef {
			if getAttr(elem, attr) != "" {
				// "abstract" uses a different error format in libxml2.
				if attr == "abstract" || attr == "substitutionGroup" || attr == "final" {
					c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
						"The attribute '"+attr+"' is not allowed."))
				} else {
					c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", attr,
						"Only the attributes 'minOccurs', 'maxOccurs' and 'id' are allowed in addition to 'ref'."))
				}
			}
		}

		// Validate 'block' attribute value (even though it's not allowed with ref).
		if v := getAttr(elem, "block"); v != "" && !isValidBlock(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}

		// default and fixed mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// Children not allowed with ref (except annotation).
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "complexType") || isXSDElement(ce, "simpleType") {
				c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The content is not valid. Expected is (annotation?)."))
			}
		}
	} else if name != "" {
		// Named local element constraints.

		// default and fixed mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.schemaErrors.WriteString(schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."))
		}

		// Validate 'block' attribute value.
		if v := getAttr(elem, "block"); v != "" && !isValidBlock(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		}

		// Validate 'final' attribute value.
		if v := getAttr(elem, "final"); v != "" && !isValidFinal(v) {
			c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, line, local, "element", "final",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction))'."))
		}

		// type and inline complexType/simpleType mutually exclusive.
		if getAttr(elem, "type") != "" {
			for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
				if child.Type() != helium.ElementNode {
					continue
				}
				ce := child.(*helium.Element)
				if isXSDElement(ce, "complexType") || isXSDElement(ce, "simpleType") {
					c.schemaErrors.WriteString(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."))
				}
			}
		}

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
