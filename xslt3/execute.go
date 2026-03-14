package xslt3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

type execContextKey struct{}

// execContext holds XSLT transformation state. Stored inside context.Context.
type execContext struct {
	stylesheet       *Stylesheet
	sourceDoc        *helium.Document
	resultDoc        *helium.Document
	currentNode      helium.Node // XSLT current() node
	contextNode      helium.Node // XPath context node
	position         int
	size             int
	localVars        *varScope
	globalVars       map[string]xpath3.Sequence
	globalVarDefs    map[string]*Variable // unevaluated global variable definitions (lazy)
	globalParamDefs  map[string]*Param    // unevaluated global param definitions (lazy)
	globalEvaluating map[string]bool      // circular dependency detection
	collectingVars   bool                 // reentrancy guard for collectAllVars
	currentMode      string
	currentTemplate  *Template                  // currently executing template (for next-match)
	xpathDefaultNS   string                     // current xpath-default-namespace
	tunnelParams     map[string]xpath3.Sequence // tunnel parameters passed through apply-templates
	depth            int                        // recursion depth
	outputStack      []*outputFrame
	keyTables        map[string]*keyTable
	docCache         map[string]*helium.Document
	msgHandler       func(string, bool)
	transformCfg     *transformConfig
	transformCtx     context.Context // parent context from Transform caller (for cancellation/deadlines)
}

func withExecContext(ctx context.Context, ec *execContext) context.Context {
	return context.WithValue(ctx, execContextKey{}, ec)
}

func getExecContext(ctx context.Context) *execContext {
	v, _ := ctx.Value(execContextKey{}).(*execContext)
	return v
}

// varScope is a variable scope chain.
type varScope struct {
	vars   map[string]xpath3.Sequence
	parent *varScope
}

func (vs *varScope) lookup(name string) (xpath3.Sequence, bool) {
	for s := vs; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

func (ec *execContext) pushVarScope() {
	ec.localVars = &varScope{
		vars:   make(map[string]xpath3.Sequence),
		parent: ec.localVars,
	}
}

func (ec *execContext) popVarScope() {
	if ec.localVars != nil {
		ec.localVars = ec.localVars.parent
	}
}

func (ec *execContext) setVar(name string, value xpath3.Sequence) {
	if ec.localVars == nil {
		ec.localVars = &varScope{vars: make(map[string]xpath3.Sequence)}
	}
	ec.localVars.vars[name] = value
}

func (ec *execContext) resolvePrefix(prefix string) string {
	if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
		return uri
	}
	return ""
}

// currentOutput returns the current output frame.
func (ec *execContext) currentOutput() *outputFrame {
	return ec.outputStack[len(ec.outputStack)-1]
}

// addNode adds a node to the current output insertion point.
func (ec *execContext) addNode(node helium.Node) error {
	out := ec.currentOutput()
	return out.current.AddChild(node)
}

// newXPathContext creates a context.Context with xpath3 settings for evaluating
// XPath expressions within the XSLT transformation.
func (ec *execContext) newXPathContext(node helium.Node) context.Context {
	vars := ec.collectAllVars()
	ctx := ec.transformCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withExecContext(ctx, ec)
	ctx = xpath3.WithVariables(ctx, vars)
	ctx = xpath3.WithFunctions(ctx, ec.xsltFunctions())
	if fnsNS := ec.xsltFunctionsNS(); len(fnsNS) > 0 {
		ctx = xpath3.WithFunctionsNS(ctx, fnsNS)
	}
	if len(ec.stylesheet.namespaces) > 0 || ec.xpathDefaultNS != "" {
		ns := make(map[string]string, len(ec.stylesheet.namespaces)+1)
		for k, v := range ec.stylesheet.namespaces {
			ns[k] = v
		}
		if ec.xpathDefaultNS != "" {
			ns[""] = ec.xpathDefaultNS
		}
		ctx = xpath3.WithNamespaces(ctx, ns)
	}
	if ec.position > 0 {
		ctx = xpath3.WithPosition(ctx, ec.position)
	}
	if ec.size > 0 {
		ctx = xpath3.WithSize(ctx, ec.size)
	}
	return ctx
}

