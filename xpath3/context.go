package xpath3

import (
	"context"
	"time"
)

type contextKey struct{}

// fnContextKey is used exclusively by the unexported withFnContext/getFnContext
// pair to pass evalContext to built-in function implementations. It is never
// exposed to external callers.
type fnContextKey struct{}

// QualifiedName identifies a function in a specific namespace.
type QualifiedName struct {
	URI  string
	Name string
}

// Context holds XPath evaluation settings provided by the caller.
type Context struct {
	namespaces       map[string]string
	variables        map[string]Sequence
	functions        map[string]Function
	functionsNS      map[QualifiedName]Function
	opLimit          int
	implicitTimezone *time.Location
}

// ContextOption configures a Context.
type ContextOption func(*Context)

// WithNamespaces binds namespace prefixes to URIs for the evaluation.
func WithNamespaces(ns map[string]string) ContextOption {
	return func(c *Context) {
		c.namespaces = ns
	}
}

// WithVariables binds variable names to pre-constructed Sequence values.
func WithVariables(vars map[string]Sequence) ContextOption {
	return func(c *Context) {
		c.variables = vars
	}
}

// WithOpLimit sets the maximum number of operations before the evaluator
// returns ErrOpLimit. Zero means unlimited.
func WithOpLimit(limit int) ContextOption {
	return func(c *Context) {
		c.opLimit = limit
	}
}

// WithFunctions registers user-defined functions by local name.
func WithFunctions(fns map[string]Function) ContextOption {
	return func(c *Context) {
		c.functions = fns
	}
}

// WithFunctionsNS registers user-defined functions by qualified name.
func WithFunctionsNS(fns map[QualifiedName]Function) ContextOption {
	return func(c *Context) {
		c.functionsNS = fns
	}
}

// WithImplicitTimezone sets the implicit timezone for the dynamic context.
// This is used by functions like fn:adjust-dateTime-to-timezone when called
// with a single argument. If not set, the system local timezone is used.
func WithImplicitTimezone(loc *time.Location) ContextOption {
	return func(c *Context) {
		c.implicitTimezone = loc
	}
}

// NewContext creates a new context.Context carrying XPath evaluation settings.
func NewContext(ctx context.Context, opts ...ContextOption) context.Context {
	c := &Context{}
	for _, opt := range opts {
		opt(c)
	}
	return context.WithValue(ctx, contextKey{}, c)
}

// GetContext retrieves the XPath Context from a context.Context, or nil if absent.
func GetContext(ctx context.Context) *Context {
	v, _ := ctx.Value(contextKey{}).(*Context)
	return v
}

// withFnContext stores the evalContext in a context.Context so built-in
// functions can access the evaluation state (position, size, context node).
func withFnContext(ctx context.Context, ec *evalContext) context.Context {
	return context.WithValue(ctx, fnContextKey{}, ec)
}

// getFnContext retrieves the evalContext stashed by the evaluator.
// Returns nil if not in an evaluation.
func getFnContext(ctx context.Context) *evalContext {
	ec, _ := ctx.Value(fnContextKey{}).(*evalContext)
	return ec
}
