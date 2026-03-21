package xsd

import (
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'name' is required but missing."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// ref is not allowed at global level.
	if getAttr(elem, "ref") != "" {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'ref' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// minOccurs is not allowed at global level.
	if getAttr(elem, "minOccurs") != "" {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'minOccurs' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// maxOccurs is not allowed at global level.
	if getAttr(elem, "maxOccurs") != "" {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'maxOccurs' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// form is not allowed at global level.
	if getAttr(elem, "form") != "" {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'form' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Validate 'final' attribute value.
	if v := getAttr(elem, "final"); v != "" {
		if !isValidFinal(v) {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "final",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// Validate 'block' attribute value.
	if v := getAttr(elem, "block"); v != "" {
		if !isValidBlock(v) {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// default and fixed are mutually exclusive.
	if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// type and inline complexType/simpleType are mutually exclusive.
	if getAttr(elem, "type") != "" {
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "complexType") {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <complexType> child are mutually exclusive."), helium.ErrorLevelFatal))
				c.errorCount++
			}
			if isXSDElement(ce, "simpleType") {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <simpleType> child are mutually exclusive."), helium.ErrorLevelFatal))
				c.errorCount++
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// minOccurs > maxOccurs check.
		if minOcc != "" && maxOcc != "" && maxOcc != "unbounded" {
			minVal := parseOccurs(minOcc, 1)
			maxVal := parseOccurs(maxOcc, 1)
			if minVal > maxVal {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value must not be greater than the value of 'maxOccurs'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// ref and name are mutually exclusive.
		if name != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
				"The attributes 'ref' and 'name' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// Report first ref-restricted attribute found (alphabetical order).
		notAllowedWithRef := []string{"abstract", "block", "default", "final", "fixed", "form", "nillable", "substitutionGroup", "type"}
		for _, attr := range notAllowedWithRef {
			if getAttr(elem, attr) != "" {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", attr,
					"Only the attributes 'minOccurs', 'maxOccurs' and 'id' are allowed in addition to 'ref'."), helium.ErrorLevelFatal))
				c.errorCount++
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The content is not valid. Expected is (annotation?)."), helium.ErrorLevelFatal))
				c.errorCount++
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
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// Some attributes not allowed for local named elements.
		localNotAllowed := []string{"abstract", "substitutionGroup", "final"}
		for _, attr := range localNotAllowed {
			if getAttr(elem, attr) != "" {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
					"The attribute '"+attr+"' is not allowed."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// Validate 'block' attribute value.
		if v := getAttr(elem, "block"); v != "" && !isValidBlock(v) {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// default and fixed mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
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
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The attribute 'type' and the <complexType> child are mutually exclusive."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			} else if isXSDElement(ce, "simpleType") {
				if hasType {
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."), helium.ErrorLevelFatal))
					c.errorCount++
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
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'name' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// type not allowed with ref.
		if getAttr(elem, "type") != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'type' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// form not allowed with ref.
		if getAttr(elem, "form") != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'form' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// simpleType child not allowed with ref.
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			if isXSDElement(ce, "simpleType") {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
					"The content is not valid. Expected is (annotation?)."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}
	} else {
		// Attribute name must not be "xmlns".
		if getAttr(elem, "name") == "xmlns" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "attribute", "name",
				"The value of the attribute must not match 'xmlns'."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// Qualified attribute must not be in the XSI namespace.
		form := getAttr(elem, "form")
		if form == "qualified" || (form == "" && c.schema.attrFormQualified) {
			if c.schema.targetNamespace == "http://www.w3.org/2001/XMLSchema-instance" {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
					"The target namespace must not match 'http://www.w3.org/2001/XMLSchema-instance'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// default and fixed are mutually exclusive.
		if getAttr(elem, "default") != "" && getAttr(elem, "fixed") != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// If default is present, use must be optional (or absent, which defaults to optional).
		if getAttr(elem, "default") != "" {
			use := getAttr(elem, "use")
			if use != "" && use != "optional" {
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
					"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present."), helium.ErrorLevelFatal))
				c.errorCount++
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
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"The attribute 'type' and the <simpleType> child are mutually exclusive."), helium.ErrorLevelFatal))
					c.errorCount++
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "annotation",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "annotation",
			"The content is not valid. Expected is (appinfo | documentation)*."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "appinfo",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, line, local, "documentation",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Validate xml:lang value after attribute checks.
	if langValue != "" && !languageRegex.MatchString(langValue) {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "documentation",
			"{http://www.w3.org/XML/1998/namespace}lang",
			"'"+langValue+"' is not a valid value of the atomic type 'xs:language'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}
