# XPath 3.1 — Task Breakdown

## Phase 0: Shared Infrastructure (`internal/xpath`)

### 0.1 Extract axes [Moderate]
- Create `internal/xpath/axes.go`
- Move from `xpath1/axes.go`: `traverseAxis`, all `axisXxx` functions, namespace helpers
- Move from `xpath1/expr.go`: `AxisType`, constants, `axisNames`, `axisFromName`
- Export: `AxisType`, all constants, `TraverseAxis`; unexport helpers
- Add `maxNodes int` param to `TraverseAxis`
- Deps: none

### 0.2 Extract document-order [Moderate]
- Create `internal/xpath/docorder.go`
- Move: `docOrderCache`, `deduplicateNodes`, `mergeNodeSets`, `documentRoot`, `nsNodeKey`
- Export: `DocOrderCache`, `DeduplicateNodes`, `MergeNodeSets`, `DocumentRoot`
- Deps: 0.1

### 0.3 Extract string-value [Simple]
- Create `internal/xpath/stringvalue.go`
- Move: `stringValue`, `collectTextDescendants`, `localNameOf`, `nodeNamespaceURI`, `nodePrefix`
- Export all
- Deps: none

### 0.4 Extract limits [Simple]
- Create `internal/xpath/limits.go`
- Define constants + `ErrNodeSetLimit`
- Deps: none

### 0.5 Refactor xpath1 [Moderate]
- Update `xpath1/axes.go`, `xpath1/eval.go`, `xpath1/expr.go` → call `internal/xpath`
- ALL existing `xpath1` tests MUST pass unchanged
- Deps: 0.1, 0.2, 0.3, 0.4

## Phase 1: Lexer & Parser

### 1.1 Lexer [Moderate]
- Create `xpath3/token.go`, `xpath3/lexer.go`, `xpath3/lexer_test.go`
- All tokens from `xpath3-parser.md`
- Table-driven tests for new tokens
- Deps: none

### 1.2 AST [Simple]
- Create `xpath3/expr.go`
- All Expr types, NodeTest variants, SequenceType, Occurrence
- Deps: 0.1 (for `ixpath.AxisType`)

### 1.3 Parser core [Complex]
- Create `xpath3/parser.go`, `xpath3/parser_test.go`
- `Parse(input) (Expr, error)`
- Precedence levels 0–10 (sequence through union)
- Location paths, steps, node tests, predicates
- Function calls, named function references
- Deps: 1.1, 1.2

### 1.4 Parser XPath 3.1 extensions [Complex]
- Extend `xpath3/parser.go`:
  - FLWOR (for/let/where/order-by/return)
  - Quantified (some/every)
  - if-then-else, try-catch
  - instance-of, cast-as, castable-as, treat-as (+ sequence type parsing)
  - Arrow (desugar to FunctionCall)
  - Simple map, lookup, unary lookup
  - String concat, range, value comparisons
  - Inline functions, partial application
  - Map constructors, array constructors
- Deps: 1.3

## Phase 2: Type System

### 2.1 Item + AtomicValue [Moderate]
- Create `xpath3/types.go`, `xpath3/sequence.go`
- `Item`, `ItemType`, `Sequence`, `NodeItem`, `AtomicValue`
- All atomic type constants
- Sequence helpers, `EBV`
- Deps: none

### 2.2 MapItem [Moderate]
- Add to `xpath3/types.go`
- `mapKey` normalization, `Get`, `Put`, `Contains`, `Keys`, `Size`, `ForEach`, `MergeMaps`
- Tests in `xpath3/types_test.go`
- Deps: 2.1

### 2.3 ArrayItem [Moderate]
- Add to `xpath3/types.go`
- `Get` (1-based), `Put`, `Append`, `Size`, `Members`, `SubArray`, `Flatten`
- Tests in `xpath3/types_test.go`
- Deps: 2.1

### 2.4 Casting [Moderate]
- Create `xpath3/cast.go`, `xpath3/cast_test.go`
- `CastAtomic`, `CastFromString` for all valid pairs per XPath 3.1 Section 18
- Deps: 2.1

## Phase 3: Core Evaluator

### 3.1 evalContext + basic dispatch [Moderate]
- Create `xpath3/eval.go`
- `evalContext`, `newEvalContext`, `withNode`, `withVar`, `countOps`
- `eval` dispatch
- Evaluate: `LiteralExpr`, `VariableExpr`, `SequenceExpr`, `RootExpr`
- Deps: 0.1, 0.2, 0.3, 2.1

### 3.2 Location paths [Moderate]
- `evalLocationPath` using `ixpath.TraverseAxis`
- `filterByNodeTest` (all XPath 3.1 node tests)
- `applyPredicate` (numeric vs EBV)
- Deps: 3.1

### 3.3 Binary operators [Moderate]
- Create `xpath3/compare.go`
- `GeneralCompare`, `ValueCompare`, `NodeCompare`
- Arithmetic, logic, union/intersect/except, concat, range, simple map, lookup
- Deps: 3.1, 2.4

