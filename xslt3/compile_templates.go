package xslt3

import (
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
)

func (c *compiler) compileTemplate(elem *helium.Element) error {
	tmpl := &Template{
		ImportPrec:    c.importPrec,
		MinImportPrec: c.minImportPrec,
		BaseURI:       c.baseURI,
	}

	// Collect namespace declarations from this template
	c.collectNamespaces(elem)

	// Inherit or override xpath-default-namespace
	savedXPathDefaultNS := c.xpathDefaultNS
	if xdn := getAttr(elem, "xpath-default-namespace"); xdn != "" {
		c.xpathDefaultNS = xdn
	}
	tmpl.XPathDefaultNS = c.xpathDefaultNS
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	matchAttr := getAttr(elem, "match")
	if matchAttr != "" {
		p, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return err
		}
		// Validate function calls in the pattern against known functions.
		if err := c.validatePatternFunctions(p, matchAttr); err != nil {
			return err
		}
		tmpl.Match = p
	}

	tmpl.Name = resolveQName(getAttr(elem, "name"), c.nsBindings)

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
	if modeAttr != "" {
		// Resolve mode QNames to Clark notation for namespace-aware matching
		tmpl.Mode = c.resolveMode(modeAttr)
	}
	// XSLT 3.0 §6.7: if the stylesheet (or an included/imported module) has
	// default-mode, templates without an explicit mode attribute belong to it.
	if tmpl.Mode == "" && c.defaultMode != "" {
		tmpl.Mode = c.resolveMode(c.defaultMode)
	}

	hasExplicitPriority := false
	if prio := getAttr(elem, "priority"); prio != "" {
		f, err := strconv.ParseFloat(prio, 64)
		if err != nil {
			return staticError(errCodeXTSE0010, "invalid priority %q: %v", prio, err)
		}
		tmpl.Priority = f
		hasExplicitPriority = true
	} else if tmpl.Match != nil && len(tmpl.Match.Alternatives) == 1 {
		tmpl.Priority = tmpl.Match.Alternatives[0].priority
	}

	// Handle exclude-result-prefixes on xsl:template
	savedExcludes := c.localExcludes
	if erp := getAttr(elem, "exclude-result-prefixes"); erp != "" {
		newExcludes := make(map[string]struct{})
		for k, v := range c.localExcludes {
			newExcludes[k] = v
		}
		if erp == "#all" {
			for prefix := range c.stylesheet.namespaces {
				newExcludes[prefix] = struct{}{}
			}
		} else {
			for _, prefix := range strings.Fields(erp) {
				newExcludes[prefix] = struct{}{}
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

	// Handle version on xsl:template for forwards-compatible processing
	savedVersion := c.effectiveVersion
	if ver := getAttr(elem, "version"); ver != "" {
		c.effectiveVersion = ver
	}

	// Compile template body: first xsl:param elements, then instructions
	body, params, err := c.compileTemplateBody(elem)
	c.effectiveVersion = savedVersion
	c.expandText = savedExpandText
	c.localExcludes = savedExcludes
	if err != nil {
		return err
	}
	tmpl.Params = params
	tmpl.Body = body
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
		templates := []*Template{tmpl}
		if !hasExplicitPriority && len(tmpl.Match.Alternatives) > 1 {
			templates = nil
			for _, alt := range tmpl.Match.Alternatives {
				split := *tmpl // shallow copy shares Body, Params, etc.
				split.Match = &Pattern{
					source:         tmpl.Match.source,
					Alternatives:   []*PatternAlt{alt},
					xpathDefaultNS: tmpl.Match.xpathDefaultNS,
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
func (c *compiler) registerTemplateInModes(tmpl *Template, mode string) {
	if mode == "#all" {
		// Register in all existing modes plus default
		for m := range c.stylesheet.modeTemplates {
			c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
		}
		c.stylesheet.modeTemplates[""] = append(c.stylesheet.modeTemplates[""], tmpl)
		// Also store under the "#all" key so findBestTemplate's fallback
		// can find these templates for modes that don't exist yet.
		c.stylesheet.modeTemplates["#all"] = append(c.stylesheet.modeTemplates["#all"], tmpl)
		return
	}
	// XSLT 2.0+: mode can be a whitespace-separated list of mode names.
	// Each mode name can be a QName, "#default", "#unnamed", or "#all".
	modes := strings.Fields(mode)
	if len(modes) <= 1 {
		// Single mode (or empty = default mode)
		if mode == "#default" || mode == "#unnamed" {
			mode = ""
		}
		c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
	} else {
		for _, m := range modes {
			if m == "#default" || m == "#unnamed" {
				m = ""
			} else if m == "#all" {
				// In a mode list, #all means register in all modes
				c.stylesheet.modeTemplates["#all"] = append(c.stylesheet.modeTemplates["#all"], tmpl)
				continue
			}
			c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
		}
	}
}

func (c *compiler) compileTemplateBody(elem *helium.Element) ([]Instruction, []*Param, error) {
	var params []*Param
	var body []Instruction

	inParams := true
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "param" && inParams {
				p, err := c.compileParamDef(v)
				if err != nil {
					return nil, nil, err
				}
				params = append(params, p)
				continue
			}
			inParams = false
			inst, err := c.compileInstruction(v)
			if err != nil {
				return nil, nil, err
			}
			if inst != nil {
				body = append(body, inst)
			}
		case *helium.Text:
			text := string(v.Content())
			if !c.shouldStripText(text) {
				inParams = false
				inst := &LiteralTextInst{Value: text}
				if c.expandText && strings.ContainsAny(text, "{}") {
					avt, err := compileAVT(text, c.nsBindings)
					if err != nil {
						return nil, nil, err
					}
					inst.TVT = avt
				}
				body = append(body, inst)
			}
		case *helium.CDATASection:
			inParams = false
			text := string(v.Content())
			inst := &LiteralTextInst{Value: text}
			if c.expandText && strings.ContainsAny(text, "{}") {
				avt, err := compileAVT(text, c.nsBindings)
				if err != nil {
					return nil, nil, err
				}
				inst.TVT = avt
			}
			body = append(body, inst)
		}
	}

	return body, params, nil
}

func (c *compiler) compileParamDef(elem *helium.Element) (*Param, error) {
	savedNS := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = savedNS }()

	// Validate attributes on xsl:param
	if err := validateXSLTAttrs(elem, paramAllowedAttrs); err != nil {
		return nil, err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
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

	required := getAttr(elem, "required") == "yes"

	// XTSE0010: A required parameter must not have a select attribute or body content
	if required {
		selectAttr := getAttr(elem, "select")
		if selectAttr != "" {
			return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have a select attribute")
		}
		// Check for non-whitespace body content
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
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

	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:param "+name); err != nil {
		return nil, err
	}

	p := &Param{
		Name:       resolveQName(name, c.nsBindings),
		As:         asAttr,
		Required:   required,
		Tunnel:     xsdBoolTrue(getAttr(elem, "tunnel")),
		Visibility: getAttr(elem, "visibility"),
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

	// Validate attributes on xsl:variable
	if err := validateXSLTAttrs(elem, variableAllowedAttrs); err != nil {
		return err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:variable "+name); err != nil {
		return err
	}

	v := &Variable{
		Name:       resolveQName(name, c.nsBindings),
		As:         asAttr,
		Visibility: getAttr(elem, "visibility"),
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return err
		}
		v.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return err
		}
		v.Body = body
	}

	c.stylesheet.globalVars = append(c.stylesheet.globalVars, v)
	return nil
}

func (c *compiler) compileGlobalParam(elem *helium.Element) error {
	p, err := c.compileParamDef(elem)
	if err != nil {
		return err
	}
	c.stylesheet.globalParams = append(c.stylesheet.globalParams, p)
	return nil
}
