# XPath 3.1 — Architecture

## Package Layout

```
xpath3/              → XPath 3.1 (public)
internal/xpath/      → shared infra (axes, docorder, stringvalue, limits)
xpath1/              → XPath 1.0 (unchanged public API, refactored to use internal/xpath)
```

## Import Graph

```
xpath3 → internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence → helium
xpath1 → internal/xpath → helium
```

`xpath3` NEVER imports `xpath1`. `xpath1` NEVER imports `xpath3`.

## Data Flow

```
string → lexer ([]Token) → parser (Expr AST) → VM lowering (`vmProgram`) → VM execution → Sequence → Result
                                        └→ on-demand reparse for `AST()` / streamability helpers
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
| `float_value.go` | Float/double special value handling |
| `sequence.go` | `Sequence`, helpers: `SingleNode`, `SingleString`, `EBV`, `AtomizeSequence` |
| `expr.go` | All AST node types, `NodeTest` variants, `SequenceType` |
| `token.go` | `TokenType` constants (60+) |
| `lexer.go` | One-pass tokenizer |
| `parser.go` | Recursive descent parser |
| `compile_direct.go` | `Compile()` fast path for simple path-like expressions and simple predicate comparisons, with shared parser fallback |
| `compiler.go` | AST lowering to VM instructions |
| `eval.go` | `evalContext`, raw AST eval trampoline |
| `eval_dispatch.go` | Shared dispatch used by raw eval + VM |
| `eval_path.go` | Location paths, node tests, predicates, literal/variable/sequence eval |
| `eval_operators.go` | Binary/unary logic ops, concat, simple map, range, union, intersect/except, filter, path steps |
| `eval_arithmetic.go` | Integer/decimal/float arithmetic, unary negation, type promotion helpers |
| `eval_control.go` | FLWOR, quantified, if/else, try/catch, lookup expressions |
| `eval_types.go` | instanceof, cast, castable, treat-as, sequence type matching |
| `eval_funcall.go` | Function calls, dynamic calls, inline functions, partial application, map/array constructors |
| `eval_reuse.go` | Shared eval helpers reused by VM |
| `eval_state.go` | Eval context state management |
| `evaluator.go` | Expression evaluator interface |
| `vm.go` | AST lowering to indexed instruction graph + VM executor |
| `vm_dump.go` | Text disassembly for compiled VM instructions |
| `compare.go` | `GeneralCompare`, `ValueCompare`, `NodeCompare`, type promotion |
| `cast.go` | `CastAtomic`, `CastFromString` |
| `cast_numeric.go` | Numeric-specific casting |
| `cast_string.go` | String-specific casting |
| `cast_datetime.go` | Date/time casting |
| `context.go` | Context configuration (evalConfig) |
| `variables.go` | Variable binding management |
| `collation.go` | Collation support |
| `regex.go` | XPath regex→Go regex translation |
| `regex_public.go` | Public regex API |
| `static_check.go` | Static expression checks |
| `streamability.go` | Internal streamability precomputation (unexported); query helpers moved to `internal/xpathstream` |
| `stream_info.go` | Exported `StreamInfo` struct + accessor method |
| `node_identity.go` | Node identity comparison |
| `uri_resolution.go` | URI resolution for fn:doc, fn:unparsed-text |
| `arithmetic_datetime.go` | Date/time arithmetic |
| `parse_ietf_date.go` | IETF date format parsing |
| `format_datetime.go` | format-date/dateTime/time |
| `format_integer.go` | format-integer |
| `format_number.go` | format-number |
| `function_library.go` | Function library management |
| `function_signatures.go` | Function signature declarations |
| `functions.go` | `Function` interface, `FunctionContext`, registry, `builtinFunc`, `registerFn`/`registerNS` helpers |
| `functions_node.go` | node-name, local-name, namespace-uri, name, root, path, id, lang, etc. |
| `functions_string.go` | string ops, regex (matches, replace, tokenize), upper/lower-case |
| `functions_numeric.go` | abs, ceiling, floor, round, round-half-to-even |
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
| `functions_json.go` | parse-json, json-doc, json-to-xml, xml-to-json, serialize |
| `functions_error.go` | error, trace |
| `functions_misc.go` | static-base-uri, default-collation, environment-variable, current-dateTime, generate-id |
| `functions_constructors.go` | XSD typed atomic constructors |
| `functions_unparsed_text.go` | unparsed-text, unparsed-text-lines, unparsed-text-available |
| `errors.go` | `XPathError` (structured error with code), standard error constructors |
| `doc.go` | Package documentation |

## Runtime Model

- `Compile()` first tries a direct compile fast path for simple path-like expressions and simple predicate comparisons, then falls back to shared parse+lower on the same token stream
- Direct fast-path parsing emits `vmLocationPathExpr` / `vmLocationStep` payloads immediately for simple location paths instead of building AST `LocationPath` / `Step` nodes first
- Shared fallback path is `string -> lexer ([]Token) -> parser (Expr AST) -> VM lowering`
- String-based `Compile()` uses an ownership-taking lowering path that can reuse parsed slices, then keeps only `source` + `vmProgram`; AST-inspection helpers reparse on demand
- Non-trivial lowered nodes become indexed `vmInstruction`s; `vmInstruction` now stores a generic payload, not just AST `Expr`s
- Location paths and `PathExpr` path segments are lowered to VM-specific payload types (`vmLocationPathExpr`, `vmLocationStep`, `vmPathExpr`) instead of reusing AST `LocationPath` nodes in the instruction stream
- Common location-path predicates are also lowered to VM-specific inline predicate payloads for hot cases such as `[N]`, `[position() = N]`, `[@attr]`, and `[@attr = "literal"]`
- Child Expr references inside lowered payloads become `compiledExprRef` when they need their own instruction slot
- VM executes compiled refs by opcode, then reuses existing `eval_*` helpers via `exprEvaluator`
- `CompileExpr()` uses non-mutating lowering and keeps the caller-provided AST
- Raw `eval()` remains as fallback for unlowered `CompileExpr` inputs
