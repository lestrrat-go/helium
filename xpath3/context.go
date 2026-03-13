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

// CollectionResolver resolves fn:collection and fn:uri-collection lookups.
// The empty string identifies the default collection.
type CollectionResolver interface {
	ResolveCollection(uri string) (Sequence, error)
	ResolveURICollection(uri string) ([]string, error)
}

type evalConfig struct {
	namespaces         map[string]string
	variables          map[string]Sequence
	varScope           *variableScope // prebuilt from variables, reused across evaluations
	functions          map[string]Function
	functionsNS        map[QualifiedName]Function
	opLimit            int
	implicitTimezone   *time.Location
	defaultLanguage    string
	defaultCollation   string
	defaultDecimal     *DecimalFormat
	decimalFormats     map[QualifiedName]DecimalFormat
	baseURI            string
	uriResolver        URIResolver
	collectionResolver CollectionResolver
	httpClient         *http.Client
	position           int // initial context position (0 = use default 1)
	size               int // initial context size (0 = use default 1)
}

func getEvalConfig(ctx context.Context) *evalConfig {
	if ctx == nil {
		return nil
	}
	cfg, _ := ctx.Value(contextKey{}).(*evalConfig)
	return cfg
}

func deriveEvalConfig(ctx context.Context) *evalConfig {
	if cfg := getEvalConfig(ctx); cfg != nil {
		return cfg.clone()
	}
	return &evalConfig{}
}

func withEvalConfig(ctx context.Context, cfg *evalConfig) context.Context {
	cfg.rebuildVariableScope()
	return context.WithValue(ctx, contextKey{}, cfg)
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
	cp.variables = cloneVariableMap(c.variables)
	cp.functions = maps.Clone(c.functions)
	cp.functionsNS = maps.Clone(c.functionsNS)
	if c.defaultDecimal != nil {
		df := *c.defaultDecimal
		cp.defaultDecimal = &df
	}
	cp.decimalFormats = maps.Clone(c.decimalFormats)
	return &cp
}

func (c *evalConfig) rebuildVariableScope() {
	if len(c.variables) == 0 {
		c.varScope = nil
		return
	}
	c.varScope = newVariableScope(c.variables)
}

// WithNamespaces binds namespace prefixes to URIs for the evaluation.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithNamespaces(ctx context.Context, ns map[string]string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.namespaces = maps.Clone(ns)
	})
}

// WithAdditionalNamespaces merges namespace prefixes into the returned context.
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

// WithVariables binds variable names to pre-constructed Sequence values.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithVariables(ctx context.Context, vars map[string]Sequence) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.variables = cloneVariableMap(vars)
	})
}

// WithAdditionalVariables merges variable bindings into the returned context.
func WithAdditionalVariables(ctx context.Context, vars map[string]Sequence) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.variables == nil {
			c.variables = make(map[string]Sequence, len(vars))
		}
		for name, seq := range vars {
			c.variables[name] = append(Sequence(nil), seq...)
		}
	})
}

// WithOpLimit sets the maximum number of operations before the evaluator
// returns ErrOpLimit. Zero means unlimited.
func WithOpLimit(ctx context.Context, limit int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.opLimit = limit
	})
}

// WithFunctions registers user-defined functions by local name.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctions(ctx context.Context, fns map[string]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.functions = maps.Clone(fns)
	})
}

// WithFunction registers a single user-defined function by local name.
func WithFunction(ctx context.Context, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.functions == nil {
			c.functions = make(map[string]Function)
		}
		c.functions[name] = fn
	})
}

// WithFunctionsNS registers user-defined functions by qualified name.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctionsNS(ctx context.Context, fns map[QualifiedName]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.functionsNS = maps.Clone(fns)
	})
}

// WithFunctionNS registers a single user-defined function by qualified name.
func WithFunctionNS(ctx context.Context, uri, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		if c.functionsNS == nil {
			c.functionsNS = make(map[QualifiedName]Function)
		}
		c.functionsNS[QualifiedName{URI: uri, Name: name}] = fn
	})
}

// WithImplicitTimezone sets the implicit timezone for the dynamic context.
// This is used by functions like fn:adjust-dateTime-to-timezone when called
// with a single argument. If not set, the system local timezone is used.
func WithImplicitTimezone(ctx context.Context, loc *time.Location) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.implicitTimezone = loc
	})
}

// WithDefaultLanguage sets the dynamic default language used by
// fn:default-language and formatting functions when no language argument
// is supplied.
func WithDefaultLanguage(ctx context.Context, lang string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.defaultLanguage = lang
	})
}

// WithDefaultCollation sets the default collation URI used by string
// comparison and ordering operations when no explicit collation argument is
// supplied. Use a URI understood by the evaluator's collation registry.
func WithDefaultCollation(ctx context.Context, uri string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.defaultCollation = uri
	})
}

// WithDefaultDecimalFormat sets the unnamed decimal format used by
// fn:format-number and related formatting features when no named decimal
// format is requested. The DecimalFormat value is copied before storage.
func WithDefaultDecimalFormat(ctx context.Context, df DecimalFormat) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		cp := df
		c.defaultDecimal = &cp
	})
}

// WithNamedDecimalFormats registers named decimal formats keyed by expanded
// QName. These formats are used when a formatting expression references a
// specific decimal format name. The map is defensively copied before storage.
func WithNamedDecimalFormats(ctx context.Context, dfs map[QualifiedName]DecimalFormat) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.decimalFormats = maps.Clone(dfs)
	})
}

// WithBaseURI sets the static base URI for the evaluation context.
// This is used for resolving relative URIs in fn:unparsed-text, fn:doc, etc.
func WithBaseURI(ctx context.Context, uri string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.baseURI = uri
	})
}

// WithURIResolver sets a custom URI resolver for functions that load external
// resources such as fn:unparsed-text and fn:doc.
func WithURIResolver(ctx context.Context, r URIResolver) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.uriResolver = r
	})
}

// WithCollectionResolver sets a custom resolver for fn:collection and
// fn:uri-collection.
func WithCollectionResolver(ctx context.Context, r CollectionResolver) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.collectionResolver = r
	})
}

// WithHTTPClient sets the HTTP client used for fetching http:// and https://
// resources in fn:unparsed-text and similar functions. If not set, HTTP URIs
// are not supported (unless a URIResolver handles them).
func WithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.httpClient = client
	})
}

// WithPosition sets the initial context position for the evaluation.
// This is used by XSLT to pass the current position from xsl:for-each.
func WithPosition(ctx context.Context, pos int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.position = pos
	})
}

// WithSize sets the initial context size for the evaluation.
// This is used by XSLT to pass the current size from xsl:for-each.
func WithSize(ctx context.Context, size int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) {
		c.size = size
	})
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

func cloneVariableMap(vars map[string]Sequence) map[string]Sequence {
	if vars == nil {
		return nil
	}
	cloned := make(map[string]Sequence, len(vars))
	for name, seq := range vars {
		cloned[name] = append(Sequence(nil), seq...)
	}
	return cloned
}
