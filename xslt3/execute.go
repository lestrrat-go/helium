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
	stylesheet        *Stylesheet
	sourceDoc         *helium.Document
	resultDoc         *helium.Document
	currentNode       helium.Node // XSLT current() node
	contextNode       helium.Node // XPath context node
	contextItem       xpath3.Item // non-nil when context is an atomic value (for-each over atomics)
	position          int
	size              int
	localVars         *varScope
	globalVars        map[string]xpath3.Sequence
	globalVarDefs     map[string]*Variable // unevaluated global variable definitions (lazy)
	globalParamDefs   map[string]*Param    // unevaluated global param definitions (lazy)
	globalEvaluating  map[string]bool      // circular dependency detection
	collectingVars    bool                 // reentrancy guard for collectAllVars
	currentMode       string
	currentTemplate   *Template                  // currently executing template (for next-match)
	xpathDefaultNS    string                     // current xpath-default-namespace
	hasXPathDefaultNS bool                       // true when xpathDefaultNS is explicitly set
	tunnelParams      map[string]xpath3.Sequence // tunnel parameters passed through apply-templates
	currentGroup      xpath3.Sequence            // current-group() value during for-each-group
	currentGroupKey   xpath3.Sequence            // current-grouping-key() value during for-each-group
	inGroupContext    bool                       // true when inside for-each-group body
	depth             int                        // recursion depth
	outputStack       []*outputFrame
	keyTables         map[string]*keyTable
	docCache          map[string]*helium.Document
	cachedFns         map[string]xpath3.Function         // cached xsltFunctions() result
	cachedFnsNS       map[xpath3.QualifiedName]xpath3.Function // cached xsltFunctionsNS() result
	globalVarsGen     uint64                             // incremented when globalVars changes
	cachedVarsMap     map[string]xpath3.Sequence         // cached result of collectAllVars (globals only)
	cachedVarsGen     uint64                             // globalVarsGen at time cachedVarsMap was built
	accumulatorState  map[string]xpath3.Sequence         // accumulator name -> current value
	breakValue        xpath3.Sequence                    // value produced by xsl:break
	nextIterParams    map[string]xpath3.Sequence         // param values from xsl:next-iteration
	msgHandler        func(string, bool)
	transformCfg      *transformConfig
	transformCtx      context.Context                  // parent context from Transform caller (for cancellation/deadlines)
	typeAnnotations   map[helium.Node]string           // node → xs:... type annotation (schema-aware)
}

func withExecContext(ctx context.Context, ec *execContext) context.Context {
	return context.WithValue(ctx, execContextKey{}, ec)
}

func getExecContext(ctx context.Context) *execContext {
	v, _ := ctx.Value(execContextKey{}).(*execContext)
	return v
}

// annotateAttr applies a type annotation to a just-set attribute on an element.
// If typeName is empty, this is a no-op.
func (ec *execContext) annotateAttr(elem *helium.Element, typeName, localName, nsURI, value string) {
	if typeName == "" {
		return
	}
	if nsURI != "" {
		if attr := elem.GetAttributeNodeNS(localName, nsURI); attr != nil {
			ec.annotateNode(attr, typeName)
		}
	} else {
		for _, attr := range elem.Attributes() {
			if attr.Name() == localName {
				ec.annotateNode(attr, typeName)
				break
			}
		}
	}
	if typeName == "xs:ID" {
		// Register on the document that owns this element (may be a
		// temporary document during variable/function body evaluation).
		out := ec.currentOutput()
		out.doc.RegisterID(value, elem)
	}
}

// annotateNode records a type annotation for a result-tree node.
func (ec *execContext) annotateNode(node helium.Node, typeName string) {
	if typeName == "" {
		return
	}
	if ec.typeAnnotations == nil {
		ec.typeAnnotations = make(map[helium.Node]string)
	}
	ec.typeAnnotations[node] = typeName
}

