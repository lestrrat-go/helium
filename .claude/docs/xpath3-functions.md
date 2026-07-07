# XPath 3.1 — Function System

## Function Interface

```go
type Function interface {
    MinArity() int
    MaxArity() int  // -1 = variadic
    Call(ctx context.Context, args []Sequence) (Sequence, error)
}

type FunctionContext interface {
    Node()       helium.Node
    Position()   int
    Size()       int
    Namespace(prefix string) (string, bool)
    Variable(name string)   (Sequence, bool)
}
```

## Registry

Package-level `builtinFunctions3 map[QualifiedName]Function` populated in `init()`.
Registered via `registerFn`/`registerNS` (STANDARD F&O 3.1) or `registerFnExt`/
`registerNSExt` (helium EXTENSIONS beyond F&O 3.1 — forward-looking XPath/XQuery 4.0,
currently `fn:flatten` and `array:flat-map`; `builtinFunc.extension = true`).
Membership/arity queries: `BuiltinFunctionAcceptsArity(uri,name,arity)` accepts any
registered function; `StandardFunctionAcceptsArity(uri,name,arity)` EXCLUDES
extensions — use it for conformance-restricted static contexts (XSD 1.1 CTA @test).
Extensions are also STATIC-CALL-ONLY: `fn:function-lookup` (`lookupFunctionItem`,
`functions_hof.go`) skips an extension built-in (`isExtensionBuiltin`) so it is not
dynamically reachable, closing the function-lookup bypass of the static CTA gate.

## Namespace URIs

| Prefix | URI |
|--------|-----|
| `fn:` (default) | `http://www.w3.org/2005/xpath-functions` |
| `math:` | `http://www.w3.org/2005/xpath-functions/math` |
| `map:` | `http://www.w3.org/2005/xpath-functions/map` |
| `array:` | `http://www.w3.org/2005/xpath-functions/array` |
| `err:` | `http://www.w3.org/2005/xqt-errors` |
| `xs:` | `http://www.w3.org/2001/XMLSchema` |

`fn:` is default → `string()` and `fn:string()` resolve to same function.
Explicit prefixed names + `Q{uri}local` names NEVER fall back to `fn:` on miss.

Typed helper coercions for `xs:string?` + `xs:integer` enforce cardinality.
Sequences with length `> 1` → `XPTY0004`; helpers do not truncate to first item.
Signature-sensitive string/regex/URI builtin call sites + `||` string coercion also enforce single-item cardinality and propagate atomization/type errors.

`evalFunctionCall` (the static call path) enforces the declared parameter
signature for every resolved function before invoking it: it looks up
`paramTypes` via the `function_signatures.go` registry (then
`TypedFunction`/`TypedFunctionByArity`) the same way `evalNamedFunctionRef`
does, and runs `coerceToSequenceType(arg, paramTypes[i], ec)` per argument,
raising `XPTY0004` on mismatch. Functions with no registered signature
(`paramTypes == nil`) are not type-checked. `coerceToSequenceType` atomizes via
`AtomizeSequence` (so list-typed nodes/arrays expand correctly) and falls back
to `ec.schemaDeclarations.IsSubtypeOf` for user-defined schema types.

## Functions by File

### `functions_node.go`
`node-name`, `nilled`, `string`, `data`, `base-uri`, `document-uri`, `root`, `path`, `has-children`, `innermost`, `outermost`, `id`, `idref`, `lang`, `local-name`, `name`, `namespace-uri`, `number`, `generate-id`

`fn:nilled` returns the PSVI [nil] property of an element node (`()` for a non-element): true iff the node is in the evaluator's `NilledElements` set (`Evaluator.NilledElements`, populated from `xsd.Validator.NilledElements`), else false. The same set drives two other nilled-aware behaviors: `fn:data` / atomization gives a nilled element the empty typed value `()` (via `fnDataItemCheck`, checked before content-kind), and an `element(name, type)` instance-of test excludes a nilled element while `element(name, type?)` matches it (`eval_path.go` `ElementTest`). Non-schema-aware evaluation leaves the set nil → every element is not-nilled.

