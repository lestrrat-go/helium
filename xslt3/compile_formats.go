package xslt3

import (
	"context"
	"errors"
	"maps"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (c *compiler) compileCharacterMap(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "use-character-maps": {}, xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	saved := c.pushElementNamespaces(ctx, elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, xslAttrName)
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:character-map requires name attribute")
	}
	name = resolveQName(name, c.nsBindings)

	cm := &characterMapDef{
		Name:     name,
		Mappings: make(map[rune]string),
	}

	if ucm := getAttr(elem, paramUseCharacterMaps); ucm != "" {
		for n := range strings.FieldsSeq(ucm) {
			cm.UseCharacterMaps = append(cm.UseCharacterMaps, resolveQName(n, c.nsBindings))
		}
	}

	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != lexicon.NamespaceXSLT || childElem.LocalName() != lexicon.XSLTElementOutputCharacter {
			continue
		}
		charAttr := getAttr(childElem, "character")
		if charAttr == "" {
			return staticError(errCodeXTSE0010,
				"xsl:output-character requires 'character' attribute")
		}
		r := firstRune(charAttr)
		str := getAttr(childElem, "string")
		cm.Mappings[r] = str
	}

	if c.stylesheet.characterMaps == nil {
		c.stylesheet.characterMaps = make(map[string]*characterMapDef)
	}
	if c.charMapModuleKeys == nil {
		c.charMapModuleKeys = make(map[string]string)
	}
	// XTSE1580: duplicate character-map declarations with same name in the
	// same stylesheet module. Different modules may define the same name;
	// higher import precedence overrides, same precedence merges.
	if prevModule, exists := c.charMapModuleKeys[name]; exists && prevModule == c.moduleKey {
		return staticError(errCodeXTSE1580,
			"duplicate character-map declaration %q", name)
	}
	c.charMapModuleKeys[name] = c.moduleKey

	if existing, ok := c.stylesheet.characterMaps[name]; ok {
		// Merge: later-compiled mappings override earlier ones for the
		// same character. Since imports compile before the importing
		// module, the importing module's mappings naturally take precedence.
		maps.Copy(existing.Mappings, cm.Mappings)
		existing.UseCharacterMaps = append(existing.UseCharacterMaps, cm.UseCharacterMaps...)
	} else {
		c.stylesheet.characterMaps[name] = cm
	}
	return nil
}

func (c *compiler) compileKey(ctx context.Context, elem *helium.Element) error {
	defer c.pushElementVersion(elem)()
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "match": {}, "use": {}, "collation": {}, "composite": {},
		xslAttrUseWhen: {}, "default-collation": {},
	}); err != nil {
		return err
	}
	// Collect local namespace declarations (e.g., xmlns:ex="..." on xsl:key)
	c.collectNamespaces(ctx, elem)

	// Handle per-element default-collation override (standard attribute)
	savedDefaultCollation := c.defaultCollation
	if dc := getAttr(elem, "default-collation"); dc != "" {
		if uri := resolveDefaultCollation(dc); uri != "" {
			c.defaultCollation = uri
		}
	}
	defer func() { c.defaultCollation = savedDefaultCollation }()

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires name attribute")
	}
	if !xmlchar.IsValidQName(name) && !isValidEQName(name) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:key", name)
	}

	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires match attribute")
	}

	// Apply a local xpath-default-namespace on the xsl:key element (presence,
	// even an empty value, overrides the inherited default) so the @match
	// pattern's unprefixed names resolve consistently with NameTest semantics.
	xpathDefaultNS := c.xpathDefaultNS
	hasXPathDefaultNS := c.hasXPathDefaultNS
	if xdn, ok := elem.GetAttribute("xpath-default-namespace"); ok {
		xpathDefaultNS = xdn
		hasXPathDefaultNS = true
	}

	matchPat, err := compilePattern(matchAttr, elem, xpathDefaultNS, hasXPathDefaultNS, c.backwardsCompatible(), c.schemaDeclsForValidation())
	if err != nil {
		return err
	}

	expandedName := resolveQName(name, c.nsBindings)
	composite, _ := parseXSDBool(getAttr(elem, "composite"))

	// XTSE1222: if there is already a key with the same name, the composite
	// attribute must agree
	if existingDefs, ok := c.stylesheet.keys[expandedName]; ok && len(existingDefs) > 0 {
		if existingDefs[0].Composite != composite {
			return staticError(errCodeXTSE1222,
				"xsl:key declarations with name %q have conflicting composite attributes", expandedName)
		}
	}

	// Resolve collation: explicit attribute takes precedence over default-collation.
	collationURI := ""
	if collAttr := getAttr(elem, "collation"); collAttr != "" {
		// XTSE1210: collation must be a recognized URI
		if !xpath3.IsCollationSupported(collAttr) {
			return staticError(errCodeXTSE1210, "unrecognized collation URI %q on xsl:key", collAttr)
		}
		collationURI = collAttr
	} else if c.defaultCollation != "" {
		collationURI = c.defaultCollation
	}

	// XTSE1220: if there is already a key with the same name, the collation
	// must agree across all declarations
	if existingDefs, ok := c.stylesheet.keys[expandedName]; ok && len(existingDefs) > 0 {
		if existingDefs[0].Collation != collationURI {
			return staticError(errCodeXTSE1220,
				"xsl:key declarations with name %q have conflicting collation attributes", expandedName)
		}
	}

	kd := &keyDef{
		Name:      expandedName,
		Match:     matchPat,
		Composite: composite,
		Collation: collationURI,
		Compat:    c.backwardsCompatible(),
	}

	useAttr := getAttr(elem, "use")
	hasContent := c.hasEffectiveContent(ctx, elem)
	// XTSE1205: must have either use attr or content, not both, and not neither
	if useAttr != "" && hasContent {
		return staticError(errCodeXTSE1205, "xsl:key must not have both a use attribute and content")
	}
	if useAttr == "" && !hasContent {
		return staticError(errCodeXTSE1205, "xsl:key must have either a use attribute or content")
	}
	if useAttr != "" {
		useExpr, err := c.compileXPath(useAttr, c.nsBindings)
		if err != nil {
			return err
		}
		kd.Use = useExpr
	} else {
		// XSLT 2.0+: key may use body content instead of use attribute.
		body, _, err := c.compileTemplateBody(ctx, elem)
		if err != nil {
			return err
		}
		kd.Body = body
	}

	c.stylesheet.keys[expandedName] = append(c.stylesheet.keys[expandedName], kd)
	return nil
}

// outputBoolAttr reads a boolean xsl:output attribute. When the attribute is
// absent (present=false) the value is ignored; when present but not a valid
// value it returns SEPM0016. The attr name is embedded in the error verbatim,
// matching the param constants' literal values.
//
// The accepted value space depends on the stylesheet's effective XSLT version.
// XSLT 3.0 uses Serialization 3.0, whose boolean serialization parameters accept
// the full xs:boolean lexical space {yes, no, true, false, 1, 0}. XSLT 2.0 (and
// 1.0) use the earlier serialization spec, restricting them to {yes, no}; when
// yesNoOnly is true any other lexical form (including true/false/1/0) is invalid.
func outputBoolAttr(elem *helium.Element, attr string, yesNoOnly bool) (bool, bool, error) {
	v := getAttr(elem, attr)
	if v == "" {
		return false, false, nil
	}
	if yesNoOnly {
		switch strings.TrimSpace(v) {
		case lexicon.ValueYes:
			return true, true, nil
		case lexicon.ValueNo:
			return false, true, nil
		default:
			return false, false, staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@"+attr, v)
		}
	}
	b, ok := parseXSDBool(v)
	if !ok {
		return false, false, staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@"+attr, v)
	}
	return b, true, nil
}