func (ec *execContext) collectAllVars() map[string]xpath3.Sequence {
	vars := make(map[string]xpath3.Sequence)
	// Eagerly evaluate all pending global vars/params, but only at the
	// top level. Nested calls (from within evaluateGlobalVar/Param →
	// newXPathContext → collectAllVars) just snapshot what's available;
	// the lazy lookupVar path handles remaining references on demand.
	if !ec.collectingVars {
		ec.collectingVars = true
		defer func() { ec.collectingVars = false }()
		for len(ec.globalVarDefs) > 0 || len(ec.globalParamDefs) > 0 {
			progress := false
			for _, v := range ec.globalVarDefs {
				if _, err := ec.evaluateGlobalVar(v); err == nil {
					progress = true
				}
			}
			for _, p := range ec.globalParamDefs {
				if _, err := ec.evaluateGlobalParam(p); err == nil {
					progress = true
				}
			}
			if !progress {
				break // avoid infinite loop on circular deps
			}
		}
	}

	// Start with globals
	for k, v := range ec.globalVars {
		vars[k] = v
	}
	// Walk from outermost to innermost scope so inner scopes override
	var scopes []*varScope
	for s := ec.localVars; s != nil; s = s.parent {
		scopes = append(scopes, s)
	}
	for i := len(scopes) - 1; i >= 0; i-- {
		for k, v := range scopes[i].vars {
			vars[k] = v
		}
	}
	return vars
}

// executeTransform performs the XSLT transformation.
func executeTransform(ctx context.Context, source *helium.Document, ss *Stylesheet, cfg *transformConfig) (*helium.Document, error) {
	resultDoc := helium.NewDefaultDocument()

	ec := &execContext{
		stylesheet:   ss,
		sourceDoc:    source,
		resultDoc:    resultDoc,
		currentNode:  source,
		contextNode:  source,
		position:     1,
		size:         1,
		globalVars:   make(map[string]xpath3.Sequence),
		currentMode:  "",
		outputStack:  []*outputFrame{{doc: resultDoc, current: resultDoc}},
		keyTables:    make(map[string]*keyTable),
		docCache:     make(map[string]*helium.Document),
		transformCtx: ctx,
	}

	if cfg != nil && cfg.msgHandler != nil {
		ec.msgHandler = cfg.msgHandler
	}

	// Store exec context in Go context for AVT evaluation
	ctx = withExecContext(ctx, ec)

	// Initialize global variables
	if err := ec.initGlobalVars(ctx, cfg); err != nil {
		return nil, err
	}

	// Either call the initial-template or apply templates to the document root
	initialTemplateName := ""
	if cfg != nil && cfg.initialTemplate != "" {
		initialTemplateName = cfg.initialTemplate
	}

	// XSLT 3.0: if no explicit initial template, check for xsl:initial-template
	if initialTemplateName == "" {
		if _, ok := ec.stylesheet.namedTemplates["xsl:initial-template"]; ok {
			initialTemplateName = "xsl:initial-template"
		}
	}

	if initialTemplateName != "" {
		tmpl := ec.stylesheet.namedTemplates[initialTemplateName]
		if tmpl == nil {
			return nil, dynamicError(errCodeXTDE0820, "initial template %q not found", initialTemplateName)
		}
		if err := ec.executeTemplate(ctx, tmpl, source, ""); err != nil {
			return nil, err
		}
	} else {
		if err := ec.applyTemplates(ctx, source, ""); err != nil {
			return nil, err
		}
	}

	return resultDoc, nil
}

// initGlobalVars registers global variables and parameters for lazy evaluation.
// Params with caller-provided values are set immediately; all others are
// evaluated on first access to support arbitrary declaration order.
func (ec *execContext) initGlobalVars(ctx context.Context, cfg *transformConfig) error {
	ec.transformCfg = cfg
	ec.globalVarDefs = make(map[string]*Variable, len(ec.stylesheet.globalVars))
	ec.globalParamDefs = make(map[string]*Param, len(ec.stylesheet.globalParams))
	ec.globalEvaluating = make(map[string]bool)

	// Register params — set immediately if caller provided a value
	for _, p := range ec.stylesheet.globalParams {
		if cfg != nil && cfg.params != nil {
			if sv, ok := cfg.params[p.Name]; ok {
				ec.globalVars[p.Name] = xpath3.SingleString(sv)
				continue
			}
		}
		ec.globalParamDefs[p.Name] = p
	}

	// Register variables for lazy evaluation
	for _, v := range ec.stylesheet.globalVars {
		ec.globalVarDefs[v.Name] = v
	}

	return nil
}