### `functions_string.go`
`codepoints-to-string`, `string-to-codepoints`, `compare`, `codepoint-equal`, `concat`, `string-join`, `substring`, `string-length`, `normalize-space`, `normalize-unicode`, `upper-case`, `lower-case`, `translate`, `contains`, `starts-with`, `ends-with`, `substring-before`, `substring-after`, `matches`, `replace`, `tokenize`, `analyze-string` (partial; result DOM built with helium)

Regex: use Go `regexp` package by default; fall back to `github.com/dlclark/regexp2` for patterns requiring backreferences, character class subtraction, or large quantifiers. Map XPath flags (`i`,`m`,`s`,`x`) to Go equivalents.
Compiled regexes are cached by pattern + flags pair so repeated literal calls do not repay translation/compilation cost. The cache (`regexLRUCache` in `regex_cache.go`) is a bounded LRU with a 1024-entry cap and least-recently-used eviction, so many distinct dynamic patterns cannot grow process memory without limit.

**Resource bounds (XPATH3-102/105).** `string-to-codepoints` charges `fnCountOp` per produced codepoint so a huge input cannot build an item sequence below the node-set cap but above `OpLimit`. `analyze-string` STREAMS its regex matches via `compiledXPathRegex.eachStringSubmatchIndex` (the internal form of the public `Regex.EachSubmatchIndex`) rather than calling `FindAllStringSubmatchIndex` up front: when an op budget is in force it passes `fnRemainingOps(ec)+1` as the match limit and charges `fnCountOp` BEFORE building each match's result nodes, so an input with millions of matches rejects with `ErrOpLimit` (or honors cancellation) without ever materializing the O(matches) index slice. A non-streamable leading-context pattern over an oversized input surfaces `ErrRegexMatchLimit` as-is (errors.Is-compatible); other engine errors map to `FORX0002`.

### `functions_numeric.go`
`abs`, `ceiling`, `floor`, `round`, `round-half-to-even`

### `functions_constructors.go`
Typed atomic constructors for XSD types

### `functions.go`
Boolean: `boolean`, `not`, `true`, `false`
Error/trace: `error` (raises `FOER0000` with optional code/description/value), `trace` (logs and returns input)

### `functions_aggregate.go`
`count`, `avg`, `max`, `min`, `sum`, `distinct-values`

### `functions_sequence.go`
`empty`, `exists`, `head`, `tail`, `insert-before`, `remove`, `reverse`, `subsequence`, `unordered`, `zero-or-one`, `one-or-more`, `exactly-one`, `deep-equal`, `index-of`

### `functions_datetime.go`
Constructors: `dateTime`
Accessors: `year-from-dateTime`, `month-from-dateTime`, `day-from-dateTime`, `hours-from-dateTime`, `minutes-from-dateTime`, `seconds-from-dateTime`, `timezone-from-dateTime`, (same for date/time variants), `years-from-duration`, `months-from-duration`, `days-from-duration`, `hours-from-duration`, `minutes-from-duration`, `seconds-from-duration`
Formatting: `format-date`, `format-dateTime`, `format-time`
Misc: `adjust-dateTime-to-timezone`, `adjust-date-to-timezone`, `adjust-time-to-timezone`

### `functions_uri.go`
`resolve-uri`, `encode-for-uri`, `iri-to-uri`, `escape-html-uri`, `base-uri`, `document-uri`

### `functions_json.go`
`parse-json`, `json-doc`

### `functions_json_xml.go`
`json-to-xml`, `xml-to-json`

