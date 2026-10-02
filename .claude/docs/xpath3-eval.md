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
- The string-based `Compile()` path can reuse parsed slices during lowering because the compiled expression keeps
  `source` + `vmProgram` and reparses AST only for AST/streamability access
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

Lowering is structural for non-trivial nodes: recursive children usually become `compiledExprRef` indexes,
trivial leaves such as literals or variable refs can stay inline in the parent payload, and hot path forms now
use VM-specific payloads instead of AST nodes. `LocationPath` / `PathExpr` lower to `vmLocationPathExpr` /
`vmPathExpr` with `vmLocationStep` slices, and the direct-compile fast path can emit `vmLocationPathExpr`
directly before lowering child predicate expressions. Common predicates on VM location steps also lower to
inline VM predicate payloads for `[N]`, `[position() = N]`, `[@attr]`, and `[@attr = "literal"]`; other
predicates stay as lowered `Expr`s. VM execution switches on `vmOpcode` for compiled refs, then reuses
existing `eval_*` helpers for language semantics.

## Evaluation Rules by Expr Type

### SequenceExpr (comma)
Evaluate each item, concatenate sequences through `appendBoundedSeq` so the aggregate honors `maxNodes` / OpLimit /
cancellation (each operand is individually capped, but the concatenation must be bounded too).

### LocationPath
1. Start: root node (absolute) or context node (relative)
2. Per step: traverse axis → filter by NodeTest → apply predicates → `ixpath.OrderStepResult` (document order,
   no duplicates; the whole-document order index is built only when the step shape cannot prove the order,
   see `xpath3-architecture.md`)
3. Hot axes (`child`, `attribute`, `self`, `parent`) fuse traversal and node-test filtering directly in `xpath3`,
   avoiding the generic `TraverseAxis` + extra filtered-slice path
4. Return merged node-set

`evalVMLocationPathNodes` runs the steps and returns the `[]helium.Node`; `evalVMLocationPath` wraps it in node
items, and `evalVMPathExpr` (`E1/path`) takes the node list for each E1 node directly.

#### `//` fusion (`eval_path_descendant.go`)
A bare `descendant-or-self::node()` step (what `//` abbreviates) followed by a child or attribute step runs as
one pre-order walk of each context node's subtree instead of two steps, when `fusesDescendantContexts` accepts the
context list: every node passes `ixpath.OrderedFrom` (not inside entity content, not an `Entity`), and with several
nodes none is an attribute or namespace node and none lies inside another's subtree (a node whose parent differs
from the previous node's is checked against the last top-level node's ancestors). Child matches are emitted when
the walk reaches them and attributes when it reaches their element, so the result is in document order and neither
step builds the whole-document order index. Any other context list evaluates the steps one by one. The fused walk
keeps every observable effect of the two-step evaluation:
- per context node it fails with `ErrNodeSetLimit` when the subtree has more than `maxNodes` nodes and then
  charges one op per subtree node, as the descendant-or-self traversal does;
- it then fails with `ErrNodeSetLimit` when all subtrees together exceed `maxNodes`, and registers the document in
  the order cache when they hold more than one node (`DocOrderCache.ReserveDocument`, where `OrderStepResult` or
  `DeduplicateNodes` would), then charges the ops the second step's per-node enumeration charges (XDM children or
  attributes), and applies the node-set limit to the result;
- predicates of the second step run per parent, on that parent's candidate list (so `position()`/`last()` keep
  their meaning), in document order of the parents. With predicates, a first pass counts the subtrees so the
  limits, the first charges and the registration come before any predicate runs; the second pass marks each
  parent's selected children on the walk stack and emits them when they are popped.

#### Node lists between instructions (`vm_path_nodes.go`)
A node-list producer is an instruction whose result is always the `nodeItemFor` wrapping of a node list (or an
error): a location path, a union, an intersect/except, a `vmPathExpr` with a location path after its first operand,
and a filter expression over a producer (`nodeListInstruction`). `vm.evalNodeListRef` evaluates such an operand to
its `[]helium.Node`, with the recursion accounting of `evalWith`, and these consumers read the list instead of a
sequence of node items:
- A one-argument static call resolves the function after evaluating the argument; the built-in `fn:count`,
  `fn:exists`, `fn:empty`, `fn:boolean`, `fn:not` and `fn:head` (all `item()*`) compute their result from the node
  list, and any other function (including a user function that shadows one of them) gets the node items through
  `callResolvedFunction`.