// transferAnnotations copies the type annotation for srcNode to the most
// recently appended child of the current output node. Used when
// validation="preserve" is in effect during copy-of.
func (ec *execContext) transferAnnotations(srcNode helium.Node) {
	if ec.typeAnnotations == nil {
		return
	}
	ann, ok := ec.typeAnnotations[srcNode]
	if !ok || ann == "" {
		return
	}
	out := ec.currentOutput()
	last := out.current.LastChild()
	if last != nil {
		ec.annotateNode(last, ann)
	}
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
	// When separateTextNodes is set, capture each text node as a separate
	// string item to avoid DOM text-node merging.  This is needed by
	// xsl:value-of with separator + body content so that each produced
	// text value remains a distinct item for separator insertion.
	if out.separateTextNodes && node.Type() == helium.TextNode {
		out.pendingItems = append(out.pendingItems, xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: string(node.Content())})
		return nil
	}
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
	ctx = xpath3.WithVariablesBorrowed(ctx, vars)
	ctx = xpath3.WithFunctionsBorrowed(ctx, ec.xsltFunctions())
	if fnsNS := ec.xsltFunctionsNS(); len(fnsNS) > 0 {
		ctx = xpath3.WithFunctionsNSBorrowed(ctx, fnsNS)
	}
	if len(ec.typeAnnotations) > 0 {
		ctx = xpath3.WithTypeAnnotations(ctx, ec.typeAnnotations)
	}
	if len(ec.stylesheet.namespaces) > 0 || ec.hasXPathDefaultNS {
		ns := make(map[string]string, len(ec.stylesheet.namespaces)+1)
		for k, v := range ec.stylesheet.namespaces {
			// Skip the default namespace binding unless xpath-default-namespace
			// is explicitly set; otherwise the stylesheet's xmlns="..." leaks
			// into XPath name tests and changes how unprefixed names resolve.
			if k == "" && !ec.hasXPathDefaultNS {
				continue
			}
			ns[k] = v
		}
		if ec.hasXPathDefaultNS {
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
	if ec.contextItem != nil {
		ctx = xpath3.WithContextItem(ctx, ec.contextItem)
	}
	// Use the current template's module base URI when available,
	// falling back to the main stylesheet base URI.
	if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
		ctx = xpath3.WithBaseURI(ctx, ec.currentTemplate.BaseURI)
	} else if ec.stylesheet.baseURI != "" {
		ctx = xpath3.WithBaseURI(ctx, ec.stylesheet.baseURI)
	}
	if len(ec.stylesheet.decimalFormats) > 0 {
		// Separate default from named formats
		for qn, df := range ec.stylesheet.decimalFormats {
			if qn == (xpath3.QualifiedName{}) {
				ctx = xpath3.WithDefaultDecimalFormat(ctx, df)
			}
		}
		ctx = xpath3.WithNamedDecimalFormats(ctx, ec.stylesheet.decimalFormats)
	}
	return ctx
}

// baseXPathContext builds the invariant portion of an XPath evaluation context.
// It includes variables, functions, namespace bindings, and the exec context carrier.
// Position, size, and context item are layered by the caller when needed.
func (ec *execContext) baseXPathContext() context.Context {
	vars := ec.collectAllVars()
	ctx := ec.transformCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withExecContext(ctx, ec)
	ctx = xpath3.WithVariablesBorrowed(ctx, vars)
	ctx = xpath3.WithFunctionsBorrowed(ctx, ec.xsltFunctions())
	if fnsNS := ec.xsltFunctionsNS(); len(fnsNS) > 0 {
		ctx = xpath3.WithFunctionsNSBorrowed(ctx, fnsNS)
	}
	if len(ec.typeAnnotations) > 0 {
		ctx = xpath3.WithTypeAnnotations(ctx, ec.typeAnnotations)
	}
	if len(ec.stylesheet.namespaces) > 0 || ec.hasXPathDefaultNS {
		ns := make(map[string]string, len(ec.stylesheet.namespaces)+1)
		for k, v := range ec.stylesheet.namespaces {
			if k == "" && !ec.hasXPathDefaultNS {
				continue
			}
			ns[k] = v
		}
		if ec.hasXPathDefaultNS {
			ns[""] = ec.xpathDefaultNS
		}
		ctx = xpath3.WithNamespaces(ctx, ns)
	}
	// Use the current template's module base URI when available.
	if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
		ctx = xpath3.WithBaseURI(ctx, ec.currentTemplate.BaseURI)
	} else if ec.stylesheet.baseURI != "" {
		ctx = xpath3.WithBaseURI(ctx, ec.stylesheet.baseURI)
	}
	if len(ec.stylesheet.decimalFormats) > 0 {
		for qn, df := range ec.stylesheet.decimalFormats {
			if qn == (xpath3.QualifiedName{}) {
				ctx = xpath3.WithDefaultDecimalFormat(ctx, df)
			}
		}
		ctx = xpath3.WithNamedDecimalFormats(ctx, ec.stylesheet.decimalFormats)
	}
	return ctx
}

// sortXPathEvalState creates a reusable xpath3.EvalState from the base context.
// Used by sort to avoid per-item newEvalContext allocations.
func (ec *execContext) sortXPathEvalState() *xpath3.EvalState {
	return xpath3.NewEvalState(ec.baseXPathContext(), nil)
}

