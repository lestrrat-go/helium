package xslt3

import (
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

var templateAllowedAttrs = map[string]struct{}{
	"match": {}, "name": {}, "priority": {}, "mode": {}, "as": {},
	"visibility": {}, "use-when": {},
}

func (c *compiler) compileTemplate(elem *helium.Element) error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	if err := c.validateXSLTAttrs(elem, templateAllowedAttrs); err != nil {
		return err
	}
	// Collect namespace declarations and xpath-default-namespace before
	// evaluating use-when so the expression has the correct namespace context.
	c.collectNamespaces(elem)
	savedXPathDefaultNS := c.xpathDefaultNS
	if xdn := getAttr(elem, "xpath-default-namespace"); xdn != "" {
		c.xpathDefaultNS = xdn
	}

	// Evaluate use-when before compiling the template.
	if uw := getAttr(elem, "use-when"); uw != "" {
		include, err := c.evaluateUseWhen(uw)
		if err != nil {
			c.xpathDefaultNS = savedXPathDefaultNS
			return err
		}
		if !include {
			c.xpathDefaultNS = savedXPathDefaultNS
			return nil
		}
	}

	// Compute effective base URI considering xml:base on the template element.
	// Only check the template element itself (not ancestors), since c.baseURI
	// already accounts for xml:base on the stylesheet root and includes.
	templateBaseURI := c.baseURI
	if xmlBase, ok := elem.GetAttributeNS("base", helium.XMLNamespace); ok && xmlBase != "" {
		templateBaseURI = helium.BuildURI(xmlBase, c.baseURI)
	}

	tmpl := &template{
		ImportPrec:    c.importPrec,
		MinImportPrec: c.minImportPrec,
		BaseURI:       templateBaseURI,
	}
	tmpl.XPathDefaultNS = c.xpathDefaultNS
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	// Inherit or override default-collation
	savedDefaultCollation := c.defaultCollation
	if dc := getAttr(elem, "default-collation"); dc != "" {
		if uri := resolveDefaultCollation(dc); uri != "" {
			c.defaultCollation = uri
		}
	}
	tmpl.DefaultCollation = c.defaultCollation
	defer func() { c.defaultCollation = savedDefaultCollation }()

	matchAttr := getAttr(elem, "match")
	if matchAttr != "" {
		p, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return err
		}
		tmpl.Match = p
		// Defer function validation until after all xsl:function declarations are processed.
		c.pendingPatternValidations = append(c.pendingPatternValidations, pendingPatternValidation{p, matchAttr})
	}

	nameAttr := strings.TrimSpace(getAttr(elem, "name"))
	if nameAttr != "" && !isValidQName(nameAttr) && !isValidEQName(nameAttr) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:template", nameAttr)
	}
	if nameAttr != "" {
		if err := c.checkQNamePrefix(nameAttr, "xsl:template"); err != nil {
			return err
		}
	}
	tmpl.Name = resolveQName(nameAttr, c.nsBindings)

	// XTSE0080: template name must not be in the XSLT namespace
	// Exception: xsl:initial-template is explicitly allowed (XSLT 3.0 §3.11).
	if tmpl.Name != "" && strings.HasPrefix(tmpl.Name, "{"+lexicon.NamespaceXSLT+"}") && tmpl.Name != helium.ClarkName(lexicon.NamespaceXSLT, "initial-template") {
		return staticError(errCodeXTSE0080, "xsl:template name %q is in the XSLT namespace", getAttr(elem, "name"))
	}

	// XTSE0500: template must have match or name (or both).
	if matchAttr == "" && tmpl.Name == "" {
		return staticError(errCodeXTSE0500, "xsl:template must have a @match or @name attribute")
	}
	// XTSE0500: template without match must not have mode or priority.
	if matchAttr == "" {
		if _, hasMode := elem.GetAttribute("mode"); hasMode {
			return staticError(errCodeXTSE0500, "xsl:template without @match must not have @mode")
		}
		if _, hasPrio := elem.GetAttribute("priority"); hasPrio {
			return staticError(errCodeXTSE0500, "xsl:template without @match must not have @priority")
		}
	}
	// XTSE0500: template without name must not have visibility.
	if tmpl.Name == "" {
		if _, hasVis := elem.GetAttribute("visibility"); hasVis {
			return staticError(errCodeXTSE0500, "xsl:template without @name must not have @visibility")
		}
	}

	// XSLT 3.0 §3.8.2: default-mode on xsl:template affects both the
	// template's own mode (when mode is omitted) and xsl:apply-templates
	// within the template body. Read it before resolving the mode so
	// the template's own default-mode is visible to the mode defaulting.
	savedDefaultMode := c.defaultMode
	if dm := getAttr(elem, "default-mode"); dm != "" {
		c.defaultMode = dm
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	modeAttr := getAttr(elem, "mode")
	if modeAttrVal, hasMode := elem.GetAttribute("mode"); hasMode {
		// XTSE0550: empty mode list is invalid.
		if strings.TrimSpace(modeAttrVal) == "" {
			return staticError(errCodeXTSE0550, "mode attribute on xsl:template must not be empty")
		}
	}
	if modeAttr != "" {
		modeFields := strings.Fields(modeAttr)
		seenModes := make(map[string]struct{}, len(modeFields))
		hasAll := false
		for _, m := range modeFields {
			if m[0] != '#' && !isValidQName(m) && !isValidEQName(m) {
				return staticError(errCodeXTSE0550, "invalid mode name %q on xsl:template", m)
			}
			// XTSE0280: check for undeclared prefix in mode name.
			if idx := strings.IndexByte(m, ':'); idx > 0 {
				prefix := m[:idx]
				if _, ok := c.nsBindings[prefix]; !ok {
					return staticError(errCodeXTSE0280, "undeclared namespace prefix %q in mode name %q", prefix, m)
				}
			}
			if m == modeAll {
				hasAll = true
			}
			if _, dup := seenModes[m]; dup {
				return staticError(errCodeXTSE0550, "duplicate mode %q in xsl:template/@mode", m)
			}
			seenModes[m] = struct{}{}
		}
		if hasAll && len(modeFields) > 1 {
			return staticError(errCodeXTSE0550, "#all must not appear with other modes in xsl:template/@mode")
		}
		// Resolve mode QNames to Clark notation for namespace-aware matching
		tmpl.Mode = c.resolveMode(modeAttr)
	}
	// XSLT 3.0 §6.7: if the stylesheet (or an included/imported module) has
	// default-mode, templates without an explicit mode attribute belong to it.
	if tmpl.Mode == "" && c.defaultMode != "" {
		tmpl.Mode = c.resolveMode(c.defaultMode)
	}

	// Record mode usage for XTSE3085 checking (only match templates have modes)
	if matchAttr != "" {
		c.recordModeUsage(tmpl.Mode)
	}

	hasExplicitPriority := false
	if prio := getAttr(elem, "priority"); prio != "" {
		// XTSE0530: priority must be a valid xs:decimal — no exponent notation.
		if !isXSDecimal(prio) {
			return staticError(errCodeXTSE0530, "priority %q is not a valid xs:decimal", prio)
		}
		f, err := strconv.ParseFloat(prio, 64)
		if err != nil {
			return staticError(errCodeXTSE0530, "invalid priority %q: %v", prio, err)
		}
		tmpl.Priority = f
		hasExplicitPriority = true
	} else if tmpl.Match != nil && len(tmpl.Match.Alternatives) == 1 {
		tmpl.Priority = tmpl.Match.Alternatives[0].priority
	}

	// Handle exclude-result-prefixes on xsl:template.
	// Resolve prefixes to URIs at the declaration point so that the
	// exclusion applies to URIs, not prefixes (per XSLT spec).
	savedExcludes := c.localExcludes
	if erp := getAttr(elem, "exclude-result-prefixes"); erp != "" {
		newExcludes := make(map[string]struct{})
		for k, v := range c.localExcludes {
			newExcludes[k] = v
		}
		if erp == "#all" {
			for prefix := range c.stylesheet.namespaces {
				if uri, ok := c.nsBindings[prefix]; ok && uri != "" {
					newExcludes[uri] = struct{}{}
				}
			}
		} else {
			for _, prefix := range strings.Fields(erp) {
				if uri, ok := c.nsBindings[prefix]; ok && uri != "" {
					newExcludes[uri] = struct{}{}
				}
			}
		}
		c.localExcludes = newExcludes
	}

	// Handle expand-text on xsl:template (using GetAttribute to catch empty values)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return staticError(errCodeXTSE0020, "%q is not a valid value for xsl:template/@expand-text", et)
		}
	}

	// Handle xml:space on xsl:template
	savedPreserveSpace := c.preserveSpace
	if xs := getAttr(elem, lexicon.QNameXMLSpace); xs != "" {
		c.preserveSpace = (xs == "preserve")
	}

	// Handle version on xsl:template for forwards-compatible processing
	savedVersion := c.effectiveVersion
	if ver := getAttr(elem, "version"); ver != "" {
		c.effectiveVersion = ver
	}

	// Compile template body: first xsl:param elements, then instructions
	ctxDecl, body, params, err := c.compileTemplateBodyEx(elem, false)
	c.effectiveVersion = savedVersion
	c.expandText = savedExpandText
	c.preserveSpace = savedPreserveSpace
	c.localExcludes = savedExcludes
	if err != nil {
		return err
	}
	tmpl.Params = params
	tmpl.Body = body

	// Store context-item declaration on the template
	if ctxDecl != nil {
		tmpl.ContextItemAs = ctxDecl.as
		tmpl.ContextItemUse = ctxDecl.use
	}
	tmplAs := getAttr(elem, "as")
	if err := c.validateAsSequenceType(tmplAs, "xsl:template"); err != nil {
		return err
	}
	tmpl.As = tmplAs
	tmpl.Visibility = getAttr(elem, "visibility")
	// Only set Version when the template has its own version attribute.
	// Stylesheet-level version is NOT propagated to templates because
	// shadow attributes (_version) may override the stylesheet version.
	tmpl.Version = getAttr(elem, "version")
	if c.stylesheet.isPackage {
		tmpl.OwnerPackage = c.stylesheet
	}

	// Register the template
	c.stylesheet.templates = append(c.stylesheet.templates, tmpl)

	if tmpl.Name != "" {
		if existing, exists := c.stylesheet.namedTemplates[tmpl.Name]; exists {
			// Same import precedence = error; different = higher precedence wins
			if existing.ImportPrec == tmpl.ImportPrec {
				return staticError(errCodeXTSE0080, "duplicate template name %q", tmpl.Name)
			}
			if tmpl.ImportPrec > existing.ImportPrec {
				c.stylesheet.namedTemplates[tmpl.Name] = tmpl
			}
			// else keep existing (higher precedence)
		} else {
			c.stylesheet.namedTemplates[tmpl.Name] = tmpl
		}
	}

	if tmpl.Match != nil {
		// XSLT 3.0 §6.4: A pattern of the form P1 | P2 is treated as
		// separate template rules with the same body, one per alternative.
		// Split union patterns into separate template entries so each gets
		// its own default priority.
		templates := []*template{tmpl}
		if !hasExplicitPriority && len(tmpl.Match.Alternatives) > 1 {
			templates = nil
			for _, alt := range tmpl.Match.Alternatives {
				split := *tmpl // shallow copy shares Body, Params, etc.
				split.Match = &pattern{
					source:         tmpl.Match.source,
					Alternatives:   []*patternAlt{alt},
					xpathDefaultNS: tmpl.Match.xpathDefaultNS,
					nsBindings:     tmpl.Match.nsBindings,
				}
				split.Priority = alt.priority
				splitCopy := split // allocate separate heap object
				templates = append(templates, &splitCopy)
			}
		}

		mode := tmpl.Mode
		for _, t := range templates {
			c.registerTemplateInModes(t, mode)
		}
	}

	return nil
}

