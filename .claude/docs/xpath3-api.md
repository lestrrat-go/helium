# XPath 3.1 — Public API

Mirrors `xpath1` shape. Key difference: `Result` wraps `Sequence` (not union struct).

## Entry Points

```go
// Compiler — fluent builder (clone-on-write, value type)
func NewCompiler() Compiler
func (c Compiler) Compile(expr string) (*Expression, error)
func (c Compiler) MustCompile(expr string) *Expression
func (c Compiler) CompileExpr(ast Expr) (*Expression, error)

// Evaluator — fluent builder (clone-on-write, value type)
type EvaluatorOption uint32
const EvalBorrowing EvaluatorOption = 1 << iota // setters skip map/slice cloning
const DefaultEvaluatorOptions EvaluatorOption = 0 // zero value: setters clone

func NewEvaluator(flags EvaluatorOption) Evaluator
func (e Evaluator) Evaluate(ctx context.Context, expr *Expression, node helium.Node) (*Result, error)
```

`NewEvaluator` takes an `EvaluatorOption` bitmask (NOT plural `EvaluatorOptions`).
Pass `DefaultEvaluatorOptions` for safe clone-on-write behavior. Pass
`EvalBorrowing` for callers that own their data and want setters to borrow
(skip cloning) the maps/slices they receive — the caller must then not mutate
that data for the lifetime of derived evaluators, contexts, and eval states.

`Evaluate` is the terminal method; `ctx` is for cancellation/deadlines only,
not configuration. All configuration comes from the fluent setters below.

### Evaluator fluent setters

Each returns an updated copy; the original `Evaluator` is never mutated. Maps and
slices are cloned unless `EvalBorrowing` was set on `NewEvaluator`.

```go
// Bindings & resolvers
func (e Evaluator) Namespaces(ns map[string]string) Evaluator
func (e Evaluator) Variables(vars map[string]Sequence) Evaluator
func (e Evaluator) Functions(byLocal map[string]Function, byQName map[QualifiedName]Function) Evaluator
func (e Evaluator) VariableResolver(r VariableResolver) Evaluator
func (e Evaluator) FunctionResolver(r FunctionResolver) Evaluator
func (e Evaluator) URIResolver(r URIResolver) Evaluator
func (e Evaluator) CollectionResolver(r CollectionResolver) Evaluator
func (e Evaluator) HTTPClient(client *http.Client) Evaluator

// Limits & resources
func (e Evaluator) OpLimit(limit int) Evaluator
func (e Evaluator) MaxResourceBytes(n int64) Evaluator

// Time & locale / formatting
func (e Evaluator) CurrentTime(now time.Time) Evaluator
func (e Evaluator) ImplicitTimezone(loc *time.Location) Evaluator
func (e Evaluator) DefaultLanguage(lang string) Evaluator
func (e Evaluator) DefaultCollation(uri string) Evaluator
func (e Evaluator) DefaultDecimalFormat(df DecimalFormat) Evaluator
func (e Evaluator) NamedDecimalFormats(dfs map[QualifiedName]DecimalFormat) Evaluator

// Base URI
func (e Evaluator) BaseURI(uri string) Evaluator

// Dynamic focus
func (e Evaluator) Position(pos int) Evaluator
func (e Evaluator) Size(size int) Evaluator
func (e Evaluator) ContextItem(item Item) Evaluator

// Schema / typing
func (e Evaluator) TypeAnnotations(annotations map[helium.Node]string) Evaluator
func (e Evaluator) PreservedIDAnnotations(annotations map[helium.Node]string) Evaluator
func (e Evaluator) IDNodes(ids map[helium.Node]struct{}) Evaluator // PSVI is-id nodes (from xsd Validator.IDNodes); a node here is is-id for fn:id/fn:element-with-id even when its type name is not a subtype of xs:ID (singleton list of xs:ID, or a union selecting an xs:ID-derived member)
func (e Evaluator) SchemaDeclarations(d SchemaDeclarations) Evaluator
func (e Evaluator) StrictPrefixes() Evaluator
func (e Evaluator) QNameValueNoDefaultNamespace() Evaluator // XSD: unprefixed QName/NOTATION node VALUE atomizes to no namespace (off by default)
func (e Evaluator) AllowXML11Chars() Evaluator

// Misc
func (e Evaluator) DocOrderCache(cache *DocOrderCache) Evaluator
func (e Evaluator) TraceWriter(w io.Writer) Evaluator // nil → os.Stderr
```

