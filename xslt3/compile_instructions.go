package xslt3

import (
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
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
		"use-attribute-sets":        {},
		"expand-text":               {},
		"xpath-default-namespace":   {},
		"exclude-result-prefixes":   {},
		"extension-element-prefixes": {},
		"version":                   {},
		"type":                      {},
		"validation":                {},
		"default-collation":         {},
		"default-mode":              {},
		"default-validation":        {},
		"inherit-namespaces":        {},
		"use-when":                  {},
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

// validateXSLTAttrs checks that an XSLT element has only allowed unprefixed attributes
// and no attributes in the XSLT namespace. Returns XTSE0090 for unknown attributes.
func validateXSLTAttrs(elem *helium.Element, allowed map[string]struct{}) error {
	for _, attr := range elem.Attributes() {
		// Attributes in the XSLT namespace are not allowed on XSLT elements
		if attr.URI() == NSXSLT {
			return staticError(errCodeXTSE0090,
				"attribute %q in the XSLT namespace is not allowed on xsl:%s", attr.LocalName(), elem.LocalName())
		}
		// Skip attributes in other (non-null) namespaces — those are extension attributes
		if attr.URI() != "" {
			continue
		}
		if _, ok := allowed[attr.LocalName()]; !ok {
			return staticError(errCodeXTSE0090,
				"attribute %q is not allowed on xsl:%s", attr.LocalName(), elem.LocalName())
		}
	}
	return nil
}

// compileInstruction compiles a single element into an Instruction.
func (c *compiler) compileInstruction(elem *helium.Element) (Instruction, error) {
	// Push element-local namespace declarations into scope
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	// Handle xml:space inheritance
	savedPreserve := c.preserveSpace
	if xs := getAttr(elem, "xml:space"); xs != "" {
		c.preserveSpace = (xs == "preserve")
	}
	defer func() { c.preserveSpace = savedPreserve }()

	// Handle expand-text inheritance (check both unprefixed and xsl:-prefixed for LREs)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else if elem.URI() == NSXSLT {
			// XTSE0020: invalid boolean value for expand-text on XSLT element
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:%s/@expand-text", et, elem.LocalName())
		}
	} else if et, ok := elem.GetAttributeNS("expand-text", NSXSLT); ok {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			// XTSE0020: invalid boolean value for xsl:expand-text on LRE
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:expand-text", et)
		}
	}
	defer func() { c.expandText = savedExpandText }()

	// Handle per-instruction xpath-default-namespace
	// Check both unprefixed (on XSLT elements) and xsl:-prefixed (on LREs)
	savedXPathDefaultNS := c.xpathDefaultNS
	hasLocalXPNS := false
	if xdn, ok := elem.GetAttribute("xpath-default-namespace"); ok {
		c.xpathDefaultNS = xdn
		hasLocalXPNS = true
	} else if xdn, ok := elem.GetAttributeNS("xpath-default-namespace", NSXSLT); ok {
		c.xpathDefaultNS = xdn
		hasLocalXPNS = true
	}
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	// Handle per-instruction default-mode (XSLT 3.0 standard attribute)
	// On XSLT elements: default-mode="..."
	// On LREs: xsl:default-mode="..."
	// Mode names are stored as raw QNames (not Clark notation) for consistency
	// with how template mode attributes are stored.
	savedDefaultMode := c.defaultMode
	if dm := getAttr(elem, "default-mode"); dm != "" {
		c.defaultMode = dm
	} else if dm, ok := elem.GetAttributeNS("default-mode", NSXSLT); ok {
		c.defaultMode = dm
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	if elem.URI() == NSXSLT {
		inst, err := c.compileXSLTInstruction(elem)
		if err != nil {
			return nil, err
		}
		// Store effective xpath-default-namespace on instructions that support it
		c.setInstructionXPathNS(inst, hasLocalXPNS)
		return inst, nil
	}
	return c.compileLiteralResultElement(elem)
}

// setInstructionXPathNS stores the current xpath-default-namespace on
// instructions that embed xpathNS.
func (c *compiler) setInstructionXPathNS(inst Instruction, hasLocal bool) {
	set := func(ns *xpathNS) {
		ns.XPathDefaultNS = c.xpathDefaultNS
		// Mark as set when either explicitly declared locally or inherited non-empty
		ns.HasXPathDefaultNS = hasLocal || c.xpathDefaultNS != ""
	}
	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		set(&v.xpathNS)
	case *IfInst:
		set(&v.xpathNS)
	case *ValueOfInst:
		set(&v.xpathNS)
	case *ForEachInst:
		set(&v.xpathNS)
	case *ChooseInst:
		set(&v.xpathNS)
	}
}

// pushElementNamespaces adds namespace declarations from elem to nsBindings
// and to the stylesheet's runtime namespace map. Returns previous nsBindings
// for restoring.
func (c *compiler) pushElementNamespaces(elem *helium.Element) map[string]string {
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
		if uri != NSXSLT {
			newBindings[prefix] = uri
			// Also add to stylesheet namespaces for runtime XPath evaluation
			c.stylesheet.namespaces[prefix] = uri
		}
	}
	c.nsBindings = newBindings
	return saved
}