// registerTemplateInModes adds a template to the appropriate mode template lists.
func (c *compiler) registerTemplateInModes(tmpl *template, mode string) {
	if mode == modeAll {
		// Register in all existing modes plus default
		for m := range c.stylesheet.modeTemplates {
			c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
		}
		c.stylesheet.modeTemplates[""] = append(c.stylesheet.modeTemplates[""], tmpl)
		// Also store under the modeAll key so findBestTemplate's fallback
		// can find these templates for modes that don't exist yet.
		c.stylesheet.modeTemplates[modeAll] = append(c.stylesheet.modeTemplates[modeAll], tmpl)
		return
	}
	// XSLT 2.0+: mode can be a whitespace-separated list of mode names.
	// Each mode name can be a QName, "#default", "#unnamed", or "#all".
	modes := strings.Fields(mode)
	if len(modes) <= 1 {
		// Single mode (or empty = default mode)
		if mode == modeDefault || mode == modeUnnamed {
			mode = ""
		}
		c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
	} else {
		for _, m := range modes {
			if m == modeDefault || m == modeUnnamed {
				m = ""
			} else if m == modeAll {
				// In a mode list, #all means register in all modes
				c.stylesheet.modeTemplates[modeAll] = append(c.stylesheet.modeTemplates[modeAll], tmpl)
				continue
			}
			c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
		}
	}
}

