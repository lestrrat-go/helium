# Streaming implementation hand-off

Date: 2026-03-15
Branch: `feat-streaming` (off `feat-xslt3`)
Worktree: `.worktrees/feat-streaming`

## Current state

326 streaming tests are runnable (out of 2,542 total in the `strm` category;
the remaining 2,216 are skipped because their stylesheets are embedded in the
W3C test catalog XML and not extracted as files by xslt3gen).

Of the 326 runnable tests, **299 pass (92%)** and **27 fail**.

Zero regressions against the non-streaming test suite.

## What was implemented

### New XSLT instructions

- `xsl:source-document` — loads external XML by URI, runs body against it
  (DOM-materialization; no SAX-level streaming)
- `xsl:iterate` with `xsl:break`, `xsl:next-iteration`, `xsl:on-completion`
- `xsl:fork` — sequential branch execution
- `xsl:accumulator` / `xsl:accumulator-rule` — state accumulation during
  tree traversal
- `xsl:merge` / `xsl:merge-source` / `xsl:merge-key` / `xsl:merge-action` —
  N-way sorted merge with type-aware key comparison
- `xsl:map` / `xsl:map-entry` — map constructors
- `xsl:attribute-set` / `use-attribute-sets`

### New functions

- `snapshot()` — deep-copies current node with ancestor chain for navigation
- `copy-of()` — 0 or 1 argument (streaming snapshot variant)
- `accumulator-before(name)` / `accumulator-after(name)`
- `current-merge-group()` / `current-merge-group(source-name)` /
  `current-merge-key()`
- `unparsed-entity-public-id()`

### Streamability analysis (XTSE3430)

Post-compilation pass in `xslt3/streamability_analysis.go` that walks compiled
instruction trees and XPath ASTs. Detects non-streamable constructs in modes
with `streamable="yes"` and in `xsl:source-document streamable="yes"` bodies.

XPath AST inspection helpers live in `xpath3/streamability.go`.

Covers: upward axes, `last()` in predicates, multiple downward selections,
attribute-set streamability, for-each/group/map/iterate/fork streamability,
function declared-streamability (absorbing, inspection, filter, ascent,
shallow-descent).

### Other changes made during streaming work

- `on-no-match="deep-copy"` mode now handles element nodes properly
- `match="."` pattern now works (ContextItemExpr matching + correct priority)
- Composite `group-by` keys (`composite="yes"`)
- `default-mode` attribute on `xsl:stylesheet`
- `xsl:value-of` with `separator=""` (distinguished from absent separator)
- Strip-space applied to source documents loaded by `xsl:source-document`
- `CopyDTDInfo()` for preserving unparsed entities across document copies
- `CopyNode()` preserves namespace-qualified attributes
- `PathExpr` order preservation for `reverse()` in path expressions
- Tunnel parameter isolation in `apply-imports` and `next-match`
- `map:merge` default duplicate handling corrected to `use-first`
- User-defined function atomization for atomic return types
- `key()` function context-node fallback restored
- `Q{uri}local` EQName notation in `xsl:function` names

## Remaining 27 failures

### XTSE3430 streamability analysis gaps (18 tests)

These tests expect compile-time rejection (XTSE3430) but the analyzer does
not detect the violation.

**si-map-902**: `xsl:map-entry` with both key and value consuming the stream.
The map-entry key uses `head(//AUTHOR)` and value uses `data(head(//TITLE))`,
both navigating downward in the streamed document. The analyzer checks for
multiple downward selections in map entries but misses this case because
`head()` wrapping makes each individual selection appear singular. Needs:
count downward selections across key AND value of a single map-entry, treating
both as consuming.

**si-map-903**: `xsl:map` body has mixed `xsl:map-entry` and `xsl:if` children.
The implicit-fork rule for `xsl:map` requires ALL children to be
`xsl:map-entry`. An `xsl:if` containing a map-entry violates this. Needs:
check that xsl:map children are exclusively xsl:map-entry elements.

**su-shallow-descent-903, su-shallow-descent-905**: Functions declared
`streamability="shallow-descent"` but using downward axis on parameters in
ways that violate the constraint. -903 uses `$n/*` directly; -905 uses it
in a non-striding pattern. Needs: stricter shallow-descent parameter usage
checks — any `$param/*` must be pure striding (no predicates or further
nesting).

**su-filter-903**: Filter function uses `$element/..` (parent axis). Filter
functions must not navigate upward from their parameter. Needs: detect parent
axis (`..`) on function parameters in filter-declared functions.

