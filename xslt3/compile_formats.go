package xslt3

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

func (c *compiler) compileCharacterMap(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "use-character-maps": {}, "use-when": {},
	}); err != nil {
		return err
	}
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:character-map requires name attribute")
	}
	name = resolveQName(name, c.nsBindings)

	cm := &characterMapDef{
		Name:     name,
		Mappings: make(map[rune]string),
	}

	if ucm := getAttr(elem, paramUseCharacterMaps); ucm != "" {
		for _, n := range strings.Fields(ucm) {
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
		for r, s := range cm.Mappings {
			existing.Mappings[r] = s
		}
		existing.UseCharacterMaps = append(existing.UseCharacterMaps, cm.UseCharacterMaps...)
	} else {
		c.stylesheet.characterMaps[name] = cm
	}
	return nil
}

func (c *compiler) compileKey(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "match": {}, "use": {}, "collation": {}, "composite": {},
		"use-when": {}, "default-collation": {},
	}); err != nil {
		return err
	}
	// Collect local namespace declarations (e.g., xmlns:ex="..." on xsl:key)
	c.collectNamespaces(elem)

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
	if !isValidQName(name) && !isValidEQName(name) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:key", name)
	}

	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires match attribute")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
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
	}

	useAttr := getAttr(elem, "use")
	hasContent := c.hasEffectiveContent(elem)
	// XTSE1205: must have either use attr or content, not both, and not neither
	if useAttr != "" && hasContent {
		return staticError(errCodeXTSE1205, "xsl:key must not have both a use attribute and content")
	}
	if useAttr == "" && !hasContent {
		return staticError(errCodeXTSE1205, "xsl:key must have either a use attribute or content")
	}
	if useAttr != "" {
		useExpr, err := compileXPath(useAttr, c.nsBindings)
		if err != nil {
			return err
		}
		kd.Use = useExpr
	} else {
		// XSLT 2.0+: key may use body content instead of use attribute.
		body, _, err := c.compileTemplateBody(elem)
		if err != nil {
			return err
		}
		kd.Body = body
	}

	c.stylesheet.keys[expandedName] = append(c.stylesheet.keys[expandedName], kd)
	return nil
}

func (c *compiler) compileOutput(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "method": {}, "version": {}, "encoding": {},
		"omit-xml-declaration": {}, "standalone": {}, "doctype-public": {},
		"doctype-system": {}, "cdata-section-elements": {}, "indent": {},
		"media-type": {}, "byte-order-mark": {}, "escape-uri-attributes": {},
		"include-content-type": {}, "normalization-form": {},
		"undeclare-prefixes": {}, "use-character-maps": {},
		"suppress-indentation": {}, "html-version": {},
		"item-separator": {}, "json-node-output-method": {},
		"parameter-document": {}, "build-tree": {},
		"allow-duplicate-names": {},
		"use-when":              {},
	}); err != nil {
		return err
	}
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
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
		} else if !isValidQName(methodStr) && !isValidEQName(methodStr) {
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
		outDef.Encoding = "UTF-8"
	}

	// Validate and parse boolean output attributes.
	// SEPM0016: invalid boolean values.
	if v := getAttr(elem, paramIndent); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@indent", v)
		}
		outDef.Indent = b
		outDef.IndentRaw = v
	}
	if v := getAttr(elem, paramOmitXMLDeclaration); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@omit-xml-declaration", v)
		}
		outDef.OmitDeclaration = b
		outDef.OmitDeclarationExplicit = true
	}
	if v := getAttr(elem, paramUndeclarePrefixes); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@undeclare-prefixes", v)
		}
		outDef.UndeclarePrefixes = b
	}
	if v := getAttr(elem, paramByteOrderMark); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@byte-order-mark", v)
		}
		outDef.ByteOrderMark = b
	}
	// Parse escape-uri-attributes.
	if v := getAttr(elem, paramEscapeURIAttributes); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@escape-uri-attributes", v)
		}
		outDef.EscapeURIAttributes = &b
	}
	if v := getAttr(elem, paramIncludeContentType); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@include-content-type", v)
		}
		outDef.IncludeContentType = &b
	}
	// Validate standalone: must be "yes", "no", "omit", or boolean equivalents.
	if v := getAttr(elem, paramStandalone); v != "" {
		v = strings.TrimSpace(v)
		switch v {
		case lexicon.ValueYes, lexicon.ValueNo, "omit":
			// valid as-is
		case "true", "1":
			v = lexicon.ValueYes
		case "false", "0":
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
		for _, n := range strings.Fields(ucm) {
			outDef.UseCharacterMaps = append(outDef.UseCharacterMaps, resolveQName(n, c.nsBindings))
		}
	}

	if v := getAttr(elem, paramJSONNodeOutputMethod); v != "" {
		outDef.JSONNodeOutputMethod = strings.TrimSpace(v)
	}

	if v := getAttr(elem, paramAllowDuplicateNames); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@allow-duplicate-names", v)
		}
		outDef.AllowDuplicateNames = b
	}

	if v := getAttr(elem, paramSuppressIndentation); v != "" {
		names := strings.Fields(v)
		resolved := make([]string, len(names))
		for i, n := range names {
			resolved[i] = resolveQName(n, c.nsBindings)
		}
		outDef.SuppressIndentation = resolved
	}

	if v := getAttr(elem, paramBuildTree); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@build-tree", v)
		}
		outDef.BuildTree = &b
	}

	if v := getAttr(elem, paramParameterDocument); v != "" {
		outDef.ParameterDocument = v
		baseURI := stylesheetBaseURI(elem, c.baseURI)
		if err := c.loadParameterDocument(outDef, baseURI, v); err != nil {
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
	}

	c.stylesheet.outputs[name] = outDef
	return nil
}