- Union and intersect/except read producer operands as node lists (`vm.evalOperand`), and are producers
  themselves.
- A filter expression over a producer passes the node list to its predicates; a numeric literal predicate
  (`(//x)[1]`) charges the per-node ops and the first node's recursion check of `applyPredicate` and then selects
  by position without evaluating the literal per node.
- The first operand of `E1/path`, of a non-axis path step `E1/E2` and of a simple map `E1 ! E2` is read as a node
  list; each node is the context node, as for a node item.
- A producer in a position that only takes its effective boolean value (a step or filter predicate, an `if`
  condition, an `and`/`or` operand, a quantified `satisfies`) is marked `vmInstruction.ebv` at lowering
  (`vmBuilder.markEBV`) and evaluates to `xs:boolean` (non-empty node list); a predicate never takes a single
  `xs:boolean` as a position, so the outcome is the one the node sequence gives. The walk still runs to the end,
  so op charges and node-set limits are unchanged.
- `Evaluator.Evaluate` and `Expression.EvaluateReuse` keep the node list in the `Result` when the root is a
  producer and the evaluator has no type annotations (`vmProgram.execute`), so `nodeItemFor(n)` is
  `NodeItem{Node: n}`; `Result.Sequence()` builds the `ItemSlice` on first use (see `xpath3-api.md`).

Raw AST evaluation (`dispatchExpr`, `evalLocationPath`) has none of these paths and no `//` fusion; `Compile` and
`CompileExpr` lower every expression to VM instructions, so it only runs for the immediate leaves.

### UnionExpr / IntersectExceptExpr
`evalUnionExpr` gathers the nodes of both operands into one buffer (left, then right; a non-node item raises
`ErrUnionNotNodeSet`) and orders them with `ixpath.UnionNodeSets`, which returns what `ixpath.MergeNodeSets`
returns. When the concatenation is a subsequence of one element's attribute list followed by its owned-child list
(`@*|node()`, `@id|*`, `*[1]|*[last()]`) it is already in document order and free of duplicates, so it is
returned as is and the document order index is not built. Otherwise the sort keys are resolved and two operands
that are each strictly increasing are merged in one pass; anything else is deduplicated and sorted. Every result
node is wrapped again with `nodeItemFor`; the VM (`vm.unionNodes`) reads producer operands as node lists and hands
the merged list to node-list consumers unwrapped. `evalIntersectExceptExpr` filters the left operand by node identity
(`makeNodeIdentityKey`) and orders the result with `ixpath.DeduplicateNodes`. xpath1 keeps
`ixpath.MergeNodeSets` for its union.

