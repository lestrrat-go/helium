package xpath3

import (
	"context"
	"maps"
	"net/http"
	"time"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// EvaluatorOption controls Evaluator construction behavior.
type EvaluatorOption uint32

const (
	// EvalBorrowing makes all setters skip map/slice cloning.
	// The caller must not mutate borrowed data for the lifetime of
	// derived evaluators, contexts, and eval states.
	EvalBorrowing EvaluatorOption = 1 << iota
)

// DefaultEvaluatorOptions is the zero value — all setters clone by default.
const DefaultEvaluatorOptions EvaluatorOption = 0

// Evaluator configures XPath 3.1 expression evaluation.
// It is a value-style wrapper: fluent methods return updated copies
// and the original is never mutated. The terminal method Evaluate
// creates an internal evalCtx immediately; downstream evaluation
// uses that context, never the Evaluator itself.
type Evaluator struct {
	cfg *evaluatorCfg
}

type evaluatorCfg struct {
	options            EvaluatorOption
	namespaces         map[string]string
	variables          *Variables
	functions          *FunctionLibrary
	opLimit            int
	currentTime        *time.Time
	implicitTimezone   *time.Location
	defaultLanguage    string
	defaultCollation   string
	defaultDecimal     *DecimalFormat
	decimalFormats     map[QualifiedName]DecimalFormat
	baseURI            string
	uriResolver        URIResolver
	collectionResolver CollectionResolver
	httpClient         *http.Client
	position           int
	size               int
	contextItem        Item
	typeAnnotations    map[helium.Node]string
	variableResolver   VariableResolver
	functionResolver   FunctionResolver
	strictPrefixes     bool
	schemaDeclarations SchemaDeclarations
	allowXML11Chars    bool
	docOrder           *DocOrderCache
}

// NewEvaluator creates a new Evaluator with the given options.
// Use DefaultEvaluatorOptions for safe cloning behavior.
// Use EvalBorrowing for internal callers that own their data.
func NewEvaluator(flags EvaluatorOption) Evaluator {
	return Evaluator{cfg: &evaluatorCfg{options: flags}}
}

func (e Evaluator) borrowing() bool {
	return e.cfg.options&EvalBorrowing != 0
}

func (e Evaluator) clone() Evaluator {
	cp := *e.cfg
	return Evaluator{cfg: &cp}
}

// Namespaces binds namespace prefixes to URIs for the evaluation.
// The map is cloned unless EvalBorrowing is set.
func (e Evaluator) Namespaces(ns map[string]string) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.namespaces = ns
	} else {
		e.cfg.namespaces = maps.Clone(ns)
	}
	return e
}

// Variables sets the variable bindings for the evaluation.
// The Variables collection is cloned unless EvalBorrowing is set.
func (e Evaluator) Variables(v *Variables) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.variables = v
	} else {
		e.cfg.variables = v.Clone()
	}
	return e
}

// Functions sets the user-defined function library for the evaluation.
// The FunctionLibrary is cloned unless EvalBorrowing is set.
func (e Evaluator) Functions(f *FunctionLibrary) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.functions = f
	} else {
		e.cfg.functions = f.Clone()
	}
	return e
}

// VariableResolver sets a callback for lazy variable resolution.
func (e Evaluator) VariableResolver(r VariableResolver) Evaluator {
	e = e.clone()
	e.cfg.variableResolver = r
	return e
}

// FunctionResolver sets a callback for lazy function resolution.
func (e Evaluator) FunctionResolver(r FunctionResolver) Evaluator {
	e = e.clone()
	e.cfg.functionResolver = r
	return e
}

// OpLimit sets the maximum number of operations before evaluation aborts.
func (e Evaluator) OpLimit(limit int) Evaluator {
	e = e.clone()
	e.cfg.opLimit = limit
	return e
}

// CurrentTime sets the dynamic current time for fn:current-dateTime etc.
func (e Evaluator) CurrentTime(now time.Time) Evaluator {
	e = e.clone()
	t := now
	e.cfg.currentTime = &t
	return e
}

// ImplicitTimezone sets the implicit timezone for the dynamic context.
func (e Evaluator) ImplicitTimezone(loc *time.Location) Evaluator {
	e = e.clone()
	e.cfg.implicitTimezone = loc
	return e
}

// DefaultLanguage sets the default language for fn:default-language.
func (e Evaluator) DefaultLanguage(lang string) Evaluator {
	e = e.clone()
	e.cfg.defaultLanguage = lang
	return e
}

// DefaultCollation sets the default collation URI.
func (e Evaluator) DefaultCollation(uri string) Evaluator {
	e = e.clone()
	e.cfg.defaultCollation = uri
	return e
}

// DefaultDecimalFormat sets the unnamed decimal format.
func (e Evaluator) DefaultDecimalFormat(df DecimalFormat) Evaluator {
	e = e.clone()
	cp := df
	e.cfg.defaultDecimal = &cp
	return e
}

// NamedDecimalFormats sets named decimal formats.
// The map is cloned unless EvalBorrowing is set.
func (e Evaluator) NamedDecimalFormats(dfs map[QualifiedName]DecimalFormat) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.decimalFormats = dfs
	} else {
		e.cfg.decimalFormats = maps.Clone(dfs)
	}
	return e
}

// BaseURI sets the static base URI for resolving relative URIs.
func (e Evaluator) BaseURI(uri string) Evaluator {
	e = e.clone()
	e.cfg.baseURI = uri
	return e
}