`MaxResourceBytes` caps the bytes read from a single external resource fetched
through the `URIResolver` / `HTTPClient` by `fn:unparsed-text`,
`fn:unparsed-text-lines`, `fn:unparsed-text-available`, `fn:doc`,
`fn:doc-available`, and `fn:json-doc`. `0` selects the default cap; a negative
value disables the bound. Reads exceeding the cap fail rather than buffering an
unbounded body: `fn:unparsed-text` / `fn:unparsed-text-lines` surface the
over-cap error as `FOUT1170` (`fn:unparsed-text-available` returns false), while
`fn:doc` / `fn:json-doc` surface it as a retrieval error `FODC0002` (`fn:doc-available`
returns false).

## Expression

```go
type Expression struct {
    source string
    ast    Expr
    program *vmProgram
}
func (e *Expression) Validate(namespaces map[string]string) error
func (e *Expression) EvaluateReuse(ctx context.Context, state *EvalState, node helium.Node) (Result, error)
func (e *Expression) DumpVM(w io.Writer) error
func (e *Expression) AST() Expr
func (e *Expression) StreamInfo() StreamInfo
func (e *Expression) StaticReferences(namespaces map[string]string) StaticReferences
func (e *Expression) String() string

type StaticReferences struct {
    FreeVariables        []string          // variable refs not bound by an enclosing for/let/quantified/inline-function
    TypeNames            []TypeNameRef     // type names from cast/castable/instance of/treat as + element()/attribute()/document-node() kind tests (incl. nested array/map/function item types and path-step node tests)
    FunctionNames        []FunctionNameRef // callee QNames of FunctionCall (incl. constructor calls + arrow targets) and NamedFunctionRef, with ARITY (an inline-function literal call records nothing — it has no named referent)
    SchemaComponentTests []string          // schema-element(E)/schema-attribute(A) node tests (rendered "schema-element(NAME)") — reference GLOBAL declarations
}
type TypeNameRef struct{ Prefix, Name, URI string }            // URI is the RESOLVED namespace
type FunctionNameRef struct{ Prefix, Name, URI string; Arity int } // arity = arg count (arrow incl. LHS) / #arity
```

For a conformance-restricted static context, pair `StaticReferences` with
`StandardFunctionAcceptsArity(uri, name, arity)` (NOT `BuiltinFunctionAcceptsArity`):
it accepts only STANDARD F&O 3.1 functions + built-in type constructors, excluding
helium's forward-looking EXTENSION functions (e.g. `fn:flatten`, `array:flat-map`).

There is no `Expression.Evaluate`; evaluation goes through `Evaluator.Evaluate`
(allocating) or `Expression.EvaluateReuse` (reusing an `EvalState`, see Reuse
below).

`Validate` runs static namespace-prefix validation against the given bindings,
catching undeclared prefixes in function calls, type names, etc. before
evaluation. The same validation runs automatically inside `Evaluate` /
`EvaluateReuse`.

`Compile()` first tries a direct fast path for simple path-like expressions on the lexer token stream, then falls back to parse+lower through the VM backend on the same lexer if the fast path does not apply. It does not retain the parsed AST on the `Expression`; `AST()` reparses from `source` on demand. `CompileExpr()` keeps the caller-provided AST and lowers it without mutating the input tree.

`StreamInfo()` returns a snapshot of precomputed streamability properties (axis usage bitmask, downward steps, function names, etc.). Streamability query helpers live in `internal/xpathstream`, not on the xpath3 package.

`StaticReferences(namespaces)` walks the expression's AST (reparsing from `source` when no AST is retained) and reports its FREE variable references (those not bound by an enclosing for/let/quantified binding or inline-function parameter), its type-name references (cast/castable/instance of/treat as, kind-test type annotations, nested array/map/function item types, and path-step node tests), and its function-call callees (FunctionCall including constructor calls and arrow targets, plus NamedFunctionRef). Every type and function name is RESOLVED to a namespace URI using the supplied in-scope `namespaces` (the same bindings `Validate` takes) plus xpath3's predeclared prefixes — handling prefixed, unprefixed (type → default element namespace; function → fn), and braced-URI `Q{uri}local` forms uniformly — and reports each function reference's static ARITY, so a caller does a pure URI+existence check with no name-form handling of its own. It is side-effect free and intended for schema-compile-time analysis (not the eval hot path); the XSD 1.1 conditional-type-assignment compiler uses it to reject an `xs:alternative` @test that references a variable, an unknown/non-built-in type (XPST0008, via `IsKnownXSDType`), or an unknown, wrong-arity, or non-standard (extension) standard-library function / built-in constructor (XPST0017, via `StandardFunctionAcceptsArity`), or a schema-element()/schema-attribute() node test (which references a global declaration outside the CTA static context), which the CTA static context disallows.

`DumpVM()` writes a textual disassembly of compiled VM instructions. Use it for debugging or tooling around lowered expressions.

