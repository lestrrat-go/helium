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
The `xs:string?` helpers (`coerceArgToString`/`coerceArgToStringRequired` →
`coerceAtomizedString`, plus `seqToStringErr` and the `||` `concatToString`
path) atomize FIRST — arrays flatten to their members, list-typed nodes expand
to their tokens — and count cardinality AFTER, so a length `> 1` becomes
`XPTY0004` while an empty-array member flattens away (`f(([], "x"))` coerces to
the single `"x"`, not `XPTY0004`). Helpers do not truncate to first item, and a
raw pre-atomization `seqLen > 1` gate must NOT wrap them (it would pre-empt the
flattening). This is why the signature-less doc/collection URI family
(`fn:doc`/`fn:doc-available` via direct `coerceAtomizedString`,
`fn:collection`/`fn:uri-collection` via `collectionURIArg`) and the URI builtins
(`fn:encode-for-uri`/`iri-to-uri`/`escape-html-uri`) delegate cardinality
entirely to the atomize-then-count string coercion. Emptiness is detected AFTER
atomization too (via the `empty` result of `coerceAtomizedString`), so an
empty-array or nilled/empty-content node argument yields the function's spec
empty result (`()`), not the empty-string path.
Signature-sensitive string/regex/URI builtin call sites + `||` string coercion
also propagate atomization/type errors (`FOTY0012`/`FOTY0013`).

The same rule governs **options-map string values** (F&O 3.1 §2.5 option/function
conversion): an `xs:string` option value atomizes FIRST (an empty-array member
flattens away) and cardinality applies AFTER, so `map{"opt": ([], "v")}` coerces
to the single `"v"`, not `XPTY0004`. No raw pre-atomization `seqLen != 1` gate
wraps these. `fn:json-to-xml`/`fn:parse-json`/`map:merge` `duplicates` route
through `coerceArgToStringRequired`; `fn:serialize`'s string map parameters
(`method`/`item-separator`/`encoding`, via `readSerializeStringOption`) and its
`standalone` union value (`resolveSerializeStandaloneMap`) atomize via the
ctx-aware `atomizeTypedValue`/typed-value stream then enforce the singleton,
keeping each atom's type (e.g. an `xs:QName` `method`) so the `atomicToString`/
type checks are unchanged. Because atomization is content-kind-aware, an
element-only-typed node used as an option value has no typed value and raises
`FOTY0012`; every extractor lets that dynamic error surface (`isNoTypedValueError`)
instead of masking it as its own bad-option error (`XPTY0004`/`FOJS0005`).

`evalFunctionCall` (the static call path) enforces the declared parameter
signature for every resolved function before invoking it: it looks up
`paramTypes` via the `function_signatures.go` registry (then
`TypedFunction`/`TypedFunctionByArity`) the same way `evalNamedFunctionRef`
does, and runs `coerceFuncallArg(arg, paramTypes[i], ec)` per argument.
Functions with no registered signature (`paramTypes == nil`) are not
type-checked. `coerceFuncallArg` delegates to `coerceToSequenceTypeE`, which
atomizes through `typedValueItemCheckFor(ec)` (so list-typed nodes/arrays expand
correctly, arrays flatten before cardinality is applied, and atomizing an
element-only-typed node raises `FOTY0012`) and falls back to
`ec.schemaDeclarations.IsSubtypeOf` for user-defined schema types. A plain type
mismatch surfaces as `XPTY0004`; a genuine coercion/atomization error
(`FOTY0012`, `FOTY0013`, `FORG0001`, …) propagates unchanged rather than being
collapsed into `XPTY0004`. The same `coerceToSequenceTypeE` propagation applies
to the inline-function parameter/return paths and `coerceFunctionItem`
(`function-lookup`, dynamic function items).

## Functions by File

### `functions_node.go`
`node-name`, `nilled`, `string`, `data`, `base-uri`, `document-uri`, `root`, `path`, `has-children`, `innermost`, `outermost`, `id`, `idref`, `lang`, `local-name`, `name`, `namespace-uri`, `number`, `generate-id`

