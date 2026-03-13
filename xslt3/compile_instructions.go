package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
)

// compileInstruction compiles a single element into an Instruction.
func (c *compiler) compileInstruction(elem *helium.Element) (Instruction, error) {
	// Push element-local namespace declarations into scope
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	if elem.URI() == NSXSLT {
		return c.compileXSLTInstruction(elem)
	}
	return c.compileLiteralResultElement(elem)
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
		// For now, treat result-document body as regular output
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		return &SequenceInst{Body: body}, nil
	case "where-populated":
		// xsl:where-populated: execute body and only include if non-empty
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		return &WherePopulatedInst{Body: body}, nil
	case "on-empty":
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		return &SequenceInst{Body: body}, nil
	case "try":
		return c.compileTry(elem)
	case "for-each-group":
		return c.compileForEachGroup(elem)
	case "map":
		// xsl:map: stub for now
		return &SequenceInst{}, nil
	case "map-entry":
		return &SequenceInst{}, nil
	case "sort":
		// xsl:sort is handled by parent instructions
		return nil, nil
	default:
		return nil, staticError(errCodeXTSE0090, "unknown XSLT instruction xsl:%s", elem.LocalName())
	}
}

// compileChildren compiles all children of an element into instructions.
func (c *compiler) compileChildren(parent *helium.Element) ([]Instruction, error) {
	var body []Instruction
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			inst, err := c.compileInstruction(v)
			if err != nil {
				return nil, err
			}
			if inst != nil {
				body = append(body, inst)
			}
		case *helium.Text:
			text := string(v.Content())
			if strings.TrimSpace(text) != "" {
				body = append(body, &LiteralTextInst{Value: text})
			}
		case *helium.CDATASection:
			body = append(body, &LiteralTextInst{Value: string(v.Content())})
		}
	}
	return body, nil
}