// contextItemDecl holds the parsed xsl:context-item declaration from a template body.
type contextItemDecl struct {
	as  string // type constraint (e.g., "xs:string", "element()")
	use string // "required", "optional", "absent"
}

func (c *compiler) compileTemplateBody(elem *helium.Element) ([]instruction, []*param, error) {
	_, body, params, err := c.compileTemplateBodyEx(elem, false)
	return body, params, err
}

func (c *compiler) compileTemplateBodyEx(elem *helium.Element, isFunction bool) (*contextItemDecl, []instruction, []*param, error) {
	var params []*param
	var body []instruction
	var ctxDecl *contextItemDecl

	// Pre-scan: find the last declaration/param child so we can strip
	// whitespace-only text in the prologue even under xml:space="preserve".
	var lastDeclNode helium.Node
	for ch := range helium.Children(elem) {
		if e, ok := ch.(*helium.Element); ok && e.URI() == lexicon.NamespaceXSLT {
			ln := e.LocalName()
			if ln == "context-item" || ln == "param" {
				lastDeclNode = ch
			}
		}
	}

	inParams := true
	sawContextItem := false
	pastDecls := lastDeclNode == nil // true if no declarations at all
	sawContent := false              // true once non-whitespace text or non-param/context-item element seen
	for child := range helium.Children(elem) {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == lexicon.NamespaceXSLT && v.LocalName() == lexicon.XSLTElementContextItem {
				// XTSE0010: xsl:context-item is not allowed in xsl:function
				if isFunction {
					return nil, nil, nil, staticError(errCodeXTSE0010, "xsl:context-item is not allowed in xsl:function")
				}
				if sawContextItem {
					return nil, nil, nil, staticError(errCodeXTSE0010, "duplicate xsl:context-item")
				}
				if sawContent {
					return nil, nil, nil, staticError(errCodeXTSE0010, "xsl:context-item must appear before other children")
				}
				if len(params) > 0 {
					return nil, nil, nil, staticError(errCodeXTSE0010, "xsl:context-item must appear before xsl:param")
				}
				if err := c.validateContextItem(v); err != nil {
					return nil, nil, nil, err
				}
				asVal := getAttr(v, "as")
				useVal := strings.TrimSpace(getAttr(v, "use"))
				// Leave empty use as "" — runtime determines the default
				// based on whether template is match or named.
				ctxDecl = &contextItemDecl{as: asVal, use: useVal}
				sawContextItem = true
				if child == lastDeclNode {
					pastDecls = true
				}
				continue
			}
			if v.URI() == lexicon.NamespaceXSLT && v.LocalName() == lexicon.XSLTElementParam && inParams {
				// XTSE0020: static is only valid on global params, not template/function params
				if _, hasStatic := v.GetAttribute("static"); hasStatic {
					pname := getAttr(v, "name")
					return nil, nil, nil, staticError(errCodeXTSE0020,
						"xsl:param %q: static attribute is not allowed on template/function parameters", pname)
				}
				p, err := c.compileParamDef(v)
				if err != nil {
					return nil, nil, nil, err
				}
				params = append(params, p)
				if child == lastDeclNode {
					pastDecls = true
				}
				continue
			}
			if v.URI() == lexicon.NamespaceXSLT && v.LocalName() == lexicon.XSLTElementParam {
				return nil, nil, nil, staticError(errCodeXTSE0010,
					"xsl:param must appear before other content in xsl:template/xsl:function")
			}
			inParams = false
			sawContent = true
			inst, err := c.compileInstruction(v)
			if err != nil {
				return nil, nil, nil, err
			}
			if inst != nil {
				body = append(body, inst)
			}
		case *helium.Text:
			text := string(v.Content())
			// Strip whitespace-only text in the declaration/param prologue
			// even under xml:space="preserve" (XSLT 3.0 §9.5).
			if !pastDecls && strings.TrimSpace(text) == "" {
				continue
			}
			if !c.shouldStripText(text) {
				inParams = false
				sawContent = true
				inst := &literalTextInst{Value: text}
				if c.expandText && strings.ContainsAny(text, "{}") {
					avt, err := compileAVT(text, c.nsBindings)
					if err != nil {
						return nil, nil, nil, err
					}
					inst.TVT = avt
				}
				body = append(body, inst)
			}
		case *helium.CDATASection:
			inParams = false
			sawContent = true
			text := string(v.Content())
			inst := &literalTextInst{Value: text}
			if c.expandText && strings.ContainsAny(text, "{}") {
				avt, err := compileAVT(text, c.nsBindings)
				if err != nil {
					return nil, nil, nil, err
				}
				inst.TVT = avt
			}
			body = append(body, inst)
		}
	}

	// XTSE0580: check for duplicate parameter names.
	seen := make(map[string]struct{}, len(params))
	for _, p := range params {
		if _, dup := seen[p.Name]; dup {
			return nil, nil, nil, staticError(errCodeXTSE0580, "duplicate parameter name %q", p.Name)
		}
		seen[p.Name] = struct{}{}
	}

	return ctxDecl, body, params, nil
}

