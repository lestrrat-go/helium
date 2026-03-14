package xslt3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// xsltFunctions returns the XSLT-specific functions that need to be
// registered with the XPath evaluator by local name (no namespace prefix).
func (ec *execContext) xsltFunctions() map[string]xpath3.Function {
	return map[string]xpath3.Function{
		"current":             &xsltFunc{min: 0, max: 0, fn: ec.fnCurrent},
		"document":            &xsltFunc{min: 1, max: 2, fn: ec.fnDocument},
		"key":                 &xsltFunc{min: 2, max: 3, fn: ec.fnKey},
		"generate-id":         &xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID},
		"system-property":     &xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty},
		"unparsed-entity-uri": &xsltFunc{min: 1, max: 1, fn: ec.fnUnparsedEntityURI},
		"element-available":   &xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable},
		"function-available":  &xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable},
		"type-available":      &xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable},
	}
}

type xsltFunc struct {
	min int
	max int
	fn  func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error)
}

func (f *xsltFunc) MinArity() int { return f.min }
func (f *xsltFunc) MaxArity() int { return f.max }
func (f *xsltFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.fn(ctx, args)
}

// current() returns the current node (the node being matched/processed).
func (ec *execContext) fnCurrent(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if ec.currentNode == nil {
		return xpath3.EmptySequence(), nil
	}
	return xpath3.SingleNode(ec.currentNode), nil
}

// document(uri, base?) loads an external XML document.
func (ec *execContext) fnDocument(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	uri, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Empty string means the stylesheet document itself (XSLT spec §14.1)
	if uri == "" {
		return xpath3.SingleNode(ec.stylesheet.sourceDoc), nil
	}

	// Check cache
	if doc, ok := ec.docCache[uri]; ok {
		return xpath3.SingleNode(doc), nil
	}

	// Resolve relative URI against stylesheet base URI
	resolvedURI := uri
	if ec.stylesheet.baseURI != "" && !strings.Contains(uri, "://") && !filepath.IsAbs(uri) {
		baseDir := filepath.Dir(ec.stylesheet.baseURI)
		resolvedURI = filepath.Join(baseDir, uri)
	}
	data, err := os.ReadFile(resolvedURI)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot load document %q: %v", uri, err)
	}

	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot parse document %q: %v", uri, err)
	}

	if ec.docCache == nil {
		ec.docCache = make(map[string]*helium.Document)
	}
	ec.docCache[uri] = doc
	return xpath3.SingleNode(doc), nil
}

// key(name, value, doc?) looks up nodes by key.
func (ec *execContext) fnKey(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) < 2 {
		return nil, dynamicError(errCodeXTDE1170, "key() requires at least 2 arguments")
	}

	nameAV, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(nameAV)
	if err != nil {
		return nil, err
	}

	if len(args[1]) == 0 {
		return xpath3.EmptySequence(), nil
	}
	valAV, err := xpath3.AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	value, err := xpath3.AtomicToString(valAV)
	if err != nil {
		return nil, err
	}

	nodes, err := ec.lookupKey(name, value)
	if err != nil {
		return nil, err
	}

	seq := make(xpath3.Sequence, len(nodes))
	for i, n := range nodes {
		seq[i] = xpath3.NodeItem{Node: n}
	}
	return seq, nil
}

// generate-id(node?) returns a unique string identifier for a node.
func (ec *execContext) fnGenerateID(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var node helium.Node
	if len(args) == 0 || len(args[0]) == 0 {
		node = ec.contextNode
	} else {
		ni, ok := args[0][0].(xpath3.NodeItem)
		if !ok {
			return nil, dynamicError("XPTY0004", "generate-id() argument must be a node")
		}
		node = ni.Node
	}
	if node == nil {
		return xpath3.SingleString(""), nil
	}
	// Use fmt.Sprintf %p to get a unique pointer-based ID
	id := fmt.Sprintf("id%p", node)
	return xpath3.SingleString(id), nil
}

// system-property(name) returns XSLT system properties.
func (ec *execContext) fnSystemProperty(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Strip prefix
	local := name
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		local = name[idx+1:]
	}

	switch local {
	case "version":
		return xpath3.SingleString("3.0"), nil
	case "vendor":
		return xpath3.SingleString("helium"), nil
	case "vendor-url":
		return xpath3.SingleString("https://github.com/lestrrat-go/helium"), nil
	case "product-name":
		return xpath3.SingleString("helium-xslt3"), nil
	case "product-version":
		return xpath3.SingleString("0.1"), nil
	case "is-schema-aware":
		return xpath3.SingleString("no"), nil
	case "supports-serialization":
		return xpath3.SingleString("yes"), nil
	case "supports-backwards-compatibility":
		return xpath3.SingleString("yes"), nil
	default:
		return xpath3.SingleString(""), nil
	}
}

// unparsed-entity-uri(name) returns the URI of an unparsed entity.
func (ec *execContext) fnUnparsedEntityURI(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Not commonly used; return empty string
	return xpath3.SingleString(""), nil
}

// xslUserFunc wraps an xsl:function for use as an xpath3.Function.
type xslUserFunc struct {
	def *XSLFunction
	ec  *execContext
}

