package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
)

func (c *compiler) compileApplyTemplates(elem *helium.Element) (*ApplyTemplatesInst, error) {
	mode := strings.TrimSpace(getAttr(elem, "mode"))
	// Validate mode name is a valid QName.
	if mode != "" && mode[0] != '#' {
		if !isValidQName(mode) && !isValidEQName(mode) {
			return nil, staticError(errCodeXTSE0550, "invalid mode name %q on xsl:apply-templates", mode)
		}
		// XTSE0280: check for undeclared prefix.
		if idx := strings.IndexByte(mode, ':'); idx > 0 {
			prefix := mode[:idx]
			if _, ok := c.nsBindings[prefix]; !ok {
				return nil, staticError(errCodeXTSE0280, "undeclared namespace prefix %q in mode name %q", prefix, mode)
			}
		}
	}
	// When mode is absent, use the current default-mode (set by
	// compileInstruction from ancestor or self default-mode attributes).
	if mode == "" && c.defaultMode != "" {
		mode = c.defaultMode
	}
	// Resolve mode QName to Clark notation (namespace-aware)
	mode = c.resolveMode(mode)
	// Record mode usage for XTSE3085 checking
	if mode != "#current" {
		c.recordModeUsage(mode)
	}
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
	sortCount := 0
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			// XTSE0010: non-element content not allowed in xsl:apply-templates
			if isNonWhitespaceTextNode(child) {
				return nil, staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:apply-templates")
			}
			continue
		}
		if childElem.URI() != NSXSLT {
			return nil, staticError(errCodeXTSE0010, "non-XSLT element %q is not allowed as a child of xsl:apply-templates", childElem.Name())
		}
		switch childElem.LocalName() {
		case "sort":
			// XTSE1017: @stable only on first xsl:sort
			if sortCount > 0 {
				if _, has := childElem.GetAttribute("stable"); has {
					return nil, staticError(errCodeXTSE1017,
						"the stable attribute is permitted only on the first xsl:sort element")
				}
			}
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			if sk != nil {
				inst.Sort = append(inst.Sort, sk)
				sortCount++
			}
		case "with-param":
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			if wp != nil {
				inst.Params = append(inst.Params, wp)
			}
		default:
			// XTSE0010: only xsl:sort and xsl:with-param are allowed
			return nil, staticError(errCodeXTSE0010, "xsl:%s is not allowed as a child of xsl:apply-templates", childElem.LocalName())
		}
	}

	return inst, nil
}

