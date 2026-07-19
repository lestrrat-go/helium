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
(`item-separator`/`encoding`, via `readSerializeStringOption`), its `method`
union-of-string-or-QName (`resolveSerializeMethodMap` — which inspects the ATOM so
a namespaced `xs:QName` keeps its namespace as an extension EQName instead of being
stringified to its local part), and its `standalone` union value
(`resolveSerializeStandaloneMap`) atomize via the ctx-aware
`atomizeTypedValue`/typed-value stream then enforce the singleton, keeping each
atom's type so the `atomicToString`/type checks are unchanged. Because atomization
is content-kind-aware, an
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

**Resource bounds (XPATH3-102/105).** `string-to-codepoints` charges `fnCountOp` per produced codepoint so a huge input cannot build an item sequence below the node-set cap but above `OpLimit`. `analyze-string` STREAMS its regex matches via `compiledXPathRegex.eachStringSubmatchIndex` (the internal form of the public `Regex.EachSubmatchIndex`) rather than calling `FindAllStringSubmatchIndex` up front: when an op budget is in force it passes `fnRemainingOps(ec)+1` as the match limit and charges `fnCountOp` BEFORE building each match's result nodes, so an input with millions of matches rejects with `ErrOpLimit` (or honors cancellation) without ever materializing the O(matches) index slice. A non-streamable leading-context pattern over an oversized input surfaces `ErrRegexMatchLimit` as-is (errors.Is-compatible); other engine errors map to `FORX0002`. Nested capturing groups produce NESTED `fn:group` elements (F&O 3.1 §5.6.5): `analyzeStringGroupParents` derives each group's parent from the pattern's STATIC parenthesis structure (skipping escapes, XSD character-class subexpressions, and `(?…` non-capturing/modifier groups), not from match positions — `(b)(x?)` (siblings) and `(b(x?))` (nested) yield identical submatch spans, so position-only reconstruction cannot tell them apart. It MUST tokenize the SAME pattern the engine compiles, so it applies the identical preprocessing first: the `q` flag makes the whole pattern a literal (no groups), and the `x` flag runs `stripFreeSpacing` before scanning — otherwise a raw-pattern parse diverges (e.g. `( a \ ) (b) )` with `x` compiles to `(a\)(b))`, group 2 nested in group 1, but the raw form looks like two siblings and would duplicate text). The XPath 3.1 `x` flag (§5.6.1.1) removes unescaped whitespace outside character classes — EXACTLY the four XML whitespace chars `#x9/#xA/#xD/#x20` (`isXPathRegexWhitespace`, deliberately narrower than `unicode.IsSpace`: U+00A0 NBSP and other Unicode spaces stay literal) — and does NOT enable `#` comments (a Perl/Java extension absent from XPath 3.1 — `#` stays literal). That single shared whitespace definition governs BOTH the core regex-compilation stripping (`stripFreeSpacing`, used by fn:matches/replace/tokenize/analyze-string) and analyze-string group derivation. `buildAnalyzeStringMatch` builds the group tree by pattern nesting (`buildAnalyzeStringTreeByPattern`), then `renderAnalyzeStringGroup` distributes the matched substring text so each `fn:group` holds the text of its span not covered by a nested child, in document order. The fundamental invariant is that the `fn:match` string value equals the matched input substring; `buildAnalyzeStringMatch` ENFORCES it via `analyzeStringGroupText`: if the pattern-derived tree's string value ever disagrees with the input (a tokenization edge that would duplicate/drop text), it falls back to `buildAnalyzeStringTreeByPosition` (span-containment nesting, which always tiles the matched substring exactly) and, as a final floor, to a group-less match holding just the text — output whose string value differs from the input is never emitted. The result tree is still `xs:untyped`/untyped attributes (no PSVI type annotation from the built-in analyze-string-result schema).

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