func (c *compiler) compileApplyTemplates(elem *helium.Element) (*ApplyTemplatesInst, error) {
	inst := &ApplyTemplatesInst{
		Mode: getAttr(elem,"mode"),
	}

	selectAttr := getAttr(elem,"select")
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
	name := getAttr(elem,"name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:call-template requires name attribute")
	}

	inst := &CallTemplateInst{Name: name}

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

	selectAttr := getAttr(elem,"select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if sep := getAttr(elem, "separator"); sep != "" {
		avt, err := compileAVT(sep, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Separator = avt
	}

	// Check for body content (XSLT 2.0+)
	if selectAttr == "" {
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

	return &TextInst{
		Value:                 sb.String(),
		DisableOutputEscaping: getAttr(elem,"disable-output-escaping") == "yes",
	}, nil
}

func (c *compiler) compileElement(elem *helium.Element) (*ElementInst, error) {
	nameAttr := getAttr(elem,"name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:element requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ElementInst{Name: nameAVT}

	nsAttr := getAttr(elem,"namespace")
	if nsAttr != "" {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	return inst, nil
}

func (c *compiler) compileAttribute(elem *helium.Element) (*AttributeInst, error) {
	nameAttr := getAttr(elem,"name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:attribute requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &AttributeInst{Name: nameAVT}

	nsAttr := getAttr(elem,"namespace")
	if nsAttr != "" {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	selectAttr := getAttr(elem,"select")
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

	selectAttr := getAttr(elem,"select")
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
	nameAttr := getAttr(elem,"name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:processing-instruction requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &PIInst{Name: nameAVT}

	selectAttr := getAttr(elem,"select")
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
	testAttr := getAttr(elem,"test")
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
			testAttr := getAttr(childElem,"test")
			if testAttr == "" {
				return nil, staticError(errCodeXTSE0110, "xsl:when requires test attribute")
			}
			expr, err := compileXPath(testAttr, c.nsBindings)
			if err != nil {
				return nil, err
			}
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.When = append(inst.When, &WhenClause{Test: expr, Body: body})
		case "otherwise":
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.Otherwise = body
		}
	}

	return inst, nil
}

func (c *compiler) compileForEach(elem *helium.Element) (*ForEachInst, error) {
	selectAttr := getAttr(elem,"select")
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
			if strings.TrimSpace(text) != "" {
				inst.Body = append(inst.Body, &LiteralTextInst{Value: text})
			}
		case *helium.CDATASection:
			inst.Body = append(inst.Body, &LiteralTextInst{Value: string(v.Content())})
		}
	}

	return inst, nil
}

func (c *compiler) compileLocalVariable(elem *helium.Element) (*VariableInst, error) {
	name := getAttr(elem,"name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	inst := &VariableInst{Name: name}

	selectAttr := getAttr(elem,"select")
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
	name := getAttr(elem,"name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	inst := &ParamInst{
		Name:     name,
		Required: getAttr(elem,"required") == "yes",
	}

	selectAttr := getAttr(elem,"select")
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
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	return &CopyInst{Body: body}, nil
}

func (c *compiler) compileCopyOf(elem *helium.Element) (*CopyOfInst, error) {
	selectAttr := getAttr(elem,"select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:copy-of requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	return &CopyOfInst{Select: expr}, nil
}

func (c *compiler) compileNumber(elem *helium.Element) (*NumberInst, error) {
	inst := &NumberInst{
		Level: getAttr(elem,"level"),
	}
	if inst.Level == "" {
		inst.Level = "single"
	}

	if countAttr := getAttr(elem,"count"); countAttr != "" {
		p, err := compilePattern(countAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Count = p
	}

	if fromAttr := getAttr(elem,"from"); fromAttr != "" {
		p, err := compilePattern(fromAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.From = p
	}

	if valueAttr := getAttr(elem,"value"); valueAttr != "" {
		expr, err := compileXPath(valueAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Value = expr
	}

	if fmtAttr := getAttr(elem,"format"); fmtAttr != "" {
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

	return inst, nil
}

func (c *compiler) compileMessage(elem *helium.Element) (*MessageInst, error) {
	inst := &MessageInst{}

	selectAttr := getAttr(elem,"select")
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

	termAttr := getAttr(elem,"terminate")
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
	nameAttr := getAttr(elem,"name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:namespace requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &NamespaceInst{Name: nameAVT}

	selectAttr := getAttr(elem,"select")
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
	sk := &SortKey{}

	selectAttr := getAttr(elem,"select")
	if selectAttr == "" {
		selectAttr = "."
	}
	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}
	sk.Select = expr

	if order := getAttr(elem,"order"); order != "" {
		avt, err := compileAVT(order, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Order = avt
	}

	if dataType := getAttr(elem,"data-type"); dataType != "" {
		avt, err := compileAVT(dataType, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.DataType = avt
	}

	if caseOrder := getAttr(elem,"case-order"); caseOrder != "" {
		avt, err := compileAVT(caseOrder, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.CaseOrder = avt
	}

	if lang := getAttr(elem,"lang"); lang != "" {
		avt, err := compileAVT(lang, c.nsBindings)
		if err != nil {
			return nil, err
		}
		sk.Lang = avt
	}

	return sk, nil
}

func (c *compiler) compileWithParam(elem *helium.Element) (*WithParam, error) {
	name := getAttr(elem,"name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:with-param requires name attribute")
	}

	wp := &WithParam{Name: name}

	selectAttr := getAttr(elem,"select")
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

func (c *compiler) compileTry(elem *helium.Element) (*TryCatchInst, error) {
	inst := &TryCatchInst{}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "catch" {
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.Catch = body
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

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ForEachGroupInst{Select: expr}
	inst.GroupBy = getAttr(elem, "group-by")

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
			if strings.TrimSpace(text) != "" {
				inst.Body = append(inst.Body, &LiteralTextInst{Value: text})
			}
		}
	}

	return inst, nil
}

func (c *compiler) compileLiteralResultElement(elem *helium.Element) (*LiteralResultElement, error) {
	lre := &LiteralResultElement{
		Name:       elem.Name(),
		Namespace:  elem.URI(),
		Prefix:     elem.Prefix(),
		LocalName:  elem.LocalName(),
		Namespaces: make(map[string]string),
	}

	// Collect element-level xsl:exclude-result-prefixes (cumulative with parent)
	savedExcludes := c.localExcludes
	if erp, ok := elem.GetAttributeNS("exclude-result-prefixes", NSXSLT); ok {
		// Copy parent excludes and add new ones
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

	// Compile attributes (those not in XSLT namespace) with AVTs
	for _, attr := range elem.Attributes() {
		if attr.URI() == NSXSLT {
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

	// Compile children
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	lre.Body = body

	return lre, nil
}