func (f *xslUserFunc) MinArity() int { return len(f.def.Params) }
func (f *xslUserFunc) MaxArity() int { return len(f.def.Params) }

func (f *xslUserFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Retrieve the XSLT exec context from the context.Context
	ec := f.ec
	if ecFromCtx := getExecContext(ctx); ecFromCtx != nil {
		ec = ecFromCtx
	}

	// Recursion depth check
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return nil, dynamicError(errCodeXTDE0820, "recursion depth exceeded in xsl:function %s", f.def.Name.Name)
	}
	defer func() { ec.depth-- }()

	// Save and restore execution state
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedPos := ec.position
	savedSize := ec.size
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.position = savedPos
		ec.size = savedSize
	}()

	// Push new variable scope for parameters
	ec.pushVarScope()
	defer ec.popVarScope()

	// Bind parameters
	for i, param := range f.def.Params {
		if i < len(args) {
			ec.setVar(param.Name, args[i])
		} else if param.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := param.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return nil, err
			}
			ec.setVar(param.Name, result.Sequence())
		} else {
			ec.setVar(param.Name, xpath3.EmptySequence())
		}
	}

	// Execute the function body, collecting result into a temporary document
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, _ := tmpDoc.CreateElement("_xsl_fn_result")
	_ = tmpDoc.SetDocumentElement(tmpRoot)

	ec.outputStack = append(ec.outputStack, &outputFrame{current: tmpRoot, doc: tmpDoc})
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range f.def.Body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}

	// Collect results from the temporary tree
	return ec.collectSequenceFromNode(tmpRoot), nil
}

// collectSequenceFromNode converts children of a node to an XPath sequence.
func (ec *execContext) collectSequenceFromNode(node helium.Node) xpath3.Sequence {
	var seq xpath3.Sequence
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			text := string(child.Content())
			seq = append(seq, xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: text})
		default:
			seq = append(seq, xpath3.NodeItem{Node: child})
		}
	}
	return seq
}

// XSLT instruction elements recognized by element-available().
var xsltElements = map[string]struct{}{
	"analyze-string": {}, "apply-imports": {}, "apply-templates": {},
	"attribute": {}, "call-template": {}, "choose": {}, "comment": {},
	"copy": {}, "copy-of": {}, "document": {}, "element": {},
	"fallback": {}, "for-each": {}, "for-each-group": {}, "if": {},
	"import": {}, "include": {}, "message": {}, "namespace": {},
	"next-match": {}, "number": {}, "otherwise": {}, "output": {},
	"param": {}, "perform-sort": {}, "processing-instruction": {},
	"result-document": {}, "sequence": {}, "sort": {}, "strip-space": {},
	"preserve-space": {}, "stylesheet": {}, "template": {}, "text": {},
	"transform": {}, "value-of": {}, "variable": {}, "when": {},
	"with-param": {}, "try": {}, "catch": {}, "where-populated": {},
	"on-empty": {}, "on-non-empty": {}, "merge": {}, "merge-source": {},
	"merge-action": {}, "merge-key": {}, "assert": {}, "accumulator": {},
	"accumulator-rule": {}, "fork": {}, "iterate": {}, "break": {},
	"next-iteration": {}, "map": {}, "map-entry": {}, "array": {},
	"accept": {}, "expose": {}, "override": {}, "use-package": {},
	"package": {}, "global-context-item": {}, "context-item": {},
	"source-document": {},
}

// element-available(name) returns true if the named XSLT element is available.
func (ec *execContext) fnElementAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// Resolve prefix:local to namespace + local
	local := name
	ns := NSXSLT
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local = name[idx+1:]
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			ns = uri
		}
	}
	if ns != NSXSLT {
		return xpath3.SingleBoolean(false), nil
	}
	_, ok := xsltElements[local]
	return xpath3.SingleBoolean(ok), nil
}

// function-available(name, arity?) returns true if the named function is available.
func (ec *execContext) fnFunctionAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// Check XSLT functions by local name
	fns := ec.xsltFunctions()
	if _, ok := fns[name]; ok {
		return xpath3.SingleBoolean(true), nil
	}

	// Check user-defined functions (prefixed)
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			qn := xpath3.QualifiedName{URI: uri, Name: local}
			if _, ok := ec.stylesheet.functions[qn]; ok {
				return xpath3.SingleBoolean(true), nil
			}
		}
	}

	// Check XPath built-in functions — they're always available.
	// For simplicity, return true for common fn: functions.
	return xpath3.SingleBoolean(false), nil
}

// type-available(name) — not supported (no schema awareness), always returns false.
func (ec *execContext) fnTypeAvailable(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	return xpath3.SingleBoolean(false), nil
}

// xsltFunctionsNS returns user-defined xsl:function definitions as xpath3 functions
// keyed by qualified name.
func (ec *execContext) xsltFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	if len(ec.stylesheet.functions) == 0 {
		return nil
	}
	fns := make(map[xpath3.QualifiedName]xpath3.Function, len(ec.stylesheet.functions))
	for qn, def := range ec.stylesheet.functions {
		fns[qn] = &xslUserFunc{def: def, ec: ec}
	}
	return fns
}

