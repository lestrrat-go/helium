package xslt3

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// executeInstruction dispatches execution of a compiled XSLT instruction.
func (ec *execContext) executeInstruction(ctx context.Context, inst Instruction) error {
	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		return ec.execApplyTemplates(ctx, v)
	case *CallTemplateInst:
		return ec.execCallTemplate(ctx, v)
	case *ValueOfInst:
		return ec.execValueOf(ctx, v)
	case *TextInst:
		return ec.execText(v)
	case *LiteralTextInst:
		return ec.execLiteralText(v)
	case *ElementInst:
		return ec.execElement(ctx, v)
	case *AttributeInst:
		return ec.execAttribute(ctx, v)
	case *CommentInst:
		return ec.execComment(ctx, v)
	case *PIInst:
		return ec.execPI(ctx, v)
	case *IfInst:
		return ec.execIf(ctx, v)
	case *ChooseInst:
		return ec.execChoose(ctx, v)
	case *ForEachInst:
		return ec.execForEach(ctx, v)
	case *VariableInst:
		return ec.execVariable(ctx, v)
	case *ParamInst:
		return ec.execParam(ctx, v)
	case *CopyInst:
		return ec.execCopy(ctx, v)
	case *CopyOfInst:
		return ec.execCopyOf(ctx, v)
	case *LiteralResultElement:
		return ec.execLiteralResultElement(ctx, v)
	case *MessageInst:
		return ec.execMessage(ctx, v)
	case *NumberInst:
		return ec.execNumber(ctx, v)
	case *SequenceInst:
		for _, child := range v.Body {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil
	case *XSLSequenceInst:
		return ec.execXSLSequence(ctx, v)
	case *PerformSortInst:
		return ec.execPerformSort(ctx, v)
	case *NextMatchInst:
		return ec.execNextMatch(ctx, v)
	case *ApplyImportsInst:
		return ec.execApplyImports(ctx, v)
	case *WherePopulatedInst:
		return ec.execWherePopulated(ctx, v)
	case *TryCatchInst:
		return ec.execTryCatch(ctx, v)
	case *ForEachGroupInst:
		return ec.execForEachGroup(ctx, v)
	case *NamespaceInst:
		return ec.execNamespace(ctx, v)
	default:
		return dynamicError(errCodeXTDE0820, "unsupported instruction type %T", inst)
	}
}

func (ec *execContext) execApplyTemplates(ctx context.Context, inst *ApplyTemplatesInst) error {
	var nodes []helium.Node

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		ns, ok := xpath3.NodesFrom(result.Sequence())
		if !ok {
			return dynamicError(errCodeXTDE0820, "apply-templates select must return nodes")
		}
		nodes = ns
	} else {
		nodes = selectDefaultNodes(ec.contextNode)
	}

	// Apply sort keys if present
	if len(inst.Sort) > 0 {
		var err error
		nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
		if err != nil {
			return err
		}
	}

	mode := inst.Mode
	if mode == "" || mode == "#current" {
		mode = ec.currentMode
	}

	// Process with-param values
	var paramValues map[string]xpath3.Sequence
	if len(inst.Params) > 0 {
		paramValues = make(map[string]xpath3.Sequence, len(inst.Params))
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			paramValues[wp.Name] = val
		}
	}

	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(nodes)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
	}()

	for i, node := range nodes {
		ec.position = i + 1

		// Push param values if any
		if len(paramValues) > 0 {
			ec.pushVarScope()
			for name, val := range paramValues {
				ec.setVar(name, val)
			}
		}

		if err := ec.applyTemplates(ctx, node, mode); err != nil {
			if len(paramValues) > 0 {
				ec.popVarScope()
			}
			return err
		}

		if len(paramValues) > 0 {
			ec.popVarScope()
		}
	}

	return nil
}

