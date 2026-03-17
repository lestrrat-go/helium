# Feature: document() harness gaps + xsl:document residuals

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-doc-harness`

## Goal

Fix remaining document() and xsl:document failures that stem from harness
gaps (missing initial-mode support) and content-model issues.

## Current State

### document() failures (6)

- **document-0801/0802/0803**: Same stylesheet (`document-0801.xsl`) with
  different expected outputs. The W3C catalog specifies different `initial-mode`
  values for each test case ("a", "b", "c"). The test harness does not support
  `initial-mode` — it always uses the default mode.
- **document-2401**: Requires `xsl:package` support (blocked on package work)
- **document-2402**: Wrong initial template — test expects template "a" but
  harness doesn't pass it correctly
- **document-0307**: Whitespace difference — `out = "true true"` vs `out = "truetrue"`
  (missing separator in `xsl:value-of`)

### xsl_document failures (6)

- **xsl-document-0606**: Requires `use-when` compile-time evaluation (not implemented)
- **xsl-document-0601**: Comment content construction with `xsl:document` inside
  `xsl:comment` — content ordering wrong
- Others: content-model and type-checking issues

## Required Outcomes

### 1. Initial Mode Support in Test Harness

- Add `InitialMode` field to `w3cTest` struct
- Update generator to extract `initial-mode` from W3C catalog `<test>` elements
- Update test runner to pass initial mode to `Transform()`
- Add `WithInitialMode` option if not already present

### 2. document-0307 Separator Fix

- `xsl:value-of` with `select` producing a sequence should use space separator
  by default — check if the test's `separator` attribute is being honored

### 3. xsl-document Content Model

- Fix comment content construction when `xsl:document` appears inside `xsl:comment`
- The `xsl:document` body should be atomized to string for comment content,
  stripping child elements/comments from the source selection

### 4. Generator: initial-mode Extraction

The W3C catalog uses:
```xml
<test>
  <initial-mode name="a"/>
  ...
</test>
```

The generator must:
- Parse `<initial-mode>` elements from `<test>`
- Emit `InitialMode: "a"` in the generated test struct
- The runner must use this mode when calling `Transform()`

## Key Files

- `tools/xslt3gen/main.go` — initial-mode extraction
- `xslt3/w3c_helpers_test.go` — `w3cTest` struct, runner, `InitialMode` field
- `xslt3/options.go` — `WithInitialMode` if needed
- `xslt3/execute.go` — initial mode dispatch
- `xslt3/execute_instructions.go` — `execDocument` content model

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_document$' -v -count=1 -p 1 -timeout 600s > .tmp/document.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_xsl_document$' -v -count=1 -p 1 -timeout 600s > .tmp/xsl-document.txt 2>&1
```

## Acceptance

- document-0801/0802/0803 pass with correct initial-mode dispatch
- document-0307 separator issue fixed
- xsl-document-0601 content model fixed
- Combined document + xsl_document failures drop from 12 to ≤4