// contextItemAllowedAttrs lists the valid attributes for xsl:context-item.
var contextItemAllowedAttrs = map[string]struct{}{
	"as":  {},
	"use": {},
}

// validateContextItem checks compile-time constraints on xsl:context-item.
func (c *compiler) validateContextItem(elem *helium.Element) error {
	// XTSE0090: reject unknown attributes (e.g. select)
	if err := c.validateXSLTAttrs(elem, contextItemAllowedAttrs); err != nil {
		return err
	}

	asAttr := getAttr(elem, "as")
	if asAttr != "" {
		// XTSE0020: occurrence indicators (?, *, +) not allowed
		trimmed := strings.TrimSpace(asAttr)
		if len(trimmed) > 0 {
			last := trimmed[len(trimmed)-1]
			if last == '?' || last == '*' || last == '+' {
				return staticError(errCodeXTSE0020,
					"xsl:context-item/@as must not have an occurrence indicator: %q", asAttr)
			}
		}
	}

	useAttr := strings.TrimSpace(getAttr(elem, "use"))
	if useAttr != "" && useAttr != ctxItemRequired && useAttr != ctxItemOptional && useAttr != ctxItemAbsent {
		return staticError(errCodeXTSE0020,
			"xsl:context-item/@use must be 'required', 'optional', or 'absent': %q", useAttr)
	}

	// XTSE0020: if use="absent", cannot also have as=
	if useAttr == ctxItemAbsent && asAttr != "" {
		return staticError(errCodeXTSE0020,
			"xsl:context-item with use=\"absent\" must not have @as")
	}

	// xsl:context-item must not have child elements or text content
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			return staticError(errCodeXTSE0010, "xsl:context-item must be empty")
		}
		if child.Type() == helium.TextNode {
			if !c.shouldStripText(string(child.Content())) {
				return staticError(errCodeXTSE0010, "xsl:context-item must be empty")
			}
		}
	}

	return nil
}

