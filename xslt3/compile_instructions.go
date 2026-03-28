package xslt3

import (
	"context"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// Allowed attribute sets for XSLT elements (unprefixed attributes only).
var (
	withParamAllowedAttrs = map[string]struct{}{
		"name": {}, "select": {}, "as": {}, "tunnel": {},
	}
	paramAllowedAttrs = map[string]struct{}{
		"name": {}, "select": {}, "as": {}, "required": {}, "tunnel": {}, "static": {},
	}
	variableAllowedAttrs = map[string]struct{}{
		"name": {}, "select": {}, "as": {}, "static": {}, "visibility": {},
	}
	// XSLT-namespace attributes allowed on literal result elements
	lreAllowedXSLTAttrs = map[string]struct{}{
		"use-attribute-sets":         {},
		"expand-text":                {},
		"xpath-default-namespace":    {},
		"exclude-result-prefixes":    {},
		"extension-element-prefixes": {},
		"version":                    {},
		"type":                       {},
		"validation":                 {},
		"default-collation":          {},
		"default-mode":               {},
		"default-validation":         {},
		"inherit-namespaces":         {},
		"use-when":                   {},
	}
)

// validateBooleanAttr checks that a boolean attribute value is valid xs:boolean.
// Valid values are "yes", "no", "true", "false", "1", "0" (with optional whitespace).
// Returns XTSE0020 if the value is not valid.
func validateBooleanAttr(elemName, attrName, value string) error {
	if _, ok := parseXSDBool(value); !ok {
		return staticError(errCodeXTSE0020, "%q is not a valid value for %s/@%s", value, elemName, attrName)
	}
	return nil
}

// xsltStandardAttrs are standard attributes that may appear on any XSLT element
// per XSLT 3.0 §3.5.
var xsltStandardAttrs = map[string]struct{}{
	"default-collation":          {},
	"default-mode":               {},
	"default-validation":         {},
	"exclude-result-prefixes":    {},
	"expand-text":                {},
	"extension-element-prefixes": {},
	"use-when":                   {},
	"xpath-default-namespace":    {},
	"version":                    {},
}

// validateXSLTAttrs checks that an XSLT element has only allowed unprefixed attributes
// and no attributes in the XSLT namespace. Returns XTSE0090 for unknown attributes.
// Standard attributes (use-when, expand-text, etc.) are always allowed per XSLT 3.0 §3.5.
// In forwards-compatible mode (version > 3.0), unknown attributes are silently ignored
// per XSLT 3.0 §3.8.
func (c *compiler) validateXSLTAttrs(_ context.Context, elem *helium.Element, allowed map[string]struct{}) error {
	fc := isForwardsCompatible(c.effectiveVersion)
	if !fc {
		// Also check the element's own version attribute
		if elemVer := getAttr(elem, "version"); elemVer != "" {
			fc = isForwardsCompatible(elemVer)
		}
	}
	for _, attr := range elem.Attributes() {
		// Attributes in the XSLT namespace are not allowed on XSLT elements
		if attr.URI() == lexicon.NamespaceXSLT {
			if fc {
				continue
			}
			return staticError(errCodeXTSE0090,
				"attribute %q in the XSLT namespace is not allowed on xsl:%s", attr.LocalName(), elem.LocalName())
		}
		// Skip attributes in other (non-null) namespaces — those are extension attributes
		if attr.URI() != "" {
			continue
		}
		name := attr.LocalName()
		// Shadow attributes (prefixed with _) are compile-time AVTs
		if strings.HasPrefix(name, "_") {
			continue
		}
		if _, ok := allowed[name]; ok {
			continue
		}
		if _, ok := xsltStandardAttrs[name]; ok {
			continue
		}
		if fc {
			continue
		}
		return staticError(errCodeXTSE0090,
			"attribute %q is not allowed on xsl:%s", name, elem.LocalName())
	}
	return nil
}

// validateValidationAttr checks that the validation attribute value is one of
// the four legal values. Returns XTSE0020 for invalid values.
func validateValidationAttr(elemName, validation string) error {
	switch validation {
	case validationStrict, validationLax, validationPreserve, validationStrip:
		return nil
	default:
		return staticError(errCodeXTSE0020, "invalid value %q for %s/@validation: must be strict, lax, preserve, or strip", validation, elemName)
	}
}

// checkTypeAttrSchemaAware verifies that the type attribute is only used when
// the stylesheet has imported at least one schema, unless typeName resolves to
// a built-in XSD type (xs: / xsd: namespace). Returns XTSE0215 otherwise.
func (c *compiler) checkTypeAttrSchemaAware(_ context.Context, elemName, typeName string) error {
	if isBuiltinXSDTypeName(typeName, c.nsBindings) {
		// Built-in XSD types are always valid; mark schema-aware so
		// type annotations are tracked at runtime.
		c.stylesheet.schemaAware = true
		return nil
	}
	if len(c.stylesheet.schemas) > 0 || c.stylesheet.schemaAware {
		return nil
	}
	return staticError(errCodeXTSE0215, "%s/@type requires a schema import (xsl:import-schema)", elemName)
}

// isBuiltinXSDTypeName reports whether typeName refers to the built-in XSD
// namespace (http://www.w3.org/2001/XMLSchema). It recognises the conventional
// xs: and xsd: prefixes directly, and also resolves any other prefix via ns.
func isBuiltinXSDTypeName(typeName string, ns map[string]string) bool {
	if strings.HasPrefix(typeName, "xs:") || strings.HasPrefix(typeName, "xsd:") {
		return true
	}
	idx := strings.IndexByte(typeName, ':')
	if idx < 0 {
		return false
	}
	prefix := typeName[:idx]
	uri, ok := ns[prefix]
	return ok && uri == lexicon.NamespaceXSD
}

// checkValidationTypeExclusive verifies that validation and type are not both
// specified on the same instruction. Returns XTSE0220 if both are present.
func checkValidationTypeExclusive(elemName, validation, typeName string) error {
	if validation != "" && typeName != "" {
		return staticError(errCodeXTSE0220, "%s: validation and type attributes are mutually exclusive", elemName)
	}
	return nil
}

// compileInstruction compiles a single element into an instruction.
func (c *compiler) compileInstruction(ctx context.Context, elem *helium.Element) (instruction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Push element-local namespace declarations into scope
	saved := c.pushElementNamespaces(ctx, elem)
	defer func() { c.nsBindings = saved }()

	// Evaluate use-when: on XSLT elements check "use-when" attribute,
	// on LREs check "xsl:use-when" (in XSLT namespace).
	if elem.URI() == lexicon.NamespaceXSLT {
		if uw := getAttr(elem, "use-when"); uw != "" {
			include, err := c.evaluateUseWhen(ctx, uw)
			if err != nil {
				return nil, err
			}
			if !include {
				return nil, nil //nolint:nilnil
			}
		}
	} else {
		if uw, ok := elem.GetAttributeNS("use-when", lexicon.NamespaceXSLT); ok {
			include, err := c.evaluateUseWhen(ctx, uw)
			if err != nil {
				return nil, err
			}
			if !include {
				return nil, nil //nolint:nilnil
			}
		}
	}

	// XTSE0090: attributes in the XSLT namespace are not allowed on XSLT elements
	if elem.URI() == lexicon.NamespaceXSLT {
		for _, attr := range elem.Attributes() {
			if attr.URI() == lexicon.NamespaceXSLT {
				return nil, staticError(errCodeXTSE0090,
					"attribute %q in the XSLT namespace is not allowed on xsl:%s", attr.LocalName(), elem.LocalName())
			}
		}
	}

	// Resolve shadow attributes: _foo="avt" → foo="evaluated value"
	// Shadow attributes are evaluated at compile time using static params.
	if elem.URI() == lexicon.NamespaceXSLT {
		if err := c.resolveShadowAttributes(ctx, elem); err != nil {
			return nil, err
		}
	}

	// Handle version inheritance for forwards-compatible processing
	savedVersion := c.effectiveVersion
	if ver := getAttr(elem, "version"); ver != "" {
		c.effectiveVersion = ver
	}
	defer func() { c.effectiveVersion = savedVersion }()

	// Handle xml:space inheritance
	savedPreserve := c.preserveSpace
	if xs := getAttr(elem, lexicon.QNameXMLSpace); xs != "" {
		c.preserveSpace = (xs == lexicon.SpacePreserve)
	}
	defer func() { c.preserveSpace = savedPreserve }()

	// Handle expand-text inheritance (check both unprefixed and xsl:-prefixed for LREs)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else if elem.URI() == lexicon.NamespaceXSLT {
			// XTSE0020: invalid boolean value for expand-text on XSLT element
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:%s/@expand-text", et, elem.LocalName())
		}
	} else if et, ok := elem.GetAttributeNS("expand-text", lexicon.NamespaceXSLT); ok {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			// XTSE0020: invalid boolean value for xsl:expand-text on LRE
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:expand-text", et)
		}
	}
	defer func() { c.expandText = savedExpandText }()

	// Handle per-instruction xpath-default-namespace
	// On XSLT elements: check unprefixed attribute.
	// On LREs: only check xsl:-prefixed (in XSLT namespace).
	savedXPathDefaultNS := c.xpathDefaultNS
	hasLocalXPNS := false
	if elem.URI() == lexicon.NamespaceXSLT {
		if xdn, ok := elem.GetAttribute("xpath-default-namespace"); ok {
			c.xpathDefaultNS = xdn
			hasLocalXPNS = true
		}
	} else if xdn, ok := elem.GetAttributeNS("xpath-default-namespace", lexicon.NamespaceXSLT); ok {
		c.xpathDefaultNS = xdn
		hasLocalXPNS = true
	}
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	// Handle per-instruction default-collation
	savedDefaultCollation := c.defaultCollation
	if dc, ok := elem.GetAttribute("default-collation"); ok {
		if uri := resolveDefaultCollation(dc); uri != "" {
			c.defaultCollation = uri
		}
	} else if dc, ok := elem.GetAttributeNS("default-collation", lexicon.NamespaceXSLT); ok {
		if uri := resolveDefaultCollation(dc); uri != "" {
			c.defaultCollation = uri
		}
	}
	defer func() { c.defaultCollation = savedDefaultCollation }()

	// Handle per-instruction default-mode (XSLT 3.0 standard attribute)
	// On XSLT elements: default-mode="..."
	// On LREs: xsl:default-mode="..."
	// Mode names are stored as raw QNames (not Clark notation) for consistency
	// with how template mode attributes are stored.
	savedDefaultMode := c.defaultMode
	if dm := getAttr(elem, "default-mode"); dm != "" {
		c.defaultMode = dm
	} else if dm, ok := elem.GetAttributeNS("default-mode", lexicon.NamespaceXSLT); ok {
		c.defaultMode = dm
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	// Handle extension-element-prefixes on both XSLT instructions and LREs.
	// On XSLT elements: extension-element-prefixes="..."
	// On LREs: xsl:extension-element-prefixes="..." — must be processed
	// before the extension element check so that an LRE declaring its own
	// namespace as extension is recognized as an extension element.
	savedExtURIs := c.extensionURIs
	if elem.URI() == lexicon.NamespaceXSLT {
		if eep := getAttr(elem, "extension-element-prefixes"); eep != "" {
			newExtURIs := make(map[string]struct{})
			for k, v := range c.extensionURIs {
				newExtURIs[k] = v
			}
			for _, prefix := range strings.Fields(eep) {
				if uri, ok := c.nsBindings[prefix]; ok && uri != "" {
					newExtURIs[uri] = struct{}{}
				}
			}
			c.extensionURIs = newExtURIs
		}
	} else if eep, ok := elem.GetAttributeNS("extension-element-prefixes", lexicon.NamespaceXSLT); ok {
		newExtURIs := make(map[string]struct{})
		for k, v := range c.extensionURIs {
			newExtURIs[k] = v
		}
		for _, prefix := range strings.Fields(eep) {
			if uri, uriOK := c.nsBindings[prefix]; uriOK && uri != "" {
				newExtURIs[uri] = struct{}{}
			}
		}
		c.extensionURIs = newExtURIs
	}
	defer func() { c.extensionURIs = savedExtURIs }()

	var inst instruction
	var err error
	if elem.URI() == lexicon.NamespaceXSLT {
		inst, err = c.compileXSLTInstruction(ctx, elem)
		if err != nil {
			return nil, err
		}
	} else if uri := elem.URI(); uri != "" && c.extensionURIs != nil {
		if _, isExt := c.extensionURIs[uri]; isExt {
			// Extension element: compile xsl:fallback children.
			return c.compileForwardsCompat(ctx, elem)
		}
		inst, err = c.compileLiteralResultElement(ctx, elem)
		if err != nil {
			return nil, err
		}
	} else {
		inst, err = c.compileLiteralResultElement(ctx, elem)
		if err != nil {
			return nil, err
		}
	}
	// Store effective xpath-default-namespace on instructions that support it
	c.setInstructionXPathNS(ctx, inst, hasLocalXPNS)
	// Record source location for $err:line-number / $err:module in xsl:catch
	if si, ok := inst.(interface{ setSourceInfo(int, string) }); ok {
		module := ""
		if doc := elem.OwnerDocument(); doc != nil {
			module = doc.URL()
		}
		si.setSourceInfo(elem.Line(), module)
	}
	// Compute effective static base URI from xml:base on the stylesheet element.
	// This is set generically so that static-base-uri() returns the correct
	// value for any XSLT instruction or LRE that carries xml:base.
	if effectiveBase := stylesheetBaseURI(elem, c.baseURI); effectiveBase != c.baseURI {
		if si, ok := inst.(interface{ getStaticBaseURI() string }); ok {
			// Only set if not already set by a specific compile function
			if si.getStaticBaseURI() == "" {
				if setter, ok2 := inst.(interface{ setStaticBaseURI(string) }); ok2 {
					setter.setStaticBaseURI(effectiveBase)
				}
			}
		}
	}
	// Wrap in collation scope if default-collation changed on this element.
	// This ensures the runtime ec.defaultCollation is set correctly for
	// XPath expressions evaluated by the wrapped instruction.
	if c.defaultCollation != savedDefaultCollation {
		inst = &collationScopeInst{
			DefaultCollation: c.defaultCollation,
			Inner:            inst,
		}
	}
	return inst, nil
}

