package xpath

import (
	"errors"
	"fmt"

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

// ErrUnknownFunction is returned when a function call references an
// unregistered function name.
var ErrUnknownFunction = errors.New("xpath: unknown function")

// ErrUnknownFunctionNamespace is returned when a namespaced function call
// uses a prefix that cannot be resolved to a namespace URI.
var ErrUnknownFunctionNamespace = errors.New("xpath: unknown function namespace prefix")

// ErrUnsupportedExpr is returned when an unsupported expression type is encountered.
var ErrUnsupportedExpr = errors.New("xpath: unsupported expression type")

// ErrUnsupportedBinaryOp is returned when an unsupported binary operator is encountered.
var ErrUnsupportedBinaryOp = errors.New("xpath: unsupported binary operator")

// ErrUndefinedVariable is returned when a variable reference cannot be resolved.
var ErrUndefinedVariable = errors.New("xpath: undefined variable")

// ErrUnsupportedVariableType is returned when a variable holds an unsupported type.
var ErrUnsupportedVariableType = errors.New("xpath: unsupported variable type")

// ErrFilterNotNodeSet is returned when a filter expression is applied to a non-node-set.
var ErrFilterNotNodeSet = errors.New("xpath: filter expression requires a node-set")

// ErrUnionNotNodeSet is returned when the union operator is applied to non-node-sets.
var ErrUnionNotNodeSet = errors.New("xpath: union operator requires node-sets")

// ErrPathNotNodeSet is returned when a path expression base is not a node-set.
var ErrPathNotNodeSet = errors.New("xpath: path expression requires a node-set")

// ErrInvalidFunctionContext is returned when a function receives an unexpected context type.
var ErrInvalidFunctionContext = errors.New("xpath: invalid function context type")

// ErrExprTooDeep is returned when expression nesting exceeds the maximum parse depth.
var ErrExprTooDeep = errors.New("xpath: expression nesting too deep")

// ErrUnexpectedToken is returned when the parser encounters an unexpected token.
var ErrUnexpectedToken = errors.New("xpath: unexpected token")

// ErrUnexpectedChar is returned when the lexer encounters an unexpected character.
var ErrUnexpectedChar = errors.New("xpath: unexpected character")

// ErrUnterminatedString is returned when a string literal is not closed.
var ErrUnterminatedString = errors.New("xpath: unterminated string")

// ErrUnknownAxis is returned when an axis name is unrecognized.
var ErrUnknownAxis = errors.New("xpath: unknown axis")

// ErrExpectedToken is returned when the parser expected a specific token but found another.
var ErrExpectedToken = errors.New("xpath: expected token")

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
// (libxml2: xmlXPathObject)
type Result struct {
	Type    ResultType
	NodeSet []helium.Node
	Bool bool
	Number  float64
	String  string
}

// QualifiedName is a namespace-qualified name used as a key for
// registering namespace-qualified XPath functions.
type QualifiedName struct {
	URI  string
	Name string
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
	Variable(name string) (any, bool)
}

// Context carries namespace bindings, variable bindings, and custom function
// registrations for expression evaluation.
// (libxml2: xmlXPathContext)
type Context struct {
	Namespaces  map[string]string          // prefix → URI
	Variables   map[string]any     // name → value ([]helium.Node, string, float64, bool)
	OpLimit     int                        // 0 = unlimited (default); matches libxml2's opLimit
	Functions   map[string]Function        // unqualified custom functions
	FunctionsNS map[QualifiedName]Function // namespace-qualified custom functions
}

// RegisterFunction registers an unqualified custom XPath function.
// Returns an error if the name conflicts with a built-in function.
func (c *Context) RegisterFunction(name string, fn Function) error {
	if _, ok := builtinFunctions[name]; ok {
		return fmt.Errorf("cannot override built-in function %q: %w", name, ErrUnknownFunction)
	}
	if c.Functions == nil {
		c.Functions = make(map[string]Function)
	}
	c.Functions[name] = fn
	return nil
}

// RegisterFunctionNS registers a namespace-qualified custom XPath function.
func (c *Context) RegisterFunctionNS(uri, name string, fn Function) {
	if c.FunctionsNS == nil {
		c.FunctionsNS = make(map[QualifiedName]Function)
	}
	c.FunctionsNS[QualifiedName{URI: uri, Name: name}] = fn
}

// Expression is a compiled XPath expression, reusable across evaluations.
type Expression struct {
	source string
	ast    Expr
}

// Compile parses an XPath expression string into a reusable Expression.
// (libxml2: xmlXPathCompile / xmlXPathCtxtCompile)
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
// (libxml2: xmlXPathCompiledEval)
func (e *Expression) Evaluate(node helium.Node) (*Result, error) {
	ctx := newEvalContext(node)
	return eval(ctx, e.ast)
}

// EvaluateWith evaluates with explicit namespace/variable bindings.
func (e *Expression) EvaluateWith(node helium.Node, xctx *Context) (*Result, error) {
	ctx := newEvalContext(node)
	if xctx != nil {
		ctx.namespaces = xctx.Namespaces
		ctx.variables = xctx.Variables
		ctx.opLimit = xctx.OpLimit
		ctx.functions = xctx.Functions
		ctx.functionsNS = xctx.FunctionsNS
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
// (libxml2: xmlXPathCompiledEval)
func Evaluate(node helium.Node, expr string) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(node)
}

// EvaluateWith evaluates with explicit namespace/variable bindings.
func EvaluateWith(node helium.Node, expr string, xctx *Context) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.EvaluateWith(node, xctx)
}
