package xslt3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

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
	case *ResultDocumentInst:
		return ec.execResultDocument(ctx, v)
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
	case *SourceDocumentInst:
		return ec.execSourceDocument(ctx, v)
	case *IterateInst:
		return ec.execIterate(ctx, v)
	case *ForkInst:
		return ec.execFork(ctx, v)
	case *BreakInst:
		return ec.execBreak(ctx, v)
	case *NextIterationInst:
		return ec.execNextIteration(ctx, v)
	case *MergeInst:
		return ec.execMerge(ctx, v)
	case *MapInst:
		return ec.execMap(ctx, v)
	case *MapEntryInst:
		return ec.execMapEntry(ctx, v)
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
		// Default select is child::node() which requires a node context item.
		// If the context item is an atomic value (contextNode is nil), raise XTTE0510.
		if ec.contextNode == nil {
			return dynamicError("XTTE0510", "apply-templates with default select requires a node context item")
		}
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
	if mode == "#current" {
		mode = ec.currentMode
	}
	// When mode is absent (empty), use the stylesheet's default-mode
	// (not the current mode — #current must be explicit)

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
	savedGroupKey := ec.currentGroupKey
	savedGroup := ec.currentGroup
	savedInGroupCtx := ec.inGroupContext
	ec.size = len(nodes)
	// Per XSLT 3.0: current-grouping-key() and current-group() are only
	// available in the body of xsl:for-each-group itself, not in templates
	// invoked by apply-templates within that body.
	ec.currentGroupKey = nil
	ec.currentGroup = nil
	ec.inGroupContext = false
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentGroupKey = savedGroupKey
		ec.currentGroup = savedGroup
		ec.inGroupContext = savedInGroupCtx
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
		if wp.As != "" {
			// With as attribute: evaluate as raw sequence (individual items)
			return ec.evaluateBody(ctx, wp.Body)
		}
		// No as: wrap in document node (temporary tree)
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
		if inst.HasSeparator {
			if inst.Separator != nil {
				var err error
				separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
				if err != nil {
					return err
				}
			} else {
				separator = ""
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
		if inst.HasSeparator && inst.Separator != nil {
			var err error
			separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		val, err := ec.evaluateBodySeparateText(ctx, inst.Body)
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
	value := inst.Value
	if inst.TVT != nil {
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

	if inst.TypeName != "" {
		ec.annotateNode(elem, inst.TypeName)
	}

	// Push new output context for children
	out := ec.currentOutput()
	savedCurrent := out.current
	out.current = elem
	defer func() { out.current = savedCurrent }()

	// Apply attribute sets (before body so body can override)
	if len(inst.UseAttributeSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttributeSets); err != nil {
			return err
		}
	}
	if len(inst.UseAttrSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttrSets); err != nil {
			return err
		}
	}

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
			if err := elem.SetAttributeNS(localName, value, ns); err != nil {
				return err
			}
			ec.annotateAttr(elem, inst.TypeName, localName, nsURI, value)
			return nil
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
			if err := elem.SetAttributeNS(localName, value, ns); err != nil {
				return err
			}
			ec.annotateAttr(elem, inst.TypeName, localName, uri, value)
			return nil
		}
	}

	// Remove existing attribute with same name to allow replacement
	elem.RemoveAttribute(name)
	if err := elem.SetAttribute(name, value); err != nil {
		return err
	}
	ec.annotateAttr(elem, inst.TypeName, name, "", value)
	return nil
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
				ec.contextNode = nil
				ec.currentNode = nil
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
				if err := ec.execCopyNode(ctx, v.Node, copyNodeOpts{
					body:           inst.Body,
					copyNamespaces: inst.CopyNamespaces,
				}); err != nil {
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

	if ec.contextNode == nil {
		return dynamicError(errCodeXTTE0945, "xsl:copy: no context item")
	}
	return ec.execCopyNode(ctx, ec.contextNode, copyNodeOpts{
		body:           inst.Body,
		useAttrSets:    inst.UseAttrSets,
		copyNamespaces: inst.CopyNamespaces,
	})
}

// effectiveValidation returns the validation mode for a copy/copy-of instruction,
// falling back to the stylesheet default when the instruction has none.
func (ec *execContext) effectiveValidation(instValidation string) string {
	if instValidation != "" {
		return instValidation
	}
	return ec.stylesheet.defaultValidation
}

type copyNodeOpts struct {
	body           []Instruction
	useAttrSets    []string
	copyNamespaces bool
}

