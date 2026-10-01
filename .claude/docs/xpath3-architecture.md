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
| `axes.go` | `AxisType` enum, `TraverseAxis(ctx, axis, node, maxNodes)`, `AppendAxis(ctx, dst, axis, node, maxNodes)`, all 13 axis functions, namespace helpers; child and descendant walks enumerate through `helium.Children` (owned-child boundary), so an entity reference has no children or descendants |
| `docorder.go` | `DocOrderCache`, `DeduplicateNodes`, `MergeNodeSets`, `DocumentRoot` |
| `union.go` | `UnionNodeSets` (xpath3 union: `MergeNodeSets` result, skipping the index for one element's attributes then children, and merging two sorted operands in one pass), `inElementOrder`, `mergeIncreasingRuns` |
| `steporder.go` | `OrderStepResult` (orders one location step's result, skipping the index when the step shape proves the order), `allOrderedContexts`, `inEntityContent`, `sameDepth`, `isReverseAxis` |
| `stringvalue.go` | `StringValue(Node)` (an element's or document's string-value is its `Content()`: Text/CDATA descendants with entity references expanded through owned children only; a document's leaves out its DTD), `LocalNameOf`, `NodeNamespaceURI`, `NodePrefix` |
| `limits.go` | `DefaultMaxRecursionDepth=5000`, `DefaultMaxNodeSetLength=10_000_000`, `ErrNodeSetLimit` |

### `TraverseAxis` signature

```go
func TraverseAxis(ctx context.Context, axis AxisType, node helium.Node, maxNodes int) ([]helium.Node, error)
func AppendAxis(ctx context.Context, dst []helium.Node, axis AxisType, node helium.Node, maxNodes int) ([]helium.Node, error)
```

`AppendAxis` appends the traversal to a caller-supplied buffer so a multi-step
walk accumulates in one slice instead of one slice per context node. The
descendant and descendant-or-self axes write straight into `dst`; the others
delegate to `TraverseAxis` and append its result. `maxNodes` caps what THIS
traversal appends, so the limit means the same thing in both functions.

### `DocOrderCache` signature

```go
type DocOrderCache struct { ... }
func (c *DocOrderCache) BuildFrom(root helium.Node)
func (c *DocOrderCache) Position(n helium.Node) int
func (c *DocOrderCache) Compare(a, b helium.Node) int
func (c *DocOrderCache) Less(a, b helium.Node) bool
func (c *DocOrderCache) Reset() // clear cache; callers MUST call after mutating the document
func DeduplicateNodes(nodes []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
func MergeNodeSets(a, b []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
func UnionNodeSets(nodes []helium.Node, split int, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
func DocumentRoot(n helium.Node) helium.Node
func OrderStepResult(out, inputs []helium.Node, axis AxisType, cache *DocOrderCache, maxNodes int) ([]helium.Node, error)
```

`OrderStepResult` returns what `DeduplicateNodes(out, cache, maxNodes)` returns for a
location step whose context list `inputs` is sorted and duplicate-free. It skips the
whole-document index when the result has at most one node, when the step ran from one
input (reverse axes are reversed in place), or when the axis is child, attribute, self,
namespace or parent and every input has the same depth (parent results drop adjacent
duplicates). Both skips require every input to be a document, element, attribute,
namespace, text, CDATA, comment, PI, entity-reference or DTD node (`allOrderedContexts`).
An `Entity` input, and any input inside an entity's parsed content (an `Entity` among
its ancestors, `inEntityContent`; the one-input skip walks the ancestors, the same-depth
skip checks while `sameDepth` walks them), always sorts: the index places an entity's
content at the last reference to it while a raw axis walk from inside the content climbs
through the `Entity` and the DTD, so a following step from there, or the child steps of
two entities, or of an entity and an element, come out of document order. A skipping
step reserves its document's registration order in the cache (unexported
`reserveDocument`) without indexing it, so the order between documents stays the one
indexing would have produced. `Position`, `Compare` and every indexing path index a
reserved document on first use, under its reserved order. xpath1 and xpath3 end every
axis step of a location path with it. In xpath3 the other node-ordering sites keep
`DeduplicateNodes`: a path step whose step expression is not an axis step (`E1/(a|b)`,
`E1/f()`, `E1/$v`, `evalPathStepExpr`), the merge of the per-node results of `E1/E2`
(`evalPathExpr`), and intersect/except. xpath3 union uses `UnionNodeSets` and xpath1
union uses `MergeNodeSets`.

`UnionNodeSets(nodes, split, ...)` takes both operands in one buffer (`nodes[:split]`
then `nodes[split:]`) and returns what `MergeNodeSets(nodes[:split], nodes[split:], ...)`
returns, with the same documents registered in the cache in the same order. When
`nodes` is a subsequence of one element's attributes (in `ForEachAttribute` order)
followed by that element's owned children (in `helium.Children` order),
`inElementOrder`, it returns `nodes` with its capacity clamped and only reserves the
document (unexported `reserveDocumentOf`, which does nothing for an already-indexed
node). The index numbers an element, its attributes and its owned children's subtrees
in exactly that order on every walk that reaches the element, and leaves all of them
unindexed together otherwise, so that order is the sorted order in both cases; namespace
nodes and nodes of another element fail the check. Otherwise it resolves the sort keys
and, when the keys of each operand are strictly increasing, merges them in one pass
(`mergeIncreasingRuns`), emitting a node common to both once and falling back on two
different nodes with equal keys (namespace nodes of one element, unindexed nodes).
Anything else takes the `MergeNodeSets` body over the resolved keys
(`mergeNodeSetsWithKeys`).

### `StringValue` signatures

```go
func StringValue(n helium.Node) string
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
| `consts.go` | Shared string constants (keywords, JSON kind labels, option values) |
| `lexer.go` | One-pass tokenizer |
| `parser.go` | Recursive descent parser |
| `compile_direct.go` | `Compile()` fast path for simple path-like expressions and simple predicate comparisons, with shared parser fallback |
| `eval.go` | `evalContext`, raw AST eval trampoline |
| `eval_path.go` | Location paths, node tests, predicates, literal/variable/sequence eval |
| `eval_operators.go` | Binary/unary logic ops, concat, simple map, range, union, intersect/except, filter, path steps |
| `eval_arithmetic.go` | Integer/decimal/float arithmetic, unary negation, type promotion helpers |
| `eval_control.go` | FLWOR, quantified, if/else, try/catch, lookup expressions |
| `eval_types.go` | instanceof, cast, castable, treat-as, sequence type matching |
| `eval_funcall.go` | Function calls, dynamic calls, inline functions, partial application, map/array constructors |
| `eval_reuse.go` | Shared eval helpers reused by VM |
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
| `regex.go` | Adapter to `internal/xsdregex` (XPath regex→Go regex translation); wraps errors as FORX0002. Shared with `xsd` so pattern facets use the same translator |
| `regex_cache.go` | Bounded LRU cache (`regexLRUCache`) for compiled XPath regexes keyed by pattern+flags; 1024-entry cap with LRU eviction |
| `static_check.go` | Static expression checks |
| `streamability.go` | Internal streamability precomputation (unexported) plus exported `StreamInfo` struct + accessor; query helpers moved to `internal/xpathstream` |
| `node_identity.go` | Node identity comparison |
| `uri_resolution.go` | URI resolution for fn:doc, fn:unparsed-text |
| `arithmetic_datetime.go` | Date/time arithmetic |
| `parse_ietf_date.go` | IETF date format parsing |
| `format_datetime.go` | format-date/dateTime/time |
| `format_integer.go` | format-integer |
| `format_number.go` | format-number |
| `function_library.go` | Function library management |
| `function_signatures.go` | Function signature declarations |
| `functions.go` | `Function` interface, `FunctionContext`, registry, `builtinFunc`, `registerFn`/`registerNS` helpers; boolean, not, true, false, error, trace |
| `functions_node.go` | node-name, local-name, namespace-uri, name, root, path, id, lang, etc. |
| `functions_string.go` | string ops, regex (matches, replace, tokenize), upper/lower-case |
| `functions_numeric.go` | abs, ceiling, floor, round, round-half-to-even |
| `functions_aggregate.go` | count, sum, avg, min, max, distinct-values |
| `functions_sequence.go` | empty, exists, head, tail, subsequence, insert-before, remove, reverse, etc. |
| `functions_datetime.go` | date/time constructors, accessors, arithmetic |
| `functions_uri.go` | encode-for-uri, iri-to-uri, escape-html-uri, resolve-uri, base-uri, document-uri |
| `functions_qname.go` | QName, resolve-QName, namespace-uri-for-prefix, in-scope-prefixes |
| `functions_hof.go` | for-each, filter, fold-left, fold-right, apply, function-lookup/arity/name |
| `functions_map.go` | map:merge, map:size, map:keys, map:contains, map:get, map:put, etc. |
| `functions_array.go` | array:size, array:get, array:put, array:append, array:subarray, etc. |
| `functions_math.go` | math:pi, math:exp, math:log, math:sqrt, math:sin, math:cos, etc. |
| `functions_json.go` | parse-json, json-doc |
| `functions_json_xml.go` | json-to-xml, xml-to-json |
| `functions_serialize.go` | serialize |
| `functions_misc.go` | static-base-uri, default-collation, environment-variable, current-dateTime, generate-id |
| `functions_constructors.go` | XSD typed atomic constructors (incl. xs:error) |
| `functions_unparsed_text.go` | unparsed-text, unparsed-text-lines, unparsed-text-available |
| `errors.go` | `XPathError` (structured error with code), standard error constructors |
| `doc.go` | Package documentation |

## Runtime Model

- `Compile()` first tries a direct compile fast path for simple path-like expressions and simple predicate comparisons,
  then falls back to shared parse+lower on the same token stream
- Direct fast-path parsing emits `vmLocationPathExpr` / `vmLocationStep` payloads immediately for simple location paths
  instead of building AST `LocationPath` / `Step` nodes first
- Shared fallback path is `string -> lexer ([]Token) -> parser (Expr AST) -> VM lowering`
- String-based `Compile()` uses an ownership-taking lowering path that can reuse parsed slices, then keeps only `source`
  + `vmProgram`; AST-inspection helpers reparse on demand
- Non-trivial lowered nodes become indexed `vmInstruction`s; `vmInstruction` now stores a generic payload, not just AST
  `Expr`s
- Location paths and `PathExpr` path segments are lowered to VM-specific payload types (`vmLocationPathExpr`,
  `vmLocationStep`, `vmPathExpr`) instead of reusing AST `LocationPath` nodes in the instruction stream
- Common location-path predicates are also lowered to VM-specific inline predicate payloads for hot cases such as `[N]`,
  `[position() = N]`, `[@attr]`, and `[@attr = "literal"]`
- Child Expr references inside lowered payloads become `compiledExprRef` when they need their own instruction slot
- VM executes compiled refs by opcode, then reuses existing `eval_*` helpers via `exprEvaluator`
- `CompileExpr()` uses non-mutating lowering and keeps the caller-provided AST
- Raw `eval()` remains as fallback for unlowered `CompileExpr` inputs