// evaluateGlobalVar evaluates a global variable on first access.
func (ec *execContext) evaluateGlobalVar(v *Variable) (xpath3.Sequence, error) {
	if ec.globalEvaluating[v.Name] {
		return nil, fmt.Errorf("circular dependency in global variable %q", v.Name)
	}
	ec.globalEvaluating[v.Name] = true
	defer delete(ec.globalEvaluating, v.Name)

	var val xpath3.Sequence
	ctx := context.Background()
	if v.Select != nil {
		xpathCtx := ec.newXPathContext(ec.sourceDoc)
		result, err := v.Select.Evaluate(xpathCtx, ec.sourceDoc)
		if err != nil {
			return nil, fmt.Errorf("error evaluating global variable %q: %w", v.Name, err)
		}
		val = result.Sequence()
	} else if len(v.Body) > 0 {
		var err error
		if v.As != "" {
			// With as attribute: evaluate as raw sequence
			val, err = ec.evaluateBody(ctx, v.Body)
		} else {
			// No as: wrap in document node (temporary tree)
			val, err = ec.evaluateBodyAsDocument(ctx, v.Body)
		}
		if err != nil {
			return nil, fmt.Errorf("error evaluating global variable %q body: %w", v.Name, err)
		}
	} else {
		val = xpath3.SingleString("")
	}

	ec.globalVars[v.Name] = val
	delete(ec.globalVarDefs, v.Name)
	return val, nil
}

// evaluateGlobalParam evaluates a global param on first access.
func (ec *execContext) evaluateGlobalParam(p *Param) (xpath3.Sequence, error) {
	if ec.globalEvaluating[p.Name] {
		return nil, fmt.Errorf("circular dependency in global param %q", p.Name)
	}
	ec.globalEvaluating[p.Name] = true
	defer delete(ec.globalEvaluating, p.Name)

	var val xpath3.Sequence
	ctx := context.Background()
	if p.Select != nil {
		xpathCtx := ec.newXPathContext(ec.sourceDoc)
		result, err := p.Select.Evaluate(xpathCtx, ec.sourceDoc)
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q: %w", p.Name, err)
		}
		val = result.Sequence()
	} else if len(p.Body) > 0 {
		var err error
		val, err = ec.evaluateBody(ctx, p.Body)
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q body: %w", p.Name, err)
		}
	}

	ec.globalVars[p.Name] = val
	delete(ec.globalParamDefs, p.Name)
	return val, nil
}

// evaluateBody executes instructions and captures the result as a sequence.
// When instructions produce nodes, they are wrapped as a temporary tree.
func (ec *execContext) evaluateBody(ctx context.Context, body []Instruction) (xpath3.Sequence, error) {
	// Create a temporary document to capture output
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_tmp")
	if err != nil {
		return nil, err
	}
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	// Push a new output frame with capture mode enabled
	frame := &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}

	// If we captured atomic items via xsl:sequence, return them directly
	if len(frame.pendingItems) > 0 {
		// TODO(xslt3): when the body produces both DOM nodes and atomic
		// items, this concatenates all nodes first then all atomics,
		// which loses the original construction order (e.g., node,
		// atomic, node becomes node, node, atomic). To fully conform
		// to XSLT sequence-constructor semantics, the capture mechanism
		// should record a single ordered stream of items (nodes and
		// atomics interleaved) rather than two separate buckets. This
		// is a known limitation of the initial implementation.
		if tmpRoot.FirstChild() != nil {
			var seq xpath3.Sequence
			for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, frame.pendingItems...)
			return seq, nil
		}
		return frame.pendingItems, nil
	}

	// Collect children as sequence or text value
	var sb strings.Builder
	hasNodes := false
	for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.TextNode || child.Type() == helium.CDATASectionNode {
			sb.Write(child.Content())
		} else {
			hasNodes = true
		}
	}

	if hasNodes {
		// Return as a document fragment (the temp root's children)
		var seq xpath3.Sequence
		for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
			seq = append(seq, xpath3.NodeItem{Node: child})
		}
		return seq, nil
	}

	return xpath3.SingleString(sb.String()), nil
}