func (ec *execContext) execCopyNode(ctx context.Context, node helium.Node, opts copyNodeOpts) error {
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

		if opts.copyNamespaces {
			// Copy all namespace declarations
			for _, ns := range srcElem.Namespaces() {
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return err
				}
			}
		}
		if srcElem.URI() != "" {
			// Always declare the element's own namespace
			if !hasNSDecl(elem, srcElem.Prefix(), srcElem.URI()) {
				if err := elem.DeclareNamespace(srcElem.Prefix(), srcElem.URI()); err != nil {
					return err
				}
			}
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

		// Apply attribute sets if specified
		if len(opts.useAttrSets) > 0 {
			if err := ec.applyAttributeSets(ctx, opts.useAttrSets); err != nil {
				return err
			}
		}

		for _, child := range opts.body {
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
		// xsl:copy of a document node creates a new document node.
		// DTD information (including unparsed entities) is preserved.
		srcDoc, _ := node.(*helium.Document)
		newDoc := helium.NewDefaultDocument()
		if srcDoc != nil {
			// Copy DTD information to preserve unparsed entities.
			helium.CopyDTDInfo(srcDoc, newDoc)
			newDoc.SetURL(srcDoc.URL())
		}

		out := ec.currentOutput()
		if out.captureItems {
			// We're inside a variable or function body — capture the
			// document node as an item.
			savedDoc := ec.resultDoc
			savedOutput := out.current
			ec.resultDoc = newDoc
			docRoot := newDoc.DocumentElement()
			if docRoot == nil {
				// No doc element yet; use the document node itself as output target.
				out.current = newDoc
			}
			for _, child := range opts.body {
				if err := ec.executeInstruction(ctx, child); err != nil {
					ec.resultDoc = savedDoc
					out.current = savedOutput
					return err
				}
			}
			ec.resultDoc = savedDoc
			out.current = savedOutput
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: newDoc})
			return nil
		}

		// Not in capture mode — process body in current context.
		for _, child := range opts.body {
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
			// XTDE0410: adding attribute to non-element
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s to a non-element node", attr.Name())
		}
		if elem.FirstChild() != nil {
			// XTDE0410: adding attribute after child content
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s after child nodes have been added", attr.Name())
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

	preserve := ec.effectiveValidation(inst.Validation) == "preserve"

	prevWasAtomic := false
	for _, item := range result.Sequence() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if err := ec.copyNodeToOutput(v.Node, inst.CopyNamespaces); err != nil {
				return err
			}
			if preserve {
				ec.transferAnnotations(v.Node)
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
// and attribute nodes specially. When copyNamespaces is false, namespace
// declarations are not copied onto element nodes (only those required by
// the element name and attribute names are preserved).
func (ec *execContext) copyNodeToOutput(node helium.Node, copyNamespaces ...bool) error {
	copyNS := true
	if len(copyNamespaces) > 0 {
		copyNS = copyNamespaces[0]
	}
	switch node.Type() {
	case helium.DocumentNode:
		// Copy children of the document node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.copyNodeToOutput(child, copyNS); err != nil {
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
			// XTDE0410: adding attribute to non-element
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s to a non-element node", attr.Name())
		}
		if elem.FirstChild() != nil {
			// XTDE0410: adding attribute after child content
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s after child nodes have been added", attr.Name())
		}
		return copyAttributeToElement(elem, attr)
	case helium.NamespaceNode:
		// Namespace nodes are copied as namespace declarations on the current element.
		nsw, ok := node.(*helium.NamespaceNodeWrapper)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			return nil
		}
		prefix := nsw.Name()
		uri := string(nsw.Content())
		return elem.DeclareNamespace(prefix, uri)
	default:
		if !copyNS && node.Type() == helium.ElementNode {
			return ec.copyElementNoNamespaces(node.(*helium.Element))
		}
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			return err
		}
		return ec.addNode(copied)
	}
}

// copyElementNoNamespaces deep-copies an element but omits namespace
// declarations that are not required by the element or attribute names.
func (ec *execContext) copyElementNoNamespaces(src *helium.Element) error {
	elem, err := ec.resultDoc.CreateElement(src.LocalName())
	if err != nil {
		return err
	}

	// Only declare namespace for the element's own name
	if src.URI() != "" {
		if err := elem.DeclareNamespace(src.Prefix(), src.URI()); err != nil {
			return err
		}
		if err := elem.SetActiveNamespace(src.Prefix(), src.URI()); err != nil {
			return err
		}
	}

	// Copy attributes, declaring their namespaces as needed
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			if !hasNSDecl(elem, a.Prefix(), a.URI()) {
				if err := elem.DeclareNamespace(a.Prefix(), a.URI()); err != nil {
					return err
				}
			}
			ns, nsErr := ec.resultDoc.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nsErr
			}
			if err := elem.SetAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return err
			}
		} else {
			if err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
				return err
			}
		}
	}

	if err := ec.addNode(elem); err != nil {
		return err
	}

	// Recursively copy children (also without namespaces)
	out := ec.currentOutput()
	savedCurrent := out.current
	out.current = elem
	for child := src.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ec.copyNodeToOutput(child, false); err != nil {
			out.current = savedCurrent
			return err
		}
	}
	out.current = savedCurrent
	return nil
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

	// Apply attribute sets before adding to output
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

	// Apply attribute sets (before body so body can override)
	if len(inst.UseAttributeSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttributeSets); err != nil {
			return err
		}
	}
	if len(inst.UseAttrSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttrSets); err != nil {
			return err
		}
	}

	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}

	return nil
}