// serializationYesNoOnly reports whether an effective XSLT version predates the
// Serialization 3.0 widening of the boolean serialization-parameter value space.
// XSLT 2.0 (and 1.0) restrict xsl:output boolean parameters — and standalone's
// boolean synonyms — to the yes/no lexical space; XSLT 3.0+ accepts the full
// xs:boolean lexical space. An absent/unparseable version defaults to 3.0
// (permissive), matching the compiler's default effective version.
func serializationYesNoOnly(ver string) bool {
	ver = strings.TrimSpace(ver)
	if ver == "" {
		return false
	}
	f, err := strconv.ParseFloat(ver, 64)
	if err != nil {
		return false
	}
	return f < 3.0
}

func (c *compiler) compileOutput(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, paramMethod: {}, paramVersion: {}, "encoding": {},
		"omit-xml-declaration": {}, "standalone": {}, "doctype-public": {},
		"doctype-system": {}, "cdata-section-elements": {}, "indent": {},
		"media-type": {}, "byte-order-mark": {}, "escape-uri-attributes": {},
		"include-content-type": {}, "normalization-form": {},
		"undeclare-prefixes": {}, "use-character-maps": {},
		"suppress-indentation": {}, "html-version": {},
		"item-separator": {}, "json-node-output-method": {},
		"parameter-document": {}, "build-tree": {},
		"allow-duplicate-names": {},
		xslAttrUseWhen:          {},
	}); err != nil {
		return err
	}
	saved := c.pushElementNamespaces(ctx, elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, xslAttrName)
	if name != "" {
		name = resolveQName(name, c.nsBindings)
	}
	methodStr := strings.TrimSpace(strings.ToLower(getAttr(elem, paramMethod)))
	// XTSE1570: validate method value
	if methodStr != "" {
		if !strings.Contains(methodStr, ":") {
			// No-prefix: must be a known method
			switch methodStr {
			case methodXML, methodHTML, methodXHTML, methodText, methodJSON, methodAdaptive:
				// valid
			default:
				return staticError(errCodeXTSE1570, "invalid output method %q", methodStr)
			}
		} else if !xmlchar.IsValidQName(methodStr) && !isValidEQName(methodStr) {
			return staticError(errCodeXTSE1570, "invalid output method %q", methodStr)
		}
	}
	outDef := &OutputDef{
		Name:           name,
		Method:         methodStr,
		MethodExplicit: methodStr != "",
		Encoding:       getAttr(elem, paramEncoding),
		Version:        getAttr(elem, paramVersion),
		ImportPrec:     c.importPrec,
		MethodRaw:      getAttr(elem, paramMethod),
		EncodingRaw:    getAttr(elem, paramEncoding),
		VersionRaw:     getAttr(elem, paramVersion),
	}

	if outDef.Method == "" {
		outDef.Method = methodXML
	}
	if outDef.Encoding == "" {
		outDef.Encoding = lexicon.EncodingUTF8U
	}

	// Determine the accepted lexical space for boolean serialization parameters.
	// XSLT 2.0 (and 1.0) restrict them to yes/no; XSLT 3.0+ accepts the full
	// xs:boolean lexical space. The governing XSLT version is the effective
	// (in-scope module) version — NOT the xsl:output element's own @version,
	// which is the serialization (output XML/HTML version) parameter.
	yesNoOnly := serializationYesNoOnly(c.effectiveVersion)

	// Validate and parse boolean output attributes.
	// SEPM0016: invalid boolean values.
	if b, present, err := outputBoolAttr(elem, paramIndent, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.Indent = b
		outDef.IndentRaw = getAttr(elem, paramIndent)
	}
	if b, present, err := outputBoolAttr(elem, paramOmitXMLDeclaration, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.OmitDeclaration = b
		outDef.OmitDeclarationExplicit = true
	}
	if b, present, err := outputBoolAttr(elem, paramUndeclarePrefixes, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.UndeclarePrefixes = b
	}
	if b, present, err := outputBoolAttr(elem, paramByteOrderMark, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.ByteOrderMark = b
	}
	// Parse escape-uri-attributes.
	if b, present, err := outputBoolAttr(elem, paramEscapeURIAttributes, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.EscapeURIAttributes = &b
	}
	if b, present, err := outputBoolAttr(elem, paramIncludeContentType, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.IncludeContentType = &b
	}
	// Validate standalone: must be "yes", "no", "omit", or (XSLT 3.0+ only) a
	// boolean synonym. Under the yes/no-only value space true/false/1/0 are
	// invalid, matching the boolean serialization parameters.
	if v := getAttr(elem, paramStandalone); v != "" {
		v = strings.TrimSpace(v)
		switch v {
		case lexicon.ValueYes, lexicon.ValueNo, "omit":
			// valid as-is
		case lexicon.ValueTrue, "1":
			if yesNoOnly {
				return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@standalone", v)
			}
			v = lexicon.ValueYes
		case lexicon.ValueFalse, "0":
			if yesNoOnly {
				return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@standalone", v)
			}
			v = lexicon.ValueNo
		default:
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@standalone", v)
		}
		outDef.Standalone = v
		outDef.StandaloneRaw = getAttr(elem, paramStandalone)
	} else {
		outDef.Standalone = ""
	}
	outDef.DoctypePublic = getAttr(elem, paramDoctypePublic)
	outDef.DoctypeSystem = getAttr(elem, paramDoctypeSystem)
	outDef.MediaType = getAttr(elem, paramMediaType)
	if hv := getAttr(elem, paramHTMLVersion); hv != "" {
		hv = strings.TrimSpace(hv)
		if _, err := strconv.ParseFloat(hv, 64); err != nil {
			return staticError(errCodeXTSE0020, "%q is not a valid value for xsl:output/@html-version", hv)
		}
		outDef.HTMLVersion = hv
	}

	cdataStr := getAttr(elem, paramCDATASectionElements)
	if cdataStr != "" {
		names := strings.Fields(cdataStr)
		resolved := make([]string, len(names))
		for i, n := range names {
			resolved[i] = resolveQName(n, c.nsBindings)
		}
		outDef.CDATASections = resolved
	}

	if is := getAttr(elem, paramItemSeparator); is != "" {
		outDef.ItemSeparator = &is
	}

	if nf := getAttr(elem, paramNormalizationForm); nf != "" {
		outDef.NormalizationForm = strings.ToUpper(strings.TrimSpace(nf))
	}

	if ucm := getAttr(elem, paramUseCharacterMaps); ucm != "" {
		for n := range strings.FieldsSeq(ucm) {
			outDef.UseCharacterMaps = append(outDef.UseCharacterMaps, resolveQName(n, c.nsBindings))
		}
	}

	if v := getAttr(elem, paramJSONNodeOutputMethod); v != "" {
		outDef.JSONNodeOutputMethod = strings.TrimSpace(v)
	}

	// adnExplicit tracks whether allow-duplicate-names was supplied by either the
	// xsl:output attribute or the parameter document. It is compile-time merge
	// bookkeeping only and is deliberately kept off the public OutputDef type.
	var adnExplicit bool
	if b, present, err := outputBoolAttr(elem, paramAllowDuplicateNames, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.AllowDuplicateNames = b
		adnExplicit = true
	}

	if v := getAttr(elem, paramSuppressIndentation); v != "" {
		names := strings.Fields(v)
		resolved := make([]string, len(names))
		for i, n := range names {
			resolved[i] = resolveQName(n, c.nsBindings)
		}
		outDef.SuppressIndentation = resolved
	}

	if b, present, err := outputBoolAttr(elem, paramBuildTree, yesNoOnly); err != nil {
		return err
	} else if present {
		outDef.BuildTree = &b
	}

	if v := getAttr(elem, paramParameterDocument); v != "" {
		outDef.ParameterDocument = v
		baseURI := stylesheetBaseURI(elem, c.baseURI, c.moduleRoot)
		var err error
		// This applies the parameter-document directly onto the xsl:output's
		// OutputDef (not a delta to be folded later), so the boolean presence
		// flags are not needed here and are discarded.
		if adnExplicit, _, err = c.loadParameterDocument(ctx, outDef, baseURI, v, adnExplicit); err != nil {
			return err
		}
	}

	// XTSE1560: check for conflicting output attributes at the same import
	// precedence before merging.
	if existing, ok := c.stylesheet.outputs[name]; ok && existing.ImportPrec == c.importPrec {
		// Attributes that are not allowed to conflict (all except cdata-section-elements
		// and use-character-maps). Check each explicitly set attribute.
		type attrConflict struct {
			attrName string
			oldVal   string
			newVal   string
			newIsSet bool
			oldIsSet bool
		}
		checks := []attrConflict{
			{paramMethod, existing.MethodRaw, getAttr(elem, paramMethod), getAttr(elem, paramMethod) != "", existing.MethodRaw != ""},
			{paramIndent, existing.IndentRaw, getAttr(elem, paramIndent), getAttr(elem, paramIndent) != "", existing.IndentRaw != ""},
			{paramEncoding, existing.EncodingRaw, getAttr(elem, paramEncoding), getAttr(elem, paramEncoding) != "", existing.EncodingRaw != ""},
			{paramVersion, existing.VersionRaw, getAttr(elem, paramVersion), getAttr(elem, paramVersion) != "", existing.VersionRaw != ""},
			{paramStandalone, existing.StandaloneRaw, getAttr(elem, paramStandalone), getAttr(elem, paramStandalone) != "", existing.StandaloneRaw != ""},
		}
		for _, chk := range checks {
			if chk.newIsSet && chk.oldIsSet && chk.oldVal != chk.newVal {
				return staticError(errCodeXTSE1560,
					"conflicting values for xsl:output/@%s: %q vs %q", chk.attrName, chk.oldVal, chk.newVal)
			}
		}
	}

	// Per XSLT 3.0 §9.6.1: multiple xsl:output declarations with the same
	// name are merged. use-character-maps values accumulate; other attributes
	// are taken from the later declaration when explicitly specified.
	if existing, ok := c.stylesheet.outputs[name]; ok {
		// Accumulate use-character-maps from both declarations.
		if len(existing.UseCharacterMaps) > 0 {
			outDef.UseCharacterMaps = append(existing.UseCharacterMaps, outDef.UseCharacterMaps...)
		}
		// Preserve fields from earlier declaration when not explicitly set here.
		if getAttr(elem, paramMethod) == "" {
			outDef.Method = existing.Method
			outDef.MethodExplicit = existing.MethodExplicit
		}
		if getAttr(elem, paramEncoding) == "" {
			outDef.Encoding = existing.Encoding
		}
		if getAttr(elem, paramVersion) == "" {
			outDef.Version = existing.Version
		}
		if getAttr(elem, paramIndent) == "" {
			outDef.Indent = existing.Indent
		}
		if getAttr(elem, paramOmitXMLDeclaration) == "" {
			outDef.OmitDeclaration = existing.OmitDeclaration
			outDef.OmitDeclarationExplicit = existing.OmitDeclarationExplicit
		}
		if getAttr(elem, paramStandalone) == "" {
			outDef.Standalone = existing.Standalone
		}
		if getAttr(elem, paramUndeclarePrefixes) == "" {
			outDef.UndeclarePrefixes = existing.UndeclarePrefixes
		}
		if !elem.HasAttribute("doctype-public") {
			outDef.DoctypePublic = existing.DoctypePublic
		}
		if !elem.HasAttribute("doctype-system") {
			outDef.DoctypeSystem = existing.DoctypeSystem
		}
		if getAttr(elem, paramMediaType) == "" {
			outDef.MediaType = existing.MediaType
		}
		if getAttr(elem, paramCDATASectionElements) == "" {
			outDef.CDATASections = existing.CDATASections
		}
		if getAttr(elem, paramItemSeparator) == "" {
			outDef.ItemSeparator = existing.ItemSeparator
		}
		if getAttr(elem, paramNormalizationForm) == "" {
			outDef.NormalizationForm = existing.NormalizationForm
		}
		if getAttr(elem, paramHTMLVersion) == "" {
			outDef.HTMLVersion = existing.HTMLVersion
		}
		if outDef.IncludeContentType == nil {
			outDef.IncludeContentType = existing.IncludeContentType
		}
		if getAttr(elem, paramJSONNodeOutputMethod) == "" {
			outDef.JSONNodeOutputMethod = existing.JSONNodeOutputMethod
		}
		// Preserve the earlier declaration's value only when neither the
		// attribute nor the parameter document of this declaration supplied
		// allow-duplicate-names; a parameter-document supplied value sets
		// adnExplicit and must not be clobbered.
		if !adnExplicit {
			outDef.AllowDuplicateNames = existing.AllowDuplicateNames
			adnExplicit = c.outputAllowDupExplicit[name]
		}
	}

	c.stylesheet.outputs[name] = outDef
	if c.outputAllowDupExplicit == nil {
		c.outputAllowDupExplicit = make(map[string]bool)
	}
	c.outputAllowDupExplicit[name] = adnExplicit
	return nil
}

