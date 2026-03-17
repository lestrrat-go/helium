# Feature: typed key comparison + composite keys

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-key-typed-composite`

## Goal

Fix key lookup failures caused by missing typed comparison and composite key support.

## Current State

- Key values are always compared as strings
- `composite="yes"` attribute on `xsl:key` is not implemented
- Keys defined with `use` expressions that return typed values (integers, dates)
  fail when looked up with typed arguments
- Namespace node keys don't work because `helium.Walk` doesn't visit namespace nodes

## Required Outcomes

### 1. Typed Key Comparison

Currently `key()` converts both the stored key value and the lookup value to
strings for comparison. Per XSLT 3.0 §16.3.2, key values should be compared
using the XPath `eq` operator with type-aware semantics:

- `key('k', 42)` should match a key stored as `xs:integer(42)`, not just `"42"`
- String/untyped values still compare as strings
- Mixed types that are `eq`-comparable should match

Fix the key table to store `xpath3.AtomicValue` (or at minimum the original
typed value) alongside the string representation, and use typed comparison
when the lookup value is typed.

Affected tests: key-069, key-066

### 2. Composite Keys

Implement `composite="yes"` on `xsl:key` (XSLT 3.0 §16.3.1):

- When `composite="yes"`, the `use` expression returns a sequence of values
  that form a composite key (like a multi-column index)
- `key('k', (v1, v2, v3))` matches entries where all components match
- When `composite="no"` (default), each value in the sequence is a separate
  key entry (current behavior)

The `KeyDef.Composite` field already exists in `stylesheet.go` but is not used
in the key table build or lookup logic.

Affected tests: key-093, key-096

### 3. Namespace Node Keys

`xsl:key match="namespace::*"` or patterns matching namespace nodes require
that the key index builder visits namespace nodes. Currently `helium.Walk`
skips them.

Options:
- Extend the key indexing walker to explicitly iterate `elem.Namespaces()`
- Or add a namespace-aware walk mode

Affected tests: key-087, key-090

### 4. Muenchian Grouping Consistency

key-055 uses classic Muenchian grouping with `generate-id(key('domain',.)[1])`.
The test fails because domain "Domain 2" appears twice in output. This suggests
`generate-id()` returns different IDs for the same node across calls, or the
key table returns duplicates.

Verify that:
- Key table entries are deduplicated by node identity
- `generate-id()` is stable for the same node across calls
- `key()[1]` returns a consistent first node

Affected tests: key-055

## Key Files

- `xslt3/keys.go` — key table build + lookup
- `xslt3/functions.go` — `key()` function, `generate-id()`
- `xslt3/stylesheet.go` — `KeyDef.Composite` field
- `xpath3/compare.go` — typed value comparison

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_key/(key-034|key-047|key-055|key-066|key-069|key-087|key-090|key-093|key-096)$' -v -count=1 -p 1 -timeout 600s > .tmp/key-typed.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_key$' -v -count=1 -p 1 -timeout 600s > .tmp/key-all.txt 2>&1
```

## Acceptance

- Typed key lookup works for integer/string comparisons
- Composite keys match multi-value sequences
- Namespace node keys are indexed
- Muenchian grouping produces consistent results
- key bucket failures drop from 13 to ≤5