// URIResolver sets a custom URI resolver for fn:unparsed-text, fn:doc, etc.
func (e Evaluator) URIResolver(r URIResolver) Evaluator {
	e = e.clone()
	e.cfg.uriResolver = r
	return e
}

// CollectionResolver sets a custom resolver for fn:collection.
func (e Evaluator) CollectionResolver(r CollectionResolver) Evaluator {
	e = e.clone()
	e.cfg.collectionResolver = r
	return e
}

// HTTPClient sets the HTTP client used for fetching resources.
func (e Evaluator) HTTPClient(client *http.Client) Evaluator {
	e = e.clone()
	e.cfg.httpClient = client
	return e
}

// Position sets the initial context position.
func (e Evaluator) Position(pos int) Evaluator {
	e = e.clone()
	e.cfg.position = pos
	return e
}

// Size sets the initial context size.
func (e Evaluator) Size(size int) Evaluator {
	e = e.clone()
	e.cfg.size = size
	return e
}

// ContextItem sets the context item to an atomic value instead of a node.
func (e Evaluator) ContextItem(item Item) Evaluator {
	e = e.clone()
	e.cfg.contextItem = item
	return e
}

// TypeAnnotations sets the type annotation map for schema-aware evaluation.
// The map is cloned unless EvalBorrowing is set.
func (e Evaluator) TypeAnnotations(annotations map[helium.Node]string) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.typeAnnotations = annotations
	} else {
		e.cfg.typeAnnotations = maps.Clone(annotations)
	}
	return e
}

// SchemaDeclarations sets the schema declarations provider.
func (e Evaluator) SchemaDeclarations(d SchemaDeclarations) Evaluator {
	e = e.clone()
	e.cfg.schemaDeclarations = d
	return e
}

// StrictPrefixes disables the default prefix→namespace fallback.
func (e Evaluator) StrictPrefixes() Evaluator {
	e = e.clone()
	e.cfg.strictPrefixes = true
	return e
}

// AllowXML11Chars enables XML 1.1 restricted characters.
func (e Evaluator) AllowXML11Chars() Evaluator {
	e = e.clone()
	e.cfg.allowXML11Chars = true
	return e
}

// DocOrderCache attaches a shared document-order cache.
func (e Evaluator) DocOrderCache(cache *DocOrderCache) Evaluator {
	e = e.clone()
	e.cfg.docOrder = cache
	return e
}

// Evaluate evaluates the compiled expression against the given context node.
// ctx is used for cancellation/deadlines only, not for configuration.
func (e Evaluator) Evaluate(ctx context.Context, expr *Expression, node helium.Node) (*Result, error) {
	ec := e.newEvalCtx(ctx, node)

	if err := expr.prefixPlan.Validate(ec.namespaces, ec.strictPrefixes, ec.schemaDeclarations); err != nil {
		return nil, err
	}

	seq, err := expr.evaluate(ec)
	if err != nil {
		return nil, err
	}
	return &Result{seq: seq}, nil
}

// newEvalCtx creates the internal evaluation context from the Evaluator config.
func (e Evaluator) newEvalCtx(ctx context.Context, node helium.Node) *evalContext {
	opCount := 0
	now := time.Now()

	cfg := e.cfg

	ec := &evalContext{
		goCtx:    ctx,
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		docOrder: &ixpath.DocOrderCache{},
		maxNodes: maxNodeSetLength,
		docCache: make(map[string]helium.Node),
	}

	// namespaces
	ec.namespaces = cfg.namespaces

	// variables
	if cfg.variables != nil && cfg.variables.Len() > 0 {
		ec.vars = newVariableScope(cfg.variables.toMap())
	}

	// functions
	if cfg.functions != nil {
		ec.functions = cfg.functions.localMap()
		ec.fnsNS = cfg.functions.qnameMap()
	}

	// resolvers
	ec.variableResolver = cfg.variableResolver
	ec.functionResolver = cfg.functionResolver
	ec.uriResolver = cfg.uriResolver
	ec.collectionResolver = cfg.collectionResolver
	ec.httpClient = cfg.httpClient

	// limits
	ec.opLimit = cfg.opLimit

	// time
	if cfg.currentTime != nil {
		ec.currentTime = cfg.currentTime
	} else {
		ec.currentTime = &now
	}
	ec.implicitTimezone = cfg.implicitTimezone

	// locale / formatting
	ec.defaultLanguage = cfg.defaultLanguage
	ec.defaultCollation = cfg.defaultCollation
	if cfg.defaultDecimal != nil {
		df := *cfg.defaultDecimal
		ec.defaultDecimalFormat = &df
	}
	ec.decimalFormats = cfg.decimalFormats

	// base URI
	ec.baseURI = cfg.baseURI

	// schema / typing
	ec.typeAnnotations = cfg.typeAnnotations
	ec.schemaDeclarations = cfg.schemaDeclarations
	ec.strictPrefixes = cfg.strictPrefixes
	ec.allowXML11Chars = cfg.allowXML11Chars

	// dynamic focus overrides
	if cfg.position > 0 {
		ec.position = cfg.position
	}
	if cfg.size > 0 {
		ec.size = cfg.size
	}
	if cfg.contextItem != nil {
		ec.contextItem = cfg.contextItem
		ec.node = nil
	}
	if cfg.docOrder != nil {
		ec.docOrder = cfg.docOrder
	}

	return ec
}