// loadParameterDocument loads a serialization parameter document (XSLT 3.0 §9.2)
// and applies its settings to the given OutputDef. Parameters explicitly set on
// the xsl:output element take precedence; the parameter document provides defaults.
// adnExplicit carries the allow-duplicate-names explicitness in and out so the
// parameter document does not override a value already set on xsl:output and the
// caller can fold the result into its compile-time merge bookkeeping.
func (c *compiler) loadParameterDocument(ctx context.Context, outDef *OutputDef, baseURI, href string, adnExplicit bool) (bool, paramDocPresence, error) {
	// Compile-time loading is opt-in: a URIResolver must be configured via
	// Compiler.URIResolver. There is no implicit filesystem access.
	loadBytes := func(_ context.Context, uri string) ([]byte, error) {
		if c.resolver == nil {
			return nil, staticError(errCodeXTSE0090, "cannot read parameter-document %q: no URIResolver configured (filesystem access is opt-in; set Compiler.URIResolver)", href)
		}
		rc, resolveErr := c.resolver.Resolve(uri)
		if resolveErr != nil {
			return nil, staticError(errCodeXTSE0090, "cannot read parameter-document %q: %v", href, resolveErr)
		}
		defer func() { _ = rc.Close() }()
		return readResourceBounded(rc, c.maxResourceBytes)
	}
	return loadParameterDocumentFromFile(ctx, c.parser, outDef, baseURI, href, loadBytes, false, adnExplicit, c.maxResourceBytes)
}