func (c *compiler) compileParamDef(elem *helium.Element) (*param, error) {
	savedNS := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = savedNS }()

	// Validate attributes on xsl:param
	if err := c.validateXSLTAttrs(elem, paramAllowedAttrs); err != nil {
		return nil, err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}
	if !isValidQName(name) && !isValidEQName(name) {
		return nil, staticError(errCodeXTSE0020, "invalid name %q on xsl:param", name)
	}

	// Validate boolean attribute values (including empty string)
	if reqAttr, hasReq := elem.GetAttribute("required"); hasReq {
		if err := validateBooleanAttr("xsl:param", "required", reqAttr); err != nil {
			return nil, err
		}
	}
	if tunnelAttr, hasTunnel := elem.GetAttribute("tunnel"); hasTunnel {
		if err := validateBooleanAttr("xsl:param", "tunnel", tunnelAttr); err != nil {
			return nil, err
		}
	}

	required := getAttr(elem, "required") == lexicon.ValueYes

	// XTSE0010: A required parameter must not have a select attribute or body content
	if required {
		selectAttr := getAttr(elem, "select")
		if selectAttr != "" {
			return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have a select attribute")
		}
		// Check for non-whitespace body content
		for child := range helium.Children(elem) {
			switch child.Type() {
			case helium.ElementNode:
				return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have content")
			case helium.TextNode, helium.CDATASectionNode:
				if strings.TrimSpace(string(child.Content())) != "" {
					return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have content")
				}
			}
		}
	}

	// XTSE0020: validate static attribute (boolean)
	if staticVal, hasStatic := elem.GetAttribute("static"); hasStatic {
		if err := validateBooleanAttr("xsl:param", "static", staticVal); err != nil {
			return nil, err
		}
	}

	// XTSE0010: A static parameter must not have a sequence constructor (body content).
	// Static params default to empty sequence when no select is provided.
	isStatic := xsdBoolTrue(getAttr(elem, "static"))

	// XTSE0020: static parameter must not have tunnel="yes"
	if isStatic && xsdBoolTrue(getAttr(elem, "tunnel")) {
		return nil, staticError(errCodeXTSE0020,
			"xsl:param %q with static='yes' must not have tunnel='yes'", name)
	}

	if isStatic {
		for child := range helium.Children(elem) {
			switch child.Type() {
			case helium.ElementNode:
				return nil, staticError(errCodeXTSE0010,
					"xsl:param %q with static='yes' must not have content (use select attribute instead)", name)
			case helium.TextNode, helium.CDATASectionNode:
				if strings.TrimSpace(string(child.Content())) != "" {
					return nil, staticError(errCodeXTSE0010,
						"xsl:param %q with static='yes' must not have content (use select attribute instead)", name)
				}
			}
		}
	}

	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:param "+name); err != nil {
		return nil, err
	}

	p := &param{
		Name:       resolveQName(name, c.nsBindings),
		As:         asAttr,
		Required:   required,
		Tunnel:     xsdBoolTrue(getAttr(elem, "tunnel")),
		Visibility: getAttr(elem, "visibility"),
	}

	// Capture xml:base for static base URI override during body evaluation.
	if effectiveBase := stylesheetBaseURI(elem, c.baseURI); effectiveBase != c.baseURI {
		p.StaticBaseURI = effectiveBase
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		p.Select = expr
	}

	if selectAttr == "" {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		p.Body = body
	}

	return p, nil
}