// setInstructionXPathNS stores the current xpath-default-namespace on
// instructions that embed xpathNS.
func (c *compiler) setInstructionXPathNS(_ context.Context, inst instruction, hasLocal bool) {
	set := func(ns *xpathNS) {
		ns.XPathDefaultNS = c.xpathDefaultNS
		// Mark as set when either explicitly declared locally or inherited non-empty
		ns.HasXPathDefaultNS = hasLocal || c.xpathDefaultNS != ""
		ns.HasLocalXPathDefaultNS = hasLocal
	}
	switch v := inst.(type) {
	case *applyTemplatesInst:
		set(&v.xpathNS)
	case *ifInst:
		set(&v.xpathNS)
	case *valueOfInst:
		set(&v.xpathNS)
	case *forEachInst:
		set(&v.xpathNS)
	case *chooseInst:
		set(&v.xpathNS)
	case *evaluateInst:
		set(&v.xpathNS)
	case *literalResultElement:
		set(&v.xpathNS)
	}
}

// validateDescendantUseWhen recursively validates use-when XPath expressions
// on all descendant elements. This is used for elements like xsl:fallback whose
// children are not compiled but whose use-when expressions must still be
// syntactically valid per XSLT 3.0.
func (c *compiler) validateDescendantUseWhen(ctx context.Context, parent *helium.Element) error {
	for child := range helium.Children(parent) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Check use-when on XSLT elements and xsl:use-when on LREs
		var uw string
		if childElem.URI() == lexicon.NamespaceXSLT {
			uw = getAttr(childElem, "use-when")
		} else {
			uw, _ = childElem.GetAttributeNS("use-when", lexicon.NamespaceXSLT)
		}
		if uw != "" {
			if _, err := c.evaluateUseWhen(ctx, uw); err != nil {
				return err
			}
		}
		if err := c.validateDescendantUseWhen(ctx,childElem); err != nil {
			return err
		}
	}
	return nil
}

