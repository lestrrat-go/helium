# Feature: `as` Attribute Type Coercion

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-type-coercion`

## Goal

Implement `as` attribute type checking and coercion on `xsl:variable`,
`xsl:param`, `xsl:function`, `xsl:template`, and `xsl:with-param`.
This affects ~27 tests in `sequence`, ~11 in `variable`, ~6 in `tunnel`,
~3 in `param`, and scattered tests elsewhere (~50 total).

## Problem Summary

The `as` attribute declares the expected type of a variable/param/function
return value. When `as` is present:

1. **Body content produces a sequence** — multiple text/element children
   become separate items, not a single document node
2. **Type checking** — the actual value is checked against the declared type
3. **Atomization** — if `as="xs:string"` and the body produces elements,
   their string values are extracted
4. **Sequence type matching** — `as="item()*"` means any sequence,
   `as="xs:integer"` means exactly one integer, `as="element()+"` means
   one or more elements

## Current State

The `as` attribute is parsed and stored on `Variable`, `Param`, `Template`,
`XSLFunction`, and `WithParam` structs. At runtime:

- Global params with `as` go through `CastFromString` (for initial values)
- Local variables/params with `as` are NOT type-checked or coerced
- Function return values with `as` are NOT type-checked
- Body content always produces a document fragment, not a typed sequence

## Key Test: sequence-0101

```xml
<xsl:variable name="q" as="item()*">
  <xsl:text>a</xsl:text>
  <xsl:text>b</xsl:text>
  <xsl:text>c</xsl:text>
</xsl:variable>
<z><xsl:value-of select="data($q)" separator=","/></z>
```

Expected: `<z>a,b,c</z>` (three separate items)
Got: `<z>abc</z>` (one concatenated text node)

## Implementation Plan

### 1. Body-to-Sequence Conversion

When `as` is present on a variable/param, the body content should be
evaluated as a **sequence**, not as a document fragment:

- Each `xsl:text` or literal text becomes a separate `xs:string` item
- Each element constructor becomes a separate element item
- Adjacent text nodes are NOT merged (unlike normal output)

Implement `evaluateBodyAsSequence()` that returns `xpath3.Sequence`
instead of writing to the result tree.

### 2. Type Checking

After obtaining the sequence, check it against the declared `as` type:
- Parse the `as` type string into a sequence type descriptor
- Check cardinality: `?` (0-1), `*` (0+), `+` (1+), none (exactly 1)
- Check item type: `xs:string`, `xs:integer`, `element()`, `node()`, etc.
- Raise XTTE0570 (variable), XTTE0505 (param), or XTTE0780 (function)
  on type mismatch

### 3. Type Coercion

When the actual type doesn't match but is castable:
- Atomize nodes to get their string values
- Cast xs:untypedAtomic to the target type
- Wrap single items in sequences for `*` or `+` types

### 4. Sequence Type Parsing

Create a `parseSequenceType(as string)` function that returns:
```go
type SequenceType struct {
    ItemType    string // "xs:string", "element()", "node()", "item()", etc.
    Occurrence  rune   // 0 (exactly one), '?', '*', '+'
}
```

## Files

- `xslt3/execute.go` — variable evaluation with `as` type
- `xslt3/execute_instructions.go` — `evaluateBodyAsSequence()`, variable/param execution
- `xslt3/compile.go` — parse `as` attribute more precisely
- New file `xslt3/types.go` — sequence type parsing and checking

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(sequence|variable|param|tunnel)$' -v -count=1 2>&1 > .tmp/type-coercion-final.txt
```