### Child-list enumeration
The child axis and the descendant walks enumerate child lists through `helium.Children` (the owned-child
boundary), and keep only XDM kinds (`IsXDMChild`). The XDM has no entity references, so from an entity-reference
context node (entity substitution off) the child and descendant axes are empty: its only child is the DTD-owned
`Entity` node, and the walk does not follow that node's sibling links into the DTD. String-value
(`ixpath.StringValue`, an element's `Content()`) counts an entity reference as the text it expands to, so an
element or document holding references has the string-value of the tree a substituted parse builds, and a
document's DTD adds nothing.

**Cancellation:** the fused hot child/attribute loops check `ctx.Err()` once per enumerated node (the
attribute path also inside its `ForEachAttribute` callback) so a cancelled context aborts mid-enumeration
instead of scanning the whole child/attribute set before the next `countOps` boundary. Generic (non-hot) axes
delegate to `ixpath.TraverseAxis(ctx, ...)`, which performs its own in-loop `ctx.Err()` checks; on the
namespace axis those checks run inside the `NamespacePrefixesInScope` / `CollectNamespaceNodes` helper loops
(outer and inner) so `namespace::*` cancels promptly too.

### Path steps (`E1/E2`)
Every step but the last must return nodes only: `PathExpr` / `vmPathExpr` (axis-step E2) and `PathStepExpr`
(non-axis E2) return `ErrPathNotNodeSet` (an `*XPathError` with code XPTY0019) when E1 holds any non-node.
`evalPathStepExpr` evaluates E2 once per E1 node. All-node results are sorted and deduplicated, all-non-node
results keep their order, and any mix of nodes and non-nodes raises XPTY0018, whether the mix sits in one
evaluation of E2 (`/a/(1, .)`) or across evaluations for different nodes. An empty result counts as either kind.

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
Evaluate Try. On `*XPathError`: match code against catch clause codes (`*` = catch-all). Bind `$err:code`,
`$err:description`, `$err:value`, `$err:module`, `$err:line-number`, `$err:column-number` in catch scope.
`err:` prefix → `http://www.w3.org/2005/xqt-errors`.

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
Atomize both → for each pair → type promotion → value compare. True if ANY pair matches. The O(N·M) left×right
scan charges one op (via `fnCountOp`) and checks `ctx.Err()` per candidate pair, so a comparison over two
large sequences honors `OpLimit` / context cancellation instead of running unbounded.

### Value Comparison (`eq ne lt le gt ge`)
Both must be single items. Error (XPTY0004) if sequence length > 1. Operands are atomized with an early stop
(`atomizeSingletonOperand`, cap 2) so a multi-item or unbounded-lazy operand raises the cardinality error
without materializing the whole sequence.

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
lexical string so lexical facets such as patterns still see `"05"` exactly as
written, with no numeric canonicalization; already-typed sources validate the builtin cast
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
stringifies as its canonical XSD lexical (`"2001-01-01"`), never falling through
to the generic Go `%v` form. This keeps a union target's re-validation lexical correct when
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
- Hot-path node/item rebinding during predicates, path steps, and simple-map evaluation mutates the current
  `evalContext` temporarily and restores it afterward instead of allocating copied child contexts
- Default language comes from `WithDefaultLanguage`; built-ins fall back to `"en"` when unset
- `DocOrderCache` lazy, O(n) build, O(1) lookup: one flat `map[helium.Node]sortKey` covering every indexed
  document, so a position lookup is a single hash probe with no parent-chain walk. A second map records each
  document root's registration order, which orders nodes from different trees. A document can be registered
  without being indexed (`ReserveDocument`, used by `OrderStepResult` when a location step needs no sort, and by the `//` fusion); the
  first lookup that needs a position in it indexes it under the reserved order
- Inline functions and named function refs snapshot the dynamic context they close over, so later focus rebinding does
  not change captured behavior

## Safety Limits

| Limit | Default | Config |
|-------|---------|--------|
| Recursion depth | 5000 | `internal/xpath.DefaultMaxRecursionDepth` |
| Parser nesting | 200 | Constant in parser |
| Op count | unlimited | `WithOpLimit(n)` |
| Sequence/node-set size | 10M | `internal/xpath.DefaultMaxNodeSetLength` |

Range expressions (`1 to N`) and `for` clauses also check sequence size limit.

The limits hold where int is 32 bits:
- The op counter saturates at `math.MaxInt` instead of wrapping (`countOps`; xpath1 does the same), and a
  charge that would pass `math.MaxInt` returns `ErrOpLimit`.
- A `Sequence` length is an int, so `NewRangeSequence` reports a range longer than `math.MaxInt` as
  `math.MaxInt` items (`rangeLen`); every length limit still rejects it.
- An array index that fits int64 but not int raises FOAY0001 (`checkedArrayIndex`).
- A yearMonthDuration month total that does not fit int raises FODT0002 (see `xpath3-types.md`, "Duration
  month range").
- Date/time ± duration moves at most `maxSafeAddDateYears` (1e11 years where int is 64 bits, `math.MaxInt/4`
  where it is 32 bits) and passes `time.AddDate` operands that fit int (`addMonths` splits months into years
  and months, `addDays` adds days in chunks). A result year past `maxResultYear` (`math.MaxInt/2`) raises
  FODT0001; only where int is 32 bits can a result reach it.
