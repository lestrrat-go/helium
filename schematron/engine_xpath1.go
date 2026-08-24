package schematron

import (
	"context"
	"math"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/lestrrat-go/helium/internal/xpath1/number"
	"github.com/lestrrat-go/helium/xpath1"
)

// xpath1Engine backs the XPath 1.0 query language bindings ("xslt", "xslt1",
// "xpath", "xpath1", and the absent-attribute default). Its conversions
// follow libxml2's schematron.c so golden output stays byte-identical.
type xpath1Engine struct{}

func (xpath1Engine) compile(src string) (compiledExpr, error) {
	compiled, err := xpath1.Compile(src)
	if err != nil {
		return nil, err
	}
	return &xpath1Expr{src: src, expr: compiled}, nil
}

func (xpath1Engine) runner(namespaces map[string]string) runner {
	// One document-order cache is shared by every expression evaluated
	// against the instance. Without it xpath1 indexes the whole document
	// again on every evaluation, which makes validation quadratic in
	// document size.
	return xpath1Runner{ev: xpath1.NewEvaluator().Namespaces(namespaces), docOrder: &ixpath.DocOrderCache{}}
}

type xpath1Expr struct {
	src  string
	expr *xpath1.Expression
}

func (e *xpath1Expr) source() string { return e.src }

type xpath1Runner struct {
	ev       xpath1.Evaluator
	docOrder *ixpath.DocOrderCache
}

func (r xpath1Runner) evaluate(ctx context.Context, expr compiledExpr, node helium.Node) (value, error) {
	e, ok := expr.(*xpath1Expr)
	if !ok {
		return nil, errWrongEngine
	}
	result, err := r.ev.Evaluate(ixpath.WithDocOrderCache(ctx, r.docOrder), e.expr, node)
	if err != nil {
		return nil, err
	}
	return xpath1Value{result: result}, nil
}

func (r xpath1Runner) bind(name string, v value) runner {
	xv, ok := v.(xpath1Value)
	if !ok {
		return r
	}
	return xpath1Runner{ev: r.ev.AdditionalVariables(map[string]any{name: xv.variable()}), docOrder: r.docOrder}
}

type xpath1Value struct {
	result *xpath1.Result
}

func (v xpath1Value) nodeSet() []helium.Node {
	if v.result.Type != xpath1.NodeSetResult {
		return nil
	}
	return v.result.NodeSet
}

// effectiveBoolean converts an XPath 1.0 result to a boolean. XPath 1.0
// boolean conversion is total, so the error is always nil.
func (v xpath1Value) effectiveBoolean() (bool, error) {
	switch v.result.Type {
	case xpath1.BooleanResult:
		return v.result.Bool, nil
	case xpath1.NumberResult:
		return v.result.Number != 0 && !math.IsNaN(v.result.Number), nil
	case xpath1.StringResult:
		return v.result.String != "", nil
	case xpath1.NodeSetResult:
		return len(v.result.NodeSet) > 0, nil
	}
	return false, nil
}

// stringValue converts an XPath 1.0 result to a string.
func (v xpath1Value) stringValue() string {
	switch v.result.Type {
	case xpath1.StringResult:
		return v.result.String
	case xpath1.NumberResult:
		// XPath 1.0 string(number): reuse the canonical xmlXPathFormatNumber
		// port (integers without a decimal point, "NaN"/"Infinity"/"-Infinity",
		// no trailing ".0") instead of Go's default formatting.
		return number.ToString(v.result.Number)
	case xpath1.BooleanResult:
		// XPath 1.0 string(boolean): "true"/"false" (lowercase).
		if v.result.Bool {
			return "true"
		}
		return "false"
	case xpath1.NodeSetResult:
		if len(v.result.NodeSet) == 0 {
			return ""
		}
		// XPath 1.0: a node-set converts to the string-value of the node
		// first in document order.
		return ixpath.StringValue(v.result.NodeSet[0])
	}
	return ""
}

// nodeName extracts a node name from an XPath 1.0 result.
func (v xpath1Value) nodeName() string {
	if v.result.Type != xpath1.NodeSetResult || len(v.result.NodeSet) == 0 {
		return ""
	}
	return nodeDisplayName(v.result.NodeSet[0])
}

// variable converts a result to a value suitable for variable binding.
func (v xpath1Value) variable() any {
	switch v.result.Type {
	case xpath1.NodeSetResult:
		return v.result.NodeSet
	case xpath1.StringResult:
		return v.result.String
	case xpath1.NumberResult:
		return v.result.Number
	case xpath1.BooleanResult:
		return v.result.Bool
	}
	return nil
}
