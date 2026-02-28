package xpath

import (
	"errors"

	helium "github.com/lestrrat-go/helium"
)

// ErrNotNodeSet is returned when an expression result is not a node-set.
var ErrNotNodeSet = errors.New("xpath: result is not a node-set")

// ErrRecursionLimit is returned when expression evaluation exceeds the
// maximum recursion depth (5000), matching libxml2's XPATH_MAX_RECURSION_DEPTH.
var ErrRecursionLimit = errors.New("xpath: recursion limit exceeded")

// ErrOpLimit is returned when the operation counter exceeds the limit
// configured via Context.OpLimit, matching libxml2's opLimit mechanism.
var ErrOpLimit = errors.New("xpath: operation limit exceeded")

// ErrNodeSetLimit is returned when a node-set exceeds the maximum length
// (10,000,000), matching libxml2's XPATH_MAX_NODESET_LENGTH.
var ErrNodeSetLimit = errors.New("xpath: node-set length limit exceeded")

// ResultType identifies the type of an XPath evaluation result.
type ResultType int

// NodeSetResult, BooleanResult, NumberResult, and StringResult identify the
// four possible XPath result types.
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
	OpLimit    int                    // 0 = unlimited (default); matches libxml2's opLimit
}

// Function is the interface for an XPath function implementation.
// Arguments are pre-evaluated before Eval is called.
type Function interface {
	Eval(ctx FunctionContext, args []*Result) (*Result, error)
}

// FunctionFunc adapts a function to the Function interface.
type FunctionFunc func(ctx FunctionContext, args []*Result) (*Result, error)

// Eval calls f(ctx, args).
func (f FunctionFunc) Eval(ctx FunctionContext, args []*Result) (*Result, error) {
	return f(ctx, args)
}

// FunctionContext provides read-only access to function-evaluation state.
type FunctionContext interface {
	Node() helium.Node
	Position() int
	Size() int
	Namespace(prefix string) (string, bool)
	Variable(name string) (interface{}, bool)
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
		ctx.opLimit = xctx.OpLimit
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