func (c *compiler) compileCallTemplate(elem *helium.Element) (*CallTemplateInst, error) {
	if err := validateXSLTAttrs(elem, map[string]struct{}{
		"name": {},
	}); err != nil {
		return nil, err
	}
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:call-template requires name attribute")
	}

	inst := &CallTemplateInst{Name: resolveQName(name, c.nsBindings)}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			if isNonWhitespaceTextNode(child) {
				return nil, staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:call-template")
			}
			continue
		}
		if childElem.URI() != NSXSLT {
			return nil, staticError(errCodeXTSE0010, "non-XSLT element %q is not allowed as a child of xsl:call-template", childElem.Name())
		}
		if childElem.LocalName() != "with-param" {
			return nil, staticError(errCodeXTSE0010, "xsl:%s is not allowed as a child of xsl:call-template", childElem.LocalName())
		}
		wp, err := c.compileWithParam(childElem)
		if err != nil {
			return nil, err
		}
		if wp != nil {
			inst.Params = append(inst.Params, wp)
		}
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
	inst := &ChooseInst{DefaultCollation: c.defaultCollation}
	hasOtherwise := false

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		// XTSE0010: xsl:choose must contain only xsl:when and xsl:otherwise elements
		switch v := child.(type) {
		case *helium.Text, *helium.CDATASection:
			if !isWhitespaceOnly(string(child.Content())) {
				return nil, staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:choose")
			}
			continue
		case *helium.Element:
			_ = v
			// handled below
		default:
			continue
		}
		childElem := child.(*helium.Element)
		if childElem.URI() != NSXSLT {
			return nil, staticError(errCodeXTSE0010, "non-XSLT element %q is not allowed as a child of xsl:choose", childElem.Name())
		}

		switch childElem.LocalName() {
		case "when":
			// XTSE0010: xsl:when must not appear after xsl:otherwise
			if hasOtherwise {
				return nil, staticError(errCodeXTSE0010, "xsl:when must not appear after xsl:otherwise in xsl:choose")
			}
			// Push element-local namespace declarations into scope
			savedBindings := c.pushElementNamespaces(childElem)
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
				c.nsBindings = savedBindings
				c.xpathDefaultNS = savedNS
				c.expandText = savedET
				return nil, staticError(errCodeXTSE0110, "xsl:when requires test attribute")
			}
			expr, err := compileXPath(testAttr, c.nsBindings)
			if err != nil {
				c.nsBindings = savedBindings
				c.xpathDefaultNS = savedNS
				c.expandText = savedET
				return nil, err
			}
			// Capture per-clause namespace bindings for runtime resolution
			var clauseNS map[string]string
			if len(c.nsBindings) > 0 {
				clauseNS = make(map[string]string, len(c.nsBindings))
				for k, v := range c.nsBindings {
					clauseNS[k] = v
				}
			}
			whenCollation := getAttr(childElem, "default-collation")
			body, err := c.compileChildren(childElem)
			wc := &WhenClause{Test: expr, Body: body, Namespaces: clauseNS, DefaultCollation: whenCollation}
			wc.XPathDefaultNS = c.xpathDefaultNS
			wc.HasXPathDefaultNS = hasLocal
			c.nsBindings = savedBindings
			c.xpathDefaultNS = savedNS
			c.expandText = savedET
			if err != nil {
				return nil, err
			}
			inst.When = append(inst.When, wc)
		case "otherwise":
			// XTSE0010: at most one xsl:otherwise is allowed
			if hasOtherwise {
				return nil, staticError(errCodeXTSE0010, "xsl:choose must not contain more than one xsl:otherwise")
			}
			hasOtherwise = true
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
		default:
			// XTSE0010: only xsl:when and xsl:otherwise are allowed inside xsl:choose
			return nil, staticError(errCodeXTSE0010, "xsl:%s is not allowed as a child of xsl:choose", childElem.LocalName())
		}
	}

	// XTSE0010: xsl:choose must contain at least one xsl:when
	if len(inst.When) == 0 {
		return nil, staticError(errCodeXTSE0010, "xsl:choose must contain at least one xsl:when")
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

	// First pass: collect sort keys. Validate sort comes before content.
	pastSortContent := false
	sortCount := 0
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "sort" {
			if pastSortContent {
				return nil, staticError(errCodeXTSE0010, "xsl:sort must come before other content in xsl:for-each")
			}
			// XTSE1017: @stable only on first xsl:sort
			if sortCount > 0 {
				if _, has := childElem.GetAttribute("stable"); has {
					return nil, staticError(errCodeXTSE1017,
						"the stable attribute is permitted only on the first xsl:sort element")
				}
			}
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			if sk != nil {
				inst.Sort = append(inst.Sort, sk)
				sortCount++
			}
		} else {
			pastSortContent = true
		}
	}

	// Second pass: compile body (skip sort elements).
	// Find the last xsl:sort element so we know where the body starts.
	// Whitespace-only text nodes before/between sorts are stripped even with
	// xml:space="preserve" per XSLT §4.2 (element-only content).
	var lastSort helium.Node
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if childElem, ok := child.(*helium.Element); ok {
			if childElem.URI() == NSXSLT && childElem.LocalName() == "sort" {
				lastSort = child
			}
		}
	}
	pastSort := lastSort == nil // true once we've passed all sort elements
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child == lastSort {
			pastSort = true
			continue
		}
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "sort" {
				continue
			}
			pastSort = true
			childInst, err := c.compileInstruction(v)
			if err != nil {
				return nil, err
			}
			if childInst != nil {
				inst.Body = append(inst.Body, childInst)
			}
		case *helium.Text:
			text := string(v.Content())
			// Strip whitespace-only text nodes in the sort region (before/between sorts)
			if !pastSort && strings.TrimSpace(text) == "" {
				continue
			}
			if !c.shouldStripText(text) {
				inst.Body = append(inst.Body, &LiteralTextInst{Value: text})
			}
		case *helium.CDATASection:
			inst.Body = append(inst.Body, &LiteralTextInst{Value: string(v.Content())})
		}
	}

	return inst, nil
}