func (ec *execContext) collectAllVars() map[string]xpath3.Sequence {
	// Eagerly evaluate all pending global vars/params, but only at the
	// top level. Nested calls (from within evaluateGlobalVar/Param →
	// newXPathContext → collectAllVars) just snapshot what's available;
	// the lazy lookupVar path handles remaining references on demand.
	if !ec.collectingVars && (len(ec.globalVarDefs) > 0 || len(ec.globalParamDefs) > 0) {
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

	// Fast path: when there are no local variables and globals haven't
	// changed since the last call, return the cached map directly.
	if ec.localVars == nil && ec.cachedVarsGen == ec.globalVarsGen && ec.cachedVarsMap != nil {
		return ec.cachedVarsMap
	}

	vars := make(map[string]xpath3.Sequence, len(ec.globalVars))
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

	// Cache the result when it's globals-only (no local scopes)
	if ec.localVars == nil {
		ec.cachedVarsMap = vars
		ec.cachedVarsGen = ec.globalVarsGen
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
		keyTables:        make(map[string]*keyTable),
		docCache:         make(map[string]*helium.Document),
		accumulatorState: make(map[string]xpath3.Sequence),
		transformCtx:     ctx,
	}

	if cfg != nil && cfg.msgHandler != nil {
		ec.msgHandler = cfg.msgHandler
	}

	// Apply xsl:strip-space to the source document so that whitespace-only
	// text nodes are removed before template matching and XPath evaluation.
	if len(ss.stripSpace) > 0 && source != nil {
		ec.stripWhitespaceFromDoc(source)
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
		xsltInitial := "{" + NSXSLT + "}initial-template"
		if _, ok := ec.stylesheet.namedTemplates[xsltInitial]; ok {
			initialTemplateName = xsltInitial
		} else if _, ok := ec.stylesheet.namedTemplates["xsl:initial-template"]; ok {
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
// castParamValue casts a string param value to the declared XSD type.
// Handles simple atomic types like "xs:integer", "xs:double", "xs:string".
// Occurrence indicators (*, +, ?) are stripped.
func castParamValue(s string, asType string) (xpath3.Sequence, error) {
	// Strip occurrence indicator
	t := strings.TrimRight(asType, "*+?")
	t = strings.TrimSpace(t)
	if t == "" {
		return nil, fmt.Errorf("empty type")
	}
	av, err := xpath3.CastFromString(s, t)
	if err != nil {
		return nil, err
	}
	return xpath3.Sequence{av}, nil
}

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
				val := xpath3.SingleString(sv)
				// Cast to declared type if specified
				if p.As != "" {
					castVal, err := castParamValue(sv, p.As)
					if err == nil {
						val = castVal
					}
				}
				ec.globalVars[p.Name] = val
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
	ctx := ec.transformCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withExecContext(ctx, ec)
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
	ec.globalVarsGen++
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
	ctx := ec.transformCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withExecContext(ctx, ec)
	if p.Select != nil {
		xpathCtx := ec.newXPathContext(ec.sourceDoc)
		result, err := p.Select.Evaluate(xpathCtx, ec.sourceDoc)
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q: %w", p.Name, err)
		}
		val = result.Sequence()
	} else if len(p.Body) > 0 {
		var err error
		if p.As != "" {
			// With as attribute: evaluate as raw sequence
			val, err = ec.evaluateBody(ctx, p.Body)
		} else {
			// No as: wrap in document node (temporary tree)
			val, err = ec.evaluateBodyAsDocument(ctx, p.Body)
		}
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q body: %w", p.Name, err)
		}
	}

	ec.globalVars[p.Name] = val
	ec.globalVarsGen++
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

	// Return all children as node items
	var seq xpath3.Sequence
	for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
		seq = append(seq, xpath3.NodeItem{Node: child})
	}
	// Also collect attributes that were set on the temporary root
	// (e.g., from xsl:attribute with as="attribute()")
	for _, attr := range tmpRoot.Attributes() {
		seq = append(seq, xpath3.NodeItem{Node: attr})
	}
	if len(seq) > 0 {
		return seq, nil
	}

	return xpath3.EmptySequence(), nil
}