The streaming JSON parser (`parseJSONValue`/`parseJSONToken`) is shared by
`fn:parse-json`/`fn:json-doc` and `fn:json-to-xml`, but the two consumers type
JSON numbers differently, gated by `jsonOptions.retainNumberLexical`:

- `fn:parse-json` / `fn:json-doc` (flag false): per F&O 3.1 §17.5 JSON has a
  single number type, so every JSON number becomes an `xs:double` (`jsonToXDM`),
  including integral values like `0`/`-0` — never `xs:integer`.
- `fn:json-to-xml` (flag true, set in `parseJSONToXMLOptions`): the `<number>`
  element retains the number's EXACT lexical form (`23E0`, `0.23e+02`, `-0`,
  `1000000000000` — never scientific notation), per F&O 3.1 / W3C bug 28179. The
  parser yields a private `jsonLexicalNumber{lexical}` item that
  `buildJSONToXMLTree` writes verbatim, bypassing the `xs:double` canonicalizer.

### `functions_json_xml.go`
`json-to-xml`, `xml-to-json`

`json-to-xml` builds every result element (`map`/`array`/`string`/`number`/
`boolean`/`null`) in the fn namespace (`http://www.w3.org/2005/xpath-functions`)
and DECLARES that default namespace on EVERY element (not only the root), so a
descendant selected by an XPath step (e.g. `json-to-xml(...)//j:string`) still
carries its `xmlns` declaration when `fn:serialize` renders it in isolation
(`buildJSONToXMLTree`). The `duplicates` option defaults to `retain` (F&O 3.1
§17.6.1: reject if `validate` is true, else retain) — unlike `fn:parse-json`,
whose default is `use-first` — so duplicate JSON object keys are preserved as
repeated entries (`MapItem.entries` keeps duplicates; `forEach0` emits all).
The `escaped` / `escaped-key` attributes are emitted ONLY under `escape=true()`
AND ONLY when the string value / key attribute actually contains a backslash
(F&O 3.1 §17.6.1: "Any string element whose string value contains a backslash
character must have the attribute value escaped='true'"); a value/key with no
backslash carries no such attribute, and `escape=false()` never emits either.

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
NBSP-padded forms are rejected — `serializeBooleanValue`), `method` validates as
a built-in method OR a prefixed-QName/non-null-EQName EXTENSION name — a bare
non-built-in NCName (e.g. `bogus`) is invalid (`serializeMethodValid` /
`isExtensionMethodName`) — and the recognized
but unapplied parameters (`byte-order-mark`, `escape-uri-attributes`,
`include-content-type`, `html-version`, `json-node-output-method`, `media-type`,
`normalization-form`, `doctype-public`, `doctype-system`) are validated and
ignored rather than rejected as unsupported.
The xml path builds a `helium.Writer` via `newSerializeXMLWriter` wiring
`OutputVersion`/`Standalone`/`OmitStandalone`/`AllowPrefixUndeclarations`/
`CDATASectionElements`/`SuppressIndentElements`/`CharacterMap`/`Normalization` — the shared
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
`serializeHTMLNode` / `writeHTMLNodeTree`) emits its DOCTYPE immediately before the
first element and injects a `<meta http-equiv="Content-Type">` into `<head>`.
A DOCTYPE is emitted in two cases (Serialization 3.1 §7.4.6): (a) an explicit
`doctype-system`/`doctype-public` — emitted regardless of the document element's
name AND for a bare element input, since sequence normalization wraps the element
in a document; or (b) the DEFAULT `<!DOCTYPE html>` (HTML5) / HTML 4.01 declaration,
which is emitted ONLY when the document element's local name is `html`
(case-insensitive). Because sequence normalization (§2) wraps the WHOLE sequence in
a single document node, the DOCTYPE is emitted AT MOST ONCE for the entire
sequence — before the first element — not once per item; the `doctypeEmitted` flag
threaded through the item loop (and `serializeHTMLNode`'s `allowDoctype` parameter)
enforces this. The `<meta>` injection likewise applies only to an
html-rooted document (a fragment / non-html root has no `<head>`). The html
parameters are APPLIED:
`include-content-type` (default yes) gates the meta injection; `escape-uri-attributes`
(default yes) selects `helium/html` `Writer.EscapeURIAttributes`; and the DOCTYPE
is chosen by `htmlDoctype` — an explicit `doctype-public`/`doctype-system` yields
a PUBLIC/SYSTEM declaration whose DOCTYPE name is the fixed token `html` (the
html-method / HTML5 doctype rule — never the source element's arbitrary case, so
`<HtMl>` still emits `<!DOCTYPE html …>`), otherwise HTML5 (`html-version` ≥ 5, the default)
yields `<!DOCTYPE html>` and HTML 4 yields the HTML 4.01 declaration; the
Content-Type meta uses the `media-type` param (default `text/html`) and
`insertHTMLContentTypeMeta` DISCARDS every existing `<meta http-equiv=
"Content-Type">` (matched case-insensitively and whitespace-trimmed via
`isHTMLContentTypeMeta`/`removeHTMLContentTypeMetas`) before inserting the
computed one as the head's first child, so no stale declaration survives. The
doctype/meta work on a copy of the document (never mutating the input), then
serialize via the `helium/html` writer. Character maps ARE applied on the html
path (text + attribute content) via the `helium/html` `Writer.CharacterMap` knob;
per Serialization 3.1 §7 (character-expansion phase) a URI attribute value that is
URI-escaped (`escape-uri-attributes=yes`, the default) SKIPS character mapping, so
character maps apply to non-URI attributes normally and to URI attributes only
when escaping is disabled. The `helium` XML writer's XHTML serialization path
(`writer_xhtml.go`) likewise applies `Writer.CharacterMap` to XHTML attribute
values (including the synthesized id-from-name / xml:lang attributes); it performs
no URI percent-encoding, so the §7 URI exclusion is not reachable there.
The `xml` method emits `doctype-public`/`doctype-system` ONLY when
`doctype-system` is present — per Serialization 3.1 §5.1 `doctype-public` MUST be
ignored unless `doctype-system` is also specified (`serializeNodeItem` injects an
internal-subset DTD named after the document element on a COPY via
`documentWithDoctype`, so the XML writer emits `<!DOCTYPE name PUBLIC/SYSTEM ...>`
between the declaration and the root). The map form defaults `omit-xml-declaration`
to true (an empty map equals omitting the argument; W3C serialize-xml-127a); the
element form keeps the Serialization-spec default (declaration emitted).
The DTD / internal subset of a SOURCE document node is NOT part of the XDM data
model (a document node's children are element/PI/comment/text nodes only), so
`fn:serialize` never reproduces it: `newSerializeXMLWriter` sets
`Writer.IncludeDTD(opts.doctypeSystem != "" && opts.methodEmitsDoctype())`,
dropping any source DTD unless a `doctype-system` param is present on a
DOCTYPE-emitting method (in which case the writer emits the freshly injected
empty-internal-subset DTD from the doctype path); the adaptive and json methods,
whose node items serialize through the same writer, ignore `doctype-system` and
never emit a DTD. W3C fn-doc-25/26/29, parse-xml-006/008/009/013.

**In-scope namespaces on an isolated element.** `fn:serialize` serializes an
element as if it were the root of a tree, so an element selected from a larger
document must carry the namespace declarations for EVERY namespace in scope on
it — including those inherited from ancestors OUTSIDE the serialized subtree (in
XDM an element node's namespace nodes comprise ALL its in-scope namespaces, and
the XML output method emits a declaration for each, matching Saxon). helium's DOM
stores only the declarations literal on each element, so `serializeNodeItem`
routes an `*helium.Element` through `elementWithInScopeNamespaces`: an element
with no inherited in-scope namespace is returned unchanged (byte-identical — a
document root, root element, or standalone element), otherwise a namespace-
complete deep COPY is serialized. `inheritedInScopeNamespaces` walks the
ancestor chain collecting each PREFIXED declaration not shadowed closer to the
element and not already on it; the NEAREST ancestor declaration for a prefix
decides it, so an XML 1.1 undeclaration (`xmlns:p=""`, empty URI) marks the prefix
decided and removes it from scope BEFORE the empty-URI skip — a further-out
ancestor binding for that prefix cannot resurrect it. Only a nearest declaration
with a non-empty URI is added. The over-declaring deep copy re-declares the
element's own/active namespaces and the inherited prefixes are added on top. Only
prefixed declarations are force-added
(a prefixed decl never rebinds the element's own name); the default (empty-prefix)
namespace and undeclarations are left to the copy's active-namespace binding.
`pruneRedundantNamespaceDecls` then walks the copy and drops, from every element
BELOW the root, any declaration whose prefix is already bound to the same URI by
an ancestor within the serialized subtree — the redundant `xmlns` the
over-declaring copy leaves on each namespaced descendant. The root keeps every
declaration (it is the top of the serialized tree and must be namespace-complete);
the XML output method emits a declaration for a descendant only when its namespace
node differs from its parent's, so a bare descendant (`<xsl:text>`) serializes
without repeating an in-scope ancestor binding (W3C strip-space-029). This runs
BEFORE the doctype handling so `applyDoctype` copies the namespace-complete
element. W3C fn-union-node-args-015/016/017, fn-intersect-node-args-015/016,
XQueryComment002.

**SEPM0009** (`fnSerialize`, gated to `methodEmitsXMLDeclaration` — the xml/xhtml/
default methods) fires when `omit-xml-declaration=yes` AND EITHER (a) `standalone`
is yes/no, OR (b) the effective `version` is other than `1.0` AND `doctype-system`
is present. Both sub-conditions are gated on `omit-xml-declaration=yes`; the
`doctype-system` sub-condition is ADDITIONALLY gated on `version != "1.0"` (a
DOCTYPE without an XML declaration is well-formed in XML 1.0), so
`serialize(., map{"omit-xml-declaration":true(),"doctype-system":"x"})` at the
default version 1.0 emits the DOCTYPE and is NOT an error, while the same at
`version":"1.1"` is SEPM0009. It uses the EFFECTIVE omit (incl. the map-form
default true) and resolved standalone. The html method (no XML declaration) is
unaffected.

An extension `method` (a prefixed QName) passes value validation but is an
UNSUPPORTED value: helium implements no extension output methods, so `fnSerialize`
raises `SEPM0016` at dispatch rather than silently falling through to xml. The
`version` param, for the XML-declaration-emitting methods, is validated as a
supported XML output version (`1.0`/`1.1`); any other value is `SESU0013`
(`isSupportedXMLOutputVersion`), not a bogus version pseudo-attribute.

**Item separator (sequence normalization §2, step 3).** The join between adjacent
serialized items is `joinSerializedItems`, shared by the xml/xhtml/html/text
sequence functions and the unspecified-default path (`serializeAdaptiveSequence`
with `method==""`). When the `item-separator` parameter is EXPLICITLY specified
(`opts.itemSeparatorSet`, set in both the map and element parse paths) it is
inserted between EVERY adjacent pair of items regardless of kind. When ABSENT, a
single space is inserted ONLY between two adjacent atomic-value-derived strings —
never between two nodes, nor between a node and an atomic value — so
`serialize((1,2,3))` is `1 2 3` while `serialize((<a/>,<b/>))` is `<a/><b/>` with
no spurious separator. Each sequence function tracks a parallel `atomic []bool`
(true iff the item is an `AtomicValue`) to drive the rule. The EXPLICIT `adaptive`
method (and nested map/array serialization) is exempt and joins every item with
the `item-separator` directly.

**Output methods.** `xml` (default) and `adaptive`/`json` are full; `html` is
applied as above; `text` (`serializeTextSequence`) concatenates the string values
of the items (joined per the item-separator rule above) and no markup (character
maps applied);
`xhtml` is serialized as `xml` — a defensible approximation, as helium implements
no XHTML-specific serialization rules. Sequence normalization (Serialization 3.1
§2) governs maps/arrays/functions under EVERY markup method —
`xml`/`xhtml`/`html`/`text` and the unspecified default — in two steps.
(1) **Array flattening** (`flattenSerializeArrays`, reusing `ArrayItem.Flatten`):
an array is NOT a rejected kind; it is replaced by its member items RECURSIVELY
(nested arrays flatten too), which then serialize as if supplied directly — the
flattened atomic/node members then obey the item-separator rule above (adjacent
atomics space-separated when the separator is absent, nodes never separated). (2) The
shared `serializeItemKindError` guard then rejects with `SENR0001` a bare attribute
node, a namespace node, OR a function item — INCLUDING a map (a map is a function
item; an array is not) — delegating node kinds to `serializeNodeKindError`. So a
map, or a map member surfaced by array flattening, does NOT fall through to
adaptive serialization under the html method, nor is it accepted by the unspecified
default (which dispatches through `serializeAdaptiveItem`, guarded when the method
is not adaptive; the default flattens arrays before that dispatch). The `adaptive`
and `json` methods are exempt from BOTH steps: they serialize maps/arrays natively
(`json` → JSON, `adaptive` → the `map{…}`/`[…]` form) and keep `FOER0000` for a
bare function item.

**Normalization + character maps (all methods incl. JSON).** `normalization-form`
is APPLIED for the methods that support it — xml/xhtml/html/text, the unspecified
default, AND `json` (Serialization 3.1 §9.1.9) — `NFC`/`NFD`/`NFKC`/`NFKD` via
`golang.org/x/text/unicode/norm`; `none`/`""` is a no-op; the W3C-specific
`fully-normalized` form is not provided by that package and is the `SESU0011`
unsupported-normalization error (rejected up front in `fnSerialize` via
`isSupportedSerializeNormForm`, so it fires for every applicable method even with
no text to normalize). Only the `adaptive` method ignores it. Normalization is
SCOPED to text-node and attribute-value character content (Serialization 3.1 §4
character-expansion phase) — element/attribute NAMES, comment/PI markup, the
DOCTYPE, and the XML declaration are NEVER normalized. The markup methods
(xml/xhtml/html and the unspecified default) apply it INSIDE their writer: the
`helium` XML writer via `Writer.Normalization(form)` (normalizing each text node
and attribute value in `escapeText`/`escapeAttrValue`, and CDATA content, before
escaping) and the `helium/html` writer via `Writer.Normalization(form)`
(`dumpText`/`dumpAttributes`). The `text` and `json` methods emit pure character
data (text) or ASCII-delimited character data (json), so
`applySerializeNormalization` normalizes their WHOLE output — equivalent to
node-scoped normalization for those methods. `use-character-maps` is likewise
applicable to the `json` output method (§9.1.11, matching Saxon): a mapped
character is replaced by its verbatim replacement in JSON string content (values
and object keys) INSTEAD of being JSON-escaped — e.g. a map `"/"→"/"` prevents
escaping `/` as `\/` (`encodeJSONStringForSerialization` consults `opts.charMap`).
A character-map REPLACEMENT string is NOT subjected to Unicode Normalization or
re-escaping (Serialization 3.1 §11): when a normalization pass and a character map
are BOTH in force, `withCharMapSentinels` substitutes each mapped key with a unique
sentinel rune (Supplementary Private Use Area-A, unaffected by normalization)
during serialization (XML/HTML/text/json alike), then `expandCharMapSentinels`
restores the verbatim replacement AFTER normalization — so replacements pass
through un-normalized while the surrounding content is normalized (the markup
writer decides character-map matches on the PRE-normalization content —
`normalizeContent` splits the content into segments at each mapped key,
normalizes each non-mapped run on its own, and emits each key's replacement —
here the xpath3 sentinel — verbatim as its own segment, never touched by the
normalize pass or the escaper, so a mapped rune CREATED by normalization is
ordinary content, not newly matched, regardless of what runes the content
contains: Serialization 3.1 §4 applies character mapping — rule c — before
normalization — rule d — and never re-applies it).
`json-node-output-method` is validated against its OWN
narrower domain (`xml`/`html`/`xhtml`/`text` or an extension QName — NOT
`json`/`adaptive`, via `serializeJSONNodeOutputMethodValid`); only its default
(`xml`) is honored, so a non-default value (`html`/`xhtml`/`text`/extension) that
would change a node's serialization is an explicit `SEPM0016` unsupported-feature
error when a node is actually serialized under json (helium has no nested-node JSON
serialization for the other methods) — never silent xml. When no node is serialized
under json it is a harmless no-op.

**Value validation (Serialization 3.1 schema types), map AND element form.** Every
recognized parameter is applied, a spec-justified no-op, or a documented gap, and
its value is schema-type validated consistently across both forms: booleans/
`yes-no-omit` accept `{yes,no,true,false,1,0}` (+ `omit`), uppercase rejected;
`method` as a built-in method or a prefixed-QName/EQName extension (bare
non-built-in NCName invalid; the map form additionally accepts an `xs:QName` value,
where a namespaced QName is an extension → `SEPM0016` — but per F&O 3.1 an
`xs:QName` method value MUST have a non-absent namespace, so ANY no-namespace
QName, even `QName("","xml")`, is `XPTY0004`: built-in methods are supplied as
strings, not no-namespace QNames); `json-node-output-method` against its narrower
`{xml,html,xhtml,text}`-or-extension domain (map form also `xs:QName`-aware, via
`resolveSerializeJSONNodeMethodMap` — same F&O rule: a namespaced QName is an
extension, and a no-namespace QName value is `XPTY0004`, keeping its namespace
instead of stringifying to the local part); `html-version` as `xs:decimal`
(`isValidXSDecimal`); `normalization-form` against
`{NFC,NFD,NFKC,NFKD,fully-normalized,none,""}`; `version` as a supported XML
output version (else `SESU0013`). The map-form `cdata-section-elements` /
`suppress-indentation` values atomize FIRST (`resolveSerializeQNameNames` via
`atomizeTypedValue`), so an ARRAY value flattens to its member xs:QNames (W3C
serialize-xml-106a).
`byte-order-mark` validated as boolean and ignored (UTF-8 emits no BOM);
`media-type` applied to the html meta; `doctype-public`/`doctype-system` applied
to the html/xml doctype — but a `doctype-system` value containing BOTH a `"` and a
`'` cannot be an XML SystemLiteral, so it is the `SEPM0016` invalid-value error
(Serialization 3.1 §3), not malformed output. The `encoding` param drives the XML
declaration's encoding pseudo-attribute (§5.1.6: the declaration includes an
encoding declaration) via `Writer.OutputEncoding` — the map form defaults it to
`UTF-8` so the declaration always carries `encoding="…"`, while the element form
falls back to the document's own encoding when the param is absent. Content
whitespace checks use `isXSDWhitespaceOnly`
(`#x20/#x9/#xA/#xD` only) so NBSP-only content is significant, not ignorable.
A map-form option entry present with the EMPTY SEQUENCE selects that parameter's
DEFAULT (F&O 3.1 fn:serialize present-empty = use default, not an error), across
every map option type — the bool/string/number/method/json-node-method/character-map
readers report a present-empty value as not-found so the caller keeps its default.

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
