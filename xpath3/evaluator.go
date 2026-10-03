package xpath3

import (
	"context"
	"io"
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
	options                EvaluatorOption
	namespaces             map[string]string
	variables              map[string]Sequence
	varScope               *variableScope             // root scope built once from variables by Variables(); shared read-only by every evaluation
	functions              map[string]Function        // unqualified calls
	functionsNS            map[QualifiedName]Function // namespaced calls
	opLimit                int
	currentTime            *time.Time
	implicitTimezone       *time.Location
	defaultLanguage        string
	defaultCollation       string
	defaultDecimal         *DecimalFormat
	decimalFormats         map[QualifiedName]DecimalFormat
	baseURI                string
	uriResolver            URIResolver
	collectionResolver     CollectionResolver
	httpClient             *http.Client
	position               int
	size                   int
	contextItem            Item
	typeAnnotations        map[helium.Node]string
	nilledElements         map[helium.Node]struct{} // elements whose xsi:nil="true" was confirmed by XSD validation
	preservedIDAnnotations map[helium.Node]string   // ID/IDREF annotations preserved after input-type-annotations="strip"
	idNodes                map[helium.Node]struct{} // PSVI is-id nodes (from xsd validation); supplements the type-annotation is-id check for list/union types
	variableResolver       VariableResolver
	functionResolver       FunctionResolver
	strictPrefixes         bool
	qnameValueNoDefaultNS  bool
	schemaDeclarations     SchemaDeclarations
	allowXML11Chars        bool
	xpath10Compat          bool // XPath 1.0 compatibility mode (XSLT backwards-compatible processing)
	docOrder               *DocOrderCache
	traceWriter            io.Writer
	maxNodes               int   // 0 means use the package default (maxNodeSetLength)
	maxResourceBytes       int64 // per-resource read cap for fn:unparsed-text / fn:doc / fn:json-doc; 0 = unparsedtext default
	parser                 *helium.Parser
}

// NewEvaluator creates a new Evaluator with the given options.
// Use DefaultEvaluatorOptions for safe cloning behavior.
// Use EvalBorrowing for internal callers that own their data.
func NewEvaluator(flags EvaluatorOption) Evaluator {
	return Evaluator{cfg: &evaluatorCfg{options: flags}}
}

func (e Evaluator) borrowing() bool {
	return e.cfg != nil && e.cfg.options&EvalBorrowing != 0
}

func (e Evaluator) clone() Evaluator {
	if e.cfg == nil {
		return Evaluator{cfg: &evaluatorCfg{}}
	}
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

// Variables sets the variable bindings for the evaluation, keyed by
// expanded-QName string. The map is cloned (each value deep-copied) unless
// EvalBorrowing is set, in which case the caller must not mutate it for the
// lifetime of derived evaluators.
func (e Evaluator) Variables(vars map[string]Sequence) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.variables = vars
	} else {
		e.cfg.variables = cloneVariableMap(vars)
	}
	// Build the root variable scope once here instead of per evaluation. A
	// variableScope is never mutated after construction (evaluation only links
	// new child scopes to it), so every evaluation can share it.
	e.cfg.varScope = newVariableScope(e.cfg.variables)
	return e
}

// Functions sets the user-defined functions for the evaluation: byLocal is
// consulted for unqualified calls, byQName for namespaced calls. Both maps are
// cloned unless EvalBorrowing is set.
func (e Evaluator) Functions(byLocal map[string]Function, byQName map[QualifiedName]Function) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.functions = byLocal
		e.cfg.functionsNS = byQName
	} else {
		e.cfg.functions = maps.Clone(byLocal)
		e.cfg.functionsNS = maps.Clone(byQName)
	}
	return e
}