## Result

```go
type Result struct { seq Sequence }

func (r *Result) Sequence() Sequence
func (r *Result) IsNodeSet() bool                  // true for empty sequence + all-node sequences
func (r *Result) Nodes() ([]helium.Node, error)     // ErrNotNodeSet if non-nodes present
func (r *Result) IsAtomic() bool
func (r *Result) Atomics() ([]AtomicValue, error)
func (r *Result) IsBoolean() (bool, bool)            // value, ok
func (r *Result) IsNumber() (float64, bool)           // xs:double value, ok
func (r *Result) IsString() (string, bool)
func (r Result) StringValue() string                  // XPath string value of the sequence
func (r Result) Copy() Result                          // deep copy with independent backing (see Reuse)
```

## Regex

Compiled XPath/XML Schema regex, exported for reuse by other packages (notably
xslt3's `xsl:analyze-string`). Flags follow the F&O spec: `i` (case-insensitive),
`m` (multi-line), `s` (dot-all), `x` (free-spacing), `q` (literal).

```go
func CompileRegex(pattern, flags string) (*Regex, error)

func (r *Regex) MatchString(s string) (bool, error)
func (r *Regex) FindAllSubmatchIndex(s string, n int) ([][]int, error) // n<0 = unlimited
func (r *Regex) EachSubmatchIndex(s string, limit int, fn func(m []int) bool) error // limit<=0 = uncapped
```

- `FindAllSubmatchIndex` returns every match, each a flat slice of `(start, end)`
  byte-index pairs for the full match and each capture group (unmatched group =
  `-1, -1`); `nil` means no match.
- `EachSubmatchIndex` **streams** the same successive matches one at a time,
  calling `fn` once per match with the same per-match layout. The slice handed to
  `fn` is valid only for that call — copy it to retain it. Iteration stops early
  (returning `nil`) as soon as `fn` returns false. For the streaming engines
  matches are produced incrementally and never accumulated, so live memory stays
  bounded regardless of match count; this lets a caller enforce a match-count
  budget or honor a cancelled context DURING enumeration. Leading-context
  patterns (e.g. a multi-line `^`, which matches at every line start) cannot be
  streamed incrementally on RE2, so they are matched against the whole string by
  Go's `regexp` in one bounded `FindAllStringSubmatchIndex` pass — staying linear
  (no backtracking-ReDoS blowup for RE2-compatible patterns like `^(a+)+b`).
  `limit` caps how many matches that pass may materialize: a caller enforcing a
  budget of N passes `limit = N+1` so the allocation stays proportional to the
  budget rather than to the input's match count (`limit<=0` = uncapped).

## Configuration types & resolvers

Evaluation is configured exclusively through the `Evaluator` fluent setters
above — there are NO `With*(ctx, ...)` context helpers in `xpath3`. The setters
take these supporting types:

```go
type QualifiedName struct { URI, Name string }

type URIResolver interface {
    ResolveURI(uri string) (io.ReadCloser, error)
}

type CollectionResolver interface {
    ResolveCollection(uri string) (Sequence, error)
    ResolveURICollection(uri string) ([]string, error)
}

type VariableResolver interface { /* lazy variable lookup */ }
type FunctionResolver interface { /* lazy function lookup */ }
type SchemaDeclarations interface { /* schema-aware type info */ }
// OPTIONAL companion to SchemaDeclarations. A provider may also implement it so
// the fn:data / typed-value path can raise FOTY0012 for element-only content
// (no typed value). Reached via type assertion; non-implementers are unaffected.
type ContentTypeKindProvider interface { SchemaTypeContentKind(typeName string) (ContentTypeKind, bool) }

type DecimalFormat = icu.DecimalFormat
type DocOrderCache = internal/xpath.DocOrderCache
```

Pass bindings as plain maps: variables via `Evaluator.Variables(map[string]Sequence)`
(keyed by expanded-QName string) and user functions via
`Evaluator.Functions(byLocal, byQName)` (`byLocal` for unqualified calls,
`byQName map[QualifiedName]Function` for namespaced ones). Maps are cloned unless
`EvalBorrowing` is set. Supply lazy lookups with `Evaluator.VariableResolver` /
`Evaluator.FunctionResolver`.

**Resource loading is opt-in.** Without an explicit `URIResolver` or `HTTPClient` (`Evaluator.URIResolver` / `Evaluator.HTTPClient`), `fn:doc`, `fn:doc-available`, `fn:json-doc`, and the `fn:unparsed-text*` family error with `FODC0002` / `FOUT1170` for every URI — there is no implicit `http.DefaultClient` and no implicit `os.ReadFile`. Built-in helpers in `internal/unparsedtext`:
- `FileURIResolver{BaseDir}` — file resolver confined to `BaseDir` (refuses `..` traversal and absolute paths outside it)
- `NewFileResolver(fs.FS)` — file resolver backed by `io/fs`
- `NewHTTPResolver(*http.Client)` — http(s) resolver; caller owns transport, timeouts, redirect policy

User functions supplied via `Evaluator.Functions` take precedence over built-ins: the resolver checks user-registered functions before the built-in registry, so a user function CAN override a built-in in the `fn:` namespace.

## Low-allocation reuse

For evaluating the same `Expression` repeatedly (e.g. per node in a loop), build
an `EvalState` once and reuse it. This skips per-call allocation of the internal
eval context.

```go
type EvalState struct { /* opaque; NOT safe for concurrent use */ }

func (e Evaluator) NewEvalState(node helium.Node) *EvalState
func (e *Expression) EvaluateReuse(ctx context.Context, state *EvalState, node helium.Node) (Result, error)

func (s *EvalState) SetContextItem(item Item)
func (s *EvalState) SetPosition(pos int)
func (s *EvalState) SetSize(size int)

func (r Result) StringValue() string   // XPath string value; fast path for single node
func (r Result) Copy() Result          // deep copy with independent backing storage
```

`NewEvalState` builds the state from the `Evaluator`'s config (same
initialization path as `Evaluate`), seeding the base focus. Each
`EvaluateReuse(ctx, state, node)` resets per-evaluation fields and starts from
that base focus; a non-nil `node` overrides the context node for that call only.
When `CurrentTime` was not explicitly configured, the clock is refreshed per
call so `fn:current-dateTime()` tracks wall-clock time.

