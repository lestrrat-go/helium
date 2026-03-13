# XPath 3.1 — Architecture

## Package Layout

```
xpath3/              → XPath 3.1 (public)
internal/xpath/      → shared infra (axes, docorder, stringvalue, limits)
xpath1/              → XPath 1.0 (unchanged public API, refactored to use internal/xpath)
```

## Import Graph

```
xpath3 → internal/xpath → helium
xpath1 → internal/xpath → helium
```

`xpath3` NEVER imports `xpath1`. `xpath1` NEVER imports `xpath3`.

## Data Flow

```
string → lexer ([]Token) → parser (Expr AST) → eval(evalContext, Expr) → Sequence → Result
```

## `internal/xpath` Files

| File | Contents |
|------|----------|
| `axes.go` | `AxisType` enum, `TraverseAxis(axis, node, maxNodes)`, all 13 axis functions, namespace helpers |
| `docorder.go` | `DocOrderCache`, `DeduplicateNodes`, `MergeNodeSets`, `DocumentRoot` |
| `stringvalue.go` | `StringValue(Node)`, `appendTextDescendants` (unexported, iterative stack-based traversal), `LocalNameOf`, `NodeNamespaceURI`, `NodePrefix` |
| `limits.go` | `DefaultMaxRecursionDepth=5000`, `DefaultMaxNodeSetLength=10_000_000`, `ErrNodeSetLimit` |

### `TraverseAxis` signature

```go
func TraverseAxis(axis AxisType, node helium.Node, maxNodes int) ([]helium.Node, error)
```

### `DocOrderCache` signature

```go
type DocOrderCache struct { ... }
func (c *DocOrderCache) BuildFrom(root helium.Node)
func (c *DocOrderCache) Position(n helium.Node) int
func DeduplicateNodes(nodes []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
func MergeNodeSets(a, b []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
func DocumentRoot(n helium.Node) helium.Node
```

### `StringValue` signatures

```go
func StringValue(n helium.Node) string
// unexported: appendTextDescendants(b *strings.Builder, root helium.Node)
func LocalNameOf(n helium.Node) string
func NodeNamespaceURI(n helium.Node) string
func NodePrefix(n helium.Node) string
```

## `xpath3` Files

| File | Contents |
|------|----------|
| `xpath3.go` | Public API: `Compile`, `Evaluate`, `Find`, `Expression`, `Result`, `Context`, errors |
| `types.go` | `Item`, `AtomicValue`, `NodeItem`, `FunctionItem`, `MapItem`, `ArrayItem`, atomic type consts |
| `sequence.go` | `Sequence`, helpers: `SingleNode`, `SingleString`, `EBV`, `AtomizeSequence` |
| `expr.go` | All AST node types, `NodeTest` variants, `SequenceType` |
| `token.go` | `TokenType` constants (60+) |
| `lexer.go` | One-pass tokenizer |
| `parser.go` | Recursive descent parser |
| `eval.go` | `evalContext`, main `eval()` dispatch switch |
| `eval_path.go` | Location paths, node tests, predicates, literal/variable/sequence eval |
| `eval_operators.go` | Binary/unary logic ops, concat, simple map, range, union, intersect/except, filter, path steps |
| `eval_arithmetic.go` | Integer/decimal/float arithmetic, unary negation, type promotion helpers |
| `eval_control.go` | FLWOR, quantified, if/else, try/catch, lookup expressions |
| `eval_types.go` | instanceof, cast, castable, treat-as, sequence type matching |
| `eval_funcall.go` | Function calls, dynamic calls, inline functions, partial application, map/array constructors |
| `compare.go` | `GeneralCompare`, `ValueCompare`, `NodeCompare`, type promotion |
| `cast.go` | `CastAtomic`, `CastFromString` |
| `functions.go` | `Function` interface, `FunctionContext`, registry, `builtinFunc`, `registerFn`/`registerNS` helpers |
| `functions_node.go` | node-name, local-name, namespace-uri, name, root, path, id, lang, etc. |
| `functions_string.go` | string ops, regex (matches, replace, tokenize), upper/lower-case |
| `functions_numeric.go` | abs, ceiling, floor, round, round-half-to-even, format-integer, format-number |
| `functions_boolean.go` | boolean, not, true, false |
| `functions_aggregate.go` | count, sum, avg, min, max, distinct-values |
| `functions_sequence.go` | empty, exists, head, tail, subsequence, insert-before, remove, reverse, etc. |
| `functions_datetime.go` | date/time constructors, accessors, arithmetic |
| `functions_uri.go` | encode-for-uri, iri-to-uri, escape-html-uri, resolve-uri, base-uri, document-uri |
| `functions_qname.go` | QName, resolve-QName, namespace-uri-for-prefix, in-scope-prefixes |
| `functions_hof.go` | for-each, filter, fold-left, fold-right, apply, function-lookup/arity/name |
| `functions_map.go` | map:merge, map:size, map:keys, map:contains, map:get, map:put, etc. |
| `functions_array.go` | array:size, array:get, array:put, array:append, array:subarray, etc. |
| `functions_math.go` | math:pi, math:exp, math:log, math:sqrt, math:sin, math:cos, etc. |
| `functions_error.go` | error, trace |
| `functions_misc.go` | static-base-uri, default-collation, environment-variable, current-dateTime, generate-id |
| `errors.go` | `XPathError` (structured error with code), standard error constructors |