`json-to-xml` with the `validate:true()` option type-annotates the result tree:
each result element records the type defined by the json-to-xml result schema
(`schema-for-json.xsd`, target namespace = fn namespace) keyed by node into
`ec.typeAnnotations`, so downstream `instance of element(j:map, j:mapType)` /
`element(j:string, j:stringType)` / `element(j:boolean, xs:boolean)` tests over
the produced tree succeed. The mapping is a fixed function of each node's JSON
kind (`mapType`/`arrayType`/`stringType`/`numberType`/`nullType` in the fn
namespace; `boolean` → `xs:boolean`), so no general XSD validation pass runs.
`ec.typeAnnotations` (handed in by the caller and shared across concurrent
`Evaluate` calls) is copied into a fresh per-evaluation map before the new nodes'
annotations are merged in — the shared config map is never mutated.

### `functions_serialize.go`
`serialize`

`json-doc` uses same URI resolution/resource-loading stack as `doc` + `unparsed-text`:
`WithBaseURI` → relative resolution, `WithURIResolver` → resolver for all schemes, `WithHTTPClient` → opt-in HTTP fetch.

**Secure by default.** With no `URIResolver` and no `HTTPClient`, `fn:doc`, `fn:doc-available`, `fn:json-doc`, `fn:unparsed-text`, `fn:unparsed-text-available`, and `fn:unparsed-text-lines` cannot reach the filesystem or network — they error with `FODC0002` / `FOUT1170`. To allow access, supply a `URIResolver` (e.g. `unparsedtext.FileURIResolver{BaseDir:...}`, `unparsedtext.NewFileResolver(fs.FS)`, or `unparsedtext.NewHTTPResolver(*http.Client)`), or set an explicit `HTTPClient` whose transport/timeouts/redirect policy you control. `fn:doc` parses retrieved bytes with `BlockXXE(true).AllowNetwork(false)` so the returned document cannot pull additional externals.

### `functions_qname.go`
`QName`, `resolve-QName`, `prefix-from-QName`, `local-name-from-QName`, `namespace-uri-from-QName`, `namespace-uri-for-prefix`, `in-scope-prefixes`

### `functions_hof.go`
`for-each`, `filter`, `fold-left`, `fold-right`, `apply`, `function-lookup`, `function-arity`, `function-name`

**Resource bounds.** Accumulating higher-order / map / array built-ins honor the same limits as the evaluator's accumulation sites: per-iteration `ec.countOps` (op limit + context cancellation) and a `maxNodes` length cap (sequence/node-set length → `ErrNodeSetLimit`). Shared helpers `fnMaxNodes(ec)` / `fnCountOp(ctx, ec)` (in `functions_hof.go`) default to `maxNodeSetLength` and stay safe when `ec == nil` (function called outside an evaluation). Covers `for-each`, `for-each-pair`, `filter`, `fold-left`/`fold-right`, `map:for-each`, `map:find`, `array:join`, `array:flat-map`, `array:filter`, `array:for-each`/`for-each-pair`, `array:fold-left`/`fold-right`.

Built-ins that clone or materialize a whole sub-sequence in **one shot** rather than item-by-item (`array:for-each`, `array:for-each-pair`, `array:join`, `array:flat-map`, `map:find`) additionally charge the sub-sequence length against the op-counter via `fnCountOps(ctx, ec, n)` **before** the bulk `NewArray`/`cloneSequence`/append. This rejects a result/member-list/value that is below `maxNodes` but above `OpLimit` with `ErrOpLimit` — length-only (`maxNodes`) checking would otherwise let it be cloned unbounded. Because `NewArray` clones **each** member sequence, `array:join` and `array:flat-map` charge `seqLen(member)` **per member** (not one op per member): a single member holding many items below `maxNodes` but above `OpLimit` is still rejected.

Sites that build a result array with one **member** per callback result / matched value (`array:for-each`, `array:for-each-pair`, `map:find`) bound the **member count** independently of the item count: a callback returning an empty sequence (or an empty matched map value) adds zero items but still adds a member, so `len(results)+1 > maxNodes` is checked before each append, otherwise many empty results could build an array with more than `maxNodes` members. `array:flat-map` charges one op per callback-result item up front (including items that are empty arrays whose member expansion appends nothing), so many empty arrays cannot bypass `OpLimit`.