`fn:nilled` returns the PSVI [nil] property of an element node (`()` for a non-element): true iff the node is in the evaluator's `NilledElements` set (`Evaluator.NilledElements`, populated from `xsd.Validator.NilledElements`), else false. The same set drives two other nilled-aware behaviors: `fn:data` / atomization gives a nilled element the empty typed value `()` (via `typedValueItemCheck`, checked before content-kind), and an `element(name, type)` instance-of test excludes a nilled element while `element(name, type?)` matches it (`eval_path.go` `ElementTest`). Non-schema-aware evaluation leaves the set nil → every element is not-nilled.

`fn:id`/`fn:element-with-id` (`idLookup`) resolve is-id nodes from three sources: DTD-declared IDs (`GetElementByID`), type annotations whose name is xs:ID or a subtype (`annotationMatchesIDType`), and the PSVI is-id set supplied via `Evaluator.IDNodes` (`ec.idNodes`). The is-id set is required for cases the type name alone cannot express — a SINGLETON list of xs:ID and a union that selects an xs:ID-derived member (a multi-item list / non-ID union member is not is-id); the xsd validator computes it (`Validator.IDNodes`). `idElementsFromTypeAnnotations` unions annotation-derived and set-derived candidates, then `idNodeResult` maps each is-id node to its result (`fn:id` → the ID element / bearing element; `fn:element-with-id` → that element's parent).

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
`validate:true()` with `duplicates:'retain'` is a dynamic error `FOJS0005`
(`parseJSONToXMLOptions`): validation against the json result schema requires
unique keys, so the combination cannot succeed.

### `functions_serialize.go`
`serialize`

Serialization parameters (map form and `output:serialization-parameters`
element form) are both parsed and **applied**. String/boolean params
(`method`/`item-separator`/`encoding`/`indent`/`omit-xml-declaration`/
`allow-duplicate-names`) and these additional params drive the output:
`standalone` (resolved to yes/no/omit → the XML-declaration pseudo-attribute;
the Serialization default is `omit`, so a source declaration's standalone is not
retained unless the parameter requests yes/no), `undeclare-prefixes`
(XML 1.1 `xmlns:pfx=""` undeclarations), `cdata-section-elements` and
`suppress-indentation` (resolved to an EXACT expanded `{uri}local` name set —
Clark notation with `{}local` for the no-namespace case; matching is by exact
expanded name so `QName("","b")` never matches a namespaced `<p:b>`; the
element form is an `xs:list` of QName-OR-`Q{uri}local`-EQName, split ONLY on XSD
list whitespace (`#x20/#x9/#xA/#xD` via `xsdListFields`, so an NBSP stays in the
token and fails validation), validating NCName parts, rejecting an EQName with a
brace in the URI part, resolving an unprefixed lexical QName through the in-scope
DEFAULT namespace, binding the reserved `xml` prefix implicitly, and treating
`xmlns:p=""` as an UNDECLARATION that leaves `p` unbound rather than bound to the
empty URI), and `use-character-maps` (resolved to a `map[rune]string`; an element
form `output:character-map` requires only `@character` — an absent or empty
`@map-string` maps the character to the empty replacement, i.e. deletion). The
element-form value parsing follows the Serialization 3.1 schema types: the
boolean/`yes-no-omit` families accept the full lowercase lexical space
(`yes`/`no`/`true`/`false`/`1`/`0`, plus `omit` for standalone; uppercase and
NBSP-padded forms are rejected — `serializeBooleanValue`), `method` validates
against the built-in methods or a QName/EQName extension name, and the recognized
but unapplied parameters (`byte-order-mark`, `escape-uri-attributes`,
`include-content-type`, `html-version`, `json-node-output-method`, `media-type`,
`normalization-form`, `doctype-public`, `doctype-system`) are validated and
ignored rather than rejected as unsupported.
The xml path builds a `helium.Writer` via `newSerializeXMLWriter` wiring
`OutputVersion`/`Standalone`/`OmitStandalone`/`AllowPrefixUndeclarations`/
`CDATASectionElements`/`SuppressIndentElements`/`CharacterMap` — the shared
`helium.Writer` (root `writer.go`/`writer_escape.go`) implements those knobs
(character maps substitute a mapped rune with its raw replacement in text and
attribute content; cdata-section-elements emit direct text children as CDATA;
suppress-indentation disables indentation for the named subtree; `Standalone`
forces yes/no, `OmitStandalone` forces omission, matching by exact expanded name).
The effective output `version` (the `version` param, default `1.0`) drives BOTH
the XML declaration text AND the XML 1.1 escaping/undeclaration rules via
`Writer.OutputVersion`, so a source document's own version is overridden and the
declaration and escaping stay consistent (`Writer.OutputVersion("")` — every
non-`fn:serialize` caller — keeps the document's version, byte-identical).
`undeclare-prefixes` is honored only when that effective version is 1.1;
requesting it at an effective 1.0 (the default when `version` is unspecified) is
the `SEPM0010` static error. `method="html"` (`serializeHTMLSequence` /
`serializeHTMLNode`) emits its DOCTYPE and injects a `<meta http-equiv=
"Content-Type">` into `<head>` — but ONLY when the document element's local name
is `html` (case-insensitive); any other root (or a fragment node) is serialized
under the html method with no DOCTYPE/meta. The html parameters are APPLIED:
`include-content-type` (default yes) gates the meta injection; `escape-uri-attributes`
(default yes) selects `helium/html` `Writer.EscapeURIAttributes`; and the DOCTYPE
is chosen by `htmlDoctype` — an explicit `doctype-public`/`doctype-system` yields
a PUBLIC/SYSTEM declaration, otherwise HTML5 (`html-version` ≥ 5, the default)
yields `<!DOCTYPE html>` and HTML 4 yields the HTML 4.01 declaration; the
Content-Type meta uses the `media-type` param (default `text/html`) and UPDATES an
existing `<meta http-equiv="Content-Type">` in place (rather than leaving a stale
one) so a `media-type` change is honored. The doctype/meta work on a copy of the
document (never mutating the input), then serialize via the `helium/html` writer.
The `xml` method applies `doctype-public`/`doctype-system` too: `serializeNodeItem`
injects an internal-subset DTD (named after the document element) on a COPY via
`documentWithDoctype`, so the XML writer emits `<!DOCTYPE name PUBLIC/SYSTEM ...>`
between the declaration and the root. The map form defaults `omit-xml-declaration`
to true (an empty map equals omitting the argument; W3C serialize-xml-127a); the
element form keeps the Serialization-spec default (declaration emitted). Character
maps are not applied on the html path.

**SEPM0009** (`fnSerialize`, gated to `methodEmitsXMLDeclaration` — the xml/xhtml/
default methods) fires when `omit-xml-declaration=yes` AND (`standalone` is yes/no
OR `doctype-system` is present): neither a standalone nor a document-type
declaration is possible without an XML declaration (Serialization 3.1 §5.1). It
uses the EFFECTIVE omit (incl. the map-form default true) and resolved standalone,
so `serialize(., map{"doctype-system":"x"})` is SEPM0009 while the html method
(no XML declaration) is unaffected.

**Output methods.** `xml` (default) and `adaptive`/`json` are full; `html` is
applied as above; `text` (`serializeTextSequence`) concatenates the string values
of the items with the `item-separator` and no markup (character maps applied,
no SENR0001 node-kind restriction); `xhtml` is serialized as `xml` — a defensible
approximation, as helium implements no XHTML-specific serialization rules.

**Spec-honest gaps (no silent wrong output).** `normalization-form` other than
`none`/`""` is the `SESU0011` unsupported-normalization serialization error
(helium performs no Unicode normalization), not silently-unnormalized output.
`json-node-output-method` is validated but only its default (`xml`) is honored — a
node embedded in JSON is always serialized with the xml method (helium has no
nested-node JSON serialization for html/xhtml/text); this is a documented
no-op limitation flagged at the call site.

**Value validation (Serialization 3.1 schema types), map AND element form.** Every
recognized parameter is applied, a spec-justified no-op, or a documented gap, and
its value is schema-type validated consistently across both forms: booleans/
`yes-no-omit` accept `{yes,no,true,false,1,0}` (+ `omit`), uppercase rejected;
`method`/`json-node-output-method` validate as a built-in method or a QName/EQName;
`html-version` as `xs:decimal` (`isValidXSDecimal`); `normalization-form` against
`{NFC,NFD,NFKC,NFKD,fully-normalized,none,""}`; `version` as the output version.
`byte-order-mark` validated as boolean and ignored (UTF-8 emits no BOM);
`media-type` applied to the html meta; `doctype-public`/`doctype-system` applied
to the html doctype. Content whitespace checks use `isXSDWhitespaceOnly`
(`#x20/#x9/#xA/#xD` only) so NBSP-only content is significant, not ignorable.

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