// loadParameterDocumentFromFile loads a serialization parameter document and
// applies its settings to the given OutputDef. This standalone version can be
// called at both compile-time and runtime; the loadBytes callback performs the
// actual retrieval so each caller supplies its own (opt-in) loader.
//
// runtime selects the error taxonomy: at compile time (runtime=false) a load or
// parse failure is a static error (XTSE0090); at runtime (runtime=true, e.g. an
// xsl:result-document parameter-document AVT) it is a dynamic error (FODC0002).
// A runtime failure must NOT also satisfy errors.Is(err, ErrStaticError), so the
// runtime path never applies the static wrapper. A distinguishable cause such as
// [ErrResourceTooLarge] survives either way via errors.Join.
//
// adnExplicit (allow-duplicate-names explicitness) is threaded in and out rather
// than stored on the public OutputDef: when already true on input the parameter
// document must not override the value, and the returned value reports whether
// the parameter document supplied it so the caller can fold it into its own
// compile-time merge bookkeeping.
//
// The returned paramDocPresence reports which plain-boolean serialization
// parameters this parameter-document explicitly supplied. These presence flags
// are kept off the public OutputDef (a plain bool cannot distinguish "omitted"
// from "explicit false") and travel alongside the resulting delta so
// foldParamDocOverrides leaves an inherited xsl:output default intact for any
// boolean the parameter-document omits.
func loadParameterDocumentFromFile(ctx context.Context, injected *helium.Parser, outDef *OutputDef, baseURI, href string, loadBytes func(context.Context, string) ([]byte, error), runtime bool, adnExplicit bool, resourceLimit int64) (bool, paramDocPresence, error) {
	var presence paramDocPresence
	// Decide absoluteness with xsd.URIScheme (RFC 3986), not filepath.IsAbs or a
	// "://" substring check: an absolute-URI href may carry a scheme with no
	// "//" authority (e.g. "urn:params", "file:/p/p.xml") and must pass through
	// unchanged, while a relative href against a URI base must keep the base
	// scheme/authority. Only when both base and ref are local filesystem paths
	// is filepath.Join used.
	uri := href
	if baseURI != "" {
		switch {
		case xsd.URIScheme(href) != "" || xsd.URIScheme(baseURI) != "":
			resolved, rErr := xsd.ResolveSchemaURI(href, baseURI)
			if rErr == nil {
				uri = resolved
			}
		case !uripath.IsAbsolutePath(href):
			// Local filesystem base: resolve with forward-slash (path)
			// semantics so the result uses '/' on every OS; on Windows
			// filepath.Dir/Join would emit '\'. uripath.IsAbsolutePath
			// recognizes both POSIX- and Windows-absolute hrefs regardless
			// of GOOS.
			uri = uripath.JoinLocalBaseDir(path.Dir(uripath.ToSlash(baseURI)), href)
		}
	}

	data, err := loadBytes(ctx, uri)
	if err != nil {
		// At runtime the failure is dynamic (FODC0002); it must not also match
		// ErrStaticError, so re-wrap the raw cause via dynamicErrorCause rather
		// than returning the static wrapper. The cause (e.g. ErrResourceTooLarge)
		// survives the join either way.
		if runtime {
			return adnExplicit, presence, dynamicErrorCause(errCodeFODC0002, err, "cannot read parameter-document %q: %v", href, err)
		}
		// The compile-time loader already returns an *XSLTError (XTSE0090);
		// return it as-is rather than wrapping it in a second XTSE0090.
		var xe *XSLTError
		if errors.As(err, &xe) {
			return adnExplicit, presence, err
		}
		return adnExplicit, presence, staticErrorCause(errCodeXTSE0090, err, "cannot read parameter-document %q: %v", href, err)
	}
	// Serialization parameter documents have a fixed W3C schema and never use
	// external entities; parse with the injected base parser (or XXE blocked by
	// default when none is injected).
	doc, err := secureXMLParser(injected, "", resourceLimit).Parse(ctx, data)
	if err != nil {
		if runtime {
			return adnExplicit, presence, dynamicError(errCodeFODC0002, "cannot parse parameter-document %q: %v", href, err)
		}
		return adnExplicit, presence, staticError(errCodeXTSE0090, "cannot parse parameter-document %q: %v", href, err)
	}
	root := doc.DocumentElement()
	if root == nil || root.LocalName() != "serialization-parameters" || root.URI() != lexicon.NamespaceSerialization {
		if runtime {
			return adnExplicit, presence, dynamicError(errCodeFODC0002, "parameter-document %q: root element must be {%s}serialization-parameters", href, lexicon.NamespaceSerialization)
		}
		return adnExplicit, presence, staticError(errCodeXTSE0090, "parameter-document %q: root element must be {%s}serialization-parameters", href, lexicon.NamespaceSerialization)
	}

	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if elem.URI() != lexicon.NamespaceSerialization {
			continue
		}
		val := getAttr(elem, "value")
		switch elem.LocalName() {
		case paramMethod:
			if outDef.MethodRaw == "" && val != "" {
				outDef.Method = strings.ToLower(strings.TrimSpace(val))
				outDef.MethodExplicit = true
			}
		case paramIndent:
			if outDef.IndentRaw == "" && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.Indent = b
					presence.indent = true
				}
			}
		case paramOmitXMLDeclaration:
			if !outDef.OmitDeclarationExplicit && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.OmitDeclaration = b
					outDef.OmitDeclarationExplicit = true
				}
			}
		case paramEncoding:
			if outDef.EncodingRaw == "" && val != "" {
				outDef.Encoding = strings.TrimSpace(val)
			}
		case paramStandalone:
			if outDef.StandaloneRaw == "" && val != "" {
				v := strings.TrimSpace(val)
				switch v {
				case lexicon.ValueYes, lexicon.ValueNo, "omit":
					outDef.Standalone = v
				}
			}
		case paramCDATASectionElements:
			if len(outDef.CDATASections) == 0 && val != "" {
				outDef.CDATASections = strings.Fields(val)
			}
		case paramDoctypePublic:
			if outDef.DoctypePublic == "" && val != "" {
				outDef.DoctypePublic = val
			}
		case paramDoctypeSystem:
			if outDef.DoctypeSystem == "" && val != "" {
				outDef.DoctypeSystem = val
			}
		case paramMediaType:
			if outDef.MediaType == "" && val != "" {
				outDef.MediaType = val
			}
		case paramVersion:
			if outDef.VersionRaw == "" && val != "" {
				outDef.Version = strings.TrimSpace(val)
			}
		case paramUndeclarePrefixes:
			if val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.UndeclarePrefixes = b
					presence.undeclarePrefixes = true
				}
			}
		case paramUseCharacterMaps:
			// Character maps defined in the parameter document.
			// Each child <character-map> has @character and @map-string.
			if outDef.ResolvedCharMap == nil {
				outDef.ResolvedCharMap = make(map[rune]string)
			}
			for mapChild := range helium.Children(elem) {
				mapElem, ok := mapChild.(*helium.Element)
				if !ok {
					continue
				}
				if mapElem.LocalName() != "character-map" {
					continue
				}
				charStr := getAttr(mapElem, "character")
				mapStr := getAttr(mapElem, "map-string")
				if charStr != "" {
					runes := []rune(charStr)
					if len(runes) == 1 {
						outDef.ResolvedCharMap[runes[0]] = mapStr
					}
				}
			}
		case paramAllowDuplicateNames:
			if !adnExplicit && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.AllowDuplicateNames = b
					presence.allowDuplicateNames = true
					adnExplicit = true
				}
			}
		case paramItemSeparator:
			if outDef.ItemSeparator == nil && val != "" {
				outDef.ItemSeparator = &val
			}
		case paramByteOrderMark:
			if val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.ByteOrderMark = b
					presence.byteOrderMark = true
				}
			}
		case paramEscapeURIAttributes:
			if outDef.EscapeURIAttributes == nil && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.EscapeURIAttributes = &b
				}
			}
		case paramIncludeContentType:
			if outDef.IncludeContentType == nil && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.IncludeContentType = &b
				}
			}
		case paramNormalizationForm:
			if outDef.NormalizationForm == "" && val != "" {
				outDef.NormalizationForm = strings.ToUpper(strings.TrimSpace(val))
			}
		case paramSuppressIndentation:
			if len(outDef.SuppressIndentation) == 0 && val != "" {
				outDef.SuppressIndentation = strings.Fields(val)
			}
		case paramHTMLVersion:
			if outDef.HTMLVersion == "" && val != "" {
				outDef.HTMLVersion = strings.TrimSpace(val)
			}
		case paramBuildTree:
			if outDef.BuildTree == nil && val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.BuildTree = &b
				}
			}
		case paramJSONNodeOutputMethod:
			if outDef.JSONNodeOutputMethod == "" && val != "" {
				outDef.JSONNodeOutputMethod = strings.TrimSpace(val)
			}
		}
	}
	return adnExplicit, presence, nil
}

