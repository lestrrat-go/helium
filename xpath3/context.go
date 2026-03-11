package xpath3

import (
	"context"
	"io"
	"maps"
	"net/http"
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

// URIResolver resolves URIs to readable content for fn:unparsed-text and fn:doc.
// The resolved URI is the absolute URI after base URI resolution.
type URIResolver interface {
	ResolveURI(uri string) (io.ReadCloser, error)
}

// Context holds XPath evaluation settings provided by the caller.
type Context struct {
	namespaces       map[string]string
	variables        map[string]Sequence
	functions        map[string]Function
	functionsNS      map[QualifiedName]Function
	opLimit          int
	implicitTimezone *time.Location
	baseURI          string
	uriResolver      URIResolver
	httpClient       *http.Client
}

// ContextOption configures a Context.
type ContextOption func(*Context)

// WithNamespaces binds namespace prefixes to URIs for the evaluation.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithNamespaces(ns map[string]string) ContextOption {
	return func(c *Context) {
		c.namespaces = maps.Clone(ns)
	}
}

// WithVariables binds variable names to pre-constructed Sequence values.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithVariables(vars map[string]Sequence) ContextOption {
	return func(c *Context) {
		c.variables = maps.Clone(vars)
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
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctions(fns map[string]Function) ContextOption {
	return func(c *Context) {
		c.functions = maps.Clone(fns)
	}
}

// WithFunctionsNS registers user-defined functions by qualified name.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctionsNS(fns map[QualifiedName]Function) ContextOption {
	return func(c *Context) {
		c.functionsNS = maps.Clone(fns)
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

// WithBaseURI sets the static base URI for the evaluation context.
// This is used for resolving relative URIs in fn:unparsed-text, fn:doc, etc.
func WithBaseURI(uri string) ContextOption {
	return func(c *Context) {
		c.baseURI = uri
	}
}

// WithURIResolver sets a custom URI resolver for functions that load external
// resources such as fn:unparsed-text and fn:doc.
func WithURIResolver(r URIResolver) ContextOption {
	return func(c *Context) {
		c.uriResolver = r
	}
}

// WithHTTPClient sets the HTTP client used for fetching http:// and https://
// resources in fn:unparsed-text and similar functions. If not set, HTTP URIs
// are not supported (unless a URIResolver handles them).
func WithHTTPClient(client *http.Client) ContextOption {
	return func(c *Context) {
		c.httpClient = client
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