The cap must trigger **before** materializing a lazy result, so the bound holds for genuinely lazy inputs (e.g. an `EvalBorrowing` variable bound to a large `NewRangeSequence`, or a `VariableResolver` result) — not only after a slice has already been allocated:
- Callback-result accumulators (`for-each`, `for-each-pair`, `map:for-each`) use `appendBoundedSeq(dst, src, maxNodes)` (in `eval_operators.go`), which iterates `seqItems(src)` one item at a time and checks `maxNodes` before each append, so a callback returning an unbounded lazy `Sequence` is rejected without ever materializing it. `appendBounded` (the slice-taking variant) is still used where the source is already a materialized slice.
- Fold accumulators check `seqLen(acc) > maxNodes` after each step; `seqLen` is O(1) on a lazy range, so an oversized lazy accumulator is rejected without materialization.
- The iterative walkers `array:flatten` and `map:find` (`mapFindIter`) use an explicit stack of `seqCursor` frames (a member/value list + member/item index, in `functions_array.go`) that yields one `Item` per `next()` via `Sequence.Get`, instead of expanding child sequences into temporary `[]Item` slices. This bounds depth (no goroutine-stack recursion) and width (no per-level slice amplification), and stays op-counted and `maxNodes`-bounded. Maps and arrays are eager value types — `NewMap`/`NewArray` clone member/value sequences at construction — so member/value sequences are not lazy in practice; the cursor still avoids the intermediate slice amplification.

### `functions_map.go`
`map:merge`, `map:size`, `map:keys`, `map:contains`, `map:get`, `map:put`, `map:entry`, `map:remove`, `map:for-each`, `map:find`

### `functions_array.go`
`array:size`, `array:get`, `array:put`, `array:append`, `array:subarray`, `array:remove`, `array:insert-before`, `array:head`, `array:tail`, `array:reverse`, `array:join`, `array:flat-map`, `array:filter`, `array:fold-left`, `array:fold-right`, `array:for-each`, `array:for-each-pair`, `array:sort`

### `functions_math.go`
`math:pi`, `math:exp`, `math:exp10`, `math:log`, `math:log10`, `math:pow`, `math:sqrt`, `math:sin`, `math:cos`, `math:tan`, `math:asin`, `math:acos`, `math:atan`, `math:atan2`

All delegate to Go `math` package.

### `functions_misc.go`
`static-base-uri`, `default-collation`, `available-environment-variables`, `environment-variable`, `current-dateTime`, `current-date`, `current-time`, `implicit-timezone`, `generate-id`

### `format_number.go`
`format-number` (full ICU-style decimal format patterns)

### `format_integer.go`
`format-integer` (word, ordinal, cardinal patterns)

### `format_datetime.go`
`format-date`, `format-dateTime`, `format-time`

### `functions_unparsed_text.go`
`unparsed-text`, `unparsed-text-lines`, `unparsed-text-available`

`fn:unparsed-text-lines` bounds line production by the **effective** budget in
force — `min(fnMaxNodes(ec), fnRemainingOps(ec))` — not just `maxNodes`, so a
small `OpLimit` (far below the 10M default node-set cap) stops splitting after
~`OpLimit` lines instead of first allocating a `[]string` proportional to the
resource's full line count. `LoadTextLinesBounded` produces at most `limit+1`
lines; the produced count is then charged via `fnCountOps`, surfacing
`ErrOpLimit` (op budget binding) or `ErrNodeSetLimit` (node-set cap binding).
`fnRemainingOps` (in `functions_hof.go`) returns the remaining op budget and
whether one is in force (false when `OpLimit` is unset or `ec == nil`).

## FunctionItem Mechanics

Inline functions, named refs, partial applications → all produce `FunctionItem`.

`FunctionItem.invoke` is a closure: `func(ctx context.Context, args []Sequence) (Sequence, error)`.

Evaluator calls uniformly regardless of origin.