func (ec *execContext) evaluateWithParam(ctx context.Context, wp *WithParam) (xpath3.Sequence, error) {
	if wp.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := wp.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		return result.Sequence(), nil
	}
	if len(wp.Body) > 0 {
		return ec.evaluateBody(ctx, wp.Body)
	}
	return xpath3.EmptySequence(), nil
}

func (ec *execContext) execCallTemplate(ctx context.Context, inst *CallTemplateInst) error {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return dynamicError(errCodeXTDE0820, "recursion depth exceeded")
	}
	defer func() { ec.depth-- }()

	tmpl, ok := ec.stylesheet.namedTemplates[inst.Name]
	if !ok {
		return dynamicError(errCodeXTDE0060, "named template %q not found", inst.Name)
	}

	ec.pushVarScope()
	defer ec.popVarScope()

	// Set with-param values (override template's default param values)
	paramOverrides := make(map[string]xpath3.Sequence)
	for _, wp := range inst.Params {
		val, err := ec.evaluateWithParam(ctx, wp)
		if err != nil {
			return err
		}
		paramOverrides[wp.Name] = val
	}

	// Set template params with defaults, then override
	for _, p := range tmpl.Params {
		if val, ok := paramOverrides[p.Name]; ok {
			ec.setVar(p.Name, val)
			continue
		}
		if p.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := p.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			ec.setVar(p.Name, result.Sequence())
		} else if len(p.Body) > 0 {
			val, err := ec.evaluateBody(ctx, p.Body)
			if err != nil {
				return err
			}
			ec.setVar(p.Name, val)
		} else {
			ec.setVar(p.Name, xpath3.EmptySequence())
		}
	}

	// Execute template body
	for _, bodyInst := range tmpl.Body {
		if err := ec.executeInstruction(ctx, bodyInst); err != nil {
			return err
		}
	}

	return nil
}