// evaluateBodyAsDocument executes instructions and wraps the result in a
// document node (temporary tree), as required by the XSLT spec for variables
// and params with content body and no select/as attributes.
func (ec *execContext) evaluateBodyAsDocument(ctx context.Context, body []Instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()

	frame := &outputFrame{doc: tmpDoc, current: tmpDoc, captureItems: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}

	// If the body produced atomic items via xsl:sequence, return them
	// directly rather than wrapping in a document node.
	// TODO(xslt3): when both DOM children and atomic items exist, the
	// atomics are silently dropped (only the document node is returned).
	// A proper implementation should either merge them into an ordered
	// sequence or raise an error when atomics appear in a document-node
	// context. This is a known limitation of the initial implementation.
	if len(frame.pendingItems) > 0 && tmpDoc.FirstChild() == nil {
		return frame.pendingItems, nil
	}

	return xpath3.Sequence{xpath3.NodeItem{Node: tmpDoc}}, nil
}

// applyTemplates matches and executes templates for a node.
func (ec *execContext) applyTemplates(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	// Strip whitespace-only text nodes per xsl:strip-space
	if ec.shouldStripWhitespace(node) {
		return nil
	}

	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return dynamicError(errCodeXTDE0820, "recursion depth exceeded")
	}
	defer func() { ec.depth-- }()

	switch mode {
	case "#current":
		mode = ec.currentMode
	case "#default", "":
		mode = ec.stylesheet.defaultMode
	}

	// Collect param values if any
	var pv map[string]xpath3.Sequence
	if len(paramValues) > 0 {
		pv = paramValues[0]
	}

	// Find best matching template
	tmpl := ec.findBestTemplate(node, mode)
	if tmpl != nil {
		return ec.executeTemplate(ctx, tmpl, node, mode, pv)
	}

	// Use built-in template rules
	return ec.applyBuiltinRules(ctx, node, mode, paramValues...)
}

// findBestTemplate finds the highest-priority matching template for a node.
func (ec *execContext) findBestTemplate(node helium.Node, mode string) *Template {
	// Set currentNode to the candidate so current() works in pattern predicates
	savedCurrent := ec.currentNode
	ec.currentNode = node
	defer func() { ec.currentNode = savedCurrent }()

	templates := ec.stylesheet.modeTemplates[mode]
	for _, tmpl := range templates {
		if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return tmpl
		}
	}
	return nil
}

// executeTemplate executes a template with the given node as context.
const maxRecursionDepth = 1000

func (ec *execContext) executeTemplate(ctx context.Context, tmpl *Template, node helium.Node, mode string, paramOverrides ...map[string]xpath3.Sequence) error {
	// Save and restore context
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedMode := ec.currentMode
	savedPos := ec.position
	savedSize := ec.size
	savedTemplate := ec.currentTemplate
	savedTunnel := ec.tunnelParams
	savedXPathDefaultNS := ec.xpathDefaultNS
	ec.currentNode = node
	ec.contextNode = node
	ec.currentMode = mode
	ec.currentTemplate = tmpl
	ec.xpathDefaultNS = tmpl.XPathDefaultNS
	ec.position = 1
	ec.size = 1
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.currentMode = savedMode
		ec.currentTemplate = savedTemplate
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	// Collect param overrides
	var po map[string]xpath3.Sequence
	if len(paramOverrides) > 0 {
		po = paramOverrides[0]
	}

	// Set param values: use with-param overrides when available, else defaults.
	// Tunnel params receive from ec.tunnelParams, not from regular param overrides.
	for _, p := range tmpl.Params {
		if p.Tunnel {
			// Tunnel param: receive from tunnel context
			if ec.tunnelParams != nil {
				if val, ok := ec.tunnelParams[p.Name]; ok {
					ec.setVar(p.Name, val)
					continue
				}
			}
		} else if po != nil {
			if val, ok := po[p.Name]; ok {
				ec.setVar(p.Name, val)
				continue
			}
		}
		// Use default value
		if p.Select != nil {
			xpathCtx := ec.newXPathContext(node)
			result, err := p.Select.Evaluate(xpathCtx, node)
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
	for _, inst := range tmpl.Body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return err
		}
	}

	return nil
}

