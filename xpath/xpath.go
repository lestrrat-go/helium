package xpath

import (
	"errors"

	helium "github.com/lestrrat-go/helium"
)

// ErrNotNodeSet is returned when an expression result is not a node-set.
var ErrNotNodeSet = errors.New("xpath: result is not a node-set")

// ResultType identifies the type of an XPath evaluation result.
type ResultType int

const (
	NodeSetResult ResultType = iota
	BooleanResult
	NumberResult
	StringResult
)

// Result holds the outcome of an XPath evaluation.
type Result struct {
	Type    ResultType
	NodeSet []helium.Node
	Boolean bool
	Number  float64
	String  string
}

// Context carries namespace bindings and variable bindings
// for expression evaluation.
type Context struct {
	Namespaces map[string]string      // prefix → URI
	Variables  map[string]interface{} // name → value ([]helium.Node, string, float64, bool)
}

// Expression is a compiled XPath expression, reusable across evaluations.
type Expression struct {
	source string
	ast    Expr
}

// Compile parses an XPath expression string into a reusable Expression.
func Compile(expr string) (*Expression, error) {
	ast, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return &Expression{source: expr, ast: ast}, nil
}

// MustCompile is like Compile but panics on error.
func MustCompile(expr string) *Expression {
	e, err := Compile(expr)
	if err != nil {
		panic("xpath: Compile(" + expr + "): " + err.Error())
	}
	return e
}

// Evaluate evaluates the compiled expression against the given context node.
func (e *Expression) Evaluate(node helium.Node) (*Result, error) {
	ctx := newEvalContext(node)
	return eval(ctx, e.ast)
}

// EvaluateWithContext evaluates with explicit namespace/variable bindings.
func (e *Expression) EvaluateWithContext(node helium.Node, xctx *Context) (*Result, error) {
	ctx := newEvalContext(node)
	if xctx != nil {
		ctx.namespaces = xctx.Namespaces
		ctx.variables = xctx.Variables
	}
	return eval(ctx, e.ast)
}

// String returns the original XPath expression string.
func (e *Expression) String() string {
	return e.source
}

// Find is a convenience function: compile + evaluate, returning a node-set.
// Returns an error if the expression does not evaluate to a node-set.
func Find(node helium.Node, expr string) ([]helium.Node, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	r, err := compiled.Evaluate(node)
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, ErrNotNodeSet
	}
	return r.NodeSet, nil
}

// Evaluate is a convenience function: compile + evaluate in one call.
func Evaluate(node helium.Node, expr string) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(node)
}

// EvaluateWithContext evaluates with explicit namespace/variable bindings.
func EvaluateWithContext(node helium.Node, expr string, xctx *Context) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.EvaluateWithContext(node, xctx)
}