// evaluateUseWhen evaluates a use-when XPath expression at compile time.
// Returns (true, nil) to include the element, (false, nil) to exclude,
// or (false, error) to propagate a compile-time error.
// Provides XSLT static context functions: function-available, system-property,
// type-available, element-available per XSLT 3.0 §3.14.
func (c *compiler) evaluateUseWhen(ctx context.Context, expr string) (bool, error) {
	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		// XPST0003: invalid XPath in use-when is a static error
		return false, staticError(errCodeXPST0003,
			"invalid XPath expression in use-when: %s: %v", expr, err)
	}

	eval := c.useWhenEvaluator(ctx)

	result, err := eval.Evaluate(ctx, compiled, nil)
	if err != nil {
		// Propagate all errors from use-when evaluation.
		// XSLT 3.0 spec requires static errors (XPST0003, XPST0008,
		// XPST0017, XPST0051, XPST0081) and dynamic errors to be reported.
		return false, err
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return true, nil //nolint:nilerr // use-when defaults to true on EBV failure
	}
	return b, nil
}

func (c *compiler) useWhenFunctionAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil //nolint:nilerr // function-available returns false on atomization failure
	}
	name, _ := xpath3.AtomicToString(av)

	// XTDE1400: function name must be a valid QName
	if !strings.HasPrefix(name, "Q{") && !isValidQName(name) {
		return nil, dynamicError(errCodeXTDE1400,
			"function-available: %q is not a valid QName", name)
	}

	// Check for undeclared prefix (XTDE1400)
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		if _, ok := c.nsBindings[prefix]; !ok {
			return nil, dynamicError(errCodeXTDE1400,
				"undeclared namespace prefix %q in function-available(%q)", prefix, name)
		}
	}

	// Only XSLT static context functions are available in use-when.
	// Runtime-only functions (current, key, document, generate-id, etc.)
	// are NOT available in the use-when static context per XSLT 3.0 spec 3.4.6.
	switch name {
	case "function-available", "system-property", "type-available",
		"element-available", "available-system-properties":
		return xpath3.SingleBoolean(true), nil
	}

	// XPath built-in functions: resolve prefixed names to namespace URI.
	localName := name
	ns := ""
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		localName = name[idx+1:]
		ns = c.nsBindings[prefix] // already validated above
	} else if strings.HasPrefix(name, "Q{") {
		// EQName: Q{uri}local
		end := strings.IndexByte(name, '}')
		if end >= 0 {
			ns = name[2:end]
			localName = name[end+1:]
		}
	}
	if ns != "" {
		if xpath3.IsBuiltinFunctionNS(ns, localName) {
			return xpath3.SingleBoolean(true), nil
		}
	} else {
		if xpath3.IsBuiltinFunction(localName) {
			return xpath3.SingleBoolean(true), nil
		}
	}

	return xpath3.SingleBoolean(false), nil
}

