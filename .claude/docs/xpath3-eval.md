# XPath 3.1 — Evaluator

## evalContext

```go
type evalContext struct {
    node       helium.Node           // context item (nil if absent)
    position   int
    size       int
    vars       *variableScope        // scoped variable bindings
    namespaces map[string]string
    functions  map[string]Function
    fnsNS      map[QualifiedName]Function
    depth      int
    opCount    *int
    opLimit    int
    docOrder   *ixpath.DocOrderCache
    maxNodes   int
    defaultLanguage string
}

func newEvalContext(node helium.Node) *evalContext
func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext
func (ec *evalContext) withVar(name string, val Sequence) *evalContext  // new scope
```

`context.Context` is passed as a function parameter (`ctx`) through the eval chain, not stored in the struct.

## Dispatch

- `Compile()` lowers AST to `vmProgram` and collects prefix-validation requirements during the same pass
- The string-based `Compile()` path can reuse parsed slices during lowering because the compiled expression keeps `source` + `vmProgram` and reparses AST only for AST/streamability access
- `Expression.Evaluate()` executes `vmProgram` when present
- `evalWith()` enforces recursion depth for raw eval + VM eval
- `dispatchExpr()` contains shared Expr-type dispatch
- Raw `eval(ec, expr)` remains fallback for unlowered AST execution

### VM Shape

```go
type vmProgram struct {
    root int
    instructions []vmInstruction
}
type vmInstruction struct {
    op      vmOpcode
    payload any
}
type compiledExprRef struct { index int }
```

Lowering is structural for non-trivial nodes: recursive children usually become `compiledExprRef` indexes, trivial leaves such as literals or variable refs can stay inline in the parent payload, and hot path forms now use VM-specific payloads instead of AST nodes. `LocationPath` / `PathExpr` lower to `vmLocationPathExpr` / `vmPathExpr` with `vmLocationStep` slices, and the direct-compile fast path can emit `vmLocationPathExpr` directly before lowering child predicate expressions. Common predicates on VM location steps also lower to inline VM predicate payloads for `[N]`, `[position() = N]`, `[@attr]`, and `[@attr = "literal"]`; other predicates stay as lowered `Expr`s. VM execution switches on `vmOpcode` for compiled refs, then reuses existing `eval_*` helpers for language semantics.

## Evaluation Rules by Expr Type

### SequenceExpr (comma)
Evaluate each item, concatenate sequences through `appendBoundedSeq` so the aggregate honors `maxNodes` / OpLimit / cancellation (each operand is individually capped, but the concatenation must be bounded too).

### LocationPath
1. Start: root node (absolute) or context node (relative)
2. Per step: traverse axis → filter by NodeTest → apply predicates → dedup doc order
3. Hot axes (`child`, `attribute`, `self`, `parent`) fuse traversal and node-test filtering directly in `xpath3`, avoiding the generic `TraverseAxis` + extra filtered-slice path
4. Return merged node-set

**Cancellation:** the fused hot child/attribute loops check `ctx.Err()` once per enumerated node (the attribute path also inside its `ForEachAttribute` callback) so a cancelled context aborts mid-enumeration instead of scanning the whole child/attribute set before the next `countOps` boundary. Generic (non-hot) axes delegate to `ixpath.TraverseAxis(ctx, ...)`, which performs its own in-loop `ctx.Err()` checks; on the namespace axis those checks run inside the `NamespacePrefixesInScope` / `CollectNamespaceNodes` helper loops (outer and inner) so `namespace::*` cancels promptly too.

### Predicates
- Numeric atomic → compare to position (1-based)
- Otherwise → compute EBV

### BinaryExpr
Arithmetic (`+ - * div idiv mod`): atomize both sides → numeric promotion → compute.
Logic (`and or`): EBV of both sides.
Comparison: delegate to `GeneralCompare` or `ValueCompare`.

### SimpleMapExpr (`!`)
Evaluate left → for each item, set as context → evaluate right → concatenate. NO doc-order dedup.

### LookupExpr (`?`)
- MapItem → `Get(key)`
- ArrayItem → `Get(index)` (key must be xs:integer)
- Sequence of maps/arrays → apply to each, concatenate
- `?*` (All=true) → all values/members

### UnaryLookupExpr
Context item lookup. Used inside predicates on maps/arrays.

### ConcatExpr (`||`)
Atomize both → string → concatenate.

### RangeExpr (`to`)
Evaluate start/end as xs:integer → produce sequence of integers. Apply `maxNodes` limit.