// applyAttributeSets applies named attribute sets to the current output element.
func (ec *execContext) applyAttributeSets(ctx context.Context, names []string) error {
	for _, name := range names {
		asDef := ec.stylesheet.attributeSets[name]
		if asDef == nil {
			continue
		}
		// Apply referenced attribute sets first (use-attribute-sets on the set itself)
		if len(asDef.UseAttrSets) > 0 {
			if err := ec.applyAttributeSets(ctx, asDef.UseAttrSets); err != nil {
				return err
			}
		}
		// Execute the attribute instructions
		for _, inst := range asDef.Attrs {
			if err := ec.executeInstruction(ctx, inst); err != nil {
				return err
			}
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
			// Errors evaluating message content are recoverable
			value = err.Error()
		} else {
			value = stringifyResult(result)
		}
	}
	if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			// Errors evaluating message body are recoverable
			value += err.Error()
		} else {
			value += stringifySequence(val)
		}
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

	// XSLT 3.0: select attribute specifies which node to number.
	// Evaluate select before the nil-node check so it works inside xsl:function.
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
			f := math.Round(dv.DoubleVal())
			// XTDE0980: value must be non-negative
			if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
				return dynamicError("XTDE0980", "xsl:number value is not a non-negative integer: %v", dv.DoubleVal())
			}
			if f > math.MaxInt64 || f < math.MinInt64 {
				// For values outside int64 range, format directly as big integer
				bf := new(big.Float).SetFloat64(f)
				bi, _ := bf.Int(nil)
				text, tErr := ec.resultDoc.CreateText([]byte(bi.String()))
				if tErr != nil {
					return tErr
				}
				return ec.addNode(text)
			}
			nums = append(nums, int(f))
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
		// Special case: count="." matches any non-document node
		if inst.Count.source == "." && target.Type() != helium.DocumentNode {
			return true
		}
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
			// Include the from node itself if it matches count
			if ec.numberNodeMatches(inst, n, node) {
				ancestors = append(ancestors, n)
			}
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

	// Default separator: propagate the last separator for additional levels
	defaultSep := "."
	if len(tokens) > 1 {
		// Use the last token's separator (the one before the final format token)
		defaultSep = tokens[len(tokens)-2].separator
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
	if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}
	return r > 127 && (unicode.IsLetter(r) || unicode.IsNumber(r))
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
	case "w":
		return numberToWords(num, false)
	case "W":
		return numberToWords(num, true)
	case "Ww":
		w := numberToWords(num, false)
		// Title case: capitalize first letter of each word
		words := strings.Fields(w)
		for i, word := range words {
			if len(word) > 0 {
				runes := []rune(word)
				runes[0] = unicode.ToUpper(runes[0])
				words[i] = string(runes)
			}
		}
		return strings.Join(words, " ")
	default:
		runes := []rune(token)
		firstRune := runes[0]

		// Non-ASCII digit: use as base of a decimal numbering system
		// (e.g., ٠ = U+0660 for Arabic-Indic digits)
		if firstRune > 127 && unicode.IsDigit(firstRune) {
			return formatWithDigitSystem(num, digitZeroOf(firstRune), len(runes))
		}
		// Non-ASCII number (not a digit): ordinal numbering from that codepoint
		// (e.g., ① = U+2460 for circled digits)
		if firstRune > 127 && unicode.IsNumber(firstRune) {
			return formatWithOrdinalSystem(num, firstRune)
		}
		// Non-ASCII letter: ordinal numbering from that codepoint
		if firstRune > 127 && unicode.IsLetter(firstRune) {
			return formatWithOrdinalSystem(num, firstRune)
		}

		// ASCII numeric format: determine minimum width from token (e.g., "001" = width 3)
		minWidth := len(token)
		s := strconv.Itoa(num)
		for len(s) < minWidth {
			s = "0" + s
		}
		if groupSep != "" && groupSize > 0 {
			s = applyGroupingSeparator(s, groupSep, groupSize)
		}
		return s
	}
}

// digitZeroOf returns the zero digit for the Unicode digit block containing r.
func digitZeroOf(r rune) rune {
	// Unicode digit blocks are groups of 10 consecutive codepoints.
	// The zero digit is always at an offset of (r - digit_value) from r.
	// For standard digit blocks, digit_value = r % 10 when aligned at 0.
	// Use unicode.Digit to get the numeric value.
	for d := rune(0); d <= 9; d++ {
		if r-d >= 0 && unicode.IsDigit(r-d) {
			// Verify this is actually the zero of the block
			candidate := r - d
			if !unicode.IsDigit(candidate - 1) || candidate == 0 {
				return candidate
			}
		}
	}
	// Fallback: assume digit value is r mod 10 offset
	return r - (r % 10)
}