**su-absorbing-205, su-absorbing-908**: Absorbing functions that reference
the same streaming parameter multiple times. -205 has nested recursive
`tail()` calls; -908 has multiple explicit variable references to the same
consuming parameter. Needs: count references to consuming parameters more
precisely, including through variable bindings.

**si-for-each-904, si-iterate-904**: Both use `count($current/*[current()])`
inside a loop body. The `current()` function references the outer context,
which is the streamed node, making the predicate non-motionless. Needs:
detect `current()` inside predicates on downward selections.

**si-iterate-035**: Binds streamed elements to `xsl:iterate` parameters
(`$highest`, `$lowest` of type `element()?`) and then compares them. Storing
streamed nodes in parameters that persist across iterations is non-streamable.
Needs: detect element()/node() typed iterate parameters that are bound to
streamed input.

**si-fork-951, si-fork-954**: Fork branches that consume the stream via
`current-group()` or downward context selections. -951 uses `current-group()`
in multiple fork branches; -954 uses downward selection from context. Needs:
more precise fork branch consumption counting.

**si-fork-116**: Fork with `for-each-group` where `current-group()` is used
in a consuming position inside the fork. Needs: detect `current-group()`
consumption within fork branches.

**si-for-each-group-030, si-for-each-group-031, si-for-each-group-052,
si-for-each-group-063**: Various for-each-group patterns that violate
streaming. -030 uses `current-group()` in a result-document href; -031 is
similar; -052 opens a nested `xsl:source-document` using `current-group()[1]`;
-063 has nested `for-each-group` consuming `current-group()`. Needs: detect
`current-group()` usage in non-consuming positions (href AVTs, nested
source-document, nested for-each-group).

**si-value-of-101**: Uses `outermost(descendant::record)` in a streamable
template body. `outermost()` is a grounding function but `descendant::record`
is a crawling selection. The combination should be rejected. Needs: detect
that grounding functions do not make their crawling arguments streamable when
used in a streaming value-of.

### Other expected errors (3 tests)

**si-lre-906**: Expects XTSE0020 — a literal result element references an
attribute set marked `streamable="Yes"` (note capital Y). The compiler
currently only recognizes `"yes"` (lowercase). Needs: case-insensitive or
`"Yes"`/`"1"`/`"true"` handling for `streamable` attribute values, OR
specific XTSE0020 detection for streamable attribute-set references.

**si-fork-813**: Expects XTDE3365 — accumulator-based map access through
dynamic function call fails at runtime. Currently produces a different error.
Needs: accumulator evaluation during streaming traversal, or at minimum the
correct XTDE3365 error code.

**si-fork-816**: Expects a passing result but gets XPTY0004. Uses accumulators
to build a map of transaction-to-item mappings, then dynamic function call.
The `xs:date` cast fails because accumulator values arrive as
`xs:untypedAtomic`. Needs: proper accumulator evaluation with type coercion,
or function coercion rules for `xs:date()` constructor accepting
`xs:untypedAtomic`.

### Assertion failures (3 tests)

**sf-snapshot-0102**: `snapshot()` produces a `<wrong>` element where it
shouldn't. The stylesheet defines a user function `f:snap()` that calls
`snapshot()` and checks properties (name, local-name, namespace-uri).
The snapshot's ancestor chain or namespace handling produces unexpected
nodes. Needs: investigation of what `<wrong>` is generated and why —
likely a namespace or name() discrepancy in the snapshot copy.

**si-group-037**: Composite-key `group-adjacent` with text nodes. Expected
output has `<span>` elements with interleaved text, but actual output is
missing some text nodes. Needs: investigation of how `group-adjacent` handles
text node boundaries with composite keys — the text content between elements
may not be correctly included in groups.

**si-merge-006**: Merge of FAX log text lines with transaction XML. Uses
`unparsed-text-lines()` as one merge source, `xsl:source-document` as another.
Multiple issues: `expand-text="true"` is not recognized (only `"yes"` works),
and `xsl:where-populated` doesn't suppress empty elements. Needs: accept
`"true"`/`"1"` as boolean-true for `expand-text`, and fix `where-populated`
to suppress elements whose only children are whitespace text.

### Pre-existing type errors (2 tests)

**si-merge-002, si-merge-003**: Both fail with `XPTY0004: first arg must be
xs:date, got xs:untypedAtomic` in the `dateTime()` constructor. The merge
key expressions use `xs:dateTime(@timestamp)` and `dateTime(../@date, time)`.
The `@timestamp` arrives as `xs:untypedAtomic` because there is no schema
validation. The `fn:dateTime()` function should accept `xs:untypedAtomic`
and cast it, but currently rejects it. Needs: function coercion in
`fn:dateTime()` (and possibly `xs:date()` / `xs:time()`) to accept
`xs:untypedAtomic` arguments.