func (c *compiler) compileAttributeSet(ctx context.Context, elem *helium.Element) error {
	defer c.pushElementVersion(elem)()
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "use-attribute-sets": {}, xslAttrVisibility: {},
		"streamable": {}, xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	name := getAttr(elem, xslAttrName)
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:attribute-set requires name attribute")
	}
	if !xmlchar.IsValidQName(name) && !isValidEQName(name) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:attribute-set", name)
	}
	if err := c.checkQNamePrefix(ctx, name, "xsl:attribute-set"); err != nil {
		return err
	}
	name = resolveQName(name, c.nsBindings)

	streamable := false
	if s := getAttr(elem, "streamable"); s != "" {
		v, ok := parseXSDBool(s)
		if !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for streamable attribute on xsl:attribute-set", s)
		}
		streamable = v
	}

	asd := &attributeSetDef{
		Name:       name,
		Visibility: getAttr(elem, "visibility"),
		Streamable: streamable,
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for n := range strings.FieldsSeq(uas) {
			asd.UseAttrSets = append(asd.UseAttrSets, resolveQName(n, c.nsBindings))
		}
	}

	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			// XTSE0010: text content not allowed in xsl:attribute-set
			if isNonWhitespaceTextNode(child) {
				return staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:attribute-set")
			}
			continue
		}
		if childElem.URI() != lexicon.NamespaceXSLT || childElem.LocalName() != lexicon.XSLTElementAttribute {
			return staticError(errCodeXTSE0010, "only xsl:attribute is allowed as a child of xsl:attribute-set")
		}
		inst, err := c.compileAttribute(ctx, childElem)
		if err != nil {
			return err
		}
		asd.Attrs = append(asd.Attrs, inst)
	}

	if c.stylesheet.attributeSets == nil {
		c.stylesheet.attributeSets = make(map[string]*attributeSetDef)
	}
	// Build a part for this declaration
	effectiveBase := stylesheetBaseURI(elem, c.baseURI, c.moduleRoot)
	part := attributeSetPart{
		UseAttrSets:   asd.UseAttrSets,
		Attrs:         asd.Attrs,
		StaticBaseURI: effectiveBase,
	}
	// Merge same-named attribute-sets (XSLT spec: union of all definitions).
	// When an existing entry comes from a used package (OwnerPackage != nil)
	// and the new definition is local, the local definition replaces the
	// package's private attribute-set rather than merging with it.
	if existing, ok := c.stylesheet.attributeSets[name]; ok {
		if existing.OwnerPackage != nil {
			// Replace package-scoped attribute-set with local definition.
			asd.Parts = []attributeSetPart{part}
			c.stylesheet.attributeSets[name] = asd
		} else {
			existing.Attrs = append(existing.Attrs, asd.Attrs...)
			existing.UseAttrSets = append(existing.UseAttrSets, asd.UseAttrSets...)
			existing.Parts = append(existing.Parts, part)
		}
	} else {
		asd.Parts = []attributeSetPart{part}
		c.stylesheet.attributeSets[name] = asd
	}
	return nil
}

