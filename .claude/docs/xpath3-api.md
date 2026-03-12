# XPath 3.1 — Public API

Mirrors `xpath1` shape. Key difference: `Result` wraps `Sequence` (not union struct).

## Entry Points

```go
func Compile(expr string) (*Expression, error)
func MustCompile(expr string) *Expression
func Evaluate(ctx context.Context, node helium.Node, expr string) (*Result, error)
func Find(ctx context.Context, node helium.Node, expr string) ([]helium.Node, error)
```

## Expression

```go
type Expression struct {
    source string
    ast    Expr
}
func (e *Expression) Evaluate(ctx context.Context, node helium.Node) (*Result, error)
func (e *Expression) String() string
```

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
```

## Context

```go
type Context struct {
    namespaces       map[string]string
    variables        map[string]Sequence
    functions        map[string]Function
    functionsNS      map[QualifiedName]Function
    opLimit          int
    implicitTimezone *time.Location
    defaultLanguage  string
    defaultCollation string
    defaultDecimal   *DecimalFormat
    decimalFormats   map[QualifiedName]DecimalFormat
    baseURI          string
    uriResolver      URIResolver
    httpClient       *http.Client
}

type ContextOption func(*Context)

func WithNamespaces(ns map[string]string) ContextOption       // defensively copied
func WithVariables(vars map[string]Sequence) ContextOption    // defensively copied
func WithOpLimit(limit int) ContextOption
func WithFunctions(fns map[string]Function) ContextOption     // defensively copied
func WithFunctionsNS(fns map[QualifiedName]Function) ContextOption // defensively copied
func WithImplicitTimezone(loc *time.Location) ContextOption
func WithDefaultLanguage(lang string) ContextOption
func WithDefaultCollation(uri string) ContextOption
func WithDefaultDecimalFormat(df DecimalFormat) ContextOption
func WithNamedDecimalFormats(dfs map[QualifiedName]DecimalFormat) ContextOption // defensively copied
func WithBaseURI(uri string) ContextOption
func WithURIResolver(r URIResolver) ContextOption
func WithHTTPClient(client *http.Client) ContextOption

func NewContext(ctx context.Context, opts ...ContextOption) context.Context
func GetContext(ctx context.Context) *Context

type QualifiedName struct { URI, Name string }

type URIResolver interface {
    ResolveURI(uri string) (io.ReadCloser, error)
}
```

User functions registered via `WithFunctionsNS` CANNOT override built-ins in `fn:` namespace.

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

Standard codes: `XPTY0004` (type error), `FOER0000` (general), `FOTY0013` (atomization context), `FOAR0001` (array index OOB), `FOAR0002` (array index type), `FOMX0001` (map duplicate key reject).

`try-catch` matches `*XPathError` by code. Non-`*XPathError` errors propagate through unchanged.
