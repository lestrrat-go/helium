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
	// Apply per-instruction xpath-default-namespace if set
	if h, ok := inst.(interface{ xpathNSIsSet() bool }); ok && h.xpathNSIsSet() {
		savedNS := ec.xpathDefaultNS
		savedHas := ec.hasXPathDefaultNS
		ec.xpathDefaultNS = inst.(xpathNSHolder).getXPathDefaultNS()
		ec.hasXPathDefaultNS = true
		defer func() {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
		}()
	}
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
	case *OnEmptyInst:
		return ec.execOnEmpty(ctx, v)
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

	var atomicItems xpath3.Sequence // XSLT 3.0: atomic values from select
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		ns, ok := xpath3.NodesFrom(seq)
		if ok {
			nodes = ns
		} else {
			// XSLT 3.0: separate nodes from atomic values
			for _, item := range seq {
				if ni, ok := item.(xpath3.NodeItem); ok {
					nodes = append(nodes, ni.Node)
				} else {
					atomicItems = append(atomicItems, item)
				}
			}
		}
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

	// Process with-param values, separating tunnel from regular params
	var paramValues map[string]xpath3.Sequence
	var newTunnelParams map[string]xpath3.Sequence
	if len(inst.Params) > 0 {
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				if newTunnelParams == nil {
					newTunnelParams = make(map[string]xpath3.Sequence)
				}
				newTunnelParams[wp.Name] = val
			} else {
				if paramValues == nil {
					paramValues = make(map[string]xpath3.Sequence)
				}
				if _, dup := paramValues[wp.Name]; dup {
					return dynamicError(errCodeXTDE0410, "duplicate parameter %q in xsl:apply-templates", wp.Name)
				}
				paramValues[wp.Name] = val
			}
		}
	}

	// Merge new tunnel params with existing tunnel params (new values override)
	savedTunnel := ec.tunnelParams
	if newTunnelParams != nil {
		merged := make(map[string]xpath3.Sequence)
		for k, v := range ec.tunnelParams {
			merged[k] = v
		}
		for k, v := range newTunnelParams {
			merged[k] = v
		}
		ec.tunnelParams = merged
	}

	// Filter whitespace-only text nodes per xsl:strip-space before
	// setting position/size, so position()/last() reflect the filtered list.
	filtered := nodes[:0]
	for _, node := range nodes {
		if !ec.shouldStripWhitespace(node) {
			filtered = append(filtered, node)
		}
	}
	nodes = filtered

	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(nodes)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
	}()

	for i, node := range nodes {
		ec.position = i + 1

		if err := ec.applyTemplates(ctx, node, mode, paramValues); err != nil {
			return err
		}
	}

	// XSLT 3.0: process atomic values — try template matching first,
	// then fall back to built-in text output
	for _, item := range atomicItems {
		if tmpl := ec.findAtomicTemplate(item, mode); tmpl != nil {
			if err := ec.executeAtomicTemplate(ctx, tmpl, item, mode); err != nil {
				return err
			}
			continue
		}
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			continue
		}
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		if err := ec.addNode(text); err != nil {
			return err
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
		// Per XSLT spec: with-param body without select produces a
		// document node (temporary tree), same as variables.
		return ec.evaluateBodyAsDocument(ctx, wp.Body)
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

	// Separate tunnel from regular with-param values.
	// call-template forwards tunnel params from the caller's context.
	paramOverrides := make(map[string]xpath3.Sequence)
	savedTunnel := ec.tunnelParams
	hasTunnelOverrides := false
	for _, wp := range inst.Params {
		val, err := ec.evaluateWithParam(ctx, wp)
		if err != nil {
			return err
		}
		if wp.Tunnel {
			if !hasTunnelOverrides {
				merged := make(map[string]xpath3.Sequence)
				for k, v := range ec.tunnelParams {
					merged[k] = v
				}
				ec.tunnelParams = merged
				hasTunnelOverrides = true
			}
			ec.tunnelParams[wp.Name] = val
		} else {
			if _, dup := paramOverrides[wp.Name]; dup {
				return dynamicError(errCodeXTDE0410, "duplicate parameter %q in xsl:call-template", wp.Name)
			}
			paramOverrides[wp.Name] = val
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	// Set template params with defaults, overrides, or tunnel values
	for _, p := range tmpl.Params {
		if p.Tunnel {
			// Tunnel param: receive from tunnel context
			if ec.tunnelParams != nil {
				if val, ok := ec.tunnelParams[p.Name]; ok {
					ec.setVar(p.Name, val)
					continue
				}
			}
		} else if val, ok := paramOverrides[p.Name]; ok {
			ec.setVar(p.Name, val)
			continue
		}
		// Use default value
		if p.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := p.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			ec.setVar(p.Name, result.Sequence())
		} else if len(p.Body) > 0 {
			// Per XSLT spec: param body without select produces a
			// document node (temporary tree).
			val, err := ec.evaluateBodyAsDocument(ctx, p.Body)
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

	if inst.Select != nil {
		// Default separator for select is " "
		separator := " "
		if inst.Separator != nil {
			var err error
			separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResultWithSep(result, separator)
	} else if len(inst.Body) > 0 {
		// Default separator for body content is "" (zero-length string)
		separator := ""
		if inst.Separator != nil {
			var err error
			separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequenceWithSep(val, separator)
	}
	// XSLT 3.0: xsl:value-of always produces a text node, even if empty.
	// Skip only when select evaluates to empty sequence.
	if value == "" && inst.Select != nil {
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
	value := inst.Value
	if inst.TVT != nil {
		// Text value template: evaluate like an AVT
		ctx := ec.transformCtx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = withExecContext(ctx, ec)
		var err error
		value, err = inst.TVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
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

	hasNS := false
	if inst.Namespace != nil {
		nsURI, err := inst.Namespace.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if nsURI != "" {
			hasNS = true
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
			hasNS = true
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
			if err := elem.SetActiveNamespace(prefix, uri); err != nil {
				return err
			}
		}
	}

	// If this element has no namespace but there's a default namespace in scope,
	// we need to undeclare it with xmlns=""
	if !hasNS && prefix == "" && ec.hasDefaultNSInScope() {
		if err := elem.DeclareNamespace("", ""); err != nil {
			return err
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
		sep := " "
		if inst.Separator != nil {
			sep, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResultWithSep(result, sep)
	} else if len(inst.Body) > 0 {
		sep := ""
		if inst.Separator != nil {
			sep, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		value = stringifySequenceWithSep(val, sep)
	}

	// The current output node must be an element
	out := ec.currentOutput()
	elem, ok := out.current.(*helium.Element)
	if !ok {
		return dynamicError(errCodeXTDE0820, "xsl:attribute must be added to an element")
	}

	// XTRE0540: cannot add attribute after child content has been added
	if elem.FirstChild() != nil {
		return dynamicError(errCodeXTRE0540, "cannot add attribute to element after children have been added")
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
			// Attributes in a namespace require a non-empty prefix (unlike
			// elements, the default namespace does not apply to attributes).
			if prefix == "" {
				prefix = "ns0"
			}
			// If the prefix is already bound to a different URI on this element,
			// generate a unique prefix to avoid conflicts.
			prefix = uniqueNSPrefix(elem, prefix, nsURI)
			// Ensure the namespace is declared on the element
			if !hasNSDecl(elem, prefix, nsURI) {
				if err := elem.DeclareNamespace(prefix, nsURI); err != nil {
					return err
				}
			}
			// Remove existing attribute with same expanded name to allow replacement
			elem.RemoveAttributeNS(localName, nsURI)
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
			// Remove existing attribute with same expanded name to allow replacement
			elem.RemoveAttributeNS(localName, uri)
			ns, err := ec.resultDoc.CreateNamespace(prefix, uri)
			if err != nil {
				return err
			}
			return elem.SetAttributeNS(localName, value, ns)
		}
	}

	// Remove existing attribute with same name to allow replacement
	elem.RemoveAttribute(name)
	return elem.SetAttribute(name, value)
}

// copyAttributeToElement copies an attribute to an element, preserving its
// namespace URI and prefix. For non-namespaced attributes, falls back to
// SetAttribute.
func copyAttributeToElement(elem *helium.Element, attr *helium.Attribute) error {
	if uri := attr.URI(); uri != "" {
		prefix := attr.Prefix()
		// Extract local name by stripping prefix from Name()
		name := attr.Name()
		localName := name
		if prefix != "" {
			localName = name[len(prefix)+1:]
		}
		ns := helium.NewNamespace(prefix, uri)
		return elem.SetAttributeNS(localName, attr.Value(), ns)
	}
	return elem.SetAttribute(attr.Name(), attr.Value())
}

// hasNSDecl checks if an element already has a namespace declaration for
// the given prefix and URI.
func hasNSDecl(elem *helium.Element, prefix, uri string) bool {
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == prefix && ns.URI() == uri {
			return true
		}
	}
	return false
}

// uniqueNSPrefix returns a prefix for nsURI that doesn't conflict with
// in-scope namespace declarations on elem or its ancestors. If prefix is
// already bound to nsURI, it's returned as-is. If it's bound to a different
// URI, a suffix like _1, _2, ... is appended until a unique prefix is found.
func uniqueNSPrefix(elem *helium.Element, prefix, nsURI string) string {
	if prefixBoundTo(elem, prefix) == nsURI {
		return prefix
	}
	if uri := prefixBoundTo(elem, prefix); uri != "" && uri != nsURI {
		for i := 1; ; i++ {
			candidate := prefix + "_" + strconv.Itoa(i)
			if prefixBoundTo(elem, candidate) == "" {
				return candidate
			}
		}
	}
	return prefix
}

// prefixBoundTo walks the element and its ancestors to find what URI
// a prefix is bound to. Returns "" if not found.
func prefixBoundTo(elem *helium.Element, prefix string) string {
	for node := helium.Node(elem); node != nil; node = node.Parent() {
		e, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		for _, ns := range e.Namespaces() {
			if ns.Prefix() == prefix {
				return ns.URI()
			}
		}
	}
	return ""
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

	// Sanitize comment content per XSLT 3.0 spec §11.1:
	// Replace any occurrence of "--" with "- -" and ensure the value
	// doesn't end with "-" (add a trailing space if so).
	value = sanitizeComment(value)

	comment, err := ec.resultDoc.CreateComment([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(comment)
}

// sanitizeComment replaces "--" sequences with "- -" and ensures the
// value does not end with "-", per XSLT comment construction rules.
func sanitizeComment(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	prevDash := false
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			if prevDash {
				sb.WriteByte(' ')
			}
			sb.WriteByte('-')
			prevDash = true
		} else {
			sb.WriteByte(s[i])
			prevDash = false
		}
	}
	result := sb.String()
	if len(result) > 0 && result[len(result)-1] == '-' {
		result += " "
	}
	return result
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
		// Apply per-when xpath-default-namespace
		savedNS := ec.xpathDefaultNS
		savedHas := ec.hasXPathDefaultNS
		if when.HasXPathDefaultNS {
			ec.xpathDefaultNS = when.XPathDefaultNS
			ec.hasXPathDefaultNS = true
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := when.Test.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return err
		}
		b, err := xpath3.EBV(result.Sequence())
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return err
		}
		if b {
			for _, child := range when.Body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					ec.xpathDefaultNS = savedNS
					ec.hasXPathDefaultNS = savedHas
					return err
				}
			}
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return nil
		}
		ec.xpathDefaultNS = savedNS
		ec.hasXPathDefaultNS = savedHas
	}
	// otherwise
	savedNS := ec.xpathDefaultNS
	savedHas := ec.hasXPathDefaultNS
	if inst.HasOtherwiseXPNS {
		ec.xpathDefaultNS = inst.OtherwiseXPNS
		ec.hasXPathDefaultNS = true
	}
	for _, child := range inst.Otherwise {
		if err := ec.executeInstruction(ctx, child); err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return err
		}
	}
	ec.xpathDefaultNS = savedNS
	ec.hasXPathDefaultNS = savedHas
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

	if len(inst.Sort) > 0 {
		if isNodes {
			nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
			if err != nil {
				return err
			}
		} else {
			seq, err = sortItems(ctx, ec, seq, inst.Sort)
			if err != nil {
				return err
			}
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
		savedItem := ec.contextItem
		defer func() { ec.contextItem = savedItem }()
		for i, item := range seq {
			ec.position = i + 1
			if ni, ok := item.(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
				ec.contextItem = nil
			} else {
				ec.contextItem = item
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
		if inst.As == "" {
			// Per XSLT spec: variable with content body (no select, no as)
			// produces a document node (temporary tree)
			var err error
			val, err = ec.evaluateBodyAsDocument(ctx, inst.Body)
			if err != nil {
				return err
			}
		} else {
			// With as attribute: evaluate as raw sequence
			var err error
			val, err = ec.evaluateBody(ctx, inst.Body)
			if err != nil {
				return err
			}
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
	if inst.Select != nil {
		// XSLT 3.0: xsl:copy with select — iterate over selected items
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if len(seq) == 0 {
			return nil // empty sequence: skip body
		}
		for _, item := range seq {
			switch v := item.(type) {
			case xpath3.NodeItem:
				if err := ec.execCopyNode(ctx, v.Node, inst.Body); err != nil {
					return err
				}
			case xpath3.AtomicValue:
				// Atomic values: output as text, body is not evaluated
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

	return ec.execCopyNode(ctx, ec.contextNode, inst.Body)
}

func (ec *execContext) execCopyNode(ctx context.Context, node helium.Node, body []Instruction) error {
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

		for _, child := range body {
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
		for _, child := range body {
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
		return copyAttributeToElement(elem, attr)
	}

	return nil
}

func (ec *execContext) execCopyOf(ctx context.Context, inst *CopyOfInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	prevWasAtomic := false
	for _, item := range result.Sequence() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if err := ec.copyNodeToOutput(v.Node); err != nil {
				return err
			}
		case xpath3.AtomicValue:
			s, err := xpath3.AtomicToString(v)
			if err != nil {
				return err
			}
			if prevWasAtomic {
				sep, tErr := ec.resultDoc.CreateText([]byte(" "))
				if tErr != nil {
					return tErr
				}
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text, err := ec.resultDoc.CreateText([]byte(s))
			if err != nil {
				return err
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
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
		return copyAttributeToElement(elem, attr)
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
		// Ensure the namespace declaration is present for serialization.
		// SetActiveNamespace only sets n.ns; we also need it in nsDefs.
		if !hasNSDecl(elem, inst.Prefix, inst.Namespace) && !ec.isNSDeclaredInScope(inst.Prefix, inst.Namespace) {
			if err := elem.DeclareNamespace(inst.Prefix, inst.Namespace); err != nil {
				return err
			}
		}
	} else if inst.Prefix == "" && ec.hasDefaultNSInScope() {
		// No namespace on this LRE but default namespace in scope — undeclare it
		if err := elem.DeclareNamespace("", ""); err != nil {
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
			// Ensure the attribute's namespace is declared on the element
			if !hasNSDecl(elem, attr.Prefix, attr.Namespace) && !ec.isNSDeclaredInScope(attr.Prefix, attr.Namespace) {
				if err := elem.DeclareNamespace(attr.Prefix, attr.Namespace); err != nil {
					return err
				}
			}
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

	// Execute body in element context with a new variable scope
	out := ec.currentOutput()
	savedCurrent := out.current
	out.current = elem
	ec.pushVarScope()
	defer func() {
		ec.popVarScope()
		out.current = savedCurrent
	}()

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

// hasDefaultNSInScope returns true if there is a default namespace (xmlns="...")
// with a non-empty URI declared on any ancestor in the result tree.
func (ec *execContext) hasDefaultNSInScope() bool {
	out := ec.currentOutput()
	for node := out.current; node != nil; node = node.Parent() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		// Check the element's own namespace (if default, i.e. no prefix)
		if elem.Prefix() == "" && elem.URI() != "" {
			return true
		}
		// Check namespace declarations
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == "" && ns.URI() != "" {
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
		termStr = strings.TrimSpace(termStr)
		terminate = termStr == "yes" || termStr == "true" || termStr == "1"
	}

	if ec.msgHandler != nil {
		ec.msgHandler(value, terminate)
	}

	if terminate {
		errorCode := "XTMM9000"
		if inst.ErrorCode != nil {
			code, err := inst.ErrorCode.evaluate(ctx, ec.contextNode)
			if err == nil && code != "" {
				errorCode = code
			}
		}
		return &XSLTError{
			Code:    errorCode,
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

	// XSLT 3.0: select attribute specifies which node to number
	if inst.Select != nil && inst.Value == nil {
		xpathCtx := ec.newXPathContext(node)
		result, err := inst.Select.Evaluate(xpathCtx, node)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if len(seq) > 0 {
			if ni, ok := seq[0].(xpath3.NodeItem); ok {
				node = ni.Node
			}
		}
	}

	var nums []int

	if inst.Value != nil {
		// value attribute: evaluate expression and use result directly
		xpathCtx := ec.newXPathContext(node)
		result, err := inst.Value.Evaluate(xpathCtx, node)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		for _, item := range seq {
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				continue
			}
			dv, err := xpath3.CastAtomic(av, xpath3.TypeDouble)
			if err != nil {
				continue
			}
			nums = append(nums, int(math.Round(dv.DoubleVal())))
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

	// Apply start-at offset (XSLT 3.0): default is 1, so start-at="0" subtracts 1
	if inst.StartAt != nil {
		saStr, err := inst.StartAt.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		// start-at can be a space-separated list of integers, one per level
		saParts := strings.Fields(saStr)
		for i, n := range nums {
			offset := 0
			if i < len(saParts) {
				offset, _ = strconv.Atoi(saParts[i])
			} else if len(saParts) > 0 {
				offset, _ = strconv.Atoi(saParts[len(saParts)-1])
			}
			// start-at shifts numbering: number = number + startAt - 1
			nums[i] = n + offset - 1
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

// hasFromAncestor checks if the node or any ancestor matches the from pattern.
func (ec *execContext) hasFromAncestor(inst *NumberInst, node helium.Node) bool {
	// Check the node itself and all ancestors
	for n := node; n != nil; n = n.Parent() {
		if ec.numberFromMatches(inst, n) {
			return true
		}
	}
	// Also check preceding siblings of the node (from can match a sibling)
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
	switch token {
	case "a":
		return toLowerAlpha(num)
	case "A":
		return toUpperAlpha(num)
	case "i":
		return strings.ToLower(toRoman(num))
	case "I":
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

	out := ec.currentOutput()

	// In capture mode, accumulate items directly instead of writing to DOM
	if out.captureItems {
		out.pendingItems = append(out.pendingItems, result.Sequence()...)
		return nil
	}

	prevWasAtomic := false
	for _, item := range result.Sequence() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if v.Node.Type() == helium.AttributeNode {
				// Attribute nodes: add as attribute to current element
				attr := v.Node.(*helium.Attribute)
				elem, ok := out.current.(*helium.Element)
				if ok {
					if err := copyAttributeToElement(elem, attr); err != nil {
						return err
					}
				}
				continue
			}
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
			// Insert space separator between consecutive atomic values
			if prevWasAtomic {
				sep, tErr := ec.resultDoc.CreateText([]byte(" "))
				if tErr != nil {
					return tErr
				}
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
		}
	}
	return nil
}

// outputSequence writes a sequence of items to the current output.
func (ec *execContext) outputSequence(seq xpath3.Sequence) error {
	prevWasAtomic := false
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
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
			if prevWasAtomic {
				sep, tErr := ec.resultDoc.CreateText([]byte(" "))
				if tErr != nil {
					return tErr
				}
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
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
	} else if len(inst.Body) > 0 {
		// Body acts as sequence constructor: evaluate to get the items to sort
		seq, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		ns, ok := xpath3.NodesFrom(seq)
		if !ok {
			return nil
		}
		nodes = ns
	} else {
		return nil
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

	// Output sorted nodes
	for _, node := range nodes {
		if err := ec.copyNodeToOutput(node); err != nil {
			return err
		}
	}

	return nil
}

func (ec *execContext) execNextMatch(ctx context.Context, inst *NextMatchInst) error {
	// xsl:next-match: find the next matching template after the current one
	node := ec.currentNode
	mode := ec.currentMode

	// Process with-param (tunnel and regular)
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				if ec.tunnelParams == nil {
					ec.tunnelParams = make(map[string]xpath3.Sequence)
				}
				ec.tunnelParams[wp.Name] = val
			} else {
				if pv == nil {
					pv = make(map[string]xpath3.Sequence)
				}
				pv[wp.Name] = val
			}
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	templates := ec.stylesheet.modeTemplates[mode]
	foundCurrent := false
	for _, tmpl := range templates {
		if tmpl == ec.currentTemplate {
			foundCurrent = true
			continue
		}
		if foundCurrent && tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return ec.executeTemplate(ctx, tmpl, node, mode, pv)
		}
	}

	// No next match found — apply built-in rules
	return ec.applyBuiltinRules(ctx, node, mode, pv)
}

func (ec *execContext) execApplyImports(ctx context.Context, inst *ApplyImportsInst) error {
	// xsl:apply-imports: find a matching template with lower import precedence
	// than the currently executing template.
	if ec.currentTemplate == nil {
		return nil
	}

	node := ec.currentNode
	mode := ec.currentMode
	maxPrec := ec.currentTemplate.ImportPrec

	// Process with-param (tunnel and regular)
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				if ec.tunnelParams == nil {
					ec.tunnelParams = make(map[string]xpath3.Sequence)
				}
				ec.tunnelParams[wp.Name] = val
			} else {
				if pv == nil {
					pv = make(map[string]xpath3.Sequence)
				}
				pv[wp.Name] = val
			}
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	templates := ec.stylesheet.modeTemplates[mode]
	for _, tmpl := range templates {
		if tmpl.ImportPrec >= maxPrec {
			continue
		}
		if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return ec.executeTemplate(ctx, tmpl, node, mode, pv)
		}
	}

	// No imported template found, use built-in rules
	return ec.applyBuiltinRules(ctx, node, mode, pv)
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

	// Check if anything "populated" was produced per XSLT 3.0 semantics.
	// An element is populated if it has at least one child element or non-empty text node.
	// Comments, PIs, attributes, and namespace nodes do not count.
	populated := false
	for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
		if isPopulated(child) {
			populated = true
			break
		}
	}
	if !populated {
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

// isPopulated checks if a node is "populated" per XSLT 3.0 xsl:where-populated semantics.
// An element is populated if it has at least one child element or non-empty text node.
// A text node is populated if its content has non-zero length.
// Comments, PIs, and other node types are not considered populated.
func isPopulated(node helium.Node) bool {
	switch node.Type() {
	case helium.ElementNode:
		elem := node.(*helium.Element)
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.Type() {
			case helium.ElementNode:
				return true
			case helium.TextNode:
				if len(child.Content()) > 0 {
					return true
				}
			}
		}
		return false
	case helium.TextNode:
		return len(node.Content()) > 0
	default:
		return false
	}
}

func (ec *execContext) execOnEmpty(ctx context.Context, inst *OnEmptyInst) error {
	// xsl:on-empty fires only if the current output container has no significant content.
	out := ec.currentOutput()
	current := out.current
	if current == nil {
		return nil
	}

	// Check if the current container has any populated children
	elem, ok := current.(*helium.Element)
	if !ok {
		return nil
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if isPopulated(child) {
			return nil // has significant content, on-empty does not fire
		}
	}

	// Container is empty — execute on-empty body
	if inst.Select != nil {
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
	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execTryCatch(ctx context.Context, inst *TryCatchInst) error {
	// Execute try body; if it fails, execute catch
	tryErr := func() error {
		if inst.Select != nil {
			// xsl:try select="..." — evaluate expression and output result
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			return ec.outputSequence(result.Sequence())
		}
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

	// Set XSLT 3.0 error variables in catch scope
	const errNS = "http://www.w3.org/2005/xqt-errors"
	ec.pushVarScope()
	defer ec.popVarScope()

	errCode := "XSLT0000"
	errDesc := tryErr.Error()
	if xErr, ok := tryErr.(*XSLTError); ok {
		errCode = xErr.Code
		errDesc = xErr.Message
	} else if xpErr, ok := tryErr.(*xpath3.XPathError); ok {
		errCode = xpErr.Code
		errDesc = xpErr.Message
	}
	ec.setVar("{"+errNS+"}code", xpath3.SingleString(errCode))
	ec.setVar("{"+errNS+"}description", xpath3.SingleString(errDesc))
	ec.setVar("{"+errNS+"}value", xpath3.EmptySequence())
	ec.setVar("{"+errNS+"}module", xpath3.SingleString(""))
	ec.setVar("{"+errNS+"}line-number", xpath3.SingleInteger(0))
	ec.setVar("{"+errNS+"}column-number", xpath3.SingleInteger(0))

	// Execute catch body
	if inst.CatchSelect != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.CatchSelect.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		return ec.outputSequence(result.Sequence())
	}
	for _, child := range inst.Catch {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) execForEachGroup(ctx context.Context, inst *ForEachGroupInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	seq := result.Sequence()

	// Build groups based on the grouping mode
	var groups []fegGroup

	switch {
	case inst.GroupBy != nil:
		groups, err = ec.groupBy(ctx, seq, inst.GroupBy)
	case inst.GroupAdjacent != nil:
		groups, err = ec.groupAdjacent(ctx, seq, inst.GroupAdjacent)
	case inst.GroupStartingWith != nil:
		groups = ec.groupStartingWith(seq, inst.GroupStartingWith)
	case inst.GroupEndingWith != nil:
		groups = ec.groupEndingWith(seq, inst.GroupEndingWith)
	default:
		// No grouping attribute — treat entire sequence as one group
		groups = []fegGroup{{items: seq}}
	}
	if err != nil {
		return err
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	savedGroup := ec.currentGroup
	savedGroupKey := ec.currentGroupKey
	ec.size = len(groups)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
	}()

	for i, g := range groups {
		ec.position = i + 1
		ec.currentGroup = g.items
		ec.currentGroupKey = g.key

		// Context item is the first item of the group
		if len(g.items) > 0 {
			if ni, ok := g.items[0].(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
			}
		}

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

type fegGroup struct {
	key   xpath3.Sequence
	items xpath3.Sequence
}

// groupBy implements group-by: items are grouped by the string value of the
// group-by expression evaluated with each item as context. When the group-by
// expression returns a sequence of multiple values, the item is added to
// a group for each value.
func (ec *execContext) groupBy(_ context.Context, seq xpath3.Sequence, groupByExpr *xpath3.Expression) ([]fegGroup, error) {
	type entry struct {
		key   string
		items xpath3.Sequence
	}
	var order []string
	groupMap := make(map[string]*entry)

	for _, item := range seq {
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
		}
		xpathCtx := ec.newXPathContext(node)
		result, err := groupByExpr.Evaluate(xpathCtx, node)
		if err != nil {
			return nil, err
		}

		// Each value in the result creates a separate group key.
		// An item may appear in multiple groups.
		resultSeq := result.Sequence()
		if len(resultSeq) == 0 {
			// No grouping key — item is not included in any group
			continue
		}
		for _, keyItem := range resultSeq {
			keyVal := stringifyItem(keyItem)
			if e, ok := groupMap[keyVal]; ok {
				e.items = append(e.items, item)
			} else {
				groupMap[keyVal] = &entry{key: keyVal, items: xpath3.Sequence{item}}
				order = append(order, keyVal)
			}
		}
	}

	groups := make([]fegGroup, len(order))
	for i, k := range order {
		e := groupMap[k]
		groups[i] = fegGroup{
			key:   xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: e.key}},
			items: e.items,
		}
	}
	return groups, nil
}

// groupAdjacent implements group-adjacent: consecutive items with equal
// grouping key values form a group.
func (ec *execContext) groupAdjacent(ctx context.Context, seq xpath3.Sequence, adjExpr *xpath3.Expression) ([]fegGroup, error) {
	var groups []fegGroup
	var currentKey string
	var currentItems xpath3.Sequence

	for _, item := range seq {
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
		}
		xpathCtx := ec.newXPathContext(node)
		result, err := adjExpr.Evaluate(xpathCtx, node)
		if err != nil {
			return nil, err
		}
		keyVal := stringifyResult(result)
		if keyVal == currentKey && len(currentItems) > 0 {
			currentItems = append(currentItems, item)
		} else {
			if len(currentItems) > 0 {
				groups = append(groups, fegGroup{
					key:   xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}},
					items: currentItems,
				})
			}
			currentKey = keyVal
			currentItems = xpath3.Sequence{item}
		}
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{
			key:   xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}},
			items: currentItems,
		})
	}
	return groups, nil
}

// groupStartingWith implements group-starting-with: a new group starts
// whenever an item matches the pattern.
func (ec *execContext) groupStartingWith(seq xpath3.Sequence, pat *Pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.Sequence
	for _, item := range seq {
		ni, isNode := item.(xpath3.NodeItem)
		startsGroup := isNode && pat.matchPattern(ec, ni.Node)
		if startsGroup && len(currentItems) > 0 {
			groups = append(groups, fegGroup{items: currentItems})
			currentItems = nil
		}
		currentItems = append(currentItems, item)
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{items: currentItems})
	}
	return groups
}

// groupEndingWith implements group-ending-with: a group ends whenever
// an item matches the pattern.
func (ec *execContext) groupEndingWith(seq xpath3.Sequence, pat *Pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.Sequence
	for _, item := range seq {
		currentItems = append(currentItems, item)
		ni, isNode := item.(xpath3.NodeItem)
		if isNode && pat.matchPattern(ec, ni.Node) {
			groups = append(groups, fegGroup{items: currentItems})
			currentItems = nil
		}
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{items: currentItems})
	}
	return groups
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