// formatWithDigitSystem formats a number using a decimal digit system
// starting at the given zero codepoint.
func formatWithDigitSystem(num int, zero rune, minWidth int) string {
	if num < 0 {
		return "-" + formatWithDigitSystem(-num, zero, minWidth)
	}
	if num == 0 {
		s := string(zero)
		for len([]rune(s)) < minWidth {
			s = string(zero) + s
		}
		return s
	}
	var runes []rune
	n := num
	for n > 0 {
		runes = append([]rune{zero + rune(n%10)}, runes...)
		n /= 10
	}
	for len(runes) < minWidth {
		runes = append([]rune{zero}, runes...)
	}
	return string(runes)
}

// ordinalSystem describes a Unicode numbering system with potentially
// non-contiguous ranges and a special zero character.
type ordinalSystem struct {
	oneChar rune   // the character representing 1
	zero    rune   // the character representing 0 (0 if none)
	ranges  []rune // pairs of (first, last) codepoints for contiguous ranges starting at 1
}

// knownOrdinalSystems maps the "1" character to its ordinal system definition.
var knownOrdinalSystems = map[rune]ordinalSystem{
	// Circled digits: ①-⑳, ㉑-㉟, ㊱-㊿
	0x2460: {oneChar: 0x2460, zero: 0x24EA, ranges: []rune{0x2460, 0x2473, 0x3251, 0x325F, 0x32B1, 0x32BF}},
	// Parenthesized digits: ⑴-⒇ (no special zero)
	0x2474: {oneChar: 0x2474, zero: 0, ranges: []rune{0x2474, 0x2487}},
	// Full-stop digits: ⒈-⒛, zero: 🄀 (U+1F100)
	0x2488: {oneChar: 0x2488, zero: 0x1F100, ranges: []rune{0x2488, 0x249B}},
	// Double circled digits: ⓵-⓾ (no special zero)
	0x24F5: {oneChar: 0x24F5, zero: 0, ranges: []rune{0x24F5, 0x24FE}},
	// Dingbat negative circled: ❶-❿, ⓫-⓴
	0x2776: {oneChar: 0x2776, zero: 0x24FF, ranges: []rune{0x2776, 0x277F, 0x24EB, 0x24F4}},
	// Dingbat negative circled sans-serif: ➊-➓
	0x278A: {oneChar: 0x278A, zero: 0x1F10C, ranges: []rune{0x278A, 0x2793}},
	// Dingbat negative circled sans-serif (alt, starting from ➀): ➀-➉
	0x2780: {oneChar: 0x2780, zero: 0x1F10B, ranges: []rune{0x2780, 0x2789}},
	// Parenthesized ideograph: ㈠-㈩
	0x3220: {oneChar: 0x3220, zero: 0, ranges: []rune{0x3220, 0x3229}},
	// Circled ideograph: ㊀-㊉
	0x3280: {oneChar: 0x3280, zero: 0, ranges: []rune{0x3280, 0x3289}},
	// Aegean numbers: 𐄇-𐄐 (1-10)
	0x10107: {oneChar: 0x10107, zero: 0, ranges: []rune{0x10107, 0x10110}},
	// Coptic Epact numbers: 𐋡-𐋪 (1-10)
	0x102E1: {oneChar: 0x102E1, zero: 0, ranges: []rune{0x102E1, 0x102EA}},
	// Rumi numerals: 𐹠-𐹩 (1-10)
	0x10E60: {oneChar: 0x10E60, zero: 0, ranges: []rune{0x10E60, 0x10E69}},
	// Brahmi number signs: 𑁒-𑁛 (1-10)
	0x11052: {oneChar: 0x11052, zero: 0, ranges: []rune{0x11052, 0x1105B}},
	// Sinhala archaic numbers: 𑇡-𑇪 (1-10)
	0x111E1: {oneChar: 0x111E1, zero: 0, ranges: []rune{0x111E1, 0x111EA}},
	// Counting rod unit digits: 𝍠-𝍨 (1-9)
	0x1D360: {oneChar: 0x1D360, zero: 0, ranges: []rune{0x1D360, 0x1D368}},
	// Mende Kikakui digits: 𞣇-𞣏 (1-9)
	0x1E8C7: {oneChar: 0x1E8C7, zero: 0, ranges: []rune{0x1E8C7, 0x1E8CF}},
	// Digit comma: 🄂-🄊 (1-9), zero: 🄁
	0x1F102: {oneChar: 0x1F102, zero: 0x1F101, ranges: []rune{0x1F102, 0x1F10A}},
}