### FLWORExpr
1. Iterate `ForClause` domains (nested loops)
2. Bind variables (`LetClause`)
3. Filter (`WhereClause` → EBV)
4. Collect tuples
5. Sort (`OrderByClause`)
6. Evaluate `Return` per tuple → concatenate

### QuantifiedExpr
- `some`: iterate domain → evaluate satisfies → true on first match
- `every`: iterate domain → false on first non-match

### IfExpr
Evaluate condition → EBV → evaluate Then or Else branch.

### TryCatchExpr
Evaluate Try. On `*XPathError`: match code against catch clause codes (`*` = catch-all). Bind `$err:code`, `$err:description`, `$err:value`, `$err:module`, `$err:line-number`, `$err:column-number` in catch scope. `err:` prefix → `http://www.w3.org/2005/xqt-errors`.

### InstanceOfExpr
Check each item against SequenceType at runtime → boolean.

### CastExpr
Atomize → `CastAtomic(src, targetType)`. `AllowEmpty` allows empty sequence.
Disallowed targets such as `xs:anyAtomicType`, `xs:anySimpleType`, `xs:anyType`,
and `xs:NOTATION` raise `XPST0080` before operand cardinality is considered.

### CastableExpr
Same as CastExpr but return boolean for castability checks.
Disallowed targets such as `xs:NOTATION` still raise `XPST0080`.

### TreatAsExpr
Check instance-of → error `XPDY0050` if not.

### FunctionCall (static)
Resolve by name/arity in builtins or user registry → evaluate args → call.

### DynamicFunctionCall
Evaluate func expr → switch on the callee item: FunctionItem checks arity and
invokes directly; MapItem/ArrayItem are arity-1 lookup functions resolved via
`mapLookup`/`arrayLookup`. The placeholder/partial path instead adapts the callee
through `asFunctionItem`, which shares the same `mapLookup`/`arrayLookup` helpers.

### InlineFunctionExpr
Capture current variable scope snapshot → return FunctionItem with closure.

### NamedFunctionRef
Look up by name and arity → return FunctionItem.

### Partial Application
FunctionCall with PlaceholderExpr in args:
1. Evaluate non-placeholder args
2. Create FunctionItem closing over fixed-position args
3. New arity = placeholder count

Dynamic partial application (`$m(?)`, `$a(?)`) accepts any function item; maps and
arrays are adapted via `asFunctionItem`, so `$m(?)("k")` and `$a(?)(2)` work.

### MapConstructorExpr
Evaluate key/value pairs → keys must atomize to AtomicValue → build MapItem.

### ArrayConstructorExpr
- Square bracket (`[a, b, c]`): each expr → one member
- Curly bracket (`array { expr }`): evaluate as sequence → each item is singleton member

## Comparison Engine (`compare.go`)

### General Comparison (`= != < <= > >=`)
Atomize both → for each pair → type promotion → value compare. True if ANY pair matches. The O(N·M) left×right scan charges one op (via `fnCountOp`) and checks `ctx.Err()` per candidate pair, so a comparison over two large sequences honors `OpLimit` / context cancellation instead of running unbounded.

### Value Comparison (`eq ne lt le gt ge`)
Both must be single items. Error (XPTY0004) if sequence length > 1. Operands are atomized with an early stop (`atomizeSingletonOperand`, cap 2) so a multi-item or unbounded-lazy operand raises the cardinality error without materializing the whole sequence.

### Node Comparison (`is << >>`)
Compare node identity or document order.

### Type Promotion (simplified v1)
- untypedAtomic vs string → compare as string
- untypedAtomic vs numeric → cast untypedAtomic to double
- untypedAtomic vs untypedAtomic → compare as string
- untypedAtomic vs schema USER type → cast through the schema-aware cast helper
  (`SchemaDeclarations` builtin-base/facet/union path), preserving the user type's
  builtin `BaseType` for the subsequent value comparison
- Numeric promotion: integer → decimal → float → double

```go
func GeneralCompare(op TokenType, left, right Sequence) (bool, error)
func ValueCompare(op TokenType, a, b AtomicValue) (bool, error)
func NodeCompare(op TokenType, a, b helium.Node, cache *ixpath.DocOrderCache) (bool, error)
```

## Casting (`cast.go`)

```go
func CastAtomic(src AtomicValue, targetType string) (AtomicValue, error)
func CastFromString(s, targetType string) (AtomicValue, error)
```

Supported casts per XPath 3.1 Section 18. All atomic types are listed in `xpath3-types.md`.
QName casts require namespace context from evalContext.