func (c *compiler) compileGlobalVariable(elem *helium.Element) error {
	savedNS := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = savedNS }()

	// Handle expand-text inheritance for this element.
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		}
	}
	defer func() { c.expandText = savedExpandText }()

	// Validate attributes on xsl:variable
	if err := c.validateXSLTAttrs(elem, variableAllowedAttrs); err != nil {
		return err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}
	if !isValidQName(name) && !isValidEQName(name) {
		return staticError(errCodeXTSE0020, "invalid name %q on xsl:variable", name)
	}

	// XTSE0020: validate static attribute (boolean)
	if staticVal, hasStatic := elem.GetAttribute("static"); hasStatic {
		if err := validateBooleanAttr("xsl:variable", "static", staticVal); err != nil {
			return err
		}
	}

	// XTSE0020: static variable must not have visibility attribute
	if xsdBoolTrue(getAttr(elem, "static")) {
		if vis := getAttr(elem, "visibility"); vis != "" {
			return staticError(errCodeXTSE0020,
				"xsl:variable %q with static='yes' must not have visibility attribute", name)
		}
	}

	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:variable "+name); err != nil {
		return err
	}

	v := &variable{
		Name:           resolveQName(name, c.nsBindings),
		As:             asAttr,
		Visibility:     getAttr(elem, "visibility"),
		XPathDefaultNS: c.xpathDefaultNS,
	}
	if c.stylesheet.isPackage {
		v.OwnerPackage = c.stylesheet
	}

	// Capture xml:base for static base URI override during body evaluation.
	if effectiveBase := stylesheetBaseURI(elem, c.baseURI); effectiveBase != c.baseURI {
		v.StaticBaseURI = effectiveBase
	}

	// For static variables, use the pre-computed value from compile-time
	// evaluation instead of the runtime select expression. This ensures
	// static variables reference other static variables' values, not
	// runtime non-static variables with the same name.
	if xsdBoolTrue(getAttr(elem, "static")) {
		if sv, ok := c.staticVars[name]; ok {
			v.StaticValue = sv
		}
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0620: select and non-empty content are mutually exclusive.
		// Use-when-aware check: elements excluded by use-when don't count as content.
		if c.hasEffectiveContent(elem) {
			return staticError(errCodeXTSE0620, "xsl:variable %q has both @select and content", name)
		}
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return err
		}
		// XPST0008: a global variable is out of scope within its own
		// declaration. Detect self-references in the select expression.
		if xpathExprReferencesVar(expr.AST(), name, c.nsBindings) {
			return staticError(errCodeXPST0008,
				"global variable %q references itself in its select expression", name)
		}
		v.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return err
		}
		v.Body = body
	}

	// XTSE0630: duplicate global variable with same name and same import precedence.
	// Only flag duplicates within the same module (same importPrec).
	for _, existing := range c.stylesheet.globalVars {
		if existing.Name == v.Name && existing.ImportPrec == c.importPrec {
			return staticError(errCodeXTSE0630, "duplicate global variable %q", v.Name)
		}
	}
	for _, existing := range c.stylesheet.globalParams {
		if existing.Name == v.Name && existing.ImportPrec == c.importPrec {
			return staticError(errCodeXTSE0630, "duplicate global variable/param %q", v.Name)
		}
	}
	v.ImportPrec = c.importPrec
	c.stylesheet.globalVars = append(c.stylesheet.globalVars, v)
	return nil
}