func (c *compiler) useWhenSystemProperty(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleString(""), nil //nolint:nilerr // system-property returns "" on atomization failure
	}
	name, _ := xpath3.AtomicToString(av)

	resolved := resolveQName(name, c.nsBindings)
	switch resolved {
	case helium.ClarkName(lexicon.NamespaceXSLT, "version"):
		return xpath3.SingleString("3.0"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "vendor"):
		return xpath3.SingleString("helium"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "vendor-url"):
		return xpath3.SingleString("https://github.com/lestrrat-go/helium"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "product-name"):
		return xpath3.SingleString("helium"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "product-version"):
		return xpath3.SingleString("0.1"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "is-schema-aware"):
		return xpath3.SingleString(lexicon.ValueYes), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-serialization"):
		return xpath3.SingleString(lexicon.ValueYes), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-backwards-compatibility"):
		return xpath3.SingleString(lexicon.ValueNo), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-namespace-axis"):
		return xpath3.SingleString(lexicon.ValueYes), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-streaming"):
		return xpath3.SingleString(lexicon.ValueNo), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-dynamic-evaluation"):
		return xpath3.SingleString(lexicon.ValueYes), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "supports-higher-order-functions"):
		return xpath3.SingleString(lexicon.ValueYes), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "xpath-version"):
		return xpath3.SingleString("3.1"), nil
	case helium.ClarkName(lexicon.NamespaceXSLT, "xsd-version"):
		return xpath3.SingleString("1.1"), nil
	}
	return xpath3.SingleString(""), nil
}

func (c *compiler) useWhenTypeAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil //nolint:nilerr // type-available returns false on atomization failure
	}
	name, _ := xpath3.AtomicToString(av)
	resolved := resolveQName(name, c.nsBindings)
	return xpath3.SingleBoolean(xpath3.IsKnownXSDType(resolved)), nil
}

func (c *compiler) useWhenElementAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil //nolint:nilerr // element-available returns false on atomization failure
	}
	name, _ := xpath3.AtomicToString(av)
	resolved := resolveQName(name, c.nsBindings)
	// Check if it's a known XSLT instruction
	if strings.HasPrefix(resolved, "{"+lexicon.NamespaceXSLT+"}") {
		local := resolved[len("{"+lexicon.NamespaceXSLT+"}"):]
		switch local {
		case lexicon.XSLTElementApplyTemplates, lexicon.XSLTElementCallTemplate, lexicon.XSLTElementApplyImports,
			lexicon.XSLTElementForEach, lexicon.XSLTElementValueOf, lexicon.XSLTElementCopyOf, lexicon.XSLTElementNumber, lexicon.XSLTElementChoose,
			lexicon.XSLTElementIf, lexicon.XSLTElementText, lexicon.XSLTElementCopy, lexicon.XSLTElementVariable, lexicon.XSLTElementMessage, lexicon.XSLTElementFallback,
			lexicon.XSLTElementProcessingInstruction, lexicon.XSLTElementComment, lexicon.XSLTElementElement, lexicon.XSLTElementAttribute,
			lexicon.XSLTElementSort, lexicon.XSLTElementOutput, lexicon.XSLTElementKey, lexicon.XSLTElementTemplate, lexicon.XSLTElementParam, lexicon.XSLTElementWithParam,
			lexicon.XSLTElementForEachGroup, lexicon.XSLTElementSequence, lexicon.XSLTElementDocument, lexicon.XSLTElementResultDocument,
			lexicon.XSLTElementAnalyzeString, lexicon.XSLTElementNamespace, lexicon.XSLTElementPerformSort, lexicon.XSLTElementNextMatch,
			lexicon.XSLTElementTry, lexicon.XSLTElementIterate, lexicon.XSLTElementSourceDocument, lexicon.XSLTElementMerge, lexicon.XSLTElementOnEmpty,
			lexicon.XSLTElementOnNonEmpty, lexicon.XSLTElementWherePopulated, lexicon.XSLTElementEvaluate, lexicon.XSLTElementAssert,
			lexicon.XSLTElementMap, lexicon.XSLTElementMapEntry, lexicon.XSLTElementBreak, lexicon.XSLTElementNextIteration:
			return xpath3.SingleBoolean(true), nil
		}
	}
	return xpath3.SingleBoolean(false), nil
}