func (ec *execContext) execValueOf(ctx context.Context, inst *ValueOfInst) error {
	var value string

	// Determine separator (default " " in XSLT 2.0+)
	separator := " "
	if inst.Separator != nil {
		var err error
		separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResultWithSep(result, separator)
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	if value == "" {
		return nil
	}
	text, err := ec.resultDoc.CreateText([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

func (ec *execContext) execText(inst *TextInst) error {
	if inst.Value == "" {
		return nil
	}
	text, err := ec.resultDoc.CreateText([]byte(inst.Value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

func (ec *execContext) execLiteralText(inst *LiteralTextInst) error {
	if inst.Value == "" {
		return nil
	}
	text, err := ec.resultDoc.CreateText([]byte(inst.Value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

func (ec *execContext) execElement(ctx context.Context, inst *ElementInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	// Extract local name for element creation so SetActiveNamespace doesn't double the prefix
	localName := name
	prefix := ""
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix = name[:idx]
		localName = name[idx+1:]
	}

	elem, err := ec.resultDoc.CreateElement(localName)
	if err != nil {
		return err
	}

	if inst.Namespace != nil {
		nsURI, err := inst.Namespace.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if nsURI != "" {
			if err := elem.DeclareNamespace(prefix, nsURI); err != nil {
				return err
			}
			if err := elem.SetActiveNamespace(prefix, nsURI); err != nil {
				return err
			}
		}
	} else if prefix != "" {
		// Resolve prefix from stylesheet namespaces
		if uri := ec.resolvePrefix(prefix); uri != "" {
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
			if err := elem.SetActiveNamespace(prefix, uri); err != nil {
				return err
			}
		}
	}

	if err := ec.addNode(elem); err != nil {
		return err
	}

	// Push new output context for children
	out := ec.currentOutput()
	savedCurrent := out.current
	out.current = elem
	defer func() { out.current = savedCurrent }()

	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}

	return nil
}

func (ec *execContext) execAttribute(ctx context.Context, inst *AttributeInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	// The current output node must be an element
	out := ec.currentOutput()
	elem, ok := out.current.(*helium.Element)
	if !ok {
		return dynamicError(errCodeXTDE0820, "xsl:attribute must be added to an element")
	}

	if inst.Namespace != nil {
		nsURI, err := inst.Namespace.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if nsURI != "" {
			prefix := ""
			localName := name
			if idx := strings.IndexByte(name, ':'); idx >= 0 {
				prefix = name[:idx]
				localName = name[idx+1:]
			}
			ns, err := ec.resultDoc.CreateNamespace(prefix, nsURI)
			if err != nil {
				return err
			}
			return elem.SetAttributeNS(localName, value, ns)
		}
	}

	// Handle prefixed attribute names without explicit namespace
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		localName := name[idx+1:]
		if uri := ec.resolvePrefix(prefix); uri != "" {
			ns, err := ec.resultDoc.CreateNamespace(prefix, uri)
			if err != nil {
				return err
			}
			return elem.SetAttributeNS(localName, value, ns)
		}
	}

	return elem.SetAttribute(name, value)
}

func (ec *execContext) execComment(ctx context.Context, inst *CommentInst) error {
	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	comment, err := ec.resultDoc.CreateComment([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(comment)
}

func (ec *execContext) execPI(ctx context.Context, inst *PIInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	pi, err := ec.resultDoc.CreatePI(name, value)
	if err != nil {
		return err
	}
	return ec.addNode(pi)
}

func (ec *execContext) execIf(ctx context.Context, inst *IfInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Test.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execChoose(ctx context.Context, inst *ChooseInst) error {
	for _, when := range inst.When {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := when.Test.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		b, err := xpath3.EBV(result.Sequence())
		if err != nil {
			return err
		}
		if b {
			for _, child := range when.Body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					return err
				}
			}
			return nil
		}
	}
	// otherwise
	for _, child := range inst.Otherwise {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execForEach(ctx context.Context, inst *ForEachInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	seq := result.Sequence()
	nodes, isNodes := xpath3.NodesFrom(seq)

	if isNodes && len(inst.Sort) > 0 {
		nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
		if err != nil {
			return err
		}
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
	}()

	if isNodes {
		ec.size = len(nodes)
		for i, node := range nodes {
			ec.position = i + 1
			ec.currentNode = node
			ec.contextNode = node

			ec.pushVarScope()
			for _, child := range inst.Body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					ec.popVarScope()
					return err
				}
			}
			ec.popVarScope()
		}
	} else {
		ec.size = len(seq)
		for i, item := range seq {
			ec.position = i + 1
			if ni, ok := item.(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
			}

			ec.pushVarScope()
			for _, child := range inst.Body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					ec.popVarScope()
					return err
				}
			}
			ec.popVarScope()
		}
	}

	return nil
}

func (ec *execContext) execVariable(ctx context.Context, inst *VariableInst) error {
	var val xpath3.Sequence

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		val = result.Sequence()
	} else if len(inst.Body) > 0 {
		var err error
		val, err = ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
	} else {
		val = xpath3.SingleString("")
	}

	ec.setVar(inst.Name, val)
	return nil
}

func (ec *execContext) execParam(ctx context.Context, inst *ParamInst) error {
	// Check if already set (by with-param)
	if _, ok := ec.localVars.lookup(inst.Name); ok {
		return nil
	}
	// Use default
	return ec.execVariable(ctx, &VariableInst{
		Name:   inst.Name,
		Select: inst.Select,
		Body:   inst.Body,
	})
}

func (ec *execContext) execCopy(ctx context.Context, inst *CopyInst) error {
	node := ec.contextNode
	if node == nil {
		return nil
	}

	switch node.Type() {
	case helium.ElementNode:
		srcElem := node.(*helium.Element)
		// Use LocalName to avoid prefix doubling with SetActiveNamespace
		elem, err := ec.resultDoc.CreateElement(srcElem.LocalName())
		if err != nil {
			return err
		}

		// Copy namespace declarations
		for _, ns := range srcElem.Namespaces() {
			if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return err
			}
		}
		if srcElem.URI() != "" {
			if err := elem.SetActiveNamespace(srcElem.Prefix(), srcElem.URI()); err != nil {
				return err
			}
		}

		if err := ec.addNode(elem); err != nil {
			return err
		}

		// Execute body in element context
		out := ec.currentOutput()
		savedCurrent := out.current
		out.current = elem
		defer func() { out.current = savedCurrent }()

		for _, child := range inst.Body {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil

	case helium.TextNode, helium.CDATASectionNode:
		text, err := ec.resultDoc.CreateText(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(text)

	case helium.CommentNode:
		comment, err := ec.resultDoc.CreateComment(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(comment)

	case helium.ProcessingInstructionNode:
		pi := node.(*helium.ProcessingInstruction)
		newPI, err := ec.resultDoc.CreatePI(pi.Name(), string(pi.Content()))
		if err != nil {
			return err
		}
		return ec.addNode(newPI)

	case helium.DocumentNode:
		// Copy document: just process body
		for _, child := range inst.Body {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil

	case helium.AttributeNode:
		attr := node.(*helium.Attribute)
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			return nil
		}
		return elem.SetAttribute(attr.Name(), attr.Value())
	}

	return nil
}

func (ec *execContext) execCopyOf(ctx context.Context, inst *CopyOfInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	for _, item := range result.Sequence() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			if err := ec.copyNodeToOutput(v.Node); err != nil {
				return err
			}
		case xpath3.AtomicValue:
			s, err := xpath3.AtomicToString(v)
			if err != nil {
				return err
			}
			text, err := ec.resultDoc.CreateText([]byte(s))
			if err != nil {
				return err
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyNodeToOutput copies a node to the current output, handling document
// and attribute nodes specially.
func (ec *execContext) copyNodeToOutput(node helium.Node) error {
	switch node.Type() {
	case helium.DocumentNode:
		// Copy children of the document node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.copyNodeToOutput(child); err != nil {
				return err
			}
		}
		return nil
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			return nil
		}
		return elem.SetAttribute(attr.Name(), attr.Value())
	default:
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			return err
		}
		return ec.addNode(copied)
	}
}

func (ec *execContext) execLiteralResultElement(ctx context.Context, inst *LiteralResultElement) error {
	// Use LocalName so that SetActiveNamespace doesn't double the prefix
	elemName := inst.LocalName
	if elemName == "" {
		elemName = inst.Name
	}
	elem, err := ec.resultDoc.CreateElement(elemName)
	if err != nil {
		return err
	}

	// Declare namespaces (skip if parent already has the same declaration)
	for prefix, uri := range inst.Namespaces {
		if !ec.isNSDeclaredInScope(prefix, uri) {
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
		}
	}

	// Set the element's own namespace
	if inst.Namespace != "" {
		if err := elem.SetActiveNamespace(inst.Prefix, inst.Namespace); err != nil {
			return err
		}
	}

	// Evaluate and set attributes
	for _, attr := range inst.Attrs {
		val, err := attr.Value.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if attr.Namespace != "" {
			ns, err := ec.resultDoc.CreateNamespace(attr.Prefix, attr.Namespace)
			if err != nil {
				return err
			}
			if err := elem.SetAttributeNS(attr.LocalName, val, ns); err != nil {
				return err
			}
		} else {
			if err := elem.SetAttribute(attr.Name, val); err != nil {
				return err
			}
		}
	}

	if err := ec.addNode(elem); err != nil {
		return err
	}

	// Execute body in element context
	out := ec.currentOutput()
	savedCurrent := out.current
	out.current = elem
	defer func() { out.current = savedCurrent }()

	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}

	return nil
}

// isNSDeclaredInScope checks if a namespace prefix→URI binding is already
// declared on an ancestor element in the current output tree.
func (ec *execContext) isNSDeclaredInScope(prefix, uri string) bool {
	out := ec.currentOutput()
	for node := out.current; node != nil; node = node.Parent() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == prefix && ns.URI() == uri {
				return true
			}
		}
	}
	return false
}

func (ec *execContext) execMessage(ctx context.Context, inst *MessageInst) error {
	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	terminate := false
	if inst.Terminate != nil {
		termStr, err := inst.Terminate.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		terminate = termStr == "yes"
	}

	if ec.msgHandler != nil {
		ec.msgHandler(value, terminate)
	}

	if terminate {
		return &XSLTError{
			Code:    errCodeXTDE0835,
			Message: value,
			Cause:   ErrTerminated,
		}
	}
	return nil
}

func (ec *execContext) execNumber(ctx context.Context, inst *NumberInst) error {
	node := ec.contextNode
	if node == nil {
		return nil
	}

	var nums []int

	if inst.Value != nil {
		// value attribute: evaluate expression and use result directly
		xpathCtx := ec.newXPathContext(node)
		result, err := inst.Value.Evaluate(xpathCtx, node)
		if err != nil {
			return err
		}
		if f, ok := result.IsNumber(); ok {
			nums = []int{int(math.Round(f))}
		}
	} else {
		switch inst.Level {
		case "single":
			nums = ec.numberSingle(inst, node)
		case "multiple":
			nums = ec.numberMultiple(inst, node)
		case "any":
			nums = ec.numberAny(inst, node)
		default:
			nums = ec.numberSingle(inst, node)
		}
	}

	// Format the number list
	format := "1"
	if inst.Format != nil {
		var err error
		format, err = inst.Format.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	groupSep := ""
	if inst.GroupingSeparator != nil {
		var err error
		groupSep, err = inst.GroupingSeparator.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}
	groupSize := 0
	if inst.GroupingSize != nil {
		gsStr, err := inst.GroupingSize.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		groupSize, _ = strconv.Atoi(gsStr)
	}

	text, err := ec.resultDoc.CreateText([]byte(formatNumberList(nums, format, groupSep, groupSize)))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

// numberNodeMatches tests if a node matches the count pattern.
// If no count pattern, matches nodes with the same type and name as the context node.
func (ec *execContext) numberNodeMatches(inst *NumberInst, target helium.Node, contextNode helium.Node) bool {
	if inst.Count != nil {
		return inst.Count.matchPattern(ec, target)
	}
	// Default: same node type and expanded name
	if target.Type() != contextNode.Type() {
		return false
	}
	if target.Type() == helium.ElementNode {
		te := target.(*helium.Element)
		ce := contextNode.(*helium.Element)
		return te.LocalName() == ce.LocalName() && te.URI() == ce.URI()
	}
	return target.Name() == contextNode.Name()
}

// numberFromMatches tests if a node matches the from pattern.
func (ec *execContext) numberFromMatches(inst *NumberInst, node helium.Node) bool {
	if inst.From == nil {
		return false
	}
	return inst.From.matchPattern(ec, node)
}

// numberSingle implements level="single": find the first ancestor-or-self that
// matches the count pattern, then count preceding siblings that match.
func (ec *execContext) numberSingle(inst *NumberInst, node helium.Node) []int {
	// Find the first ancestor-or-self that matches count
	target := ec.numberFindAncestorOrSelf(inst, node)
	if target == nil {
		return nil
	}

	// Count preceding siblings that match count pattern
	count := 1
	for sib := target.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if ec.numberNodeMatches(inst, sib, node) {
			count++
		}
	}
	return []int{count}
}

// numberMultiple implements level="multiple": find all ancestors-or-self that match
// count (stopping at from), and for each count preceding siblings.
func (ec *execContext) numberMultiple(inst *NumberInst, node helium.Node) []int {
	var ancestors []helium.Node
	for n := node; n != nil; n = n.Parent() {
		if ec.numberFromMatches(inst, n) {
			break
		}
		if ec.numberNodeMatches(inst, n, node) {
			ancestors = append(ancestors, n)
		}
		if n.Type() == helium.DocumentNode {
			break
		}
	}

	// Reverse to get document order (outermost first)
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	nums := make([]int, len(ancestors))
	for i, anc := range ancestors {
		count := 1
		for sib := anc.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
			if ec.numberNodeMatches(inst, sib, node) {
				count++
			}
		}
		nums[i] = count
	}
	return nums
}

// numberAny implements level="any": count all matching nodes in document order
// that precede (or are) the context node, going back to the nearest from match.
// The from node itself is included in the count if it matches count.
func (ec *execContext) numberAny(inst *NumberInst, node helium.Node) []int {
	count := 0
	cur := node
	for cur != nil {
		if ec.numberNodeMatches(inst, cur, ec.contextNode) {
			count++
		}
		if ec.numberFromMatches(inst, cur) {
			break
		}
		cur = ec.prevInDocOrder(cur)
	}
	if count == 0 {
		return nil
	}
	return []int{count}
}

// prevInDocOrder returns the previous node in document order.
func (ec *execContext) prevInDocOrder(node helium.Node) helium.Node {
	// Previous sibling's deepest last descendant
	if prev := node.PrevSibling(); prev != nil {
		return ec.lastDescendant(prev)
	}
	// Otherwise, parent
	parent := node.Parent()
	if parent == nil || parent.Type() == helium.DocumentNode {
		return nil
	}
	return parent
}

// lastDescendant returns the deepest last descendant of node (or node itself if leaf).
func (ec *execContext) lastDescendant(node helium.Node) helium.Node {
	if node.Type() == helium.ElementNode {
		elem := node.(*helium.Element)
		if last := elem.LastChild(); last != nil {
			return ec.lastDescendant(last)
		}
	}
	return node
}

// numberFindAncestorOrSelf finds the first ancestor-or-self that matches
// the count pattern. If from is specified, the matching ancestor must be a
// descendant of a from-matching node.
func (ec *execContext) numberFindAncestorOrSelf(inst *NumberInst, node helium.Node) helium.Node {
	for n := node; n != nil; n = n.Parent() {
		if n.Type() == helium.DocumentNode {
			return nil
		}
		if ec.numberNodeMatches(inst, n, node) {
			// If from is specified, verify there's a from-matching ancestor
			if inst.From != nil {
				if !ec.hasFromAncestor(inst, n) {
					return nil
				}
			}
			return n
		}
	}
	return nil
}

// hasFromAncestor checks if any ancestor of node matches the from pattern.
func (ec *execContext) hasFromAncestor(inst *NumberInst, node helium.Node) bool {
	for n := node.Parent(); n != nil; n = n.Parent() {
		if ec.numberFromMatches(inst, n) {
			return true
		}
	}
	// Also check preceding siblings of ancestors (from can match a sibling, not just ancestor)
	for n := node.PrevSibling(); n != nil; n = n.PrevSibling() {
		if ec.numberFromMatches(inst, n) {
			return true
		}
	}
	return false
}

// formatNumberList formats a list of numbers according to an XSLT format string.
// The format string is parsed into prefix, (format-token, separator)* pairs, and suffix.
func formatNumberList(nums []int, format string, groupSep string, groupSize int) string {
	if len(nums) == 0 {
		return ""
	}

	// Parse format string into tokens
	type fmtToken struct {
		format    string // e.g. "1", "a", "A", "i", "I"
		separator string // separator BEFORE this format token
	}

	runes := []rune(format)
	var prefix, suffix string
	var tokens []fmtToken

	i := 0
	// Extract prefix (leading non-alphanumeric)
	for i < len(runes) && !isAlphanumeric(runes[i]) {
		i++
	}
	prefix = string(runes[:i])

	for i < len(runes) {
		// Read format token (alphanumeric sequence)
		start := i
		for i < len(runes) && isAlphanumeric(runes[i]) {
			i++
		}
		if start == i {
			break
		}
		fmtStr := string(runes[start:i])

		// Read separator (non-alphanumeric sequence)
		sepStart := i
		for i < len(runes) && !isAlphanumeric(runes[i]) {
			i++
		}
		sep := string(runes[sepStart:i])

		// If no more format tokens follow, this separator is suffix
		if i >= len(runes) {
			tokens = append(tokens, fmtToken{format: fmtStr})
			suffix = sep
		} else {
			tokens = append(tokens, fmtToken{format: fmtStr, separator: sep})
		}
	}

	if len(tokens) == 0 {
		tokens = []fmtToken{{format: "1"}}
	}

	// Default separator between levels is "."
	defaultSep := "."
	if len(tokens) > 1 {
		defaultSep = tokens[0].separator
		if defaultSep == "" {
			defaultSep = "."
		}
	}

	var buf strings.Builder
	buf.WriteString(prefix)
	for idx, num := range nums {
		if idx > 0 {
			// Use the separator from the token, or default
			if idx < len(tokens) && tokens[idx-1].separator != "" {
				buf.WriteString(tokens[idx-1].separator)
			} else {
				buf.WriteString(defaultSep)
			}
		}
		// Pick format token: if more numbers than tokens, use the last token
		tokIdx := idx
		if tokIdx >= len(tokens) {
			tokIdx = len(tokens) - 1
		}
		buf.WriteString(formatSingleNumber(num, tokens[tokIdx].format, groupSep, groupSize))
	}
	buf.WriteString(suffix)
	return buf.String()
}

func isAlphanumeric(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func formatSingleNumber(num int, token string, groupSep string, groupSize int) string {
	switch {
	case token == "a":
		return toLowerAlpha(num)
	case token == "A":
		return toUpperAlpha(num)
	case token == "i":
		return strings.ToLower(toRoman(num))
	case token == "I":
		return toRoman(num)
	default:
		// Numeric format: determine minimum width from token (e.g., "001" = width 3)
		minWidth := len(token)
		s := strconv.Itoa(num)
		// Pad with leading zeros to meet minimum width
		for len(s) < minWidth {
			s = "0" + s
		}
		// Apply grouping separator
		if groupSep != "" && groupSize > 0 {
			s = applyGroupingSeparator(s, groupSep, groupSize)
		}
		return s
	}
}

func applyGroupingSeparator(s string, sep string, size int) string {
	// Insert separator from right to left every 'size' digits
	if size <= 0 || sep == "" {
		return s
	}
	var result []byte
	for i, j := len(s)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j > 0 && j%size == 0 {
			result = append(result, []byte(sep)...)
		}
		result = append(result, s[i])
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func toLowerAlpha(n int) string {
	if n <= 0 {
		return strconv.Itoa(n)
	}
	var buf []byte
	for n > 0 {
		n--
		buf = append([]byte{byte('a' + n%26)}, buf...)
		n /= 26
	}
	return string(buf)
}

func toUpperAlpha(n int) string {
	return strings.ToUpper(toLowerAlpha(n))
}

func toRoman(n int) string {
	if n <= 0 || n >= 4000 {
		return strconv.Itoa(n)
	}
	vals := []struct {
		val int
		sym string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var buf strings.Builder
	for _, v := range vals {
		for n >= v.val {
			buf.WriteString(v.sym)
			n -= v.val
		}
	}
	return buf.String()
}

func (ec *execContext) execXSLSequence(ctx context.Context, inst *XSLSequenceInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	for _, item := range result.Sequence() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			copied, copyErr := helium.CopyNode(v.Node, ec.resultDoc)
			if copyErr != nil {
				return copyErr
			}
			if err := ec.addNode(copied); err != nil {
				return err
			}
		case xpath3.AtomicValue:
			s, sErr := xpath3.AtomicToString(v)
			if sErr != nil {
				return sErr
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ec *execContext) execPerformSort(ctx context.Context, inst *PerformSortInst) error {
	var nodes []helium.Node

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		ns, ok := xpath3.NodesFrom(result.Sequence())
		if !ok {
			return nil
		}
		nodes = ns
	} else {
		nodes = selectDefaultNodes(ec.contextNode)
	}

	if len(inst.Sort) > 0 {
		var err error
		nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
		if err != nil {
			return err
		}
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(nodes)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
	}()

	if len(inst.Body) == 0 {
		// No body: copy sorted nodes to output (xsl:perform-sort as sequence constructor)
		for _, node := range nodes {
			if err := ec.copyNodeToOutput(node); err != nil {
				return err
			}
		}
	} else {
		for i, node := range nodes {
			ec.position = i + 1
			ec.currentNode = node
			ec.contextNode = node
			ec.pushVarScope()
			for _, child := range inst.Body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					ec.popVarScope()
					return err
				}
			}
			ec.popVarScope()
		}
	}
	return nil
}

func (ec *execContext) execNextMatch(ctx context.Context, inst *NextMatchInst) error {
	// xsl:next-match: find the next matching template after the current one
	node := ec.currentNode
	mode := ec.currentMode

	templates := ec.stylesheet.modeTemplates[mode]
	foundCurrent := false
	for _, tmpl := range templates {
		if tmpl == ec.currentTemplate {
			foundCurrent = true
			continue
		}
		if foundCurrent && tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return ec.executeTemplate(ctx, tmpl, node, mode)
		}
	}

	// No next match found — apply built-in rules
	return ec.applyBuiltinRules(ctx, node, mode)
}

func (ec *execContext) execApplyImports(ctx context.Context, inst *ApplyImportsInst) error {
	// xsl:apply-imports: apply templates from imported stylesheets
	// Simplified: same as built-in rules for now
	return ec.applyBuiltinRules(ctx, ec.currentNode, ec.currentMode)
}

func (ec *execContext) execWherePopulated(ctx context.Context, inst *WherePopulatedInst) error {
	// Execute body into a temporary document, only add to output if non-empty
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_tmp")
	if err != nil {
		return err
	}
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return err
	}

	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpRoot})

	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			return err
		}
	}

	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Check if anything was produced
	if tmpRoot.FirstChild() == nil {
		return nil
	}

	// Copy produced nodes to real output
	for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
		copied, copyErr := helium.CopyNode(child, ec.resultDoc)
		if copyErr != nil {
			return copyErr
		}
		if err := ec.addNode(copied); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execTryCatch(ctx context.Context, inst *TryCatchInst) error {
	// Execute try body; if it fails, execute catch
	tryErr := func() error {
		for _, child := range inst.Try {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil
	}()

	if tryErr == nil {
		return nil
	}

	// Execute catch body
	for _, child := range inst.Catch {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execForEachGroup(ctx context.Context, inst *ForEachGroupInst) error {
	// Basic stub: just iterate without actual grouping
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	nodes, isNodes := xpath3.NodesFrom(result.Sequence())
	if !isNodes {
		return nil
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(nodes)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
	}()

	for i, node := range nodes {
		ec.position = i + 1
		ec.currentNode = node
		ec.contextNode = node
		ec.pushVarScope()
		for _, child := range inst.Body {
			if childErr := ec.executeInstruction(ctx, child); childErr != nil {
				ec.popVarScope()
				return childErr
			}
		}
		ec.popVarScope()
	}
	return nil
}

func (ec *execContext) execNamespace(ctx context.Context, inst *NamespaceInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, evalErr := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if evalErr != nil {
			return evalErr
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		val, bodyErr := ec.evaluateBody(ctx, inst.Body)
		if bodyErr != nil {
			return bodyErr
		}
		value = stringifySequence(val)
	}

	out := ec.currentOutput()
	elem, ok := out.current.(*helium.Element)
	if !ok {
		return nil
	}
	return elem.DeclareNamespace(name, value)
}