func (c *compiler) compileGlobalParam(elem *helium.Element) error {
	// XTSE0020: tunnel="yes" is not allowed on a stylesheet parameter
	if getAttr(elem, "tunnel") == lexicon.ValueYes {
		return staticError(errCodeXTSE0020, "tunnel=\"yes\" is not allowed on a stylesheet parameter")
	}
	p, err := c.compileParamDef(elem)
	if err != nil {
		return err
	}
	// XTSE0630: duplicate global param/variable with same name and same import precedence
	for _, existing := range c.stylesheet.globalParams {
		if existing.Name == p.Name && existing.ImportPrec == c.importPrec {
			return staticError(errCodeXTSE0630, "duplicate global param %q", p.Name)
		}
	}
	for _, existing := range c.stylesheet.globalVars {
		if existing.Name == p.Name && existing.ImportPrec == c.importPrec {
			return staticError(errCodeXTSE0630, "duplicate global param/variable %q", p.Name)
		}
	}
	p.ImportPrec = c.importPrec
	c.stylesheet.globalParams = append(c.stylesheet.globalParams, p)
	return nil
}

// xpathExprReferencesVar walks an XPath AST and returns true if any
// VariableExpr references the given variable name. The name parameter
// is the raw XSLT name attribute (may be prefixed like "ns:local");
// nsBindings are used to resolve prefixes for matching.
func xpathExprReferencesVar(expr xpath3.Expr, name string, nsBindings map[string]string) bool {
	if expr == nil {
		return false
	}

	// Determine the local name and optional resolved name for matching.
	// In XPath AST, VariableExpr stores Prefix and Name separately.
	rawLocal := name
	var rawPrefix string
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		rawPrefix = name[:idx]
		rawLocal = name[idx+1:]
	}

	return xpathExprRefsVarWalk(expr, rawLocal, rawPrefix)
}

