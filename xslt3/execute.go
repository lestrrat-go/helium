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
	stylesheet      *Stylesheet
	sourceDoc       *helium.Document
	resultDoc       *helium.Document
	currentNode     helium.Node // XSLT current() node
	contextNode     helium.Node // XPath context node
	position        int
	size            int
	localVars       *varScope
	globalVars      map[string]xpath3.Sequence
	currentMode     string
	currentTemplate *Template // currently executing template (for next-match)
	depth           int      // recursion depth
	outputStack     []*outputFrame
	keyTables       map[string]*keyTable
	docCache        map[string]*helium.Document
	msgHandler      func(string, bool)
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

func (ec *execContext) lookupVar(name string) (xpath3.Sequence, bool) {
	if seq, ok := ec.localVars.lookup(name); ok {
		return seq, true
	}
	if seq, ok := ec.globalVars[name]; ok {
		return seq, true
	}
	return nil, false
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
	ctx := context.Background()
	ctx = withExecContext(ctx, ec)
	ctx = xpath3.WithVariables(ctx, vars)
	ctx = xpath3.WithFunctions(ctx, ec.xsltFunctions())
	if fnsNS := ec.xsltFunctionsNS(); len(fnsNS) > 0 {
		ctx = xpath3.WithFunctionsNS(ctx, fnsNS)
	}
	if len(ec.stylesheet.namespaces) > 0 {
		ctx = xpath3.WithNamespaces(ctx, ec.stylesheet.namespaces)
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
	for k, v := range ec.globalVars {
		vars[k] = v
	}
	for s := ec.localVars; s != nil; s = s.parent {
		for k, v := range s.vars {
			if _, exists := vars[k]; !exists {
				vars[k] = v
			}
		}
	}
	return vars
}

// executeTransform performs the XSLT transformation.
func executeTransform(ctx context.Context, source *helium.Document, ss *Stylesheet, cfg *transformConfig) (*helium.Document, error) {
	resultDoc := helium.NewDefaultDocument()

	ec := &execContext{
		stylesheet:  ss,
		sourceDoc:   source,
		resultDoc:   resultDoc,
		currentNode: source,
		contextNode: source,
		position:    1,
		size:        1,
		globalVars:  make(map[string]xpath3.Sequence),
		currentMode: "",
		outputStack: []*outputFrame{{doc: resultDoc, current: resultDoc}},
		keyTables:   make(map[string]*keyTable),
		docCache:    make(map[string]*helium.Document),
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
	if cfg != nil && cfg.initialTemplate != "" {
		tmpl := ec.stylesheet.namedTemplates[cfg.initialTemplate]
		if tmpl == nil {
			return nil, dynamicError(errCodeXTDE0820, "initial template %q not found", cfg.initialTemplate)
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

// initGlobalVars evaluates global variables and parameters in dependency order.
func (ec *execContext) initGlobalVars(ctx context.Context, cfg *transformConfig) error {
	// First, process params (may be overridden by caller)
	for _, p := range ec.stylesheet.globalParams {
		var val xpath3.Sequence

		// Check if caller provided a value
		if cfg != nil && cfg.params != nil {
			if sv, ok := cfg.params[p.Name]; ok {
				val = xpath3.SingleString(sv)
				ec.globalVars[p.Name] = val
				continue
			}
		}

		if p.Select != nil {
			xpathCtx := ec.newXPathContext(ec.sourceDoc)
			result, err := p.Select.Evaluate(xpathCtx, ec.sourceDoc)
			if err != nil {
				return fmt.Errorf("error evaluating global param %q: %w", p.Name, err)
			}
			val = result.Sequence()
		} else if len(p.Body) > 0 {
			var err error
			val, err = ec.evaluateBody(ctx, p.Body)
			if err != nil {
				return fmt.Errorf("error evaluating global param %q body: %w", p.Name, err)
			}
		}
		ec.globalVars[p.Name] = val
	}

	// Then process variables
	for _, v := range ec.stylesheet.globalVars {
		if v.Select != nil {
			xpathCtx := ec.newXPathContext(ec.sourceDoc)
			result, err := v.Select.Evaluate(xpathCtx, ec.sourceDoc)
			if err != nil {
				return fmt.Errorf("error evaluating global variable %q: %w", v.Name, err)
			}
			ec.globalVars[v.Name] = result.Sequence()
		} else if len(v.Body) > 0 {
			val, err := ec.evaluateBody(ctx, v.Body)
			if err != nil {
				return fmt.Errorf("error evaluating global variable %q body: %w", v.Name, err)
			}
			ec.globalVars[v.Name] = val
		} else {
			ec.globalVars[v.Name] = xpath3.SingleString("")
		}
	}

	return nil
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

	// Push a new output frame
	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpRoot})
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
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

// applyTemplates matches and executes templates for a node.
func (ec *execContext) applyTemplates(ctx context.Context, node helium.Node, mode string) error {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return dynamicError(errCodeXTDE0820, "recursion depth exceeded")
	}
	defer func() { ec.depth-- }()

	if mode == "#current" {
		mode = ec.currentMode
	} else if mode == "#default" || mode == "" {
		mode = ec.stylesheet.defaultMode
	}

	// Find best matching template
	tmpl := ec.findBestTemplate(node, mode)
	if tmpl != nil {
		return ec.executeTemplate(ctx, tmpl, node, mode)
	}

	// Use built-in template rules
	return ec.applyBuiltinRules(ctx, node, mode)
}

// findBestTemplate finds the highest-priority matching template for a node.
func (ec *execContext) findBestTemplate(node helium.Node, mode string) *Template {
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

func (ec *execContext) executeTemplate(ctx context.Context, tmpl *Template, node helium.Node, mode string) error {
	// Save and restore context
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedMode := ec.currentMode
	savedPos := ec.position
	savedSize := ec.size
	savedTemplate := ec.currentTemplate
	ec.currentNode = node
	ec.contextNode = node
	ec.currentMode = mode
	ec.currentTemplate = tmpl
	ec.position = 1
	ec.size = 1
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.currentMode = savedMode
		ec.currentTemplate = savedTemplate
		ec.position = savedPos
		ec.size = savedSize
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	// Set default param values
	for _, p := range tmpl.Params {
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
func (ec *execContext) applyBuiltinRules(ctx context.Context, node helium.Node, mode string) error {
	switch node.Type() {
	case helium.DocumentNode, helium.ElementNode:
		// Built-in rule: apply templates to children
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.applyTemplates(ctx, child, mode); err != nil {
				return err
			}
		}
		return nil
	case helium.TextNode, helium.CDATASectionNode:
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

// selectDefaultNodes returns the default node-set for apply-templates
// (child::node()).
func selectDefaultNodes(node helium.Node) []helium.Node {
	var nodes []helium.Node
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		nodes = append(nodes, child)
	}
	return nodes
}