// evaluateBodySeparateText is like evaluateBody but keeps each produced
// text node as a separate string item instead of letting the DOM merge
// adjacent text nodes.  This is needed by xsl:value-of with separator
// so that each text value is a distinct item for separator insertion.
func (ec *execContext) evaluateBodySeparateText(ctx context.Context, body []Instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_tmp")
	if err != nil {
		return nil, err
	}
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	frame := &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: true, separateTextNodes: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}

	// Collect DOM children (non-text) and pending items in order
	if tmpRoot.FirstChild() != nil {
		var seq xpath3.Sequence
		for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
			seq = append(seq, xpath3.NodeItem{Node: child})
		}
		seq = append(seq, frame.pendingItems...)
		return seq, nil
	}
	if len(frame.pendingItems) > 0 {
		return frame.pendingItems, nil
	}
	return xpath3.EmptySequence(), nil
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

	// Per XSLT spec: in document-node context (variable/param without as),
	// atomic items from xsl:sequence are converted to text nodes and added
	// to the document node as space-separated text.
	if len(frame.pendingItems) > 0 {
		var sb strings.Builder
		for i, item := range frame.pendingItems {
			if i > 0 {
				sb.WriteByte(' ')
			}
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				_, _ = fmt.Fprint(&sb, item)
				continue
			}
			s, err := xpath3.AtomicToString(av)
			if err != nil {
				_, _ = fmt.Fprint(&sb, item)
			} else {
				sb.WriteString(s)
			}
		}
		if sb.Len() > 0 {
			text, err := tmpDoc.CreateText([]byte(sb.String()))
			if err != nil {
				return nil, err
			}
			if err := tmpDoc.AddChild(text); err != nil {
				return nil, err
			}
		}
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

	// Also check #all mode templates that might not be registered in this mode
	if mode != "#all" {
		for _, tmpl := range ec.stylesheet.modeTemplates["#all"] {
			if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
				return tmpl
			}
		}
	}

	return nil
}

// findAtomicTemplate finds a template matching an atomic value.
// XSLT 3.0 patterns like ".[. instance of xs:integer]" can match atomic items.
func (ec *execContext) findAtomicTemplate(item xpath3.Item, mode string) *Template {
	templates := ec.stylesheet.modeTemplates[mode]
	for _, tmpl := range templates {
		if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, item) {
			return tmpl
		}
	}
	if mode != "#all" {
		for _, tmpl := range ec.stylesheet.modeTemplates["#all"] {
			if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, item) {
				return tmpl
			}
		}
	}
	return nil
}

// matchAtomicPattern checks if an atomic item matches a pattern.
func (ec *execContext) matchAtomicPattern(p *Pattern, item xpath3.Item) bool {
	for _, alt := range p.Alternatives {
		compiled := xpath3.CompileExpr(alt.expr)
		// Evaluate the pattern as a boolean predicate with the item as context
		ctx := ec.newXPathContext(nil)
		ctx = xpath3.WithContextItem(ctx, item)
		result, err := compiled.Evaluate(ctx, nil)
		if err != nil {
			continue
		}
		// The pattern ".[. instance of xs:integer]" evaluates to the item
		// itself when matched, or empty when not. Check non-empty.
		if len(result.Sequence()) > 0 {
			return true
		}
	}
	return false
}

// executeAtomicTemplate executes a template with an atomic item as context.
func (ec *execContext) executeAtomicTemplate(ctx context.Context, tmpl *Template, item xpath3.Item, mode string) error {
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedMode := ec.currentMode
	savedItem := ec.contextItem
	savedTemplate := ec.currentTemplate
	ec.contextItem = item
	ec.currentMode = mode
	ec.currentTemplate = tmpl
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.currentMode = savedMode
		ec.contextItem = savedItem
		ec.currentTemplate = savedTemplate
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	for _, bodyInst := range tmpl.Body {
		if err := ec.executeInstruction(ctx, bodyInst); err != nil {
			return err
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
	savedHasXPathDefaultNS := ec.hasXPathDefaultNS
	savedGroup := ec.currentGroup
	savedGroupKey := ec.currentGroupKey
	ec.currentNode = node
	ec.contextNode = node
	ec.currentMode = mode
	ec.currentTemplate = tmpl
	ec.xpathDefaultNS = tmpl.XPathDefaultNS
	ec.hasXPathDefaultNS = tmpl.XPathDefaultNS != ""
	// XSLT spec: current-group() and current-grouping-key() are only
	// available within the body of xsl:for-each-group, not in templates
	// called from it.
	ec.currentGroup = nil
	ec.currentGroupKey = nil
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.currentMode = savedMode
		ec.currentTemplate = savedTemplate
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.hasXPathDefaultNS = savedHasXPathDefaultNS
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
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
	// Check for xsl:mode on-no-match behavior
	if md := ec.stylesheet.modeDefs[mode]; md != nil {
		return ec.applyOnNoMatch(ctx, node, mode, md.OnNoMatch, paramValues...)
	}
	if mode == "" {
		if md := ec.stylesheet.modeDefs["#default"]; md != nil {
			return ec.applyOnNoMatch(ctx, node, mode, md.OnNoMatch, paramValues...)
		}
	}
	return ec.applyOnNoMatch(ctx, node, mode, "text-only-copy", paramValues...)
}

func (ec *execContext) applyOnNoMatch(ctx context.Context, node helium.Node, mode, behavior string, paramValues ...map[string]xpath3.Sequence) error {
	switch behavior {
	case "shallow-copy":
		return ec.onNoMatchShallowCopy(ctx, node, mode, paramValues...)
	case "deep-copy":
		return ec.onNoMatchDeepCopy(node)
	case "shallow-skip":
		if node.Type() == helium.DocumentNode || node.Type() == helium.ElementNode {
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					return err
				}
			}
		}
		return nil
	case "deep-skip":
		return nil
	case "fail":
		return dynamicError("XTDE0555", "no matching template in mode %q (on-no-match=fail)", mode)
	default: // "text-only-copy"
		return ec.onNoMatchTextOnlyCopy(ctx, node, mode, paramValues...)
	}
}

