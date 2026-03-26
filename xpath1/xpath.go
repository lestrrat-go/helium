// Package xpath1 implements XPath 1.0 expression parsing and evaluation
// against helium XML document trees.
package xpath1

import (
	"context"
	"errors"
	"maps"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
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
// This is an alias to the internal sentinel so that errors.Is works
// consistently across xpath1 and xpath3.
var ErrNodeSetLimit = ixpath.ErrNodeSetLimit

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
	Bool    bool
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
// The context.Context carries both the FunctionContext (retrievable via
// GetFunctionContext) and any caller-provided values.
type Function interface {
	Eval(ctx context.Context, args []*Result) (*Result, error)
}

// FunctionFunc adapts a function to the Function interface.
type FunctionFunc func(ctx context.Context, args []*Result) (*Result, error)

// Eval calls f(ctx, args).
func (f FunctionFunc) Eval(ctx context.Context, args []*Result) (*Result, error) {
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

type evalConfigKey struct{}
type functionContextKey struct{}

// withFunctionContext stores a FunctionContext in a context.Context.
func withFunctionContext(ctx context.Context, fctx FunctionContext) context.Context {
	return context.WithValue(ctx, functionContextKey{}, fctx)
}

// GetFunctionContext retrieves the FunctionContext from a context.Context.
// Returns nil if none is set.
func GetFunctionContext(ctx context.Context) FunctionContext {
	fctx, _ := ctx.Value(functionContextKey{}).(FunctionContext)
	return fctx
}

type evalConfig struct {
	namespaces  map[string]string          // prefix -> URI
	variables   map[string]any             // name -> value ([]helium.Node, string, float64, bool)
	opLimit     int                        // 0 = unlimited (default); matches libxml2's opLimit
	functions   map[string]Function        // unqualified custom functions
	functionsNS map[QualifiedName]Function // namespace-qualified custom functions
}

func getEvalConfig(ctx context.Context) *evalConfig {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(evalConfigKey{}).(*evalConfig)
	return c
}

func deriveEvalConfig(ctx context.Context) *evalConfig {
	if c := getEvalConfig(ctx); c != nil {
		return c.clone()
	}
	return &evalConfig{}
}

func withEvalConfig(ctx context.Context, cfg *evalConfig) context.Context {
	return context.WithValue(ctx, evalConfigKey{}, cfg)
}

func updateEvalConfig(ctx context.Context, fn func(*evalConfig)) context.Context {
	cfg := deriveEvalConfig(ctx)
	fn(cfg)
	return withEvalConfig(ctx, cfg)
}

func (c *evalConfig) clone() *evalConfig {
	if c == nil {
		return &evalConfig{}
	}
	cp := *c
	cp.namespaces = maps.Clone(c.namespaces)
	cp.variables = maps.Clone(c.variables)
	cp.functions = maps.Clone(c.functions)
	cp.functionsNS = maps.Clone(c.functionsNS)
	return &cp
}

// Deprecated: Use Evaluator.Namespaces instead.
func WithNamespaces(ctx context.Context, ns map[string]string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.namespaces = maps.Clone(ns)
	})
}

// Deprecated: Use Evaluator.AdditionalNamespaces instead.
func WithAdditionalNamespaces(ctx context.Context, ns map[string]string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.namespaces == nil {
			c.namespaces = make(map[string]string, len(ns))
		}
		for k, v := range ns {
			c.namespaces[k] = v
		}
	})
}

// Deprecated: Use Evaluator.Variables instead.
func WithVariables(ctx context.Context, vars map[string]any) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.variables = maps.Clone(vars)
	})
}

// Deprecated: Use Evaluator.AdditionalVariables instead.
func WithAdditionalVariables(ctx context.Context, vars map[string]any) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.variables == nil {
			c.variables = make(map[string]any, len(vars))
		}
		for k, v := range vars {
			c.variables[k] = v
		}
	})
}

// Deprecated: Use Evaluator.OpLimit instead.
func WithOpLimit(ctx context.Context, limit int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.opLimit = limit
	})
}

// Deprecated: Use Evaluator.Function instead.
func WithFunction(ctx context.Context, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.functions == nil {
			c.functions = make(map[string]Function)
		}
		c.functions[name] = fn
	})
}

// Deprecated: Use Evaluator.Function instead.
func WithFunctions(ctx context.Context, fns map[string]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.functions = maps.Clone(fns)
	})
}

// Deprecated: Use Evaluator.FunctionNS instead.
func WithFunctionNS(ctx context.Context, uri, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.functionsNS == nil {
			c.functionsNS = make(map[QualifiedName]Function)
		}
		c.functionsNS[QualifiedName{URI: uri, Name: name}] = fn
	})
}

// Deprecated: Use Evaluator.FunctionNS instead.
func WithFunctionsNS(ctx context.Context, fns map[QualifiedName]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.functionsNS = maps.Clone(fns)
	})
}

// Evaluator evaluates compiled XPath 1.0 expressions against nodes.
// Configuration is set via fluent clone-on-write methods that return
// a new Evaluator with the updated setting, mirroring the xpath3 API.
type Evaluator struct {
	cfg *evalConfig
}

// NewEvaluator returns a new Evaluator with default (empty) configuration.
func NewEvaluator() Evaluator {
	return Evaluator{cfg: &evalConfig{}}
}