// cloneVariableMap deep-copies a variable binding map, copying each Sequence
// value (preserving the previous Variables.Clone semantics).
func cloneVariableMap(vars map[string]Sequence) map[string]Sequence {
	if vars == nil {
		return nil
	}
	cp := make(map[string]Sequence, len(vars))
	for name, seq := range vars {
		cp[name] = cloneSequence(seq)
	}
	return cp
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

// Parser sets the helium.Parser used to parse XML in fn:parse-xml,
// fn:parse-xml-fragment, and fn:doc. When unset, a default helium.NewParser()
// is used. The injected parser governs parse policy (limits, filesystem,
// XXE/network controls).
func (e Evaluator) Parser(p helium.Parser) Evaluator {
	e = e.clone()
	e.cfg.parser = &p
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

// MaxResourceBytes sets the maximum number of bytes read from a single
// external resource fetched through the URIResolver or HTTPClient by
// fn:unparsed-text, fn:unparsed-text-lines, fn:unparsed-text-available,
// fn:doc, fn:doc-available, and fn:json-doc. A value of 0 selects the default cap; a
// negative value disables the bound. Reads exceeding the cap fail before an unbounded body is
// buffered. fn:unparsed-text and fn:unparsed-text-lines
// surface the over-cap error as FOUT1170 (fn:unparsed-text-available returns
// false); fn:doc and fn:json-doc surface it as a retrieval error (FODC0002),
// and fn:doc-available returns false.
func (e Evaluator) MaxResourceBytes(n int64) Evaluator {
	e = e.clone()
	e.cfg.maxResourceBytes = n
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

// Focus sets the whole dynamic focus — context item, context position, and
// context size — in one step. It gives the same configuration as
// ContextItem(item).Position(position).Size(size) but copies the evaluator
// configuration once instead of three times, for callers that set a new focus
// on every evaluation. A nil item leaves the context to the node passed to
// Evaluate, and a position or size of 0 or less selects the default of 1.
func (e Evaluator) Focus(item Item, position, size int) Evaluator {
	e = e.clone()
	e.cfg.contextItem = item
	e.cfg.position = position
	e.cfg.size = size
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

// NilledElements sets the set of element nodes whose xsi:nil="true" was
// confirmed valid by XSD validation (the element declaration is nillable). It
// makes fn:nilled return true for those elements, gives them the empty typed
// value () under fn:data / atomization (a nilled element has no typed value),
// and excludes them from an element(name, type) instance-of test (a nilled
// element matches element(name, type?) but not element(name, type)). The map is
// cloned unless EvalBorrowing is set. Non-schema-aware evaluation leaves it nil
// and behavior is unchanged.
func (e Evaluator) NilledElements(nodes map[helium.Node]struct{}) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.nilledElements = nodes
	} else {
		e.cfg.nilledElements = maps.Clone(nodes)
	}
	return e
}

// PreservedIDAnnotations sets the preserved ID/IDREF annotations map for use
// by fn:id and fn:idref when input-type-annotations="strip" has removed the
// regular type annotations. The is-id and is-idref properties are preserved
// per the XSLT spec even when type annotations are stripped.
func (e Evaluator) PreservedIDAnnotations(annotations map[helium.Node]string) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.preservedIDAnnotations = annotations
	} else {
		e.cfg.preservedIDAnnotations = maps.Clone(annotations)
	}
	return e
}

