package xpath3

import (
	"context"
	"io"
	"maps"
	"net/http"
	"time"

	"github.com/lestrrat-go/helium"
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

// VariableResolver provides lazy variable resolution for variables not found
// in the static variable scope. This is used by the XSLT executor to lazily
// evaluate global variables on demand.
type VariableResolver interface {
	ResolveVariable(ctx context.Context, name string) (Sequence, bool, error)
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
	position           int  // initial context position (0 = use default 1)
	size               int  // initial context size (0 = use default 1)
	contextItem        Item // non-nil when context is an atomic value, not a node
	typeAnnotations    map[helium.Node]string // node → xs:... type annotation (set by xslt3)
	variableResolver   VariableResolver       // lazy resolver for variables not in static scope
	strictPrefixes     bool                   // when true, only user-declared namespaces are valid (no defaultPrefixNS fallback)
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
		// Shallow-copy the config and let individual mutators clone maps only
		// when they actually modify them.
		cp := *cfg
		return &cp
	}
	return &evalConfig{}
}

// withEvalConfig stores the mutated config back on the context.
// varsDirty means the top-level variable bindings changed, so the cached root
// variableScope must be rebuilt before the config is reused by evaluation.
// Scalar-only updates can keep sharing the existing varScope.
func withEvalConfig(ctx context.Context, cfg *evalConfig, varsDirty bool) context.Context {
	if varsDirty {
		cfg.rebuildVariableScope()
	}
	return context.WithValue(ctx, contextKey{}, cfg)
}

// updateEvalConfig shallow-copies the config payload, applies a mutation, and
// stores it back on a derived context. The callback returns whether it changed
// the stored variable bindings; only those mutations require rebuilding
// cfg.varScope. Dynamic per-evaluation bindings still live in evalContext.
func updateEvalConfig(ctx context.Context, fn func(*evalConfig) bool) context.Context {
	cfg := deriveEvalConfig(ctx)
	varsDirty := fn(cfg)
	return withEvalConfig(ctx, cfg, varsDirty)
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
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.namespaces = maps.Clone(ns)
		return false
	})
}

// WithStrictPrefixes disables the default prefix→namespace fallback (fn:, xs:,
// math:, map:, array:, err:). When set, only prefixes explicitly declared via
// WithNamespaces are considered valid. This is used by XSLT, where namespace
// prefixes must be declared on the stylesheet element.
func WithStrictPrefixes(ctx context.Context) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.strictPrefixes = true
		return false
	})
}

// WithAdditionalNamespaces merges namespace prefixes into the returned context.
func WithAdditionalNamespaces(ctx context.Context, ns map[string]string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.namespaces = maps.Clone(c.namespaces)
		if c.namespaces == nil {
			c.namespaces = make(map[string]string, len(ns))
		}
		for k, v := range ns {
			c.namespaces[k] = v
		}
		return false
	})
}

// GetNamespaces retrieves the namespace bindings from the context.
func GetNamespaces(ctx context.Context) map[string]string {
	ec := getEvalConfig(ctx)
	if ec == nil {
		return nil
	}
	return ec.namespaces
}

// WithVariables binds variable names to pre-constructed Sequence values.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithVariables(ctx context.Context, vars map[string]Sequence) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.variables = cloneVariableMap(vars)
		return true
	})
}

// WithVariablesBorrowed binds variable names to pre-constructed Sequence values
// without cloning. The caller must guarantee that the map and its sequences are
// not mutated for the lifetime of the returned context. This is an optimization
// for internal callers (e.g. xslt3) that already own the data.
func WithVariablesBorrowed(ctx context.Context, vars map[string]Sequence) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.variables = vars
		return true
	})
}

// WithAdditionalVariables merges variable bindings into the returned context.
func WithAdditionalVariables(ctx context.Context, vars map[string]Sequence) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.variables = cloneVariableMap(c.variables)
		if c.variables == nil {
			c.variables = make(map[string]Sequence, len(vars))
		}
		for name, seq := range vars {
			c.variables[name] = append(Sequence(nil), seq...)
		}
		return true
	})
}

// WithVariableResolver sets a callback for lazy variable resolution.
// When a variable is not found in the static scope, the resolver is called.
func WithVariableResolver(ctx context.Context, resolver VariableResolver) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.variableResolver = resolver
		return false
	})
}

// WithOpLimit sets the maximum number of operations before the evaluator
// returns ErrOpLimit. Zero means unlimited.
func WithOpLimit(ctx context.Context, limit int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.opLimit = limit
		return false
	})
}

// WithFunctions registers user-defined functions by local name.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctions(ctx context.Context, fns map[string]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functions = maps.Clone(fns)
		return false
	})
}

// WithFunctionsBorrowed is like WithFunctions but does not clone the map.
// The caller must guarantee the map is not mutated for the lifetime of the context.
func WithFunctionsBorrowed(ctx context.Context, fns map[string]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functions = fns
		return false
	})
}

// WithFunction registers a single user-defined function by local name.
func WithFunction(ctx context.Context, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functions = maps.Clone(c.functions)
		if c.functions == nil {
			c.functions = make(map[string]Function)
		}
		c.functions[name] = fn
		return false
	})
}