func xpathExprRefsVarWalk(e xpath3.Expr, local, prefix string) bool {
	switch n := e.(type) {
	case xpath3.VariableExpr:
		return n.Name == local && n.Prefix == prefix
	case xpath3.BinaryExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.UnaryExpr:
		return xpathExprRefsVarWalk(n.Operand, local, prefix)
	case xpath3.ConcatExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.SimpleMapExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.RangeExpr:
		return xpathExprRefsVarWalk(n.Start, local, prefix) ||
			xpathExprRefsVarWalk(n.End, local, prefix)
	case xpath3.UnionExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.IntersectExceptExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.FilterExpr:
		if xpathExprRefsVarWalk(n.Expr, local, prefix) {
			return true
		}
		for _, pred := range n.Predicates {
			if xpathExprRefsVarWalk(pred, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.PathExpr:
		if n.Filter != nil && xpathExprRefsVarWalk(n.Filter, local, prefix) {
			return true
		}
		if n.Path != nil {
			for _, step := range n.Path.Steps {
				for _, pred := range step.Predicates {
					if xpathExprRefsVarWalk(pred, local, prefix) {
						return true
					}
				}
			}
		}
		return false
	case xpath3.PathStepExpr:
		return xpathExprRefsVarWalk(n.Left, local, prefix) ||
			xpathExprRefsVarWalk(n.Right, local, prefix)
	case xpath3.LookupExpr:
		if xpathExprRefsVarWalk(n.Expr, local, prefix) {
			return true
		}
		return n.Key != nil && xpathExprRefsVarWalk(n.Key, local, prefix)
	case xpath3.UnaryLookupExpr:
		return n.Key != nil && xpathExprRefsVarWalk(n.Key, local, prefix)
	case xpath3.FunctionCall:
		for _, arg := range n.Args {
			if xpathExprRefsVarWalk(arg, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.DynamicFunctionCall:
		if xpathExprRefsVarWalk(n.Func, local, prefix) {
			return true
		}
		for _, arg := range n.Args {
			if xpathExprRefsVarWalk(arg, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.InlineFunctionExpr:
		// Check if the inline function's parameters shadow the variable.
		for _, p := range n.Params {
			if p.Name == local {
				return false // shadowed
			}
		}
		return xpathExprRefsVarWalk(n.Body, local, prefix)
	case xpath3.FLWORExpr:
		for _, cl := range n.Clauses {
			switch c := cl.(type) {
			case xpath3.ForClause:
				if xpathExprRefsVarWalk(c.Expr, local, prefix) {
					return true
				}
				if c.Var == local {
					return false // shadowed
				}
			case xpath3.LetClause:
				if xpathExprRefsVarWalk(c.Expr, local, prefix) {
					return true
				}
				if c.Var == local {
					return false // shadowed
				}
			}
		}
		return xpathExprRefsVarWalk(n.Return, local, prefix)
	case xpath3.QuantifiedExpr:
		for _, b := range n.Bindings {
			if xpathExprRefsVarWalk(b.Domain, local, prefix) {
				return true
			}
			if b.Var == local {
				return false // shadowed
			}
		}
		return xpathExprRefsVarWalk(n.Satisfies, local, prefix)
	case xpath3.IfExpr:
		return xpathExprRefsVarWalk(n.Cond, local, prefix) ||
			xpathExprRefsVarWalk(n.Then, local, prefix) ||
			xpathExprRefsVarWalk(n.Else, local, prefix)
	case xpath3.TryCatchExpr:
		if xpathExprRefsVarWalk(n.Try, local, prefix) {
			return true
		}
		for _, c := range n.Catches {
			if xpathExprRefsVarWalk(c.Expr, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.InstanceOfExpr:
		return xpathExprRefsVarWalk(n.Expr, local, prefix)
	case xpath3.CastExpr:
		return xpathExprRefsVarWalk(n.Expr, local, prefix)
	case xpath3.CastableExpr:
		return xpathExprRefsVarWalk(n.Expr, local, prefix)
	case xpath3.TreatAsExpr:
		return xpathExprRefsVarWalk(n.Expr, local, prefix)
	case xpath3.MapConstructorExpr:
		for _, pair := range n.Pairs {
			if xpathExprRefsVarWalk(pair.Key, local, prefix) ||
				xpathExprRefsVarWalk(pair.Value, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.ArrayConstructorExpr:
		for _, item := range n.Items {
			if xpathExprRefsVarWalk(item, local, prefix) {
				return true
			}
		}
		return false
	case xpath3.SequenceExpr:
		for _, item := range n.Items {
			if xpathExprRefsVarWalk(item, local, prefix) {
				return true
			}
		}
		return false
	default:
		// LiteralExpr, RootExpr, ContextItemExpr, LocationPath,
		// NamedFunctionRef, PlaceholderExpr — no sub-expressions
		return false
	}
}