// compileXSLTInstruction compiles an XSLT-namespaced instruction element.
func (c *compiler) compileXSLTInstruction(elem *helium.Element) (Instruction, error) {
	switch elem.LocalName() {
	case "apply-templates":
		return c.compileApplyTemplates(elem)
	case "call-template":
		return c.compileCallTemplate(elem)
	case "value-of":
		return c.compileValueOf(elem)
	case "text":
		return c.compileText(elem)
	case "element":
		return c.compileElement(elem)
	case "attribute":
		return c.compileAttribute(elem)
	case "comment":
		return c.compileComment(elem)
	case "processing-instruction":
		return c.compilePI(elem)
	case "if":
		return c.compileIf(elem)
	case "choose":
		return c.compileChoose(elem)
	case "for-each":
		return c.compileForEach(elem)
	case "variable":
		return c.compileLocalVariable(elem)
	case "param":
		return c.compileLocalParam(elem)
	case "copy":
		return c.compileCopy(elem)
	case "copy-of":
		return c.compileCopyOf(elem)
	case "number":
		return c.compileNumber(elem)
	case "message":
		return c.compileMessage(elem)
	case "namespace":
		return c.compileNamespace(elem)
	case "sequence":
		return c.compileSequence(elem)
	case "perform-sort":
		return c.compilePerformSort(elem)
	case "next-match":
		return c.compileNextMatch(elem)
	case "apply-imports":
		return c.compileApplyImports(elem)
	case "document":
		// xsl:document is deprecated in XSLT 3.0, treat like xsl:sequence
		return c.compileSequence(elem)
	case "result-document":
		inst := &ResultDocumentInst{}
		if href := getAttr(elem, "href"); href != "" {
			avt, err := compileAVT(href, c.nsBindings)
			if err != nil {
				return nil, err
			}
			inst.Href = avt
		}
		// Validate html-version: must be a decimal number if present and not an AVT
		if hv := getAttr(elem, "html-version"); hv != "" {
			if !strings.ContainsAny(hv, "{}") {
				if _, err := strconv.ParseFloat(hv, 64); err != nil {
					return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:result-document/@html-version", hv)
				}
			}
		}
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
		return inst, nil
	case "where-populated":
		// xsl:where-populated: execute body and only include if non-empty
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		return &WherePopulatedInst{Body: body}, nil
	case "on-empty":
		inst := &OnEmptyInst{}
		if sel := getAttr(elem, "select"); sel != "" {
			expr, err := xpath3.Compile(sel)
			if err != nil {
				return nil, err
			}
			inst.Select = expr
		} else {
			body, err := c.compileChildren(elem)
			if err != nil {
				return nil, err
			}
			inst.Body = body
		}
		return inst, nil
	case "try":
		return c.compileTry(elem)
	case "for-each-group":
		return c.compileForEachGroup(elem)
	case "map":
		return c.compileMap(elem)
	case "map-entry":
		return c.compileMapEntry(elem)
	case "sort":
		// xsl:sort is handled by parent instructions
		return nil, nil
	case "fallback":
		// xsl:fallback is only activated when the parent is unrecognized;
		// when we reach here the parent was recognized, so skip.
		return nil, nil
	case "analyze-string":
		return c.compileAnalyzeString(elem)
	case "source-document":
		return c.compileSourceDocument(elem)
	case "iterate":
		return c.compileIterate(elem)
	case "fork":
		return c.compileFork(elem)
	case "break":
		return c.compileBreak(elem)
	case "next-iteration":
		return c.compileNextIteration(elem)
	case "merge":
		return c.compileMerge(elem)
	case "merge-source":
		// xsl:merge-source must be a direct child of xsl:merge
		return nil, staticError(errCodeXTSE0010, "xsl:merge-source must be a direct child of xsl:merge")
	case "merge-action":
		// xsl:merge-action must be a direct child of xsl:merge
		return nil, staticError(errCodeXTSE0010, "xsl:merge-action must be a direct child of xsl:merge")
	case "merge-key":
		// xsl:merge-key must be a direct child of xsl:merge-source
		return nil, staticError(errCodeXTSE0010, "xsl:merge-key must be a direct child of xsl:merge-source")
	case "on-completion":
		// xsl:on-completion must be a direct child of xsl:iterate — if we reach
		// here, it was encountered outside that context.
		return nil, staticError(errCodeXTSE0010, "xsl:on-completion must be a direct child of xsl:iterate")
	default:
		return nil, staticError(errCodeXTSE0090, "unknown XSLT instruction xsl:%s", elem.LocalName())
	}
}