// loadParameterDocument loads a serialization parameter document (XSLT 3.0 §9.2)
// and applies its settings to the given OutputDef. Parameters explicitly set on
// the xsl:output element take precedence; the parameter document provides defaults.
func (c *compiler) loadParameterDocument(outDef *OutputDef, baseURI, href string) error {
	return loadParameterDocumentFromFile(c.ctx, outDef, baseURI, href)
}

// loadParameterDocumentFromFile loads a serialization parameter document and
// applies its settings to the given OutputDef. This standalone version can be
// called at both compile-time and runtime.
func loadParameterDocumentFromFile(ctx context.Context, outDef *OutputDef, baseURI, href string) error {
	uri := href
	if baseURI != "" && !strings.Contains(href, "://") && !filepath.IsAbs(href) {
		baseDir := filepath.Dir(baseURI)
		uri = filepath.Join(baseDir, href)
	}

	data, err := os.ReadFile(uri)
	if err != nil {
		return staticError(errCodeXTSE0090, "cannot read parameter-document %q: %v", href, err)
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return staticError(errCodeXTSE0090, "cannot parse parameter-document %q: %v", href, err)
	}
	root := doc.DocumentElement()
	if root == nil || root.LocalName() != "serialization-parameters" || root.URI() != lexicon.NamespaceSerialization {
		return staticError(errCodeXTSE0090, "parameter-document %q: root element must be {%s}serialization-parameters", href, lexicon.NamespaceSerialization)
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
			if val != "" {
				if b, ok := parseXSDBool(val); ok {
					outDef.AllowDuplicateNames = b
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
	return nil
}

func (c *compiler) compileAttributeSet(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "use-attribute-sets": {}, "visibility": {},
		"streamable": {}, "use-when": {},
	}); err != nil {
		return err
	}
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:attribute-set requires name attribute")
	}
	if !isValidQName(name) && !isValidEQName(name) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:attribute-set", name)
	}
	if err := c.checkQNamePrefix(name, "xsl:attribute-set"); err != nil {
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
		for _, n := range strings.Fields(uas) {
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
		inst, err := c.compileAttribute(childElem)
		if err != nil {
			return err
		}
		asd.Attrs = append(asd.Attrs, inst)
	}

	if c.stylesheet.attributeSets == nil {
		c.stylesheet.attributeSets = make(map[string]*attributeSetDef)
	}
	// Build a part for this declaration
	effectiveBase := stylesheetBaseURI(elem, c.baseURI)
	part := attributeSetPart{
		UseAttrSets:   asd.UseAttrSets,
		Attrs:         asd.Attrs,
		StaticBaseURI: effectiveBase,
	}
	// Merge same-named attribute-sets (XSLT spec: union of all definitions)
	if existing, ok := c.stylesheet.attributeSets[name]; ok {
		existing.Attrs = append(existing.Attrs, asd.Attrs...)
		existing.UseAttrSets = append(existing.UseAttrSets, asd.UseAttrSets...)
		existing.Parts = append(existing.Parts, part)
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
				if ref == "{"+lexicon.NamespaceXSLT+"}original" && asd.OriginalAttrSet != nil {
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

func (c *compiler) compileDecimalFormat(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "decimal-separator": {}, "grouping-separator": {},
		"infinity": {}, "minus-sign": {}, "NaN": {}, "percent": {},
		"per-mille": {}, "zero-digit": {}, "digit": {},
		"pattern-separator": {}, "exponent-separator": {},
		"use-when": {},
	}); err != nil {
		return err
	}
	// Push element-local namespace declarations so prefixed names resolve correctly
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
	qn := xpath3.QualifiedName{}
	if name != "" {
		if !isValidQName(name) && !isValidEQName(name) {
			return staticError(errCodeXTSE0020, "invalid name %q on xsl:decimal-format", name)
		}
		qn = xpath3.QualifiedName{Name: name}
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			prefix := name[:idx]
			local := name[idx+1:]
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
	for i := 0; i < len(roles); i++ {
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

func (c *compiler) compileSpaceHandling(elem *helium.Element, strip bool) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"elements": {},
	}); err != nil {
		return err
	}
	// XTSE0260: strip-space/preserve-space must be empty
	kind := "strip-space"
	if !strip {
		kind = "preserve-space"
	}
	if err := c.validateEmptyElement(elem, "xsl:"+kind); err != nil {
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

	for _, name := range strings.Fields(elements) {
		nt := nameTest{}
		// Handle EQName syntax: Q{uri}local
		if strings.HasPrefix(name, "Q{") {
			if closeIdx := strings.IndexByte(name, '}'); closeIdx > 0 {
				nt.Prefix = "Q{" + name[2:closeIdx] + "}"
				nt.Local = name[closeIdx+1:]
			} else {
				nt.Local = name
			}
		} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
			nt.Prefix = name[:idx]
			nt.Local = name[idx+1:]
			// XTSE0280: check that the prefix is declared (skip EQName and wildcards)
			if !strings.HasPrefix(nt.Prefix, "Q{") && nt.Prefix != "*" {
				if _, ok := c.nsBindings[nt.Prefix]; !ok {
					return staticError(errCodeXTSE0280, "undeclared namespace prefix %q in xsl:%s elements attribute", nt.Prefix, kind)
				}
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

// nameTestKey returns a canonical key for a nameTest by resolving the prefix
// to a URI. EQName prefixes of the form "Q{uri}" are used as-is.
func (c *compiler) nameTestKey(nt nameTest) string {
	uri := ""
	if strings.HasPrefix(nt.Prefix, "Q{") {
		uri = nt.Prefix[2 : len(nt.Prefix)-1]
	} else if nt.Prefix != "" {
		uri = c.nsBindings[nt.Prefix]
	} else if nt.HasURI {
		uri = nt.URI
	}
	return "{" + uri + "}" + nt.Local
}

// checkSpaceConflicts detects NameTests that appear in both xsl:strip-space
// and xsl:preserve-space at the same import precedence (XTSE0270).
func (c *compiler) checkSpaceConflicts() error {
	if len(c.stylesheet.stripSpace) == 0 || len(c.stylesheet.preserveSpace) == 0 {
		return nil
	}
	stripKeys := make(map[string]struct{}, len(c.stylesheet.stripSpace))
	for _, nt := range c.stylesheet.stripSpace {
		stripKeys[c.nameTestKey(nt)] = struct{}{}
	}
	for _, nt := range c.stylesheet.preserveSpace {
		key := c.nameTestKey(nt)
		if _, ok := stripKeys[key]; ok {
			return staticError(errCodeXTSE0270,
				"conflicting xsl:strip-space and xsl:preserve-space for %q", key)
		}
	}
	return nil
}