## Slow tests

These tests pass but take noticeably long. All process large XML documents
(ot.xml = Old Testament, citygml.xml = 3D city model, big-transactions.xml).

| Test | Time | Source doc | Notes |
|------|------|-----------|-------|
| si-choose-012 | ~3.0s | big-transactions.xml | Large DOM |
| si-iterate-037 | ~2.0s | ot.xml | tokenize + iterate |
| si-iterate-134 | ~1.3s | citygml.xml | polygon counting |
| si-iterate-135 | ~1.3s | citygml.xml | point counting |
| si-apply-imports-068/069/070 | ~1.0s | ot.xml | deep import chains |
| si-next-match-067 | ~1.0s | ot.xml | deep template chain |
| si-lre-904/905 | ~0.8s | ot.xml | XTSE3430 expected |
| si-merge-001/004/004ns/005 | ~0.5-0.7s | log-file XML | merge with date keys |
| si-iterate-131/132/133 | ~0.7s | citygml.xml | iterate over GML |

These would benefit from true SAX-level streaming (Phase 2) which avoids
building the full DOM. In the interim, the unicode-90 optimizations
(`WithVariablesBorrowed`, `itemToCodepoint` fast path, etc.) already help
with some of the overhead.

## Non-streaming work discovered during streaming

Several non-streaming features were implemented or fixed as side effects:

- `xsl:attribute-set` — was not implemented before; now working (31 new
  passing attribute-set tests)
- `xsl:map` / `xsl:map-entry` — was stubbed; now working
- `default-mode` on `xsl:stylesheet` — was not wired up
- `Q{uri}local` EQName parsing in function names
- `expand-text` should accept `"true"`/`"1"` in addition to `"yes"`
  (discovered via si-merge-006 but not yet fixed)
- `xsl:where-populated` should suppress elements with only whitespace
  children (discovered via si-merge-006 but not yet fixed)
- `fn:dateTime()` should coerce `xs:untypedAtomic` arguments (discovered
  via si-merge-002/003 but not yet fixed)

## Files added or significantly modified

### New files
- `xslt3/compile_streaming.go` — compile functions for streaming instructions
- `xslt3/execute_streaming.go` — execution logic for streaming instructions
- `xslt3/streamability_analysis.go` — XTSE3430 post-compilation analysis
- `xpath3/streamability.go` — XPath AST inspection helpers

### Heavily modified files
- `xslt3/instruction.go` — new instruction types
- `xslt3/stylesheet.go` — AccumulatorDef, AttributeSetDef, mode/function fields
- `xslt3/compile.go` — compile orchestration + shared helpers
- `xslt3/compile_functions_modes.go` — mode/function/global-context-item compilation
- `xslt3/compile_formats.go` — key/output/attribute-set/decimal-format/space handling compilation
- `xslt3/compile_instructions.go` — instruction dispatch + use-when/static context helpers
- `xslt3/compile_instructions_flow.go` — apply/call/choose/for-each/group/sort/try/evaluate compilation
- `xslt3/compile_instructions_nodes.go` — node-constructor/copy/number/LRE compilation
- `xslt3/compile_instructions_vars.go` — local variable/param/message/map/assert compilation
- `xslt3/execute.go` — execContext fields, strip-space, deep-copy mode
- `xslt3/execute_instructions.go` — new instruction dispatch, copy/sequence fixes
- `xslt3/functions.go` — new XSLT functions, copy-of/snapshot, key() fix
- `xslt3/compile_patterns.go` — match="." pattern, ContextItemExpr handling
- `copy.go` — CopyDTDInfo, namespace-qualified attribute copying
- `xpath3/eval_operators.go` — PathExpr order preservation
- `xpath3/functions_map.go` — map:merge default fix
- `tools/xslt3gen/main.go` — streaming feature enabled
- `xslt3/w3c_strm_gen_test.go` — regenerated with streaming tests
- `xslt3/w3c_helpers_test.go` — whitespace tolerance, slow test TODOs

### Test data added
- `testdata/xslt30/testdata/tests/strm/docs/` — 14 XML/text source files
  copied from W3C source (bullets.xml, loans.xml, works.xml, etc.)
- `testdata/xslt30/testdata/tests/strm/sf-snapshot/` — snapshot test data
- Various si-* subdirectories — additional test source files