// formatWithOrdinalSystem formats using a known ordinal numbering system.
// Falls back to decimal when the number exceeds the system's range.
func formatWithOrdinalSystem(num int, start rune) string {
	if num < 0 {
		return strconv.Itoa(num)
	}

	// Look up the system by the start character (which represents 1)
	sys, ok := knownOrdinalSystems[start]
	if !ok {
		// Unknown system: detect range by finding the block boundaries.
		// First check if start-1 is also a numbering character (the zero).
		hasZero := false
		prev := start - 1
		if prev > 0 && (unicode.IsNumber(prev) || unicode.IsLetter(prev)) {
			hasZero = true
		}
		// Count consecutive same-category characters from start, capped at 10.
		rangeLen := ordinalRangeLength(start)
		// If we have a zero predecessor, the system is (zero, 1..N).
		// The range from start to the end of the system is one less than
		// the total block size starting at zero.
		if hasZero {
			totalFromZero := ordinalRangeLength(prev)
			rangeFromStart := totalFromZero - 1
			if rangeFromStart < rangeLen {
				rangeLen = rangeFromStart
			}
		}

		if num == 0 {
			if hasZero {
				return string(prev)
			}
			return strconv.Itoa(0)
		}
		if num <= rangeLen {
			return string(start + rune(num-1))
		}
		return strconv.Itoa(num)
	}

	if num == 0 {
		if sys.zero != 0 {
			return string(sys.zero)
		}
		return strconv.Itoa(0)
	}

	// Map num to the correct codepoint across potentially non-contiguous ranges
	pos := num // 1-based position in the sequence
	for i := 0; i+1 < len(sys.ranges); i += 2 {
		rangeStart := sys.ranges[i]
		rangeEnd := sys.ranges[i+1]
		rangeLen := int(rangeEnd - rangeStart + 1)
		if pos <= rangeLen {
			return string(rangeStart + rune(pos-1))
		}
		pos -= rangeLen
	}

	// Number exceeds ordinal system range: fall back to decimal
	return strconv.Itoa(num)
}

// ordinalRangeLength returns how many consecutive characters starting at r
// belong to the same Unicode category (Number or Letter). This determines
// the range of an ordinal numbering system. For unknown systems, the range
// is capped at 10 to avoid accidentally including characters from adjacent
// but unrelated numbering systems.
func ordinalRangeLength(r rune) int {
	isNum := unicode.IsNumber(r)
	count := 0
	for c := r; ; c++ {
		if isNum {
			if !unicode.IsNumber(c) {
				break
			}
		} else {
			if !unicode.IsLetter(c) {
				break
			}
		}
		count++
		if count >= 10 {
			break // cap unknown systems at 10
		}
	}
	return count
}