// pushElementNamespaces adds namespace declarations from elem to nsBindings
// and to the stylesheet's runtime namespace map. Returns previous nsBindings
// for restoring.
func (c *compiler) pushElementNamespaces(_ context.Context, elem *helium.Element) map[string]string {
	nsList := elem.Namespaces()
	if len(nsList) == 0 {
		return c.nsBindings
	}
	saved := c.nsBindings
	newBindings := make(map[string]string, len(saved)+len(nsList))
	for k, v := range saved {
		newBindings[k] = v
	}
	for _, ns := range nsList {
		prefix := ns.Prefix()
		uri := ns.URI()
		newBindings[prefix] = uri
		// Also add to stylesheet namespaces for runtime XPath evaluation,
		// but only for prefixes not already declared at a higher scope.
		// Local redeclarations (e.g. xmlns:x="..." on a literal result
		// element inside a template) must not overwrite the top-level
		// stylesheet binding that user-defined functions are registered under.
		if _, exists := c.stylesheet.namespaces[prefix]; !exists {
			c.stylesheet.namespaces[prefix] = uri
		}
	}
	c.nsBindings = newBindings
	return saved
}

// compileXSLTInstruction compiles an XSLT-namespaced instruction element.
func (c *compiler) compileXSLTInstruction(ctx context.Context, elem *helium.Element) (instruction, error) {
	switch elem.LocalName() {
	case lexicon.XSLTElementApplyTemplates:
		return c.compileApplyTemplates(ctx, elem)
	case lexicon.XSLTElementCallTemplate:
		return c.compileCallTemplate(ctx, elem)
	case lexicon.XSLTElementValueOf:
		return c.compileValueOf(ctx, elem)
	case lexicon.XSLTElementText:
		return c.compileText(ctx, elem)
	case lexicon.XSLTElementElement:
		return c.compileElement(ctx, elem)
	case lexicon.XSLTElementAttribute:
		return c.compileAttribute(ctx, elem)
	case lexicon.XSLTElementComment:
		return c.compileComment(ctx, elem)
	case lexicon.XSLTElementProcessingInstruction:
		return c.compilePI(ctx, elem)
	case lexicon.XSLTElementIf:
		return c.compileIf(ctx, elem)
	case lexicon.XSLTElementChoose:
		return c.compileChoose(ctx, elem)
	case lexicon.XSLTElementForEach:
		return c.compileForEach(ctx, elem)
	case lexicon.XSLTElementVariable:
		return c.compileLocalVariable(ctx, elem)
	case lexicon.XSLTElementParam:
		// XTSE0010: xsl:param is only allowed at the top of xsl:template,
		// xsl:function, or xsl:iterate. If we reach here, it means
		// xsl:param appears in a general sequence constructor context.
		return nil, staticError(errCodeXTSE0010,
			"xsl:param is not allowed in this position; it must appear at the start of xsl:template, xsl:function, or xsl:iterate")
	case lexicon.XSLTElementCopy:
		return c.compileCopy(ctx, elem)
	case lexicon.XSLTElementCopyOf:
		return c.compileCopyOf(ctx, elem)
	case lexicon.XSLTElementNumber:
		return c.compileNumber(ctx, elem)
	case lexicon.XSLTElementMessage:
		return c.compileMessage(ctx, elem)
	case lexicon.XSLTElementNamespace:
		return c.compileNamespace(ctx, elem)
	case lexicon.XSLTElementSequence:
		return c.compileSequence(ctx, elem)
	case lexicon.XSLTElementPerformSort:
		return c.compilePerformSort(ctx, elem)
	case lexicon.XSLTElementNextMatch:
		return c.compileNextMatch(ctx, elem)
	case lexicon.XSLTElementApplyImports:
		return c.compileApplyImports(ctx, elem)
	case lexicon.XSLTElementDocument:
		return c.compileDocument(ctx, elem)
	case lexicon.XSLTElementResultDocument:
		inst := &resultDocumentInst{}
		if href := getAttr(elem, "href"); href != "" {
			avt, err := compileAVT(href, c.nsBindings)
			if err != nil {
				return nil, err
			}
			inst.Href = avt
		}
		// Validate html-version: must be a decimal number if present and not an avt
		if hv := getAttr(elem, paramHTMLVersion); hv != "" {
			if !strings.ContainsAny(hv, "{}") {
				if _, err := strconv.ParseFloat(hv, 64); err != nil {
					return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:result-document/@html-version", hv)
				}
			}
		}
		// Validate boolean output attributes on xsl:result-document.
		for _, boolAttr := range []string{paramByteOrderMark, paramEscapeURIAttributes,
			paramIncludeContentType, paramIndent, paramOmitXMLDeclaration, paramUndeclarePrefixes} {
			if v := getAttr(elem, boolAttr); v != "" {
				if !strings.ContainsAny(v, "{}") {
					if _, ok := parseXSDBool(v); !ok {
						return nil, staticError(errCodeSEPM0016, "%q is not a valid value for xsl:result-document/@%s", v, boolAttr)
					}
				}
			}
		}
		// Validate standalone on xsl:result-document.
		if v := getAttr(elem, paramStandalone); v != "" {
			if !strings.ContainsAny(v, "{}") {
				sv := strings.TrimSpace(v)
				switch sv {
				case lexicon.ValueYes, lexicon.ValueNo, lexicon.ValueOmit, "true", lexicon.ValueFalse, "1", "0":
					// valid
				default:
					return nil, staticError(errCodeSEPM0016, "%q is not a valid value for xsl:result-document/@standalone", v)
				}
			}
		}
		if fmtAttr := getAttr(elem, "format"); fmtAttr != "" {
			if strings.ContainsAny(fmtAttr, "{}") {
				avt, err := compileAVT(fmtAttr, c.nsBindings)
				if err != nil {
					return nil, err
				}
				inst.FormatAVT = avt
			} else {
				inst.Format = resolveQName(fmtAttr, c.nsBindings)
			}
		}
		inst.Method = getAttr(elem, paramMethod)
		if is := getAttr(elem, paramItemSeparator); is != "" {
			inst.ItemSeparatorSet = true
			// item-separator="#absent" means explicitly absent (use default, blocks format inheritance)
			if is != "#absent" {
				avt, err := compileAVT(is, c.nsBindings)
				if err != nil {
					return nil, err
				}
				inst.ItemSeparator = avt
			}
		}
		if v := getAttr(elem, "validation"); v != "" {
			if err := validateValidationAttr("xsl:result-document", v); err != nil {
				return nil, err
			}
			inst.Validation = v
		}
		if typeAttr := getAttr(elem, "type"); typeAttr != "" {
			inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
		}
		if ucm := getAttr(elem, paramUseCharacterMaps); ucm != "" {
			for _, n := range strings.Fields(ucm) {
				inst.UseCharacterMaps = append(inst.UseCharacterMaps, resolveQName(n, c.nsBindings))
			}
		}
		// Compile serialization parameter AVTs.
		for _, sp := range []struct {
			attr string
			dst  **avt
		}{
			{"output-version", &inst.OutputVersion},
			{paramEncoding, &inst.Encoding},
			{paramIndent, &inst.Indent},
			{paramOmitXMLDeclaration, &inst.OmitXMLDeclaration},
			{paramStandalone, &inst.Standalone},
			{paramDoctypeSystem, &inst.DoctypeSystem},
			{paramDoctypePublic, &inst.DoctypePublic},
			{paramCDATASectionElements, &inst.CDATASectionElements},
			{paramByteOrderMark, &inst.ByteOrderMark},
			{paramMediaType, &inst.MediaType},
			{paramHTMLVersion, &inst.HTMLVersion},
			{paramIncludeContentType, &inst.IncludeContentType},
			{paramAllowDuplicateNames, &inst.AllowDuplicateNames},
			{paramEscapeURIAttributes, &inst.EscapeURIAttributes},
			{paramJSONNodeOutputMethod, &inst.JSONNodeOutputMethodAVT},
			{paramNormalizationForm, &inst.NormalizationForm},
		} {
			if v := getAttr(elem, sp.attr); v != "" {
				avt, err := compileAVT(v, c.nsBindings)
				if err != nil {
					return nil, err
				}
				*sp.dst = avt
			}
		}
		// method attribute as avt (complements the static Method field)
		if v := getAttr(elem, paramMethod); v != "" {
			avt, err := compileAVT(v, c.nsBindings)
			if err != nil {
				return nil, err
			}
			inst.MethodAVT = avt
		}
		if pd := getAttr(elem, paramParameterDocument); pd != "" {
			if strings.ContainsAny(pd, "{}") {
				avt, err := compileAVT(pd, c.nsBindings)
				if err != nil {
					return nil, err
				}
				inst.ParameterDocAVT = avt
			} else {
				// Static parameter-document: load at compile time
				outDef := &OutputDef{}
				baseURI := stylesheetBaseURI(elem, c.baseURI)
				if err := c.loadParameterDocument(ctx, outDef, baseURI, pd); err != nil {
					return nil, err
				}
				inst.ParameterDocOutputDef = outDef
				// If the parameter-document sets the method, use it as the
				// compile-time method so isItemOutputMethod works.
				if outDef.Method != "" && inst.Method == "" {
					inst.Method = outDef.Method
				}
			}
		}
		if si := getAttr(elem, paramSuppressIndentation); si != "" {
			names := strings.Fields(si)
			resolved := make([]string, len(names))
			for i, n := range names {
				resolved[i] = resolveQName(n, c.nsBindings)
			}
			inst.SuppressIndentation = resolved
		}
		if v := getAttr(elem, paramBuildTree); v != "" {
			if b, ok := parseXSDBool(v); ok {
				inst.BuildTree = &b
			}
		}
		body, err := c.compileChildren(ctx, elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
		inst.NSBindings = c.nsBindings
		return inst, nil
	case lexicon.XSLTElementWherePopulated:
		// xsl:where-populated: execute body and only include if non-empty
		body, err := c.compileChildren(ctx, elem)
		if err != nil {
			return nil, err
		}
		return &wherePopulatedInst{Body: body}, nil
	case lexicon.XSLTElementOnEmpty:
		inst := &onEmptyInst{}
		if sel := getAttr(elem, "select"); sel != "" {
			expr, err := xpath3.NewCompiler().Compile(sel)
			if err != nil {
				return nil, err
			}
			inst.Select = expr
		} else {
			body, err := c.compileChildren(ctx, elem)
			if err != nil {
				return nil, err
			}
			inst.Body = body
		}
		return inst, nil
	case lexicon.XSLTElementOnNonEmpty:
		inst := &onNonEmptyInst{}
		if sel := getAttr(elem, "select"); sel != "" {
			expr, err := xpath3.NewCompiler().Compile(sel)
			if err != nil {
				return nil, err
			}
			inst.Select = expr
		} else {
			body, err := c.compileChildren(ctx, elem)
			if err != nil {
				return nil, err
			}
			inst.Body = body
		}
		return inst, nil
	case lexicon.XSLTElementTry:
		return c.compileTry(ctx, elem)
	case lexicon.XSLTElementForEachGroup:
		return c.compileForEachGroup(ctx, elem)
	case lexicon.XSLTElementMap:
		return c.compileMap(ctx, elem)
	case lexicon.XSLTElementMapEntry:
		return c.compileMapEntry(ctx, elem)
	case lexicon.XSLTElementSort:
		// XTSE0010: xsl:sort is only allowed as a child of xsl:apply-templates,
		// xsl:for-each, xsl:for-each-group, or xsl:perform-sort.
		// Those instructions handle xsl:sort directly; if we reach here, xsl:sort
		// appears in an invalid context (e.g. inside xsl:template body or xsl:call-template).
		return nil, staticError(errCodeXTSE0010,
			"xsl:sort is not allowed in this position")
	case lexicon.XSLTElementFallback:
		// xsl:fallback is only activated when the parent is unrecognized;
		// when we reach here the parent was recognized, so skip.
		// But we still need to validate use-when XPath on descendant elements
		// since use-when is evaluated during static analysis (XPST0003).
		if err := c.validateDescendantUseWhen(ctx,elem); err != nil {
			return nil, err
		}
		return nil, nil //nolint:nilnil
	case lexicon.XSLTElementContextItem:
		// xsl:context-item declares the expected context item type.
		// Currently a no-op — validation happens at the mode level.
		return nil, nil //nolint:nilnil
	case lexicon.XSLTElementAssert:
		return c.compileAssert(ctx, elem)
	case lexicon.XSLTElementAnalyzeString:
		return c.compileAnalyzeString(ctx, elem)
	case lexicon.XSLTElementEvaluate:
		return c.compileEvaluate(ctx, elem)
	case lexicon.XSLTElementSourceDocument:
		return c.compileSourceDocument(ctx, elem)
	case lexicon.XSLTElementIterate:
		return c.compileIterate(ctx, elem)
	case lexicon.XSLTElementFork:
		return c.compileFork(ctx, elem)
	case lexicon.XSLTElementBreak:
		return c.compileBreak(ctx, elem)
	case lexicon.XSLTElementNextIteration:
		return c.compileNextIteration(ctx, elem)
	case lexicon.XSLTElementMerge:
		return c.compileMerge(ctx, elem)
	case lexicon.XSLTElementMergeSource:
		// xsl:merge-source must be a direct child of xsl:merge
		return nil, staticError(errCodeXTSE0010, "xsl:merge-source must be a direct child of xsl:merge")
	case lexicon.XSLTElementMergeAction:
		// xsl:merge-action must be a direct child of xsl:merge
		return nil, staticError(errCodeXTSE0010, "xsl:merge-action must be a direct child of xsl:merge")
	case lexicon.XSLTElementMergeKey:
		// xsl:merge-key must be a direct child of xsl:merge-source
		return nil, staticError(errCodeXTSE0010, "xsl:merge-key must be a direct child of xsl:merge-source")
	case lexicon.XSLTElementOnCompletion:
		// xsl:on-completion must be a direct child of xsl:iterate — if we reach
		// here, it was encountered outside that context.
		return nil, staticError(errCodeXTSE0010, "xsl:on-completion must be a direct child of xsl:iterate")
	default:
		// Forwards-compatible processing (XSLT 3.0 §3.8): if the effective
		// version > 3.0, unknown instructions are allowed and
		// xsl:fallback children are used at runtime.
		if c.effectiveVersion > "3.0" {
			return c.compileForwardsCompat(ctx, elem)
		}
		return nil, staticError(errCodeXTSE0090, "unknown XSLT instruction xsl:%s", elem.LocalName())
	}
}

// compileForwardsCompat compiles an unknown XSLT instruction under
// forwards-compatible processing rules (XSLT 3.0 §3.8).
// It collects xsl:fallback children as the body. At runtime, if the body
// is empty, XTDE1450 is raised.
func (c *compiler) compileForwardsCompat(ctx context.Context, elem *helium.Element) (*fallbackInst, error) {
	inst := &fallbackInst{
		Name: "xsl:" + elem.LocalName(),
	}
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == lexicon.XSLTElementFallback {
			inst.HasFallback = true
			body, err := c.compileChildren(ctx, childElem)
			if err != nil {
				return nil, err
			}
			inst.Body = append(inst.Body, body...)
		}
	}
	return inst, nil
}