// applyBuiltinRules applies the built-in template rules per XSLT spec.
func (ec *execContext) applyBuiltinRules(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode, helium.ElementNode:
		// Built-in rule: apply templates to children, forwarding params
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.TextNode, helium.CDATASectionNode:
		// Check strip-space: skip whitespace-only text nodes if parent is stripped
		if ec.shouldStripWhitespace(node) {
			return nil
		}
		// Built-in rule: copy text to output
		text, err := ec.resultDoc.CreateText(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(text)
	case helium.AttributeNode:
		// Built-in rule: copy text value to output
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		text, err := ec.resultDoc.CreateText([]byte(attr.Value()))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	default:
		// Other node types: do nothing
		return nil
	}
}

// shouldStripWhitespace returns true if a text node is whitespace-only
// and its parent element matches a strip-space pattern.
func (ec *execContext) shouldStripWhitespace(node helium.Node) bool {
	// Only strip text/CDATA nodes, not elements or other node types
	if node.Type() != helium.TextNode && node.Type() != helium.CDATASectionNode {
		return false
	}
	content := node.Content()
	// Check if whitespace-only
	for _, b := range content {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	// Check parent element against strip/preserve space rules
	parent := node.Parent()
	if parent == nil || parent.Type() != helium.ElementNode {
		return false
	}
	elem := parent.(*helium.Element)
	return ec.isElementStripped(elem)
}

// isElementStripped checks if an element matches strip-space rules.
// preserve-space overrides strip-space for the same element.
func (ec *execContext) isElementStripped(elem *helium.Element) bool {
	ss := ec.stylesheet
	if len(ss.stripSpace) == 0 {
		return false
	}

	stripped := false
	stripPriority := -1
	for _, nt := range ss.stripSpace {
		if matchSpaceNameTest(nt, elem, ss.namespaces) {
			p := nameTestPriority(nt)
			if p > stripPriority {
				stripPriority = p
				stripped = true
			}
		}
	}

	if !stripped {
		return false
	}

	// Check if preserve-space overrides
	for _, nt := range ss.preserveSpace {
		if matchSpaceNameTest(nt, elem, ss.namespaces) {
			p := nameTestPriority(nt)
			if p >= stripPriority {
				return false
			}
		}
	}
	return true
}

// matchSpaceNameTest checks if an element matches a strip/preserve-space NameTest pattern.
func matchSpaceNameTest(nt NameTest, elem *helium.Element, nsBindings map[string]string) bool {
	if nt.Local == "*" && nt.Prefix == "" {
		return true // "*" matches all
	}
	if nt.Local == "*" && nt.Prefix != "" {
		// "prefix:*" matches elements in that namespace
		nsURI := nsBindings[nt.Prefix]
		return elem.URI() == nsURI
	}
	if nt.Prefix != "" {
		// "prefix:local" matches specific element in namespace
		nsURI := nsBindings[nt.Prefix]
		return elem.LocalName() == nt.Local && elem.URI() == nsURI
	}
	// "local" matches elements with that local name (no namespace)
	return elem.LocalName() == nt.Local && elem.URI() == ""
}

// nameTestPriority returns the priority of a NameTest for conflict resolution.
// Specific names > prefix:* > *
func nameTestPriority(nt NameTest) int {
	if nt.Local == "*" && nt.Prefix == "" {
		return 0 // "*"
	}
	if nt.Local == "*" {
		return 1 // "prefix:*"
	}
	return 2 // specific name
}

// selectDefaultNodes returns the default node-set for apply-templates
// (child::node()).
func selectDefaultNodes(node helium.Node) []helium.Node {
	var nodes []helium.Node
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		nodes = append(nodes, child)
	}
	return nodes
}