// numberToWords converts a number to English words.
func numberToWords(n int, upper bool) string {
	if n == 0 {
		if upper {
			return "ZERO"
		}
		return "zero"
	}
	var ones = []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	var tens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	var words func(int) string
	words = func(n int) string {
		if n < 0 {
			return "minus " + words(-n)
		}
		if n < 20 {
			return ones[n]
		}
		if n < 100 {
			w := tens[n/10]
			if n%10 != 0 {
				w += " " + ones[n%10]
			}
			return w
		}
		if n < 1000 {
			w := ones[n/100] + " hundred"
			if n%100 != 0 {
				w += " and " + words(n%100)
			}
			return w
		}
		if n < 1000000 {
			w := words(n/1000) + " thousand"
			if n%1000 != 0 {
				w += " " + words(n%1000)
			}
			return w
		}
		if n < 1000000000 {
			w := words(n/1000000) + " million"
			if n%1000000 != 0 {
				w += " " + words(n%1000000)
			}
			return w
		}
		if n < 1000000000000 {
			w := words(n/1000000000) + " billion"
			if n%1000000000 != 0 {
				w += " " + words(n%1000000000)
			}
			return w
		}
		return strconv.Itoa(n)
	}
	result := words(n)
	if upper {
		return strings.ToUpper(result)
	}
	return result
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

	// In capture mode, accumulate items directly instead of writing to DOM.
	// Only capture when we are at the function's root output level (the
	// document element wrapper). When nested inside a result element
	// (e.g. an LRE or xsl:copy body), items must be written to the DOM
	// as children of that element.
	if out.captureItems && out.doc != nil && out.current == out.doc.DocumentElement() {
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
			if v.Node.Type() == helium.DocumentNode {
				// Document nodes: output their children (per XSLT spec,
				// a document node in a sequence constructor is replaced
				// by its children).
				for child := v.Node.FirstChild(); child != nil; child = child.NextSibling() {
					copied, copyErr := helium.CopyNode(child, ec.resultDoc)
					if copyErr != nil {
						return copyErr
					}
					if err := ec.addNode(copied); err != nil {
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
	out := ec.currentOutput()

	// In capture mode, accumulate items directly (handles maps, arrays,
	// functions, and other non-DOM items that cannot be serialized to a tree).
	if out.captureItems {
		out.pendingItems = append(out.pendingItems, seq...)
		return nil
	}

	prevWasAtomic := false
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if v.Node.Type() == helium.DocumentNode {
				// Document nodes: output their children.
				for child := v.Node.FirstChild(); child != nil; child = child.NextSibling() {
					copied, copyErr := helium.CopyNode(child, ec.resultDoc)
					if copyErr != nil {
						return copyErr
					}
					if err := ec.addNode(copied); err != nil {
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
	var seq xpath3.Sequence

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq = result.Sequence()
	} else if len(inst.Body) > 0 {
		// Body acts as sequence constructor: evaluate to get the items to sort
		var err error
		seq, err = ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
	}
	if len(seq) == 0 {
		return nil
	}

	// Try to extract nodes for node-based sorting
	nodes, allNodes := xpath3.NodesFrom(seq)
	if allNodes && len(nodes) > 0 {
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

	// Atomic sequence: sort by string value and output as text items
	if len(inst.Sort) > 0 {
		var err error
		seq, err = sortItems(ctx, ec, seq, inst.Sort)
		if err != nil {
			return err
		}
	}

	// Output atomic items separated by spaces
	for i, item := range seq {
		if i > 0 {
			sep, err := ec.resultDoc.CreateText([]byte(" "))
			if err != nil {
				return err
			}
			if err := ec.addNode(sep); err != nil {
				return err
			}
		}
		av, ok := item.(xpath3.AtomicValue)
		if !ok {
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

func (ec *execContext) execNextMatch(ctx context.Context, inst *NextMatchInst) error {
	// xsl:next-match: find the next matching template after the current one
	node := ec.currentNode
	mode := ec.currentMode

	// Process with-param (tunnel and regular).
	// Copy tunnel params to avoid mutating the caller's map.
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		hasTunnel := false
		for _, wp := range inst.Params {
			if wp.Tunnel {
				hasTunnel = true
				break
			}
		}
		if hasTunnel {
			newTunnel := make(map[string]xpath3.Sequence, len(ec.tunnelParams)+len(inst.Params))
			for k, v := range ec.tunnelParams {
				newTunnel[k] = v
			}
			ec.tunnelParams = newTunnel
		}
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
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

	// Handle atomic context items (e.g., xsl:next-match when processing integers)
	if ec.contextItem != nil {
		templates := ec.stylesheet.modeTemplates[mode]
		foundCurrent := false
		for _, tmpl := range templates {
			if tmpl == ec.currentTemplate {
				foundCurrent = true
				continue
			}
			if foundCurrent && tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, ec.contextItem) {
				return ec.executeAtomicTemplate(ctx, tmpl, ec.contextItem, mode)
			}
		}
		// No next match for atomic items — output string value as built-in rule
		av, ok := ec.contextItem.(xpath3.AtomicValue)
		if !ok {
			return nil
		}
		s := fmt.Sprintf("%v", av.Value)
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	}

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

	// Process with-param (tunnel and regular).
	// Copy tunnel params to avoid mutating the caller's map.
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		hasTunnel := false
		for _, wp := range inst.Params {
			if wp.Tunnel {
				hasTunnel = true
				break
			}
		}
		if hasTunnel {
			newTunnel := make(map[string]xpath3.Sequence, len(ec.tunnelParams)+len(inst.Params))
			for k, v := range ec.tunnelParams {
				newTunnel[k] = v
			}
			ec.tunnelParams = newTunnel
		}
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
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
// A sequence is populated if it contains at least one node other than an attribute or
// namespace node that has non-empty content. Elements are always populated.
// Text, comment, and PI nodes are populated only if their content is non-empty.
func isPopulated(node helium.Node) bool {
	switch node.Type() {
	case helium.ElementNode:
		return true
	case helium.TextNode, helium.CommentNode, helium.ProcessingInstructionNode:
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

func (ec *execContext) execResultDocument(ctx context.Context, inst *ResultDocumentInst) error {
	// xsl:result-document redirects output to a secondary destination.
	// Execute the body into a temporary document so it doesn't pollute
	// the primary output. TODO: actually store/serialize secondary results.
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_result")
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

	// TODO: store the secondary result in a map keyed by the evaluated href,
	// and make it available via assert-result-document in the test harness.
	return nil
}

func (ec *execContext) execTryCatch(ctx context.Context, inst *TryCatchInst) error {
	// Execute try body into a temporary output buffer.
	// If the try succeeds, copy the buffered output to the real output.
	// If it fails, discard the buffer and execute the catch.
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_try")
	if err != nil {
		return err
	}
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return err
	}

	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpRoot})

	// Push a new variable scope for the try body so variables
	// declared inside the try are not visible in catch.
	savedVarScope := ec.localVars
	ec.pushVarScope()

	tryErr := func() error {
		if inst.Select != nil {
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

	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	if tryErr == nil {
		// Success — copy buffered output to real output
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

	// Extract error code and QName from the error
	const errNS = "http://www.w3.org/2005/xqt-errors"
	errCode := "XSLT0000"
	errDesc := tryErr.Error()
	var errQName xpath3.QNameValue
	if xErr, ok := tryErr.(*XSLTError); ok {
		errCode = xErr.Code
		errDesc = xErr.Message
		errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: errCode}
	} else if xpErr, ok := tryErr.(*xpath3.XPathError); ok {
		errCode = xpErr.Code
		errDesc = xpErr.Message
		errQName = xpErr.CodeQName()
	}
	if errQName.Local == "" {
		errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: errCode}
	}

	// Build Clark-notation error code for matching against compiled catch patterns
	errClark := errCode
	if errQName.URI != "" {
		errClark = "{" + errQName.URI + "}" + errQName.Local
	}

	// Restore variable scope to before the try body.
	// Variables declared inside the try must not be visible in catch.
	ec.localVars = savedVarScope

	// Find matching catch clause
	var matchedCatch *CatchClause
	for _, clause := range inst.Catches {
		if catchMatches(clause, errClark) {
			matchedCatch = clause
			break
		}
	}
	if matchedCatch == nil {
		// No matching catch — propagate the error
		return tryErr
	}

	// Set XSLT 3.0 error variables in catch scope
	ec.pushVarScope()
	defer ec.popVarScope()

	// $err:code is an xs:QName value with the error code
	errCodeSeq := xpath3.Sequence{xpath3.AtomicValue{
		TypeName: xpath3.TypeQName,
		Value:    errQName,
	}}
	ec.setVar("{"+errNS+"}code", errCodeSeq)
	ec.setVar("{"+errNS+"}description", xpath3.SingleString(errDesc))
	ec.setVar("{"+errNS+"}value", xpath3.EmptySequence())
	ec.setVar("{"+errNS+"}module", xpath3.SingleString(""))
	ec.setVar("{"+errNS+"}line-number", xpath3.SingleInteger(0))
	ec.setVar("{"+errNS+"}column-number", xpath3.SingleInteger(0))

	// Execute matched catch body
	if matchedCatch.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := matchedCatch.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		return ec.outputSequence(result.Sequence())
	}
	for _, child := range matchedCatch.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

// catchMatches returns true if a catch clause matches the given error code.
// errClark is the error code in Clark notation: "{uri}local" or just "local" if no namespace.
func catchMatches(clause *CatchClause, errClark string) bool {
	if len(clause.Errors) == 0 {
		return true // no errors attribute = match all
	}
	for _, pattern := range clause.Errors {
		if pattern == "*" {
			return true
		}
		// Both errClark and pattern are in the same format from resolveQName:
		// - Prefixed names → "{uri}local" (Clark notation)
		// - Unprefixed names → "local" (no namespace)
		if pattern == errClark {
			return true
		}
	}
	return false
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
		groups, err = ec.groupBy(ctx, seq, inst.GroupBy, inst.Composite)
	case inst.GroupAdjacent != nil:
		groups, err = ec.groupAdjacent(ctx, seq, inst.GroupAdjacent, inst.Composite)
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
	savedInGroupCtx := ec.inGroupContext
	ec.size = len(groups)
	ec.inGroupContext = true
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
		ec.inGroupContext = savedInGroupCtx
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
// group-by expression evaluated with each item as context. When composite is
// false and the expression returns a sequence of multiple values, the item is
// added to a group for each value. When composite is true, the entire sequence
// is treated as a single composite key.
func (ec *execContext) groupBy(_ context.Context, seq xpath3.Sequence, groupByExpr *xpath3.Expression, composite bool) ([]fegGroup, error) {
	type entry struct {
		key      string
		keySeq   xpath3.Sequence
		items    xpath3.Sequence
	}
	var order []string
	groupMap := make(map[string]*entry)

	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
	}()

	for i, item := range seq {
		ec.position = i + 1
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
		}
		xpathCtx := ec.newXPathContext(node)
		result, err := groupByExpr.Evaluate(xpathCtx, node)
		if err != nil {
			return nil, err
		}

		resultSeq := result.Sequence()
		if len(resultSeq) == 0 {
			continue
		}

		if composite {
			// Composite: entire sequence is a single key
			keyVal := compositeKeyString(resultSeq)
			if e, ok := groupMap[keyVal]; ok {
				e.items = append(e.items, item)
			} else {
				groupMap[keyVal] = &entry{key: keyVal, keySeq: atomizeSequence(resultSeq), items: xpath3.Sequence{item}}
				order = append(order, keyVal)
			}
		} else {
			// Non-composite: each value creates a separate group key
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
	}

	groups := make([]fegGroup, len(order))
	for i, k := range order {
		e := groupMap[k]
		if e.keySeq != nil {
			groups[i] = fegGroup{key: e.keySeq, items: e.items}
		} else {
			groups[i] = fegGroup{
				key:   xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: e.key}},
				items: e.items,
			}
		}
	}
	return groups, nil
}

// groupAdjacent implements group-adjacent: consecutive items with equal
// grouping key values form a group. When composite is true, the key
// expression returns a sequence treated as a single composite key.
func (ec *execContext) groupAdjacent(ctx context.Context, seq xpath3.Sequence, adjExpr *xpath3.Expression, composite bool) ([]fegGroup, error) {
	var groups []fegGroup
	var currentKey string
	var currentKeySeq xpath3.Sequence
	var currentItems xpath3.Sequence

	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
	}()

	for i, item := range seq {
		ec.position = i + 1
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
		}
		xpathCtx := ec.newXPathContext(node)
		result, err := adjExpr.Evaluate(xpathCtx, node)
		if err != nil {
			return nil, err
		}

		var keyVal string
		var keySeq xpath3.Sequence
		if composite {
			rSeq := result.Sequence()
			keyVal = compositeKeyString(rSeq)
			keySeq = atomizeSequence(rSeq)
		} else {
			keyVal = stringifyResult(result)
		}

		if keyVal == currentKey && len(currentItems) > 0 {
			currentItems = append(currentItems, item)
		} else {
			if len(currentItems) > 0 {
				var gKey xpath3.Sequence
				if currentKeySeq != nil {
					gKey = currentKeySeq
				} else {
					gKey = xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
				}
				groups = append(groups, fegGroup{key: gKey, items: currentItems})
			}
			currentKey = keyVal
			currentKeySeq = keySeq
			currentItems = xpath3.Sequence{item}
		}
	}
	if len(currentItems) > 0 {
		var gKey xpath3.Sequence
		if currentKeySeq != nil {
			gKey = currentKeySeq
		} else {
			gKey = xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
		}
		groups = append(groups, fegGroup{key: gKey, items: currentItems})
	}
	return groups, nil
}

// compositeKeyString creates a canonical string representation of a composite
// key for use as a map key. Items are separated by a NUL byte to avoid
// collisions.
func compositeKeyString(seq xpath3.Sequence) string {
	parts := make([]string, len(seq))
	for i, item := range seq {
		parts[i] = stringifyItem(item)
	}
	return strings.Join(parts, "\x00")
}

// atomizeSequence converts each item in the sequence to an atomic value.
func atomizeSequence(seq xpath3.Sequence) xpath3.Sequence {
	result := make(xpath3.Sequence, len(seq))
	for i, item := range seq {
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			result[i] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: stringifyItem(item)}
		} else {
			result[i] = av
		}
	}
	return result
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

// execMap executes an xsl:map instruction, producing a MapItem from child
// xsl:map-entry instructions.
func (ec *execContext) execMap(ctx context.Context, inst *MapInst) error {
	var entries []xpath3.MapEntry
	for _, child := range inst.Body {
		me, ok := child.(*MapEntryInst)
		if !ok {
			// Non-map-entry children are executed normally (e.g., xsl:variable)
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
			continue
		}
		// Evaluate the key
		xpathCtx := ec.newXPathContext(ec.contextNode)
		keyResult, err := me.Key.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		keySeq := keyResult.Sequence()
		if len(keySeq) != 1 {
			return dynamicError("XPTY0004", "xsl:map-entry key must be a single atomic value")
		}
		keyAV, err := xpath3.AtomizeItem(keySeq[0])
		if err != nil {
			return err
		}

		// Evaluate the value
		var valSeq xpath3.Sequence
		if me.Select != nil {
			valResult, err := me.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			valSeq = valResult.Sequence()
		} else if len(me.Body) > 0 {
			valSeq, err = ec.evaluateBody(ctx, me.Body)
			if err != nil {
				return err
			}
		}

		entries = append(entries, xpath3.MapEntry{Key: keyAV, Value: valSeq})
	}

	m := xpath3.NewMap(entries)
	out := ec.currentOutput()
	if out.captureItems {
		out.pendingItems = append(out.pendingItems, m)
		return nil
	}
	// If not in capture mode, maps can't be added to DOM — produce string representation
	text, err := ec.resultDoc.CreateText([]byte(fmt.Sprint(m)))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

// execMapEntry is a no-op when called outside xsl:map; entries are handled
// by execMap directly.
func (ec *execContext) execMapEntry(ctx context.Context, inst *MapEntryInst) error {
	// When called standalone (outside xsl:map), evaluate and produce output
	xpathCtx := ec.newXPathContext(ec.contextNode)
	var valSeq xpath3.Sequence
	var err error
	if inst.Select != nil {
		valResult, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		valSeq = valResult.Sequence()
	} else if len(inst.Body) > 0 {
		valSeq, err = ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
	}
	// Output the value as text
	s := stringifySequenceWithSep(valSeq, " ")
	if s != "" {
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	}
	return nil
}
