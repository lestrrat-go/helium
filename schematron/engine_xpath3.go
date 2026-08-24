package schematron

import (
	"context"
	"maps"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// xpath3Engine backs the XPath 3.1 query language bindings ("xslt3",
// "xpath3", "xpath31").
//
// Resource access is denied by default: fn:doc, fn:collection and
// fn:unparsed-text retrieve nothing unless the evaluator is given a URI
// resolver or an HTTP client, and this engine gives it neither. A schema
// therefore cannot read a file or reach the network through its expressions.
type xpath3Engine struct{}

func (xpath3Engine) compile(src string) (compiledExpr, error) {
	compiled, err := xpath3.NewCompiler().Compile(src)
	if err != nil {
		return nil, err
	}
	return &xpath3Expr{src: src, expr: compiled}, nil
}

func (xpath3Engine) runner(namespaces map[string]string) runner {
	// One document-order cache is shared by every expression evaluated
	// against the instance, so ordering work is done once per validation
	// rather than once per rule.
	ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(namespaces).
		DocOrderCache(xpath3.NewDocOrderCache())
	return xpath3Runner{ev: ev}
}

type xpath3Expr struct {
	src  string
	expr *xpath3.Expression
}

func (e *xpath3Expr) source() string { return e.src }

type xpath3Runner struct {
	ev   xpath3.Evaluator
	vars map[string]xpath3.Sequence
}

func (r xpath3Runner) evaluate(ctx context.Context, expr compiledExpr, node helium.Node) (value, error) {
	e, ok := expr.(*xpath3Expr)
	if !ok {
		return nil, errWrongEngine
	}
	result, err := r.ev.Evaluate(ctx, e.expr, node)
	if err != nil {
		return nil, err
	}
	return xpath3Value{result: result}, nil
}

func (r xpath3Runner) bind(name string, v value) runner {
	xv, ok := v.(xpath3Value)
	if !ok {
		return r
	}
	// xpath3.Evaluator takes the whole variable map at once, so each binding
	// extends a copy: the runner a rule started from keeps its own bindings.
	vars := maps.Clone(r.vars)
	if vars == nil {
		vars = make(map[string]xpath3.Sequence, 1)
	}
	vars[name] = xv.result.Sequence()
	return xpath3Runner{ev: r.ev.Variables(vars), vars: vars}
}

type xpath3Value struct {
	result *xpath3.Result
}

// nodeSet returns the result nodes, or nil when the result holds anything
// that is not a node.
func (v xpath3Value) nodeSet() []helium.Node {
	nodes, err := v.result.Nodes()
	if err != nil {
		return nil
	}
	return nodes
}

// effectiveBoolean applies the XPath 3.1 effective boolean value rules. A
// sequence of more than one item that starts with an atomic value has no
// effective boolean value, and the returned error carries FORG0006.
func (v xpath3Value) effectiveBoolean() (bool, error) {
	return xpath3.EBV(v.result.Sequence())
}

// stringValue is the XSLT 2.0 and later <xsl:value-of> conversion: every item
// in the sequence is atomized and the results joined with a single space.
// XPath 1.0 keeps only the first node instead.
func (v xpath3Value) stringValue() string {
	return v.result.StringValue()
}

// nodeName returns the name of the first node in the result.
func (v xpath3Value) nodeName() string {
	nodes, err := v.result.Nodes()
	if err != nil || len(nodes) == 0 {
		return ""
	}
	return nodeDisplayName(nodes[0])
}