// checkAttributeSetCycles detects cyclic use-attribute-sets references (XTSE0720).
// Must be called after all attribute-set definitions have been compiled and merged.
func checkAttributeSetCycles(ss *Stylesheet) error {
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[string]int, len(ss.attributeSets))

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return staticError(errCodeXTSE0720,
				"attribute-set %q has a circular use-attribute-sets reference", name)
		}
		state[name] = visiting
		if asd := ss.attributeSets[name]; asd != nil {
			for _, ref := range asd.UseAttrSets {
				// use-attribute-sets="xsl:original" refers to the
				// original attribute-set being overridden, not a
				// named attribute-set in the stylesheet.
				if ref == helium.ClarkName(lexicon.NamespaceXSLT, "original") && asd.OriginalAttrSet != nil {
					continue
				}
				if _, ok := ss.attributeSets[ref]; !ok {
					return staticError(errCodeXTSE0710,
						"attribute-set %q references undeclared attribute-set %q", name, ref)
				}
				if err := visit(ref); err != nil {
					return err
				}
			}
		}
		state[name] = done
		return nil
	}

	for name := range ss.attributeSets {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

// checkAttributeSetStreamable checks XTSE0730: if an attribute set specifies
// streamable="yes", every attribute set referenced in its use-attribute-sets
// must also specify streamable="yes".
func checkAttributeSetStreamable(ss *Stylesheet) error {
	for _, asd := range ss.attributeSets {
		if !asd.Streamable {
			continue
		}
		for _, ref := range asd.UseAttrSets {
			used := ss.attributeSets[ref]
			if used == nil {
				continue
			}
			if !used.Streamable {
				return staticError(errCodeXTSE0730,
					"streamable attribute-set %q references non-streamable attribute-set %q",
					asd.Name, ref)
			}
		}
	}
	return nil
}

func (c *compiler) compileDecimalFormat(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "decimal-separator": {}, "grouping-separator": {},
		"infinity": {}, "minus-sign": {}, "NaN": {}, "percent": {},
		"per-mille": {}, "zero-digit": {}, "digit": {},
		"pattern-separator": {}, "exponent-separator": {},
		xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	// Push element-local namespace declarations so prefixed names resolve correctly
	saved := c.pushElementNamespaces(ctx, elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, xslAttrName)
	qn := xpath3.QualifiedName{}
	if name != "" {
		if !xmlchar.IsValidQName(name) && !isValidEQName(name) {
			return staticError(errCodeXTSE0020, "invalid name %q on xsl:decimal-format", name)
		}
		qn = xpath3.QualifiedName{Name: name}
		if prefix, local, ok := strings.Cut(name, ":"); ok {
			if uri, ok := c.nsBindings[prefix]; ok {
				qn = xpath3.QualifiedName{URI: uri, Name: local}
			} else {
				qn.Name = local
			}
		}
	}

	if c.stylesheet.decimalFormats == nil {
		c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
		c.stylesheet.decimalFmtPrec = make(map[xpath3.QualifiedName]int)
		c.stylesheet.decimalFmtSet = make(map[xpath3.QualifiedName]map[string]struct{})
	}

	// XTSE1295: zero-digit must be a Unicode character with digit value zero
	if v := getAttr(elem, "zero-digit"); v != "" {
		zd := firstRune(v)
		if !unicode.IsDigit(zd) || !isDigitZero(zd) {
			return staticError(errCodeXTSE1295,
				"zero-digit %q is not a Unicode digit-zero character", string(zd))
		}
	}

	// Get or create the merged format
	df, exists := c.stylesheet.decimalFormats[qn]
	existingPrec := c.stylesheet.decimalFmtPrec[qn]
	if !exists {
		df = xpath3.DefaultDecimalFormat()
	}

	// XTSE1290: record potential conflicts for deferred checking after all imports.
	// Higher precedence declarations can override lower-precedence conflicts.
	if exists && c.importPrec == existingPrec {
		prevSet := c.stylesheet.decimalFmtSet[qn]
		if err := checkDecimalFormatConflictExplicit(df, elem, prevSet); err != nil {
			// Record the conflict and its precedence for deferred checking
			if c.stylesheet.decimalFmtConflicts == nil {
				c.stylesheet.decimalFmtConflicts = make(map[xpath3.QualifiedName]int)
			}
			c.stylesheet.decimalFmtConflicts[qn] = c.importPrec
		}
	}

	// If current declaration has higher or equal import precedence, merge properties.
	// Lower precedence declarations' explicitly-set properties are inherited.
	if !exists || c.importPrec >= existingPrec {
		// Merge explicitly-set properties
		if v := getAttr(elem, "decimal-separator"); v != "" {
			df.DecimalSeparator = firstRune(v)
		}
		if v := getAttr(elem, "grouping-separator"); v != "" {
			df.GroupingSeparator = firstRune(v)
		}
		if v := getAttr(elem, "percent"); v != "" {
			df.Percent = firstRune(v)
		}
		if v := getAttr(elem, "per-mille"); v != "" {
			df.PerMille = firstRune(v)
		}
		if v := getAttr(elem, "zero-digit"); v != "" {
			df.ZeroDigit = firstRune(v)
		}
		if v := getAttr(elem, "digit"); v != "" {
			df.Digit = firstRune(v)
		}
		if v := getAttr(elem, "pattern-separator"); v != "" {
			df.PatternSeparator = firstRune(v)
		}
		if v := getAttr(elem, "exponent-separator"); v != "" {
			df.ExponentSeparator = firstRune(v)
		}
		if v := getAttr(elem, "infinity"); v != "" {
			df.Infinity = v
		}
		if v := getAttr(elem, "NaN"); v != "" {
			df.NaN = v
		}
		if v := getAttr(elem, "minus-sign"); v != "" {
			df.MinusSign = firstRune(v)
		}

		c.stylesheet.decimalFormats[qn] = df
		c.stylesheet.decimalFmtPrec[qn] = c.importPrec

		// Track which properties were explicitly set
		if c.stylesheet.decimalFmtSet[qn] == nil {
			c.stylesheet.decimalFmtSet[qn] = make(map[string]struct{})
		}
		// If higher precedence, clear previously tracked properties
		if exists && c.importPrec > existingPrec {
			c.stylesheet.decimalFmtSet[qn] = make(map[string]struct{})
		}
		setProps := c.stylesheet.decimalFmtSet[qn]
		for _, attr := range []string{"decimal-separator", "grouping-separator", "percent",
			"per-mille", "zero-digit", "digit", "pattern-separator", "exponent-separator",
			"infinity", "NaN", "minus-sign"} {
			if getAttr(elem, attr) != "" {
				setProps[attr] = struct{}{}
			}
		}
	}
	return nil
}

// isDigitZero returns true if r is a Unicode digit with numeric value 0.
func isDigitZero(r rune) bool {
	// Unicode digit blocks are contiguous runs of 10 codepoints.
	// The zero digit is at a position where (r - block_start) == 0.
	// We check that r-1 is NOT a digit (meaning r is the first in its block).
	return r == '0' || (r > '0' && unicode.IsDigit(r) && !unicode.IsDigit(r-1))
}

// checkDecimalFormatCharConflicts raises XTSE1300 if any two formatting characters
// in the decimal format are the same.
func checkDecimalFormatCharConflicts(df xpath3.DecimalFormat) error {
	// Collect all formatting characters and check for duplicates
	type charRole struct {
		char rune
		role string
	}
	roles := []charRole{
		{df.DecimalSeparator, "decimal-separator"},
		{df.GroupingSeparator, "grouping-separator"},
		{df.Percent, "percent"},
		{df.PerMille, "per-mille"},
		{df.ZeroDigit, "zero-digit"},
		{df.Digit, "digit"},
		{df.PatternSeparator, "pattern-separator"},
		{df.ExponentSeparator, "exponent-separator"},
	}
	for i := range roles {
		for j := i + 1; j < len(roles); j++ {
			if roles[i].char == roles[j].char {
				return staticError(errCodeXTSE1300,
					"%s and %s use the same character %q",
					roles[i].role, roles[j].role, string(roles[i].char))
			}
		}
	}
	return nil
}

// checkDecimalFormatConflictExplicit raises XTSE1290 if any property
// explicitly set in elem conflicts with a property explicitly set in a prior
// declaration of the same name at the same import precedence.
func checkDecimalFormatConflictExplicit(existing xpath3.DecimalFormat, elem *helium.Element, prevSet map[string]struct{}) error {
	type runeCheck struct {
		attr string
		cur  rune
	}
	runeChecks := []runeCheck{
		{"decimal-separator", existing.DecimalSeparator},
		{"grouping-separator", existing.GroupingSeparator},
		{"percent", existing.Percent},
		{"per-mille", existing.PerMille},
		{"zero-digit", existing.ZeroDigit},
		{"digit", existing.Digit},
		{"pattern-separator", existing.PatternSeparator},
		{"exponent-separator", existing.ExponentSeparator},
		{"minus-sign", existing.MinusSign},
	}
	for _, rc := range runeChecks {
		v := getAttr(elem, rc.attr)
		if v == "" {
			continue
		}
		// Only conflict if the property was explicitly set in a prior declaration
		if _, wasSet := prevSet[rc.attr]; !wasSet {
			continue
		}
		newVal := firstRune(v)
		if newVal != rc.cur {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: %s was %q, now %q",
				rc.attr, string(rc.cur), string(newVal))
		}
	}
	// String properties
	if v := getAttr(elem, "infinity"); v != "" {
		if _, wasSet := prevSet["infinity"]; wasSet && v != existing.Infinity {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: infinity was %q, now %q",
				existing.Infinity, v)
		}
	}
	if v := getAttr(elem, "NaN"); v != "" {
		if _, wasSet := prevSet["NaN"]; wasSet && v != existing.NaN {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: NaN was %q, now %q",
				existing.NaN, v)
		}
	}
	return nil
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func (c *compiler) compileSpaceHandling(ctx context.Context, elem *helium.Element, strip bool) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		"elements": {},
	}); err != nil {
		return err
	}
	// XTSE0260: strip-space/preserve-space must be empty
	kind := "strip-space"
	if !strip {
		kind = "preserve-space"
	}
	if err := c.validateEmptyElement(ctx, elem, "xsl:"+kind); err != nil {
		return err
	}
	elements := getAttr(elem, "elements")
	if !elem.HasAttribute("elements") {
		return staticError(errCodeXTSE0010, "xsl:%s requires the elements attribute", kind)
	}

	// Apply per-element xpath-default-namespace
	savedXPathDefaultNS := c.xpathDefaultNS
	if xdn, ok := elem.GetAttribute("xpath-default-namespace"); ok {
		c.xpathDefaultNS = xdn
	}
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	// Resolve prefixes using the namespace context in scope at this
	// declaration. Imported modules may bind the same prefix to a different
	// URI than the importing module, so resolution must use the element's own
	// in-scope bindings rather than a single flat stylesheet-wide map.
	declNS := c.spaceDeclNamespaces(elem)

	for name := range strings.FieldsSeq(elements) {
		nt := nameTest{ImportPrec: c.importPrec}
		// Handle EQName syntax: Q{uri}local
		if strings.HasPrefix(name, "Q{") {
			if closeIdx := strings.IndexByte(name, '}'); closeIdx > 0 {
				nt.URI = name[2:closeIdx]
				nt.HasURI = true
				nt.Local = name[closeIdx+1:]
			} else {
				nt.Local = name
			}
		} else if prefix, local, ok := strings.Cut(name, ":"); ok {
			nt.Prefix = prefix
			nt.Local = local
			// "*:NCName" matches the local name in any namespace; no URI binding.
			if prefix != "*" {
				// XTSE0280: the prefix must be declared in scope here.
				uri, ok := declNS[prefix]
				if !ok {
					return staticError(errCodeXTSE0280, "undeclared namespace prefix %q in xsl:%s elements attribute", prefix, kind)
				}
				nt.URI = uri
				nt.HasURI = true
			}
		} else {
			nt.Local = name
			// Apply xpath-default-namespace for unprefixed names (not wildcards)
			if name != "*" && c.xpathDefaultNS != "" {
				nt.URI = c.xpathDefaultNS
				nt.HasURI = true
			}
		}
		if strip {
			c.stylesheet.stripSpace = append(c.stylesheet.stripSpace, nt)
		} else {
			c.stylesheet.preserveSpace = append(c.stylesheet.preserveSpace, nt)
		}
	}
	return nil
}