// compileChildren compiles all children of an element into instructions.
func (c *compiler) compileChildren(ctx context.Context, parent *helium.Element) ([]instruction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var body []instruction
	sawTerminator := false // true after xsl:break or xsl:next-iteration
	sawOnEmpty := false    // true after xsl:on-empty (must be last)
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			inst, err := c.compileInstruction(ctx, v)
			if err != nil {
				return nil, err
			}
			if inst != nil {
				// XTSE3120: nothing may follow xsl:break or xsl:next-iteration.
				if sawTerminator {
					return nil, staticError(errCodeXTSE3120, "no instruction may follow xsl:break or xsl:next-iteration")
				}
				// XTSE0010: xsl:on-empty must be the last instruction in a sequence constructor.
				// Only xsl:fallback (which compiles to nil) may follow it.
				if sawOnEmpty {
					return nil, staticError(errCodeXTSE0010, "xsl:on-empty must be the last instruction in a sequence constructor")
				}
				body = append(body, inst)
				switch inst.(type) {
				case *breakInst, *nextIterationInst:
					sawTerminator = true
				case *onEmptyInst:
					sawOnEmpty = true
				}
			}
		case *helium.Text, *helium.CDATASection:
			// Merge adjacent text/CDATA nodes (skipping comments/PIs) per XSLT §4.2:
			// comments are removed before whitespace stripping, and adjacent
			// text nodes must be merged for correct whitespace-only determination.
			text := string(child.Content())
			for next := child.NextSibling(); next != nil; next = next.NextSibling() {
				if next.Type() == helium.CommentNode || next.Type() == helium.ProcessingInstructionNode {
					continue
				}
				switch tn := next.(type) {
				case *helium.Text:
					text += string(tn.Content())
					child = next
					continue
				case *helium.CDATASection:
					text += string(tn.Content())
					child = next
					continue
				}
				break
			}
			if !c.shouldStripText(ctx, text) {
				if sawTerminator {
					return nil, staticError(errCodeXTSE3120, "no instruction may follow xsl:break or xsl:next-iteration")
				}
				if sawOnEmpty {
					return nil, staticError(errCodeXTSE0010, "xsl:on-empty must be the last instruction in a sequence constructor")
				}
				inst := &literalTextInst{Value: text}
				if c.expandText && strings.ContainsAny(text, "{}") {
					avt, err := compileAVT(text, c.nsBindings)
					if err != nil {
						return nil, err
					}
					inst.TVT = avt
				}
				body = append(body, inst)
			}
		}
	}
	return body, nil
}