func (c *compiler) compileSortKey(elem *helium.Element) (*SortKey, error) {
	// Evaluate use-when on xsl:sort before compiling the sort key.
	if uw := getAttr(elem, "use-when"); uw != "" {
		include, err := c.evaluateUseWhen(uw)
		if err != nil {
			return nil, err
		}
		if !include {
			return nil, nil
		}
	}

	// Validate boolean attribute: stable (including empty string).
	// Skip validation for AVTs (contain '{' and '}') since the value
	// is determined at runtime.
	if stableAttr, hasStable := elem.GetAttribute("stable"); hasStable {
		if !strings.ContainsAny(stableAttr, "{}") {
			if err := validateBooleanAttr("xsl:sort", "stable", stableAttr); err != nil {
				return nil, err
			}
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

	if col := getAttr(elem, "collation"); col != "" {
		avt, err := compileAVT(col, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Collation = avt
	}

	return sk, nil
}

func (c *compiler) compileWithParam(elem *helium.Element) (*WithParam, error) {
	// Check use-when before compiling: skip this with-param if excluded.
	if uw := getAttr(elem, "use-when"); uw != "" {
		include, err := c.evaluateUseWhen(uw)
		if err != nil {
			return nil, err
		}
		if !include {
			return nil, nil
		}
	}

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

	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:with-param "+name); err != nil {
		return nil, err
	}

	wp := &WithParam{
		Name: resolveQName(name, c.nsBindings),
		As:   asAttr,
	}

	wp.Tunnel = xsdBoolTrue(getAttr(elem, "tunnel"))

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0620: select and non-empty content are mutually exclusive.
		if err := c.validateEmptyElement(elem, "xsl:with-param"); err != nil {
			return nil, staticError(errCodeXTSE0620, "xsl:with-param %q has both @select and content", name)
		}
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

	// Collect sort keys and body. xsl:sort must come before other content.
	pastSort := false
	sortCount := 0
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "sort" {
			if pastSort {
				return nil, staticError(errCodeXTSE0010, "xsl:sort must come before other content in xsl:perform-sort")
			}
			// XTSE1017: @stable only on first xsl:sort
			if sortCount > 0 {
				if _, has := childElem.GetAttribute("stable"); has {
					return nil, staticError(errCodeXTSE1017,
						"the stable attribute is permitted only on the first xsl:sort element")
				}
			}
			sk, err := c.compileSortKey(childElem)
			if err != nil {
				return nil, err
			}
			if sk != nil {
				inst.Sort = append(inst.Sort, sk)
				sortCount++
			}
		} else {
			pastSort = true
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
			if wp != nil {
				inst.Params = append(inst.Params, wp)
			}
		}
	}
	return inst, nil
}

func (c *compiler) compileApplyImports(elem *helium.Element) (*ApplyImportsInst, error) {
	inst := &ApplyImportsInst{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			if isNonWhitespaceTextNode(child) {
				return nil, staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:apply-imports")
			}
			continue
		}
		if childElem.URI() != NSXSLT {
			return nil, staticError(errCodeXTSE0010, "non-XSLT element %q is not allowed as a child of xsl:apply-imports", childElem.Name())
		}
		if childElem.LocalName() != "with-param" {
			return nil, staticError(errCodeXTSE0010, "xsl:%s is not allowed as a child of xsl:apply-imports", childElem.LocalName())
		}
		wp, err := c.compileWithParam(childElem)
		if err != nil {
			return nil, err
		}
		if wp != nil {
			inst.Params = append(inst.Params, wp)
		}
	}
	return inst, nil
}