// WithFunctionsNS registers user-defined functions by qualified name.
// The map is defensively copied to prevent caller mutation from affecting evaluation.
func WithFunctionsNS(ctx context.Context, fns map[QualifiedName]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functionsNS = maps.Clone(fns)
		return false
	})
}

// WithFunctionsNSBorrowed is like WithFunctionsNS but does not clone the map.
// The caller must guarantee the map is not mutated for the lifetime of the context.
func WithFunctionsNSBorrowed(ctx context.Context, fns map[QualifiedName]Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functionsNS = fns
		return false
	})
}

// WithFunctionNS registers a single user-defined function by qualified name.
func WithFunctionNS(ctx context.Context, uri, name string, fn Function) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.functionsNS = maps.Clone(c.functionsNS)
		if c.functionsNS == nil {
			c.functionsNS = make(map[QualifiedName]Function)
		}
		c.functionsNS[QualifiedName{URI: uri, Name: name}] = fn
		return false
	})
}

// WithImplicitTimezone sets the implicit timezone for the dynamic context.
// This is used by functions like fn:adjust-dateTime-to-timezone when called
// with a single argument. If not set, the system local timezone is used.
func WithImplicitTimezone(ctx context.Context, loc *time.Location) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.implicitTimezone = loc
		return false
	})
}

// WithDefaultLanguage sets the dynamic default language used by
// fn:default-language and formatting functions when no language argument
// is supplied.
func WithDefaultLanguage(ctx context.Context, lang string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.defaultLanguage = lang
		return false
	})
}

// WithDefaultCollation sets the default collation URI used by string
// comparison and ordering operations when no explicit collation argument is
// supplied. Use a URI understood by the evaluator's collation registry.
func WithDefaultCollation(ctx context.Context, uri string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.defaultCollation = uri
		return false
	})
}

// WithDefaultDecimalFormat sets the unnamed decimal format used by
// fn:format-number and related formatting features when no named decimal
// format is requested. The DecimalFormat value is copied before storage.
func WithDefaultDecimalFormat(ctx context.Context, df DecimalFormat) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		cp := df
		c.defaultDecimal = &cp
		return false
	})
}

// WithNamedDecimalFormats registers named decimal formats keyed by expanded
// QName. These formats are used when a formatting expression references a
// specific decimal format name. The map is defensively copied before storage.
func WithNamedDecimalFormats(ctx context.Context, dfs map[QualifiedName]DecimalFormat) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.decimalFormats = maps.Clone(dfs)
		return false
	})
}

// WithBaseURI sets the static base URI for the evaluation context.
// This is used for resolving relative URIs in fn:unparsed-text, fn:doc, etc.
func WithBaseURI(ctx context.Context, uri string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.baseURI = uri
		return false
	})
}

// WithURIResolver sets a custom URI resolver for functions that load external
// resources such as fn:unparsed-text and fn:doc.
func WithURIResolver(ctx context.Context, r URIResolver) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.uriResolver = r
		return false
	})
}

// WithCollectionResolver sets a custom resolver for fn:collection and
// fn:uri-collection.
func WithCollectionResolver(ctx context.Context, r CollectionResolver) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.collectionResolver = r
		return false
	})
}

// WithHTTPClient sets the HTTP client used for fetching http:// and https://
// resources in fn:unparsed-text and similar functions. If not set, HTTP URIs
// are not supported (unless a URIResolver handles them).
func WithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.httpClient = client
		return false
	})
}

// WithPosition sets the initial context position for the evaluation.
// This is used by XSLT to pass the current position from xsl:for-each.
func WithPosition(ctx context.Context, pos int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.position = pos
		return false
	})
}

// WithSize sets the initial context size for the evaluation.
// This is used by XSLT to pass the current size from xsl:for-each.
func WithSize(ctx context.Context, size int) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.size = size
		return false
	})
}

// WithContextItem sets the context item for evaluation to an atomic value
// instead of a node. This is used by XSLT for xsl:for-each over atomic
// sequences where `.` should return the current atomic value.
func WithContextItem(ctx context.Context, item Item) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.contextItem = item
		return false
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

// FnContextNode returns the current XPath context node from a function call
// context. This is the context.Context passed to Function.Call by the evaluator.
// Returns nil if the context does not carry an evaluation state or the context
// item is not a node.
func FnContextNode(ctx context.Context) helium.Node {
	ec := getFnContext(ctx)
	if ec == nil {
		return nil
	}
	return ec.node
}

// WithTypeAnnotations sets the type annotation map used for schema-aware
// node matching and typed atomization. The map associates helium nodes with
// their XSD type annotations (e.g., "xs:ID", "xs:integer"). The map is NOT
// cloned; the caller must guarantee it is not mutated for the lifetime of
// the context.
func WithTypeAnnotations(ctx context.Context, annotations map[helium.Node]string) context.Context {
	return updateEvalConfig(ctx, func(c *evalConfig) bool {
		c.typeAnnotations = annotations
		return false
	})
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