func (c *compiler) compileLocalVariable(ctx context.Context, elem *helium.Element) (*variableInst, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	asAttr := c.resolveAsType(ctx, getAttr(elem, "as"))
	if err := c.validateAsSequenceType(ctx, asAttr, "xsl:variable "+name); err != nil {
		return nil, err
	}

	inst := &variableInst{
		Name: resolveQName(name, c.nsBindings),
		As:   asAttr,
	}

	// Capture xml:base for static base URI override during body/select evaluation.
	effectiveBase := stylesheetBaseURI(elem, c.baseURI)
	if effectiveBase != c.baseURI {
		inst.StaticBaseURI = effectiveBase
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0620: select and non-empty content are mutually exclusive.
		if err := c.validateEmptyElement(ctx, elem, "xsl:variable"); err != nil {
			return nil, staticError(errCodeXTSE0620, "xsl:variable %q has both @select and content", name)
		}
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(ctx, elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileMessage(ctx context.Context, elem *helium.Element) (*messageInst, error) {
	inst := &messageInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}
	// XSLT 3.0: both select and body content are allowed
	body, err := c.compileChildren(ctx, elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	termAttr := getAttr(elem, "terminate")
	if termAttr != "" {
		avt, err := compileAVT(termAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		// If the avt is a static value, validate it is "yes" or "no"
		if sv, ok := avt.staticValue(); ok {
			if sv != lexicon.ValueYes && sv != lexicon.ValueNo {
				return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:message/@terminate", sv)
			}
		}
		inst.Terminate = avt
	}

	errorCodeAttr := getAttr(elem, "error-code")
	if errorCodeAttr != "" {
		avt, err := compileAVT(errorCodeAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.ErrorCode = avt
	}

	return inst, nil
}

func (c *compiler) compileMap(ctx context.Context, elem *helium.Element) (*mapInst, error) {
	body, err := c.compileChildren(ctx, elem)
	if err != nil {
		return nil, err
	}
	return &mapInst{Body: body}, nil
}

func (c *compiler) compileMapEntry(ctx context.Context, elem *helium.Element) (*mapEntryInst, error) {
	inst := &mapEntryInst{}

	keyAttr := getAttr(elem, "key")
	if keyAttr != "" {
		expr, err := compileXPath(keyAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Key = expr
	} else {
		return nil, staticError(errCodeXTSE0010, "xsl:map-entry requires key attribute")
	}

	selectAttr := getAttr(elem, "select")
	hasBody := hasSignificantChildren(elem)
	if selectAttr != "" && hasBody {
		return nil, staticError(errCodeXTSE3280, "xsl:map-entry must not have both a select attribute and a sequence constructor body")
	}
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(ctx, elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

// compileAssert compiles xsl:assert.
// Required attribute: test.
// Optional attributes: select, error-code.
// The body provides the error message if no select attribute is present.
func (c *compiler) compileAssert(ctx context.Context, elem *helium.Element) (instruction, error) {
	testAttr := getAttr(elem, "test")
	if testAttr == "" {
		return nil, staticError(errCodeXTSE0010, "xsl:assert requires a test attribute")
	}

	testExpr, err := compileXPath(testAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &assertInst{
		Test:      testExpr,
		ErrorCode: errCodeXTMM9001, // default error code per XSLT 3.0 spec
	}

	// xpath-default-namespace
	if xdn := getAttr(elem, "xpath-default-namespace"); xdn != "" {
		inst.XPathDefaultNS = xdn
		inst.HasXPathDefaultNS = true
	}

	// error-code attribute
	if ec := getAttr(elem, "error-code"); ec != "" {
		inst.ErrorCode = resolveQName(ec, c.nsBindings)
	}

	// select attribute
	if sel := getAttr(elem, "select"); sel != "" {
		selExpr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = selExpr
	}

	// Compile body (error message content)
	body, err := c.compileChildren(ctx, elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	return inst, nil
}
