package xsd

import (
	"context"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// hasAttr checks whether an attribute is physically present on the element.
// Unlike getAttr, this distinguishes absent from empty-string.
func hasAttr(elem *helium.Element, name string) bool {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'name' is required but missing."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// ref is not allowed at global level.
	if getAttr(elem, attrRef) != "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'ref' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// minOccurs is not allowed at global level.
	if getAttr(elem, attrMinOccurs) != "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'minOccurs' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// maxOccurs is not allowed at global level.
	if getAttr(elem, attrMaxOccurs) != "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'maxOccurs' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// form is not allowed at global level.
	if getAttr(elem, attrForm) != "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attribute 'form' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Validate 'final' attribute value.
	if v := getAttr(elem, attrFinal); v != "" {
		if !isValidFinal(v) {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, elemElement, attrFinal,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// Validate 'block' attribute value.
	if v := getAttr(elem, attrBlock); v != "" {
		if !isValidBlock(v) {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, elemElement, attrBlock,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// default and fixed are mutually exclusive.
	if getAttr(elem, attrDefault) != "" && getAttr(elem, attrFixed) != "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
			"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// type and inline complexType/simpleType are mutually exclusive.
	if getAttr(elem, attrType) != "" {
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element) //nolint:forcetypeassert
			if isXSDElement(ce, "complexType") {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <complexType> child are mutually exclusive."), helium.ErrorLevelFatal))
				c.errorCount++
			}
			if isXSDElement(ce, "simpleType") {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
					"The attribute 'type' and the <simpleType> child are mutually exclusive."), helium.ErrorLevelFatal))
				c.errorCount++
			}
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

	if ref != "" {
		// Matches libxml2 ordering for ref elements (src-element 2.2):
		// 1. maxOccurs >= 1, minOccurs > maxOccurs
		// 2. ref+name conflict
		// 3. First ref-restricted attribute (alphabetical)
		// 4. First child content error

		// maxOccurs must be >= 1.
		if maxOcc != "" && maxOcc != attrValUnbounded {
			maxVal := parseOccurs(maxOcc, 1)
			if maxVal < 1 {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// minOccurs > maxOccurs check.
		if minOcc != "" && maxOcc != "" && maxOcc != attrValUnbounded {
			minVal := parseOccurs(minOcc, 1)
			maxVal := parseOccurs(maxOcc, 1)
			if minVal > maxVal {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "minOccurs",
					"The value must not be greater than the value of 'maxOccurs'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// ref and name are mutually exclusive.
		if name != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
				"The attributes 'ref' and 'name' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// Report first ref-restricted attribute found (alphabetical order).
		notAllowedWithRef := []string{attrAbstract, attrBlock, attrDefault, attrFinal, attrFixed, attrForm, attrNillable, attrSubstitutionGroup, attrType}
		for _, attr := range notAllowedWithRef {
			if getAttr(elem, attr) != "" {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", attr,
					"Only the attributes 'minOccurs', 'maxOccurs' and 'id' are allowed in addition to 'ref'."), helium.ErrorLevelFatal))
				c.errorCount++
				break // only report first
			}
		}

		// First child not allowed with ref (except annotation).
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element) //nolint:forcetypeassert
			if isXSDElement(ce, elemComplexType) || isXSDElement(ce, elemSimpleType) {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
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
		if maxOcc != "" && maxOcc != attrValUnbounded {
			maxVal := parseOccurs(maxOcc, 1)
			if maxVal < 1 {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "maxOccurs",
					"The value must be greater than or equal to 1."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// Some attributes not allowed for local named elements.
		localNotAllowed := []string{attrAbstract, attrSubstitutionGroup, attrFinal}
		for _, attr := range localNotAllowed {
			if getAttr(elem, attr) != "" {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
					"The attribute '"+attr+"' is not allowed."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// Validate 'block' attribute value.
		if v := getAttr(elem, attrBlock); v != "" && !isValidBlock(v) {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "element", "block",
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// default and fixed mutually exclusive.
		if getAttr(elem, attrDefault) != "" && getAttr(elem, attrFixed) != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "element",
				"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// type and inline complexType/simpleType checks.
		hasType := getAttr(elem, attrType) != ""
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element) //nolint:forcetypeassert
			if isXSDElement(ce, elemComplexType) {
				if hasType {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The attribute 'type' and the <complexType> child are mutually exclusive."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			} else if isXSDElement(ce, elemSimpleType) {
				if hasType {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "element",
						"The content is not valid. Expected is (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*))."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
		}
	}
}

// checkAttributeUse validates constraints on an xs:attribute declaration.
func (c *compiler) checkAttributeUse(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	ref := getAttr(elem, attrRef)
	line := elem.Line()
	local := elem.LocalName()

	if ref != "" {
		// ref and name are mutually exclusive.
		if getAttr(elem, attrName) != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'name' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// type not allowed with ref.
		if getAttr(elem, attrType) != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'type' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// form not allowed with ref.
		if getAttr(elem, attrForm) != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attribute 'form' is not allowed."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// simpleType child not allowed with ref.
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element) //nolint:forcetypeassert
			if isXSDElement(ce, elemSimpleType) {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
					"The content is not valid. Expected is (annotation?)."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}
	} else {
		// Attribute name must not be "xmlns".
		if getAttr(elem, attrName) == lexicon.PrefixXMLNS {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "attribute", "name",
				"The value of the attribute must not match 'xmlns'."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// Qualified attribute must not be in the XSI namespace.
		form := getAttr(elem, attrForm)
		if form == "qualified" || (form == "" && c.schema.attrFormQualified) {
			if c.schema.targetNamespace == lexicon.NamespaceXSI {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
					"The target namespace must not match '"+lexicon.NamespaceXSI+"'."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// default and fixed are mutually exclusive.
		if getAttr(elem, attrDefault) != "" && getAttr(elem, attrFixed) != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
				"The attributes 'default' and 'fixed' are mutually exclusive."), helium.ErrorLevelFatal))
			c.errorCount++
		}

		// If default is present, use must be optional (or absent, which defaults to optional).
		if getAttr(elem, attrDefault) != "" {
			use := getAttr(elem, attrUse)
			if use != "" && use != attrValOptional {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "attribute",
					"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present."), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}

		// type and inline simpleType are mutually exclusive.
		if getAttr(elem, attrType) != "" {
			for child := range helium.Children(elem) {
				if child.Type() != helium.ElementNode {
					continue
				}
				ce := child.(*helium.Element) //nolint:forcetypeassert
				if isXSDElement(ce, elemSimpleType) {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"The attribute 'type' and the <simpleType> child are mutually exclusive."), helium.ErrorLevelFatal))
					c.errorCount++
				}
			}
		}
	}
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "annotation",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "annotation",
			"The content is not valid. Expected is (appinfo | documentation)*."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Check children (appinfo, documentation).
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element) //nolint:forcetypeassert
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "appinfo",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, line, local, "documentation",
			"The attribute '"+name+"' is not allowed."), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Validate xml:lang value after attribute checks.
	if langValue != "" && !languageRegex.MatchString(langValue) {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, "documentation",
			helium.ClarkName(lexicon.NamespaceXML, lexicon.AttrLang),
			"'"+langValue+"' is not a valid value of the atomic type 'xs:language'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}