`CastAtomic` normalizes a source whose `TypeName` is a schema-derived USER type
(e.g. `Q{ns}MyInt`, stamped onto an atom by `AtomizeItem` for `data()` or by the
xsd `$value` binding) to its recorded builtin `BaseType` before dispatching to
the per-target cast helpers, which key on builtin `TypeName` and would otherwise
reject the opaque user type with XPTY0004. A user-typed atom therefore casts
exactly like its base, keeping `data()` and `$value` cast/castable behavior
consistent. The guard requires `BaseType` to be a known XSD builtin
(`IsKnownXSDType(v.BaseType)`), so an arbitrary custom non-XSD `BaseType` on a
public atom does not change dispatch; it stays opaque and falls through to the
normal XPTY0004 path. Built-in atoms are unaffected. The identity re-check after
normalization preserves the `castable as <ownType>` fast path and re-runs
`validateDateTimeStampSource`, so a user type whose base surfaces
`xs:dateTimeStamp` still enforces the mandatory-timezone FORG0001 invariant on
the identity return.

`evalCastExpr`/`evalCastableExpr` atomize their operand through the typed-value
stream via `atomizeSingletonOperand` and apply the singleton-or-empty cardinality
to the atomized result. A single schema-typed node whose typed value is a
list/union expands consistently with `data()`; a more-than-one-atom operand
raises cast XPTY0004 or makes `castable` false. Function-call args are handled
separately by signature coercion (`coerceToSequenceTypeE`, which atomizes via
`atomizeStreamCont` with the typed-value pre-check `typedValueItemCheckFor(ec)`,
so atomizing an element-only-typed node arg against an atomic parameter raises
`FOTY0012` — cardinality still applies after atomization). Its whole
`coerceToSequenceType`/`coerceFuncallArg`/public
`CoerceToSequenceType`+`CoerceToSequenceTypeContext` family threads a
`context.Context` so the schema-aware cast it may trigger participates in
cancellation.

Every path that coerces a USER-facing call argument/result routes through the
error-propagating `coerceToSequenceTypeE` (directly, or via `coerceFuncallArg`),
so a real dynamic error — `FOTY0012` (atomizing an element-only node),
`FOTY0013` (atomizing a function or map — an array flattens to its atomized
members, but a function/map member still raises it), `FORG0001` (a failed
untypedAtomic→target cast) — surfaces unchanged instead of being flattened into
a generic `XPTY0004`. This covers the direct call path (`evalFunctionCall`),
partial application and named function references (`partialApply` /
`evalNamedFunctionRef`, `eval_funcall.go`), `fn:function-lookup`
(`lookupFunctionItem`, `functions_hof.go`), inline-function parameter AND
return-type coercion (`evalInlineFunctionExpr`, `eval_funcall.go`), and
function-item→function-type adaptation (`coerceFunctionItem`, `eval_types.go`). Only the PUBLIC boolean wrappers
`CoerceToSequenceType`/`CoerceToSequenceTypeContext` discard the specific error
(they are type predicates returning `(Sequence, bool)`); `instance of` /
`castable` / `treat` type tests likewise stay boolean, where a mismatch is the
correct result and raises nothing.