// IDNodes sets the PSVI is-id node set for schema-aware fn:id / fn:element-with-id.
// A node in this set is treated as an is-id node in addition to those whose type
// annotation is xs:ID or derived from it, so a schema-validated element/attribute
// whose is-id property arises from a SINGLETON list of xs:ID or a union that
// selects an xs:ID-derived member (neither of which is a name-level subtype of
// xs:ID) is recognized. The map is cloned unless EvalBorrowing is set.
func (e Evaluator) IDNodes(ids map[helium.Node]struct{}) Evaluator {
	e = e.clone()
	if e.borrowing() {
		e.cfg.idNodes = ids
	} else {
		e.cfg.idNodes = maps.Clone(ids)
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

// QNameValueNoDefaultNamespace makes the atomization of a QName/NOTATION-typed
// node resolve an UNPREFIXED lexical value to NO namespace, instead of the node's
// in-scope default namespace. This is XSD value-space semantics — a QName VALUE,
// unlike an element/attribute NAME, does not pick up the default namespace — used
// by xsd assertion evaluation. Off by default, so general XPath/XQuery and XSLT
// atomization keep resolving an unprefixed QName value against the default
// namespace.
func (e Evaluator) QNameValueNoDefaultNamespace() Evaluator {
	e = e.clone()
	e.cfg.qnameValueNoDefaultNS = true
	return e
}

// AllowXML11Chars enables XML 1.1 restricted characters.
func (e Evaluator) AllowXML11Chars() Evaluator {
	e = e.clone()
	e.cfg.allowXML11Chars = true
	return e
}

// XPath10Compat enables XPath 1.0 compatibility mode, the evaluation semantics
// selected by XSLT backwards-compatible processing (an effective [xsl:]version
// below 2.0). When on: a single-item function parameter supplied more than one
// item keeps only the first; an xs:string(?) parameter coerces its value with
// fn:string and an xs:double(?)/xs:numeric parameter with fn:number (lexically
// invalid or empty → NaN); arithmetic operands are converted to xs:double
// (division by zero yields ±INF, non-numeric operands NaN); and general
// comparisons apply the 1.0 boolean/numeric/string conversion rules. Off by
// default, so ordinary XPath 3.1 / XSLT 3.0 / XSD evaluation is unchanged.
func (e Evaluator) XPath10Compat() Evaluator {
	e = e.clone()
	e.cfg.xpath10Compat = true
	return e
}

// DocOrderCache attaches a shared document-order cache.
func (e Evaluator) DocOrderCache(cache *DocOrderCache) Evaluator {
	e = e.clone()
	e.cfg.docOrder = cache
	return e
}

// TraceWriter sets the destination for fn:trace output.
// When nil, fn:trace writes to os.Stderr.
func (e Evaluator) TraceWriter(w io.Writer) Evaluator {
	e = e.clone()
	e.cfg.traceWriter = w
	return e
}

// Evaluate evaluates the compiled expression against the given context node.
// ctx is used for cancellation/deadlines only, not for configuration.
func (e Evaluator) Evaluate(ctx context.Context, expr *Expression, node helium.Node) (*Result, error) {
	if err := expr.requireCompiledProgram(); err != nil {
		return nil, err
	}

	ec := e.newEvalCtx(node)

	if err := expr.prefixPlan.Validate(ec.namespaces, ec.strictPrefixes, ec.schemaDeclarations); err != nil {
		return nil, err
	}

	seq, nodes, isNodes, err := expr.evaluate(ctx, ec)
	if err != nil {
		return nil, err
	}
	if !isNodes {
		return &Result{seq: seq}, nil
	}
	// One allocation holds the Result and its node list.
	box := &resultBox{nodes: resultNodes{nodes: nodes}}
	box.result.seq = &box.nodes
	return &box.result, nil
}

// EvaluateEBV evaluates the compiled expression against the given context
// node and returns the effective boolean value of its result (XPath 3.1
// §2.4.3): the value and errors that Evaluate followed by Result.EBV give.
// ctx is used for cancellation/deadlines only, not for configuration.
//
// Because only the boolean is needed, a location path or path expression
// E1/path that is the whole expression stops at the first node it selects,
// as it does in a predicate or as the argument of fn:exists. When it stops
// early, the OpLimit and node-set-limit errors and the predicate errors
// (XPath 3.1 §2.3.4) of the nodes it did not reach do not fire. A node
// result never builds node items, with or without type annotations.
func (e Evaluator) EvaluateEBV(ctx context.Context, expr *Expression, node helium.Node) (bool, error) {
	if err := expr.requireCompiledProgram(); err != nil {
		return false, err
	}

	ec := e.newEvalCtx(node)

	if err := expr.prefixPlan.Validate(ec.namespaces, ec.strictPrefixes, ec.schemaDeclarations); err != nil {
		return false, err
	}
	return expr.program.executeEBV(ctx, ec)
}

// resultBox holds the Result Evaluate returns and the node list it may
// refer to.
type resultBox struct {
	result Result
	nodes  resultNodes
}

// newEvalCtx creates the internal evaluation context from the Evaluator config.
func (e Evaluator) newEvalCtx(node helium.Node) *evalContext {
	opCount := 0

	cfg := e.cfg
	if cfg == nil {
		cfg = &evaluatorCfg{}
	}

	maxNodes := maxNodeSetLength
	if cfg.maxNodes > 0 {
		maxNodes = cfg.maxNodes
	}

	ec := &evalContext{
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		maxNodes: maxNodes,
		docCache: make(map[string]helium.Node),
	}

	// namespaces
	ec.namespaces = cfg.namespaces

	// variables: the root scope is built once by Variables() and shared.
	ec.vars = cfg.varScope

	// functions
	ec.functions = cfg.functions
	ec.fnsNS = cfg.functionsNS

	// resolvers
	ec.variableResolver = cfg.variableResolver
	ec.functionResolver = cfg.functionResolver
	ec.uriResolver = cfg.uriResolver
	ec.collectionResolver = cfg.collectionResolver
	ec.httpClient = cfg.httpClient
	ec.maxResourceBytes = cfg.maxResourceBytes
	ec.parser = cfg.parser

	// limits
	ec.opLimit = cfg.opLimit
	ec.maxRecursionDepth = DefaultMaxRecursionDepth

	// time
	if cfg.currentTime != nil {
		ec.currentTime = cfg.currentTime
	} else {
		now := time.Now()
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
	ec.nilledElements = cfg.nilledElements
	ec.schemaDeclarations = cfg.schemaDeclarations
	ec.preservedIDAnnotations = cfg.preservedIDAnnotations
	ec.idNodes = cfg.idNodes
	ec.strictPrefixes = cfg.strictPrefixes
	ec.qnameValueNoDefaultNS = cfg.qnameValueNoDefaultNS
	ec.allowXML11Chars = cfg.allowXML11Chars
	ec.xpath10Compat = cfg.xpath10Compat

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
	// Allocate a private document-order cache only when no shared one is
	// configured.
	ec.docOrder = cfg.docOrder
	if ec.docOrder == nil {
		ec.docOrder = &ixpath.DocOrderCache{}
	}

	ec.traceWriter = cfg.traceWriter

	return ec
}
