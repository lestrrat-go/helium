# XPath 3.1 — Evaluator

## evalContext

```go
type evalContext struct {
    goCtx      context.Context
    node       helium.Node           // context item (nil if absent)
    position   int
    size       int
    vars       map[string]Sequence   // scoped variable bindings
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

func newEvalContext(ctx context.Context, node helium.Node) *evalContext
func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext
func (ec *evalContext) withVar(name string, val Sequence) *evalContext  // new map per scope
```

## Dispatch

- `Compile()` lowers AST to `vmProgram`
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
    op   vmOpcode
    expr Expr
}
type compiledExprRef struct { index int }
```

Lowering is structural: one instruction per AST node. Lowered payload Exprs keep node metadata, but recursive children become `compiledExprRef` indexes. Existing `eval_*` helpers still implement semantics; only recursion target changes.

## Evaluation Rules by Expr Type

### SequenceExpr (comma)
Evaluate each item, concatenate sequences.

### LocationPath
1. Start: root node (absolute) or context node (relative)
2. Per step: `ixpath.TraverseAxis` → filter by NodeTest → apply predicates → dedup doc order
3. Return merged node-set

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
Evaluate func expr → must be FunctionItem → check arity → invoke.

### InlineFunctionExpr
Capture current variable scope snapshot → return FunctionItem with closure.

### NamedFunctionRef
Look up by name and arity → return FunctionItem.

### Partial Application
FunctionCall with PlaceholderExpr in args:
1. Evaluate non-placeholder args
2. Create FunctionItem closing over fixed-position args
3. New arity = placeholder count

### MapConstructorExpr
Evaluate key/value pairs → keys must atomize to AtomicValue → build MapItem.

### ArrayConstructorExpr
- Square bracket (`[a, b, c]`): each expr → one member
- Curly bracket (`array { expr }`): evaluate as sequence → each item is singleton member

## Comparison Engine (`compare.go`)

### General Comparison (`= != < <= > >=`)
Atomize both → for each pair → type promotion → value compare. True if ANY pair matches.

### Value Comparison (`eq ne lt le gt ge`)
Both must be single items. Error if sequence length > 1.

### Node Comparison (`is << >>`)
Compare node identity or document order.

### Type Promotion (simplified v1)
- untypedAtomic vs string → compare as string
- untypedAtomic vs numeric → cast untypedAtomic to double
- untypedAtomic vs untypedAtomic → compare as string
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

Supported casts per XPath 3.1 Section 18. All atomic types in `xpath3-types.md`. QName casts require namespace context from evalContext.

## State Management

- Stateless between `Expression.Evaluate` calls
- `evalContext` created fresh per call, discarded after
- Default language comes from `WithDefaultLanguage`; built-ins fall back to `"en"` when unset
- `DocOrderCache` lazy, O(n) build, O(1) lookup
- Inline function closures capture variable map snapshot (value semantics)

## Safety Limits

| Limit | Default | Config |
|-------|---------|--------|
| Recursion depth | 5000 | `internal/xpath.DefaultMaxRecursionDepth` |
| Parser nesting | 200 | Constant in parser |
| Op count | unlimited | `WithOpLimit(n)` |
| Sequence/node-set size | 10M | `internal/xpath.DefaultMaxNodeSetLength` |

Range expressions (`1 to N`) and `for` clauses also check sequence size limit.