func (e Evaluator) clone() Evaluator {
	return Evaluator{cfg: e.cfg.clone()}
}

// Namespaces returns a new Evaluator with the given namespace prefix->URI bindings,
// replacing any previously set namespaces.
func (e Evaluator) Namespaces(ns map[string]string) Evaluator {
	out := e.clone()
	out.cfg.namespaces = maps.Clone(ns)
	return out
}

// AdditionalNamespaces returns a new Evaluator with the given namespace bindings
// merged into the existing set.
func (e Evaluator) AdditionalNamespaces(ns map[string]string) Evaluator {
	out := e.clone()
	if out.cfg.namespaces == nil {
		out.cfg.namespaces = make(map[string]string, len(ns))
	}
	for k, v := range ns {
		out.cfg.namespaces[k] = v
	}
	return out
}

// Variables returns a new Evaluator with the given variable name->value bindings,
// replacing any previously set variables.
func (e Evaluator) Variables(vars map[string]any) Evaluator {
	out := e.clone()
	out.cfg.variables = maps.Clone(vars)
	return out
}

// AdditionalVariables returns a new Evaluator with the given variable bindings
// merged into the existing set.
func (e Evaluator) AdditionalVariables(vars map[string]any) Evaluator {
	out := e.clone()
	if out.cfg.variables == nil {
		out.cfg.variables = make(map[string]any, len(vars))
	}
	for k, v := range vars {
		out.cfg.variables[k] = v
	}
	return out
}

// OpLimit returns a new Evaluator with the given operation counter limit.
// A value of 0 means unlimited (the default).
func (e Evaluator) OpLimit(n int) Evaluator {
	out := e.clone()
	out.cfg.opLimit = n
	return out
}

// Function returns a new Evaluator with the given unqualified custom function registered.
func (e Evaluator) Function(name string, fn Function) Evaluator {
	out := e.clone()
	if out.cfg.functions == nil {
		out.cfg.functions = make(map[string]Function)
	}
	out.cfg.functions[name] = fn
	return out
}

// FunctionNS returns a new Evaluator with the given namespace-qualified custom function registered.
func (e Evaluator) FunctionNS(uri, name string, fn Function) Evaluator {
	out := e.clone()
	if out.cfg.functionsNS == nil {
		out.cfg.functionsNS = make(map[QualifiedName]Function)
	}
	out.cfg.functionsNS[QualifiedName{URI: uri, Name: name}] = fn
	return out
}

// Evaluate evaluates a compiled expression against the given context node.
func (e Evaluator) Evaluate(ctx context.Context, expr *Expression, node helium.Node) (*Result, error) {
	ectx := newEvalContextWithConfig(ctx, node, e.cfg)
	return eval(ectx, expr.ast)
}

// Find evaluates a compiled expression and returns the resulting node-set.
// Returns ErrNotNodeSet if the expression does not evaluate to a node-set.
func (e Evaluator) Find(ctx context.Context, expr *Expression, node helium.Node) ([]helium.Node, error) {
	r, err := e.Evaluate(ctx, expr, node)
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, ErrNotNodeSet
	}
	return r.NodeSet, nil
}

// Expression is a compiled XPath expression, reusable across evaluations.
type Expression struct {
	source string
	ast    Expr
}

// Compiler compiles XPath 1.0 expression strings into reusable Expression
// values, mirroring the xpath3.Compiler API.
type Compiler struct{}

// NewCompiler returns a new Compiler.
func NewCompiler() Compiler {
	return Compiler{}
}

// Compile parses an XPath expression string into a reusable Expression.
func (Compiler) Compile(expr string) (*Expression, error) {
	ast, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return &Expression{source: expr, ast: ast}, nil
}

// MustCompile is like Compile but panics on error.
func (c Compiler) MustCompile(expr string) *Expression {
	e, err := c.Compile(expr)
	if err != nil {
		panic("xpath: Compile(" + expr + "): " + err.Error())
	}
	return e
}

// Compile parses an XPath expression string into a reusable Expression.
// It is a convenience wrapper around NewCompiler().Compile().
// (libxml2: xmlXPathCompile / xmlXPathCtxtCompile)
func Compile(expr string) (*Expression, error) {
	return NewCompiler().Compile(expr)
}

// MustCompile is like Compile but panics on error.
// It is a convenience wrapper around NewCompiler().MustCompile().
func MustCompile(expr string) *Expression {
	return NewCompiler().MustCompile(expr)
}

// Evaluate evaluates the compiled expression against the given context node.
// The context.Context may carry xpath1 evaluation settings attached via
// WithNamespaces, WithVariables, WithFunction(s), or WithOpLimit.
// (libxml2: xmlXPathCompiledEval)
func (e *Expression) Evaluate(ctx context.Context, node helium.Node) (*Result, error) {
	ectx := newEvalContext(ctx, node)
	return eval(ectx, e.ast)
}

// String returns the original XPath expression string.
func (e *Expression) String() string {
	return e.source
}

// Find is a convenience function: compile + evaluate, returning a node-set.
// Returns an error if the expression does not evaluate to a node-set.
func Find(ctx context.Context, node helium.Node, expr string) ([]helium.Node, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	r, err := compiled.Evaluate(ctx, node)
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
func Evaluate(ctx context.Context, node helium.Node, expr string) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(ctx, node)
}