func (ec *execContext) onNoMatchTextOnlyCopy(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode, helium.ElementNode:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.TextNode, helium.CDATASectionNode:
		if ec.shouldStripWhitespace(node) {
			return nil
		}
		text, err := ec.resultDoc.CreateText(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(text)
	case helium.AttributeNode:
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
		return nil
	}
}

func (ec *execContext) onNoMatchShallowCopy(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.ElementNode:
		srcElem := node.(*helium.Element)
		newElem, err := ec.resultDoc.CreateElement(srcElem.LocalName())
		if err != nil {
			return err
		}
		for _, ns := range srcElem.Namespaces() {
			_ = newElem.DeclareNamespace(ns.Prefix(), ns.URI())
		}
		if srcElem.URI() != "" {
			_ = newElem.SetActiveNamespace(srcElem.Prefix(), srcElem.URI())
		}
		if err := ec.addNode(newElem); err != nil {
			return err
		}
		out := ec.currentOutput()
		savedCurrent := out.current
		out.current = newElem
		defer func() { out.current = savedCurrent }()
		// Apply templates to attributes first (so user templates can
		// intercept attribute nodes, e.g. match="w/@id"), then children.
		for _, attr := range srcElem.Attributes() {
			if err := ec.applyTemplates(ctx, attr, mode, paramValues...); err != nil {
				return err
			}
		}
		for child := srcElem.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
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
		pi, err := ec.resultDoc.CreatePI(node.Name(), string(node.Content()))
		if err != nil {
			return err
		}
		return ec.addNode(pi)
	case helium.AttributeNode:
		attr := node.(*helium.Attribute)
		out := ec.currentOutput()
		if outElem, ok := out.current.(*helium.Element); ok {
			_ = copyAttributeToElement(outElem, attr)
		}
		return nil
	default:
		return nil
	}
}

func (ec *execContext) onNoMatchDeepCopy(node helium.Node) error {
	// Deep copy: copy the entire subtree to the output without template matching.
	switch node.Type() {
	case helium.DocumentNode:
		// For document nodes, deep-copy each child.
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.onNoMatchDeepCopy(child); err != nil {
				return err
			}
		}
		return nil
	case helium.ElementNode, helium.TextNode, helium.CDATASectionNode,
		helium.CommentNode, helium.ProcessingInstructionNode:
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			return err
		}
		return ec.addNode(copied)
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		if outElem, ok := out.current.(*helium.Element); ok {
			return copyAttributeToElement(outElem, attr)
		}
		return nil
	default:
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

// stripWhitespaceFromDoc removes whitespace-only text nodes from a document
// tree according to the stylesheet's xsl:strip-space and xsl:preserve-space rules.
// This is called when loading documents so that XPath evaluation sees the
// correctly stripped tree.
func (ec *execContext) stripWhitespaceFromDoc(doc *helium.Document) {
	ec.stripWhitespaceFromNode(doc)
}

func (ec *execContext) stripWhitespaceFromNode(node helium.Node) {
	child := node.FirstChild()
	for child != nil {
		next := child.NextSibling()
		if ec.shouldStripWhitespace(child) {
			helium.UnlinkNode(child)
		} else {
			ec.stripWhitespaceFromNode(child)
		}
		child = next
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