func (c *compiler) compileTry(elem *helium.Element) (*TryCatchInst, error) {
	inst := &TryCatchInst{RollbackOutput: true}

	if rb := getAttr(elem, "rollback-output"); rb == "no" {
		inst.RollbackOutput = false
	}

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

			// Parse errors attribute (space-separated list of error codes).
			// Namespace declarations on xsl:catch itself must be visible
			// when resolving prefixed error codes (e.g. err:XTDE0555).
			if errAttr := getAttr(childElem, "errors"); errAttr != "" {
				// Build a merged namespace map including catch-element bindings
				catchNS := make(map[string]string, len(c.nsBindings))
				for k, v := range c.nsBindings {
					catchNS[k] = v
				}
				for _, ns := range childElem.Namespaces() {
					catchNS[ns.Prefix()] = ns.URI()
				}
				for _, code := range strings.Fields(errAttr) {
					clause.Errors = append(clause.Errors, resolveQName(code, catchNS))
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

	// Compile collation AVT
	if collAttr := getAttr(elem, "collation"); collAttr != "" {
		collAVT, collErr := compileAVT(collAttr, c.nsBindings)
		if collErr != nil {
			return nil, collErr
		}
		inst.Collation = collAVT
	}

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
	sortCount := 0
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "sort" {
				// XTSE1017: @stable only on first xsl:sort
				if sortCount > 0 {
					if _, has := v.GetAttribute("stable"); has {
						return nil, staticError(errCodeXTSE1017,
							"the stable attribute is permitted only on the first xsl:sort element")
					}
				}
				sk, sortErr := c.compileSortKey(v)
				if sortErr != nil {
					return nil, sortErr
				}
				if sk != nil {
					inst.Sort = append(inst.Sort, sk)
					sortCount++
				}
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
		return nil, staticError(errCodeXTSE1130, "xsl:analyze-string must contain xsl:matching-substring or xsl:non-matching-substring")
	}

	return inst, nil
}

// compileEvaluate compiles xsl:evaluate.
func (c *compiler) compileEvaluate(elem *helium.Element) (Instruction, error) {
	inst := &EvaluateInst{}

	// xpath attribute is required
	xpathAttr := getAttr(elem, "xpath")
	if xpathAttr == "" {
		return nil, staticError(errCodeXTSE0010, "xsl:evaluate requires an xpath attribute")
	}
	expr, err := compileXPath(xpathAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}
	inst.XPath = expr

	// context-item (optional)
	if ci := getAttr(elem, "context-item"); ci != "" {
		ciExpr, err := compileXPath(ci, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.ContextItem = ciExpr
	}

	// base-uri (optional AVT)
	if bu := getAttr(elem, "base-uri"); bu != "" {
		avt, err := compileAVT(bu, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.BaseURI = avt
	}

	// namespace-context (optional XPath expression producing a node)
	if nc := getAttr(elem, "namespace-context"); nc != "" {
		ncExpr, err := compileXPath(nc, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.NamespaceContext = ncExpr
	}

	// with-params (optional map expression)
	if wp := getAttr(elem, "with-params"); wp != "" {
		wpExpr, err := compileXPath(wp, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.WithParamsExpr = wpExpr
	}

	// as (optional type name)
	inst.As = getAttr(elem, "as")

	// schema-aware (optional AVT that evaluates to boolean)
	if saStr, hasSA := elem.GetAttribute("schema-aware"); hasSA {
		saAVT, err := compileAVT(saStr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.SchemaAwareAVT = saAVT
		inst.HasSchemaAware = true
	}

	// Compile child xsl:with-param elements
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}
		if childElem.LocalName() == "with-param" {
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			if wp != nil {
				inst.Params = append(inst.Params, wp)
			}
		} else if childElem.LocalName() == "fallback" {
			// skip
		} else {
			return nil, staticError(errCodeXTSE0010, "xsl:evaluate may only contain xsl:with-param or xsl:fallback, found xsl:%s", childElem.LocalName())
		}
	}

	return inst, nil
}