// spaceDeclNamespaces returns the namespace bindings in scope at an
// xsl:strip-space / xsl:preserve-space declaration. Only the element's own
// ancestor chain in the source tree (plus the implicit "xml" prefix) is
// authoritative for declaration-time resolution. Compiler-wide bindings are NOT
// consulted: they may carry prefixes from imported modules that are not in
// scope here, which would wrongly accept an undeclared prefix instead of
// raising XTSE0280.
func (c *compiler) spaceDeclNamespaces(elem *helium.Element) map[string]string {
	bindings := map[string]string{
		// The "xml" prefix is implicitly bound in every XML document.
		lexicon.PrefixXML: lexicon.NamespaceXML,
	}
	// Walk descendant-to-ancestor; nearest declaration wins.
	for n := helium.Node(elem); n != nil; n = n.Parent() {
		e, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		for _, ns := range e.Namespaces() {
			if _, exists := bindings[ns.Prefix()]; !exists {
				bindings[ns.Prefix()] = ns.URI()
			}
		}
	}
	return bindings
}

// nameTestKey returns a canonical conflict key for a nameTest. The key encodes
// the NameTest KIND in addition to its resolved name so that NameTests of
// different shapes never collide. A collision would raise a false XTSE0270 for a
// strip/preserve pair whose match priorities differ and is therefore resolved at
// runtime rather than being a genuine conflict. XTSE0270 must fire only for two
// rules of the SAME kind and SAME name at the same import precedence.
//
// The four distinguished kinds (with their effective match priorities) are:
//
//   - universal wildcard "*"        (priority -0.5)
//   - namespace wildcard "prefix:*" / "Q{uri}*" (priority -0.25)
//   - local-name wildcard "*:local" (priority -0.25)
//   - exact expanded name           (priority  0)
//
// The two -0.25 kinds share a priority but never match the same node by the same
// criterion (one fixes the namespace, the other fixes the local name), so they
// are kept as separate kinds.
func nameTestKey(nt nameTest) string {
	uri := ""
	if nt.HasURI {
		uri = nt.URI
	}
	switch {
	case nt.Prefix == "*":
		// "*:local" — local-name wildcard (namespace unconstrained).
		return "L:" + nt.Local
	case nt.Local == "*" && nt.HasURI:
		// "prefix:*" / "Q{uri}*" — namespace wildcard.
		return "N:" + uri
	case nt.Local == "*":
		// "*" — universal wildcard.
		return "U:"
	default:
		// Exact expanded name.
		return "E:" + helium.ClarkName(uri, nt.Local)
	}
}

// nameTestURI returns the effective namespace URI of a NameTest ("" when unset).
func nameTestURI(nt nameTest) string {
	if nt.HasURI {
		return nt.URI
	}
	return ""
}

// isUniversalNameTest reports whether nt is the universal wildcard "*", which
// matches every expanded name in any namespace.
func isUniversalNameTest(nt nameTest) bool {
	return nt.Local == "*" && nt.Prefix == "" && !nt.HasURI
}

// isNSWildcardNameTest reports whether nt is a namespace wildcard "Q{uri}*" /
// "prefix:*", which matches every local name in a fixed namespace.
func isNSWildcardNameTest(nt nameTest) bool {
	return nt.Local == "*" && !isUniversalNameTest(nt)
}

// isLocalWildcardNameTest reports whether nt is a local-name wildcard "*:local",
// which matches a fixed local name in any namespace.
func isLocalWildcardNameTest(nt nameTest) bool {
	return nt.Prefix == "*"
}

// ruleCoversRegion reports whether NameTest h matches EVERY expanded name in
// region's match-set (h's set ⊇ region's set). A region is one of the four
// NameTest kinds. Because a wildcard region has infinitely many names, only a
// rule at least as general as the region can cover it; finitely many narrower
// rules never can. This makes single-rule coverage both sound and complete for
// deciding whether a higher-precedence layer resolves a strip/preserve overlap.
func ruleCoversRegion(h, region nameTest) bool {
	switch {
	case isUniversalNameTest(h):
		return true // "*" matches every name
	case isNSWildcardNameTest(h):
		// Covers a region only if every name in the region lies in uri(h):
		// the same namespace wildcard, or an exact name in that namespace.
		switch {
		case isNSWildcardNameTest(region):
			return nameTestURI(h) == nameTestURI(region)
		case !isUniversalNameTest(region) && !isLocalWildcardNameTest(region) && region.Local != "*":
			// region is exact
			return nameTestURI(h) == nameTestURI(region)
		default:
			return false // universal / local-name-wildcard region spans other namespaces
		}
	case isLocalWildcardNameTest(h):
		// Covers a region only if every name in the region has local name h.Local:
		// the same local-name wildcard, or an exact name with that local.
		switch {
		case isLocalWildcardNameTest(region):
			return h.Local == region.Local
		case !isUniversalNameTest(region) && !isNSWildcardNameTest(region):
			// region is exact
			return h.Local == region.Local
		default:
			return false // universal / namespace-wildcard region spans other local names
		}
	default: // h is exact; can only cover a single exact name equal to it
		if isUniversalNameTest(region) || isNSWildcardNameTest(region) || isLocalWildcardNameTest(region) {
			return false
		}
		return nameTestURI(h) == nameTestURI(region) && h.Local == region.Local
	}
}