### 3.4 FLWOR + control flow [Moderate]
- `evalFLWOR`, `evalQuantified`, `evalIf`, `evalTryCatch`
- Deps: 3.1

### 3.5 Type expressions [Simple]
- `evalInstanceOf`, `evalCast`, `evalCastable`, `evalTreatAs`
- Deps: 3.1, 2.4

### 3.6 Function infrastructure [Moderate]
- Create `xpath3/functions.go`
- `Function` interface, `FunctionContext`
- `evalFunctionCall`, `evalDynamicFunctionCall`
- Partial application, named function ref resolution, inline function closure
- `builtinFunctions3` registry skeleton
- Deps: 3.1, 2.1

## Phase 4: Built-in Functions

### 4.1 Node, string, boolean, numeric [Complex]
- `functions_node.go`, `functions_string.go`, `functions_boolean.go`, `functions_numeric.go`, `functions_aggregate.go`, `functions_sequence.go`, `functions_uri.go`, `functions_qname.go`, `functions_error.go`, `functions_misc.go`
- Port overlapping xpath1 functions (using `ixpath.*`)
- Add XPath 2.0/3.0/3.1 additions
- Deps: 3.6

### 4.2 Higher-order functions [Moderate]
- `functions_hof.go`
- for-each, filter, fold-left, fold-right, apply, function-lookup/arity/name
- Deps: 3.6, 4.1

### 4.3 Map functions [Moderate]
- `functions_map.go`
- All `map:*` functions using MapItem API
- Deps: 2.2, 3.6

### 4.4 Array functions [Moderate]
- `functions_array.go`
- All `array:*` functions using ArrayItem API
- Deps: 2.3, 3.6

### 4.5 Math functions [Simple]
- `functions_math.go`
- All `math:*` via Go `math` package
- Deps: 3.6

### 4.6 Date/time functions [Complex]
- `functions_datetime.go`
- Constructors, accessors, arithmetic, timezone adjustment
- Uses Go `time.Time` with XSD parsing/formatting
- Deps: 2.1, 3.6

## Phase 5: Public API & Wiring

### 5.1 Public API [Simple]
- Create `xpath3/xpath3.go`
- `Compile`, `MustCompile`, `Evaluate`, `Find`
- `Expression`, `Result`, direct `WithX(ctx, value)` context mutators
- All error variables
- Deps: all Phase 1–4 tasks

### 5.2 Integration tests [Moderate]
- `xpath3/xpath3_test.go`
- End-to-end with real helium DOM trees
- Cover each feature area
- Deps: 5.1

### 5.3 Fuzz test [Simple]
- `xpath3/fuzz_test.go`
- Mirror `xpath1/fuzz_test.go`
- Seed with valid XPath 3.1 expressions
- Deps: 5.1

### 5.4 Update docs [Simple]
- `.claude/docs/packages.md`: add `xpath3/`, `internal/xpath/`
- `.claude/docs/dependencies.md`: add import edges
- Deps: 5.1

## Dependency Graph

```
0.1 ─┐
0.2 ─┤
0.3 ─┼→ 0.5
0.4 ─┘

1.1 ─┬→ 1.3 → 1.4
1.2 ─┘

2.1 → 2.2, 2.3, 2.4

0.x + 2.1 → 3.1 → 3.2, 3.3, 3.4, 3.5, 3.6

3.6 → 4.1 → 4.2
2.2 + 3.6 → 4.3
2.3 + 3.6 → 4.4
3.6 → 4.5
2.1 + 3.6 → 4.6

All → 5.1 → 5.2, 5.3, 5.4
```

## Parallelizable

- Phase 0 tasks 0.1/0.3/0.4 in parallel
- Phase 1 (lexer/parser) and Phase 2 (types) in parallel
- Phase 4 function files (4.1–4.6) largely independent after 3.6

## Rollout

1. Phase 0 lands first. `xpath1` tests guard correctness.
2. Phases 1–2 merge as skeleton (parse-only tests).
3. Phases 3–4 iterative merges per sub-feature.
4. Phase 5 final merge. `xpath3` usable end-to-end.

Rollback: `xpath3` is new package → trivially delete. Phase 0 is only existing-code change.

## Open Questions

1. **xs:decimal**: `string` (v1) or `*big.Rat`? → string for now, float64 for arithmetic
2. **Unicode collation**: codepoint only (v1). Other collations → error
3. **Stub functions**: format-number, format-date/dateTime/time, format-integer, analyze-string, serialize → return "not implemented" error
4. **QName cast**: needs namespace context from evalContext
5. **`$err:*` in catch**: inject into evalContext when evaluating catch body

## Risks

| Risk | Mitigation |
|------|------------|
| xs:duration arithmetic complexity | Implement only dayTimeDuration + yearMonthDuration; punt mixed-duration |
| Regex flag incompatibility (XPath vs Go re2) | Map flags, document differences |
| Sequence size explosions (range, for) | Check `maxNodes` limit |
| Map key collision for time.Time | Always normalize to UTC before keying |