A user-defined UNION used as a required item type (e.g. an `xsl:function`
`as="u"` parameter, reached across an `xsl:use-package`/`xsl:original` boundary)
matches value-first: `atomicMatchesTargetType` — the shared gate for `instance
of` and function coercion — admits an atomic value when it is an instance of one
of the union's `SchemaDeclarations.UnionMemberTypes` (recursively for nested
unions, `atomicMatchesUnionMember`, cycle-guarded), since a union's members are
not base-chain subtypes of the union. An `xs:untypedAtomic` argument coerced to a
non-builtin union/faceted target routes through the schema-aware cast
(`schemaAwareCast`, first castable member wins), not `CastAtomic`. This requires
the target union's type to be resolvable in `evalContext.schemaDeclarations`; the
xslt3 runtime registry therefore includes every USED package's imported schemas
(not only the main stylesheet's), so a component's declared `as=` type resolves
in the DEFINING package's schema.

For a USER-defined target type, when context-free `CastAtomic` fails and
`evalContext.schemaDeclarations` is set, `evalCastExpr`/`evalCastableExpr` use a
shared schema-aware cast helper. The helper resolves the target's builtin base:
a QName/NOTATION-derived base validates via
`SchemaDeclarations.ValidateCastWithNS` and, for `cast`, returns the
namespace-resolved `QNameValue` carrying the user type annotation; other bases
cast through the builtin for the returned atomic value. String and untypedAtomic
sources validate facets with `ValidateCastWithNS` against the original source
lexical string so lexical facets such as patterns still see `"05"` rather than a
canonicalized numeric value; already-typed sources validate the builtin cast
result's lexical form so a cross-cast such as `xs:dateTime(...) cast as MyDate`
checks the `xs:date` value actually being returned. The helper then returns the
user type annotation with `BaseType` set to that builtin base. Union targets from
`SchemaDeclarations.UnionMemberTypes` are tried recursively through that same
schema-aware path for both `cast` and `castable`, so a union member that is
itself a user-defined type still resolves its builtin base and facets. After a
member accepts, the helper validates the target union too, so facets/assertions
on the union restriction itself can still reject the cast: string/untypedAtomic
sources use the original lexical value, while already-typed sources use the
accepted member-cast result's lexical form. `cast` returns the atomic value for
the matching member.

A user-defined LIST target type (resolved via `SchemaDeclarations.ListItemType`)
casts by tokenization (F&O 3.1 §19.1.2): a string/untypedAtomic source is split on
XSD whitespace (`xsdListFields`) and each token must be castable to the item type
through the same schema-aware path, then the whole normalized value validates
against the list type's own facets via `ValidateCastWithNS`. `castToUserList`
returns the sequence of per-item atoms (each typed as the item type) for `cast`;
`castableToUserList` returns the boolean for `castable`. Any other already-typed
atomic source is not a list literal → not castable (an empty/whitespace-only
source is the empty list, castable iff the list facets accept zero items). A
multi-item list-typed operand (e.g. the xslt3 `s:intListType1("1 2 3")`
constructor, which expands to a sequence of item atoms) is rejected earlier by the
singleton-cardinality check on the atomized operand, so casting it to the same
list type is correctly NOT castable (it is a sequence, not a single value).

`AtomicToString` (canonical lexical) normalizes a schema-derived USER-typed atom
(a non-XSD `TypeName` with a known-XSD `BaseType`) to its builtin base before
formatting, mirroring `CastAtomic`'s dispatch normalization — so a user-typed
temporal/binary/QName atom (e.g. a value of a user type derived from `xs:date`)
stringifies as its canonical XSD lexical (`"2001-01-01"`) rather than the generic
Go `%v` fallback. This keeps a union target's re-validation lexical correct when
the accepted member value carries a user type annotation.

When the cast source is itself an already-resolved `QNameValue` (e.g. `data(@q)`
where the prefix is declared only on the instance node, absent from the
assertion's static namespace map), `qnameCastLexical` re-validates using the
value's own namespace URI. It binds the lexical prefix to that URI in a copy of
the static map, minting a synthetic prefix for an unprefixed value and keeping a
bare local for a no-namespace value, so the cast succeeds instead of failing to
re-resolve `prefix:local` against a map that lacks the prefix. The `(local, ns)`
used for schema lookup is derived from the already-resolved target type
(`Q{ns}local`, via `schemaAnnotationParts`). This schema-aware path is additive:
with no `schemaDeclarations`, the cast behaves exactly as before.

Casting a string/untypedAtomic to `xs:QName`/`xs:NOTATION` applies the in-scope
default element namespace to an unprefixed value (`castToQName`), except when
`evalContext.qnameValueNoDefaultNS` is set (the opt-in
`Evaluator.QNameValueNoDefaultNamespace()`, XSD value-space semantics), where an
unprefixed QName/NOTATION value has no namespace. A prefixed value still resolves
and the default xpath3 behavior is unchanged.

## State Management

- Stateless between `Expression.Evaluate` calls
- `evalContext` created fresh per call, discarded after
- Hot-path node/item rebinding during predicates, path steps, and simple-map evaluation mutates the current `evalContext` temporarily and restores it afterward instead of allocating copied child contexts
- Default language comes from `WithDefaultLanguage`; built-ins fall back to `"en"` when unset
- `DocOrderCache` lazy, O(n) build, O(1) lookup
- Inline functions and named function refs snapshot the dynamic context they close over, so later focus rebinding does not change captured behavior

## Safety Limits

| Limit | Default | Config |
|-------|---------|--------|
| Recursion depth | 5000 | `internal/xpath.DefaultMaxRecursionDepth` |
| Parser nesting | 200 | Constant in parser |
| Op count | unlimited | `WithOpLimit(n)` |
| Sequence/node-set size | 10M | `internal/xpath.DefaultMaxNodeSetLength` |

Range expressions (`1 to N`) and `for` clauses also check sequence size limit.