// checkSpaceConflicts detects NameTests that appear in both xsl:strip-space and
// xsl:preserve-space. Conflicts are resolved per-name by import precedence and
// match priority: for a given element name, the rule of highest import precedence
// (then highest match priority) decides whether it is stripped or preserved.
//
// Per the W3C XTSE0270 semantics, a STATIC conflict arises for a strip rule and a
// preserve rule that match an OVERLAPPING set of names at the SAME import
// precedence AND the SAME match priority, where neither rule outranks the other
// for the names they both match. Two cases produce such an unresolvable overlap:
//
//   - SAME resolved NameTest (same kind, same name-set): e.g. "item" vs "item",
//     or "*:item" vs "*:item". Detected via nameTestKey.
//   - DIFFERENT-shape wildcards of EQUAL priority whose match-sets intersect:
//     "*:item" (local-name wildcard) vs "Q{urn:A}*" (namespace wildcard) both
//     match Q{urn:A}item at priority -0.25, and neither is more specific than the
//     other on that name, so the strip/preserve outcome is undecidable. Detected
//     via nameTestsOverlapEqualPriority + the derived overlap region.
//
// A NameTest of HIGHER priority that intersects the other (e.g. an exact name vs
// a wildcard) is NOT a static conflict: the more specific rule wins per-name at
// RUNTIME, so those pairs are deliberately left to runtime resolution.
//
// A candidate same-precedence/priority overlap is still NOT a genuine conflict
// when a rule of STRICTLY HIGHER import precedence (of either kind) covers the
// entire overlap region: per XSLT's per-name precedence resolution, XTSE0270
// fires only at the HIGHEST precedence that matches a given name, so a
// higher-precedence rule resolves the names it matches (see spaceRegionCovered).
//
// The check must NOT be filtered by each kind's globally-highest precedence: an
// unrelated higher-precedence rule (e.g. a higher-precedence strip-space for "b")
// does not cover the name-set of an "a" vs "a" conflict, so it must not cancel
// that genuine same-precedence conflict.
func (c *compiler) checkSpaceConflicts(_ context.Context) error {
	if len(c.stylesheet.stripSpace) == 0 || len(c.stylesheet.preserveSpace) == 0 {
		return nil
	}
	// Scan every strip/preserve pair. A pair is a candidate conflict when its
	// rules carry equal import precedence and an unresolvable overlap (same
	// name-set, or different-shape wildcards of equal priority that intersect). It
	// is a genuine conflict only if no rule of higher import precedence covers the
	// overlap region.
	for _, sNT := range c.stylesheet.stripSpace {
		sKey := nameTestKey(sNT)
		for _, pNT := range c.stylesheet.preserveSpace {
			if sNT.ImportPrec != pNT.ImportPrec {
				continue
			}
			region, ok := spaceConflictRegion(sNT, pNT, sKey)
			if !ok {
				continue
			}
			if c.spaceRegionCovered(region, sNT.ImportPrec) {
				continue
			}
			return staticError(errCodeXTSE0270,
				"conflicting xsl:strip-space and xsl:preserve-space for %q at the same import precedence",
				nameTestKey(region))
		}
	}
	return nil
}

// spaceConflictRegion reports whether a strip NameTest s and a preserve NameTest
// p form an unresolvable static overlap (equal match priority, intersecting
// match-sets, neither outranking the other) and, if so, returns the NameTest
// describing the overlap region — the set of names BOTH match. sKey is
// nameTestKey(s), passed in to avoid recomputation.
//
// Only EQUAL-priority pairs can be statically unresolvable: when priorities
// differ, the higher-priority rule wins per-name at runtime. Among equal-priority
// pairs the cases are:
//
//   - same kind, same name-set (equal nameTestKey): the region is that shared
//     NameTest itself.
//   - namespace wildcard "Q{ns}*" vs local-name wildcard "*:local" (both priority
//     -0.25): they always intersect on the single exact name Q{ns}local, which is
//     the overlap region.
//
// Two namespace wildcards (or two local-name wildcards) of DIFFERENT name never
// intersect, and equal-name same-kind pairs are caught by the equal-key case, so
// no other equal-priority cross pairs reach the wildcard branch.
func spaceConflictRegion(s, p nameTest, sKey string) (nameTest, bool) {
	if sKey == nameTestKey(p) {
		// Same resolved NameTest: the overlap region is the shared name-set.
		return s, true
	}
	if nameTestPriority(s) != nameTestPriority(p) {
		// Differing priority: the more specific rule decides per-name at runtime.
		return nameTest{}, false
	}
	// Equal priority, different keys. The only such intersecting pair is a
	// namespace wildcard vs a local-name wildcard; their overlap is the single
	// exact name Q{ns}local.
	ns, ok := nsAndLocalWildcardOverlap(s, p)
	if !ok {
		return nameTest{}, false
	}
	return ns, true
}

// nsAndLocalWildcardOverlap returns the exact-name overlap region of a namespace
// wildcard "Q{ns}*" and a local-name wildcard "*:local" — the single name
// Q{ns}local that both match — in either argument order. ok is false when the
// pair is not one namespace wildcard and one local-name wildcard.
func nsAndLocalWildcardOverlap(a, b nameTest) (nameTest, bool) {
	nsw, lw := a, b
	if isLocalWildcardNameTest(a) && isNSWildcardNameTest(b) {
		nsw, lw = b, a
	}
	if !isNSWildcardNameTest(nsw) || !isLocalWildcardNameTest(lw) {
		return nameTest{}, false
	}
	// Exact name in the namespace of the namespace wildcard with the local name of
	// the local-name wildcard.
	return nameTest{
		Local:  lw.Local,
		URI:    nameTestURI(nsw),
		HasURI: nsw.HasURI,
	}, true
}

// spaceRegionCovered reports whether the entire overlap region of a conflicting
// strip/preserve pair (declared at import precedence prec) is matched by some
// rule of STRICTLY HIGHER import precedence — strip OR preserve, either kind.
// Such a rule resolves every name in the region per XSLT's per-name precedence
// rule, so no XTSE0270 conflict survives. Coverage of a (possibly wildcard)
// region is decided one rule at a time: a wildcard region is only ever covered by
// a single rule at least as general as it, so a union of narrower higher rules
// can never spuriously "cover" it (see ruleCoversRegion).
func (c *compiler) spaceRegionCovered(region nameTest, prec int) bool {
	for _, h := range c.stylesheet.stripSpace {
		if h.ImportPrec > prec && ruleCoversRegion(h, region) {
			return true
		}
	}
	for _, h := range c.stylesheet.preserveSpace {
		if h.ImportPrec > prec && ruleCoversRegion(h, region) {
			return true
		}
	}
	return false
}