// compileChildren compiles all children of an element into instructions.
func (c *compiler) compileChildren(parent *helium.Element) ([]Instruction, error) {
	var body []Instruction
	sawTerminator := false // true after xsl:break or xsl:next-iteration
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			inst, err := c.compileInstruction(v)
			if err != nil {
				return nil, err
			}
			if inst != nil {
				// XTSE3120: nothing may follow xsl:break or xsl:next-iteration.
				if sawTerminator {
					return nil, staticError(errCodeXTSE3120, "no instruction may follow xsl:break or xsl:next-iteration")
				}
				body = append(body, inst)
				switch inst.(type) {
				case *BreakInst, *NextIterationInst:
					sawTerminator = true
				}
			}
		case *helium.Text:
			text := string(v.Content())
			if !c.shouldStripText(text) {
				if sawTerminator {
					return nil, staticError(errCodeXTSE3120, "no instruction may follow xsl:break or xsl:next-iteration")
				}
				inst := &LiteralTextInst{Value: text}
				if c.expandText && strings.ContainsAny(text, "{}") {
					avt, err := compileAVT(text, c.nsBindings)
					if err != nil {
						return nil, err
					}
					inst.TVT = avt
				}
				body = append(body, inst)
			}
		case *helium.CDATASection:
			if sawTerminator {
				return nil, staticError(errCodeXTSE3120, "no instruction may follow xsl:break or xsl:next-iteration")
			}
			text := string(v.Content())
			inst := &LiteralTextInst{Value: text}
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
	return body, nil
}

func (c *compiler) compileApplyTemplates(elem *helium.Element) (*ApplyTemplatesInst, error) {
	mode := getAttr(elem, "mode")
	// When mode is absent, use the current default-mode (set by
	// compileInstruction from ancestor or self default-mode attributes).
	if mode == "" && c.defaultMode != "" {
		mode = c.defaultMode
	}
	// Resolve mode QName to Clark notation (namespace-aware)
	mode = c.resolveMode(mode)
	inst := &ApplyTemplatesInst{
		Mode: mode,
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	// Process children: xsl:sort and xsl:with-param
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}
		switch childElem.LocalName() {
		case "sort":
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			inst.Sort = append(inst.Sort, sk)
		case "with-param":
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			inst.Params = append(inst.Params, wp)
		}
	}

	return inst, nil
}

func (c *compiler) compileCallTemplate(elem *helium.Element) (*CallTemplateInst, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:call-template requires name attribute")
	}

	inst := &CallTemplateInst{Name: resolveQName(name, c.nsBindings)}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "with-param" {
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			inst.Params = append(inst.Params, wp)
		}
	}

	return inst, nil
}