**Result lifetime warning:** the `Result` returned by `EvaluateReuse` is only
valid until the next `EvaluateReuse` call on the same `EvalState` — its backing
storage may be overwritten. Extract everything you need (e.g. via
`Result.StringValue()` / `Result.Nodes()`) before the next call, or retain it
across calls with `Result.Copy()`, which returns a deep copy backed by
independent storage. An `EvalState` is not safe for concurrent use.

`Compile()` uses a direct compile fast path where possible, otherwise lowers parsed AST to VM program using an ownership-taking lowering path that can reuse parsed slices. `CompileExpr()` uses a non-mutating lowering path, with raw AST fallback for unsupported custom Expr implementations.

## Errors

```go
var (
    ErrNotNodeSet               // result is not a node-set
    ErrRecursionLimit           // recursion limit exceeded
    ErrOpLimit                  // operation limit exceeded
    ErrUnknownFunction          // unknown function
    ErrUnknownFunctionNamespace // unknown function namespace prefix
    ErrUnsupportedExpr          // unsupported expression type
    ErrUndefinedVariable        // undefined variable
    ErrTypeMismatch             // type mismatch
    ErrArityMismatch            // arity mismatch
    ErrUnexpectedToken          // unexpected token
    ErrUnexpectedChar           // unexpected character
    ErrUnterminatedString       // unterminated string
    ErrUnknownAxis              // unknown axis
    ErrExpectedToken            // expected token
    ErrExprTooDeep              // expression nesting too deep
    ErrUnionNotNodeSet          // union operands must be node-sets
    ErrPathNotNodeSet           // path expression requires node-set
    ErrUnsupportedBinaryOp      // unsupported binary operator
    ErrNodeSetLimit             // node-set size limit exceeded (alias of internal/xpath.ErrNodeSetLimit)
    ErrRegexMatchLimit          // Regex.EachSubmatchIndex full-context match alloc exceeds the safe ceiling
)
```

All sentinel errors prefixed `xpath3:` except `ErrNodeSetLimit` which uses `xpath:` prefix
(shared sentinel from `internal/xpath`, aliased for `errors.Is` compatibility with `xpath1`).
Wrap with `fmt.Errorf("%w: detail", ErrSomething)`.

## XPathError (structured)

```go
type XPathError struct {
    Code    string    // e.g. "XPTY0004", "FOER0000" (without err: prefix)
    Message string
}
func (e *XPathError) Error() string
func (e *XPathError) Is(target error) bool
```

Standard codes: `XPTY0004` (type error), `FOER0000` (general), `FOTY0012` (typed value undefined — fn:data on element-only complex content), `FOTY0013` (atomization context), `FOAR0001` (division by zero), `FOAR0002` (numeric overflow/underflow, e.g. round precision past the non-terminating cap), `FOAY0001` (array index out of bounds), `FOMX0001` (map duplicate key reject).

`try-catch` matches `*XPathError` by code. Non-`*XPathError` errors propagate through unchanged.
