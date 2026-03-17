# Feature: xsl:result-document

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-result-document`

## Goal

Complete `xsl:result-document` — 14 failing tests in `TestW3C_result_document$`.
Also fixes scattered failures in other categories that use result-document.

## Current State

`ResultDocumentInst` exists and executes its body into a temporary buffer
that is discarded. This prevents the body from polluting the primary output
but doesn't actually produce secondary results.

The `href` AVT is compiled but never evaluated.

Output properties (`method`, `encoding`, `indent`, etc.) from `xsl:output`
and from `xsl:result-document` attributes are not applied.

## What's Missing

1. **Secondary output storage** — the result document body needs to be
   stored so `assert-result-document` tests can verify it
2. **href evaluation** — the `href` AVT needs to be evaluated to determine
   the secondary output URI
3. **Output properties** — `method`, `encoding`, `indent`, `omit-xml-declaration`
   etc. on `xsl:result-document` should override `xsl:output` defaults
4. **Error detection** — XTDE1490 (duplicate URI), XTDE1560 (empty href
   with result-document in body)

## Implementation Plan

### 1. Store Secondary Results

In `execute.go` or `stylesheet.go`, add a map to store secondary results:

```go
// In execContext:
resultDocuments map[string]*helium.Document
```

In `execResultDocument`, after executing the body into the temp document,
store it keyed by the evaluated href.

### 2. Expose to Test Harness

In `xslt3.go`, add a method to retrieve secondary results:

```go
func (t *Transformer) ResultDocument(href string) *helium.Document
```

Update `w3c_helpers_test.go` to handle `assert-result-document` assertions
by looking up the secondary result document.

### 3. Output Property Handling

Parse output property attributes on `xsl:result-document`:
- `method`, `encoding`, `indent`, `omit-xml-declaration`, `cdata-section-elements`
- These override the corresponding `xsl:output` defaults

For now, the output properties affect serialization. Since our test
harness compares DOM trees (not serialized text), most output properties
don't affect test results. Focus on `method="text"` which changes how
the result is constructed.

### 4. Error Detection

- XTDE1490: Raise if `href` resolves to the same URI as a previous
  `xsl:result-document` in the same transformation
- Check at runtime in `execResultDocument`

## Key Files

- `xslt3/execute_instructions.go` — `execResultDocument()` enhancement
- `xslt3/execute.go` — `resultDocuments` map on `execContext`
- `xslt3/instruction.go` — `ResultDocumentInst` (already exists, may need more fields)
- `xslt3/compile_instructions.go` — parse additional attributes
- `xslt3/w3c_helpers_test.go` — `assert-result-document` support

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_result_document$' -v -count=1 2>&1 > .tmp/result-document-final.txt
```