func (c *compiler) compileValueOf(elem *helium.Element) (*ValueOfInst, error) {
	inst := &ValueOfInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if sep, ok := elem.GetAttribute("separator"); ok {
		avt, err := compileAVT(sep, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Separator = avt
		inst.HasSeparator = true
	}

	// XTSE0350: xsl:value-of with select must not have non-whitespace content
	if selectAttr != "" {
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.Type() {
			case helium.TextNode, helium.CDATASectionNode:
				if strings.TrimSpace(string(child.Content())) != "" {
					return nil, staticError(errCodeXTSE0350, "xsl:value-of with select attribute must not have content")
				}
			case helium.ElementNode:
				return nil, staticError(errCodeXTSE0350, "xsl:value-of with select attribute must not have content")
			}
		}
	} else {
		// Sequence constructor body (XSLT 2.0+)
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileText(elem *helium.Element) (*TextInst, error) {
	var sb strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.TextNode || child.Type() == helium.CDATASectionNode {
			sb.Write(child.Content())
		}
	}

	text := sb.String()
	inst := &TextInst{
		Value:                 text,
		DisableOutputEscaping: getAttr(elem, "disable-output-escaping") == "yes",
	}

	if c.expandText && strings.ContainsAny(text, "{}") {
		avt, err := compileAVT(text, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.TVT = avt
	}

	return inst, nil
}

func (c *compiler) compileElement(elem *helium.Element) (*ElementInst, error) {
	// xsl:break / xsl:next-iteration not allowed inside element constructors.
	savedBreak := c.breakAllowed
	c.breakAllowed = false
	defer func() { c.breakAllowed = savedBreak }()

	// Validate boolean attributes (including empty string)
	if inAttr, hasIn := elem.GetAttribute("inherit-namespaces"); hasIn {
		if err := validateBooleanAttr("xsl:element", "inherit-namespaces", inAttr); err != nil {
			return nil, err
		}
	}

	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:element requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ElementInst{Name: nameAVT}

	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}

	nsAttr := getAttr(elem, "namespace")
	if nsAttr != "" {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, name := range strings.Fields(uas) {
			inst.UseAttributeSets = append(inst.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		inst.UseAttrSets = strings.Fields(uas)
	}

	return inst, nil
}

func (c *compiler) compileAttribute(elem *helium.Element) (*AttributeInst, error) {
	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:attribute requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &AttributeInst{Name: nameAVT}

	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}

	nsAttr := getAttr(elem, "namespace")
	if nsAttr != "" {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	if sep := getAttr(elem, "separator"); sep != "" {
		sepAVT, err := compileAVT(sep, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Separator = sepAVT
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileComment(elem *helium.Element) (*CommentInst, error) {
	inst := &CommentInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compilePI(elem *helium.Element) (*PIInst, error) {
	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:processing-instruction requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &PIInst{Name: nameAVT}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileIf(elem *helium.Element) (*IfInst, error) {
	testAttr := getAttr(elem, "test")
	if testAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:if requires test attribute")
	}

	expr, err := compileXPath(testAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}

	return &IfInst{Test: expr, Body: body}, nil
}

func (c *compiler) compileChoose(elem *helium.Element) (*ChooseInst, error) {
	inst := &ChooseInst{}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}

		switch childElem.LocalName() {
		case "when":
			savedNS := c.xpathDefaultNS
			savedET := c.expandText
			hasLocal := false
			if xdn, ok := childElem.GetAttribute("xpath-default-namespace"); ok {
				c.xpathDefaultNS = xdn
				hasLocal = true
			}
			if et := getAttr(childElem, "expand-text"); et != "" {
				if v, ok := parseXSDBool(et); ok {
					c.expandText = v
				}
			}
			testAttr := getAttr(childElem, "test")
			if testAttr == "" {
				c.xpathDefaultNS = savedNS
				c.expandText = savedET
				return nil, staticError(errCodeXTSE0110, "xsl:when requires test attribute")
			}
			expr, err := compileXPath(testAttr, c.nsBindings)
			if err != nil {
				c.xpathDefaultNS = savedNS
				c.expandText = savedET
				return nil, err
			}
			body, err := c.compileChildren(childElem)
			wc := &WhenClause{Test: expr, Body: body}
			wc.XPathDefaultNS = c.xpathDefaultNS
			wc.HasXPathDefaultNS = hasLocal
			c.xpathDefaultNS = savedNS
			c.expandText = savedET
			if err != nil {
				return nil, err
			}
			inst.When = append(inst.When, wc)
		case "otherwise":
			savedNS := c.xpathDefaultNS
			savedET := c.expandText
			hasLocal := false
			if xdn, ok := childElem.GetAttribute("xpath-default-namespace"); ok {
				c.xpathDefaultNS = xdn
				hasLocal = true
			}
			if et := getAttr(childElem, "expand-text"); et != "" {
				if v, ok := parseXSDBool(et); ok {
					c.expandText = v
				}
			}
			body, err := c.compileChildren(childElem)
			if hasLocal {
				inst.OtherwiseXPNS = c.xpathDefaultNS
				inst.HasOtherwiseXPNS = true
			}
			c.xpathDefaultNS = savedNS
			c.expandText = savedET
			if err != nil {
				return nil, err
			}
			inst.Otherwise = body
		}
	}

	return inst, nil
}

func (c *compiler) compileForEach(elem *helium.Element) (*ForEachInst, error) {
	// xsl:break / xsl:next-iteration not allowed inside xsl:for-each.
	savedBreak := c.breakAllowed
	c.breakAllowed = false
	defer func() { c.breakAllowed = savedBreak }()

	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:for-each requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ForEachInst{Select: expr}

	// First pass: collect sort keys
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "sort" {
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			inst.Sort = append(inst.Sort, sk)
		}
	}

	// Second pass: compile body (skip sort elements)
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "sort" {
				continue
			}
			childInst, err := c.compileInstruction(v)
			if err != nil {
				return nil, err
			}
			if childInst != nil {
				inst.Body = append(inst.Body, childInst)
			}
		case *helium.Text:
			text := string(v.Content())
			if !c.shouldStripText(text) {
				inst.Body = append(inst.Body, &LiteralTextInst{Value: text})
			}
		case *helium.CDATASection:
			inst.Body = append(inst.Body, &LiteralTextInst{Value: string(v.Content())})
		}
	}

	return inst, nil
}

func (c *compiler) compileLocalVariable(elem *helium.Element) (*VariableInst, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	inst := &VariableInst{
		Name: resolveQName(name, c.nsBindings),
		As:   getAttr(elem, "as"),
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileLocalParam(elem *helium.Element) (*ParamInst, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	inst := &ParamInst{
		Name:     resolveQName(name, c.nsBindings),
		Required: getAttr(elem, "required") == "yes",
		As:       getAttr(elem, "as"),
	}

	if tunnelAttr := getAttr(elem, "tunnel"); tunnelAttr == "yes" {
		inst.Tunnel = true
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileCopy(elem *helium.Element) (*CopyInst, error) {
	// Validate boolean attributes (including empty string)
	if inAttr, hasIn := elem.GetAttribute("inherit-namespaces"); hasIn {
		if err := validateBooleanAttr("xsl:copy", "inherit-namespaces", inAttr); err != nil {
			return nil, err
		}
	}
	if cn, hasCN := elem.GetAttribute("copy-namespaces"); hasCN {
		if err := validateBooleanAttr("xsl:copy", "copy-namespaces", cn); err != nil {
			return nil, err
		}
	}

	inst := &CopyInst{
		CopyNamespaces:    true,
		InheritNamespaces: true,
	}

	if cn := getAttr(elem, "copy-namespaces"); cn == "no" {
		inst.CopyNamespaces = false
	}
	if in := getAttr(elem, "inherit-namespaces"); in == "no" {
		inst.InheritNamespaces = false
	}

	if v := getAttr(elem, "validation"); v != "" {
		inst.Validation = v
	}

	if selectAttr := getAttr(elem, "select"); selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, name := range strings.Fields(uas) {
			inst.UseAttributeSets = append(inst.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		inst.UseAttrSets = strings.Fields(uas)
	}

	return inst, nil
}

func (c *compiler) compileCopyOf(elem *helium.Element) (*CopyOfInst, error) {
	// XTSE0260: xsl:copy-of must have no significant content
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			return nil, staticError(errCodeXTSE0260,
				"xsl:copy-of must be empty, but contains child element %s", child.Name())
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) != "" {
				return nil, staticError(errCodeXTSE0260,
					"xsl:copy-of must be empty, but contains text content")
			}
		}
	}

	// XTSE0090: reject specific attributes that are not allowed on xsl:copy-of
	for _, attr := range elem.Attributes() {
		if attr.URI() != "" {
			continue
		}
		if attr.LocalName() == "match" || attr.LocalName() == "count" || attr.LocalName() == "from" {
			return nil, staticError(errCodeXTSE0090,
				"attribute %q is not allowed on xsl:copy-of", attr.LocalName())
		}
	}

	// Validate boolean attribute: copy-namespaces
	if cn, hasCN := elem.GetAttribute("copy-namespaces"); hasCN {
		if err := validateBooleanAttr("xsl:copy-of", "copy-namespaces", cn); err != nil {
			return nil, err
		}
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:copy-of requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &CopyOfInst{
		Select:         expr,
		CopyNamespaces: true,
	}
	if cn := getAttr(elem, "copy-namespaces"); cn == "no" {
		inst.CopyNamespaces = false
	}
	if v := getAttr(elem, "validation"); v != "" {
		inst.Validation = v
	}
	return inst, nil
}

func (c *compiler) compileNumber(elem *helium.Element) (*NumberInst, error) {
	inst := &NumberInst{
		Level: getAttr(elem, "level"),
	}
	if inst.Level == "" {
		inst.Level = "single"
	}

	if countAttr := getAttr(elem, "count"); countAttr != "" {
		p, err := compilePattern(countAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return nil, err
		}
		inst.Count = p
	}

	if fromAttr := getAttr(elem, "from"); fromAttr != "" {
		p, err := compilePattern(fromAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return nil, err
		}
		inst.From = p
	}

	if valueAttr := getAttr(elem, "value"); valueAttr != "" {
		expr, err := compileXPath(valueAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Value = expr
	}

	if fmtAttr := getAttr(elem, "format"); fmtAttr != "" {
		avt, err := compileAVT(fmtAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Format = avt
	}

	if gs := getAttr(elem, "grouping-separator"); gs != "" {
		avt, err := compileAVT(gs, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.GroupingSeparator = avt
	}

	if gsz := getAttr(elem, "grouping-size"); gsz != "" {
		avt, err := compileAVT(gsz, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.GroupingSize = avt
	}

	if ord := getAttr(elem, "ordinal"); ord != "" {
		avt, err := compileAVT(ord, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Ordinal = avt
	}

	if sa := getAttr(elem, "start-at"); sa != "" {
		avt, err := compileAVT(sa, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.StartAt = avt
	}

	if sel := getAttr(elem, "select"); sel != "" {
		expr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	return inst, nil
}

func (c *compiler) compileMessage(elem *helium.Element) (*MessageInst, error) {
	inst := &MessageInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}
	// XSLT 3.0: both select and body content are allowed
	body, err := c.compileChildren(elem)
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

func (c *compiler) compileNamespace(elem *helium.Element) (*NamespaceInst, error) {
	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:namespace requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &NamespaceInst{Name: nameAVT}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileSortKey(elem *helium.Element) (*SortKey, error) {
	// Validate boolean attribute: stable (including empty string)
	if stableAttr, hasStable := elem.GetAttribute("stable"); hasStable {
		if err := validateBooleanAttr("xsl:sort", "stable", stableAttr); err != nil {
			return nil, err
		}
	}

	sk := &SortKey{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		if len(body) > 0 {
			sk.Body = body
		} else {
			expr, err := compileXPath(".", c.nsBindings)
			if err != nil {
				return nil, err
			}
			sk.Select = expr
		}
	}

	if order := getAttr(elem, "order"); order != "" {
		avt, err := compileAVT(order, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Order = avt
	}

	if dataType := getAttr(elem, "data-type"); dataType != "" {
		avt, err := compileAVT(dataType, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.DataType = avt
	}

	if caseOrder := getAttr(elem, "case-order"); caseOrder != "" {
		avt, err := compileAVT(caseOrder, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.CaseOrder = avt
	}

	if lang := getAttr(elem, "lang"); lang != "" {
		avt, err := compileAVT(lang, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Lang = avt
	}

	return sk, nil
}

func (c *compiler) compileWithParam(elem *helium.Element) (*WithParam, error) {
	// Push element-local namespace declarations (for EQName variable refs)
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	// Validate attributes: xsl:with-param allows name, select, as, tunnel
	// but NOT required (that's only for xsl:param)
	if err := validateXSLTAttrs(elem, withParamAllowedAttrs); err != nil {
		return nil, err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:with-param requires name attribute")
	}

	// Validate tunnel attribute value if present (including empty string)
	if tunnelAttr, hasTunnel := elem.GetAttribute("tunnel"); hasTunnel {
		if err := validateBooleanAttr("xsl:with-param", "tunnel", tunnelAttr); err != nil {
			return nil, err
		}
	}

	wp := &WithParam{
		Name: resolveQName(name, c.nsBindings),
		As:   getAttr(elem, "as"),
	}

	if tunnelAttr := getAttr(elem, "tunnel"); tunnelAttr == "yes" {
		wp.Tunnel = true
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		wp.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		wp.Body = body
	}

	return wp, nil
}

func (c *compiler) compileSequence(elem *helium.Element) (Instruction, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		return &XSLSequenceInst{Select: expr}, nil
	}
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	return &SequenceInst{Body: body}, nil
}

func (c *compiler) compilePerformSort(elem *helium.Element) (*PerformSortInst, error) {
	inst := &PerformSortInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	// Collect sort keys and body
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "sort" {
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			inst.Sort = append(inst.Sort, sk)
		} else {
			childInst, err := c.compileInstruction(childElem)
			if err != nil {
				return nil, err
			}
			if childInst != nil {
				inst.Body = append(inst.Body, childInst)
			}
		}
	}

	return inst, nil
}

func (c *compiler) compileNextMatch(elem *helium.Element) (*NextMatchInst, error) {
	inst := &NextMatchInst{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "with-param" {
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			inst.Params = append(inst.Params, wp)
		}
	}
	return inst, nil
}

func (c *compiler) compileApplyImports(elem *helium.Element) (*ApplyImportsInst, error) {
	inst := &ApplyImportsInst{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "with-param" {
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			inst.Params = append(inst.Params, wp)
		}
	}
	return inst, nil
}

func (c *compiler) compileMap(elem *helium.Element) (*MapInst, error) {
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	return &MapInst{Body: body}, nil
}

func (c *compiler) compileMapEntry(elem *helium.Element) (*MapEntryInst, error) {
	inst := &MapEntryInst{}

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
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileTry(elem *helium.Element) (*TryCatchInst, error) {
	inst := &TryCatchInst{}

	// xsl:try select attribute
	if sel := getAttr(elem, "select"); sel != "" {
		expr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "catch" {
			clause := &CatchClause{}

			// Parse errors attribute (space-separated list of error codes)
			if errAttr := getAttr(childElem, "errors"); errAttr != "" {
				for _, code := range strings.Fields(errAttr) {
					clause.Errors = append(clause.Errors, resolveQName(code, c.nsBindings))
				}
			}

			// xsl:catch select attribute
			if sel := getAttr(childElem, "select"); sel != "" {
				expr, err := compileXPath(sel, c.nsBindings)
				if err != nil {
					return nil, err
				}
				clause.Select = expr
			} else {
				body, err := c.compileChildren(childElem)
				if err != nil {
					return nil, err
				}
				clause.Body = body
			}
			inst.Catches = append(inst.Catches, clause)
		} else if childElem.URI() == NSXSLT && childElem.LocalName() == "fallback" {
			// xsl:fallback inside xsl:try is silently ignored
			continue
		} else {
			childInst, err := c.compileInstruction(childElem)
			if err != nil {
				return nil, err
			}
			if childInst != nil {
				inst.Try = append(inst.Try, childInst)
			}
		}
	}

	return inst, nil
}

func (c *compiler) compileForEachGroup(elem *helium.Element) (*ForEachGroupInst, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:for-each-group requires select attribute")
	}

	// XTSE1080: exactly one of group-by, group-adjacent, group-starting-with,
	// group-ending-with must be present
	groupingCount := 0
	if getAttr(elem, "group-by") != "" {
		groupingCount++
	}
	if getAttr(elem, "group-adjacent") != "" {
		groupingCount++
	}
	if getAttr(elem, "group-starting-with") != "" {
		groupingCount++
	}
	if getAttr(elem, "group-ending-with") != "" {
		groupingCount++
	}
	if groupingCount == 0 {
		return nil, staticError(errCodeXTSE1080, "xsl:for-each-group requires one of group-by, group-adjacent, group-starting-with, or group-ending-with")
	}
	if groupingCount > 1 {
		return nil, staticError(errCodeXTSE1080, "xsl:for-each-group must have at most one of group-by, group-adjacent, group-starting-with, or group-ending-with")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ForEachGroupInst{Select: expr}

	// Validate boolean attribute: composite
	if comp, hasComp := elem.GetAttribute("composite"); hasComp {
		if err := validateBooleanAttr("xsl:for-each-group", "composite", comp); err != nil {
			return nil, err
		}
		if comp == "yes" || comp == "true" || comp == "1" {
			inst.Composite = true
		}
	}

	if gb := getAttr(elem, "group-by"); gb != "" {
		gbExpr, gbErr := compileXPath(gb, c.nsBindings)
		if gbErr != nil {
			return nil, gbErr
		}
		inst.GroupBy = gbExpr
	}
	if ga := getAttr(elem, "group-adjacent"); ga != "" {
		gaExpr, gaErr := compileXPath(ga, c.nsBindings)
		if gaErr != nil {
			return nil, gaErr
		}
		inst.GroupAdjacent = gaExpr
	}
	if gs := getAttr(elem, "group-starting-with"); gs != "" {
		gsPat, gsErr := compilePattern(gs, c.nsBindings, c.xpathDefaultNS)
		if gsErr != nil {
			return nil, gsErr
		}
		inst.GroupStartingWith = gsPat
	}
	if ge := getAttr(elem, "group-ending-with"); ge != "" {
		gePat, geErr := compilePattern(ge, c.nsBindings, c.xpathDefaultNS)
		if geErr != nil {
			return nil, geErr
		}
		inst.GroupEndingWith = gePat
	}

	// Compile body (skip sort elements)
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "sort" {
				sk, sortErr := c.compileSortKey(v)
				if sortErr != nil {
					return nil, sortErr
				}
				inst.Sort = append(inst.Sort, sk)
				continue
			}
			childInst, childErr := c.compileInstruction(v)
			if childErr != nil {
				return nil, childErr
			}
			if childInst != nil {
				inst.Body = append(inst.Body, childInst)
			}
		case *helium.Text:
			text := string(v.Content())
			if !c.shouldStripText(text) {
				inst.Body = append(inst.Body, &LiteralTextInst{Value: text})
			}
		}
	}

	return inst, nil
}

func (c *compiler) compileLiteralResultElement(elem *helium.Element) (*LiteralResultElement, error) {
	// xsl:break / xsl:next-iteration not allowed inside element constructors.
	savedBreak := c.breakAllowed
	c.breakAllowed = false
	defer func() { c.breakAllowed = savedBreak }()

	lre := &LiteralResultElement{
		Name:       elem.Name(),
		Namespace:  elem.URI(),
		Prefix:     elem.Prefix(),
		LocalName:  elem.LocalName(),
		Namespaces: make(map[string]string),
	}

	// Collect element-level xsl:exclude-result-prefixes and
	// xsl:extension-element-prefixes (cumulative with parent)
	savedExcludes := c.localExcludes
	needNewExcludes := false
	if _, ok := elem.GetAttributeNS("exclude-result-prefixes", NSXSLT); ok {
		needNewExcludes = true
	}
	if _, ok := elem.GetAttributeNS("extension-element-prefixes", NSXSLT); ok {
		needNewExcludes = true
	}
	if needNewExcludes {
		newExcludes := make(map[string]struct{})
		for k, v := range c.localExcludes {
			newExcludes[k] = v
		}
		if erp, ok := elem.GetAttributeNS("exclude-result-prefixes", NSXSLT); ok {
			if erp == "#all" {
				for prefix := range c.stylesheet.namespaces {
					newExcludes[prefix] = struct{}{}
				}
			} else {
				for _, prefix := range strings.Fields(erp) {
					newExcludes[prefix] = struct{}{}
				}
			}
		}
		if eep, ok := elem.GetAttributeNS("extension-element-prefixes", NSXSLT); ok {
			for _, prefix := range strings.Fields(eep) {
				newExcludes[prefix] = struct{}{}
			}
		}
		c.localExcludes = newExcludes
	}
	defer func() { c.localExcludes = savedExcludes }()

	isExcluded := func(prefix string) bool {
		if _, ok := c.stylesheet.excludePrefixes[prefix]; ok {
			return true
		}
		if _, ok := c.localExcludes[prefix]; ok {
			return true
		}
		return false
	}

	// Copy in-scope namespace declarations from stylesheet that are not excluded.
	// These represent namespaces that should propagate to the result tree.
	for prefix, uri := range c.stylesheet.namespaces {
		if uri == NSXSLT || prefix == "" {
			continue
		}
		if isExcluded(prefix) {
			continue
		}
		lre.Namespaces[prefix] = uri
	}
	// Override/add from directly declared namespaces on this element
	for _, ns := range elem.Namespaces() {
		uri := ns.URI()
		prefix := ns.Prefix()
		if uri == NSXSLT || prefix == "" {
			continue
		}
		if isExcluded(prefix) {
			delete(lre.Namespaces, prefix)
			continue
		}
		lre.Namespaces[prefix] = uri
	}

	// Validate and compile attributes
	for _, attr := range elem.Attributes() {
		if attr.URI() == NSXSLT {
			// XTSE0805: only certain XSLT attributes are allowed on LREs
			if _, ok := lreAllowedXSLTAttrs[attr.LocalName()]; !ok {
				return nil, staticError(errCodeXTSE0805,
					"attribute xsl:%s is not allowed on a literal result element", attr.LocalName())
			}
			continue
		}
		avt, err := compileAVT(attr.Value(), c.nsBindings)
		if err != nil {
			return nil, err
		}
		lre.Attrs = append(lre.Attrs, &LiteralAttribute{
			Name:      attr.Name(),
			Namespace: attr.URI(),
			Prefix:    attr.Prefix(),
			LocalName: attr.LocalName(),
			Value:     avt,
		})
	}

	// Apply namespace aliases (xsl:namespace-alias).
	// This must happen after namespace declarations and attributes are collected,
	// but before children are compiled.
	if len(c.stylesheet.namespaceAliases) > 0 {
		// Alias the element itself
		if resultURI, resultPfx, ok := c.resolveNamespaceAlias(lre.Namespace); ok {
			lre.Namespace = resultURI
			if resultPfx != "" {
				lre.Prefix = resultPfx
				lre.Name = resultPfx + ":" + lre.LocalName
			} else {
				lre.Prefix = ""
				lre.Name = lre.LocalName
			}
		}

		// Alias attributes
		for _, attr := range lre.Attrs {
			if attr.Namespace == "" {
				continue
			}
			if resultURI, resultPfx, ok := c.resolveNamespaceAlias(attr.Namespace); ok {
				attr.Namespace = resultURI
				if resultPfx != "" {
					attr.Prefix = resultPfx
					attr.Name = resultPfx + ":" + attr.LocalName
				} else {
					attr.Prefix = ""
					attr.Name = attr.LocalName
				}
			}
		}

		// Alias namespace declarations
		aliasedNS := make(map[string]string)
		for prefix, uri := range lre.Namespaces {
			if resultURI, resultPfx, ok := c.resolveNamespaceAlias(uri); ok {
				if resultPfx != "" {
					aliasedNS[resultPfx] = resultURI
				} else {
					aliasedNS[prefix] = resultURI
				}
			} else {
				aliasedNS[prefix] = uri
			}
		}
		lre.Namespaces = aliasedNS
	}

	// Handle xsl:use-attribute-sets
	if uas, ok := elem.GetAttributeNS("use-attribute-sets", NSXSLT); ok {
		for _, name := range strings.Fields(uas) {
			lre.UseAttributeSets = append(lre.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
		lre.UseAttrSets = strings.Fields(uas)
	}

	// Compile children
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	lre.Body = body

	return lre, nil
}

func (c *compiler) compileAnalyzeString(elem *helium.Element) (*AnalyzeStringInst, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:analyze-string requires select attribute")
	}
	regexAttr, regexFound := elem.GetAttribute("regex")
	if !regexFound {
		return nil, staticError(errCodeXTSE0110, "xsl:analyze-string requires regex attribute")
	}

	selectExpr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	regexAVT, err := compileAVT(regexAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &AnalyzeStringInst{
		Select: selectExpr,
		Regex:  regexAVT,
	}

	flagsAttr := getAttr(elem, "flags")
	if flagsAttr != "" {
		flagsAVT, err := compileAVT(flagsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Flags = flagsAVT
	}

	// Compile xsl:matching-substring and xsl:non-matching-substring children.
	// The spec requires: matching-substring? then non-matching-substring? then fallback*.
	// (XTSE0010 if out of order)
	const (
		phaseInit     = 0
		phaseMatching = 1
		phaseNonMatch = 2
		phaseFallback = 3
	)
	phase := phaseInit
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}
		switch childElem.LocalName() {
		case "matching-substring":
			if phase >= phaseMatching {
				return nil, staticError(errCodeXTSE0010, "xsl:matching-substring must precede xsl:non-matching-substring and xsl:fallback")
			}
			phase = phaseMatching
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.MatchingBody = body
		case "non-matching-substring":
			if phase >= phaseNonMatch {
				return nil, staticError(errCodeXTSE0010, "xsl:non-matching-substring out of order in xsl:analyze-string")
			}
			phase = phaseNonMatch
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.NonMatchingBody = body
		case "fallback":
			phase = phaseFallback
		}
	}

	// XTSE1130: at least one of matching-substring or non-matching-substring is required
	if len(inst.MatchingBody) == 0 && len(inst.NonMatchingBody) == 0 {
		return nil, staticError("XTSE1130", "xsl:analyze-string must contain xsl:matching-substring or xsl:non-matching-substring")
	}

	return inst, nil
}
