# Feature: document() mode dispatch + select residuals

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-doc-select`

## Goal

Fix remaining document() and select failures.

## Current State

### document() failures (6)

- **document-0801/0802/0803**: Same stylesheet with different initial modes.
  The W3C catalog specifies `<initial-mode name="a">` etc. for each test case.
  The test harness does not extract or use the initial-mode attribute.
  These tests all produce filename strings instead of loaded document content,
  suggesting `document()` returns filenames when called with mode-specific
  templates that never fire.

- **document-0307**: Output is `truetrue` instead of `true true`.
  The stylesheet uses `xsl:value-of` with two `select` expressions producing
  a sequence — the default separator (space) is not being applied.

- **document-2401**: Requires `xsl:package` — blocked on package work.

- **document-2402**: Initial template "a" not found — the test specifies
  `initial-template name="a"` but the harness passes it wrong or the
  stylesheet doesn't have that template name.

### select failures (1)

- **select-7502b**: `XTSE0870: xsl:value-of must have a select attribute or content`.
  This is an XSLT 3.0 change — `xsl:value-of` with `separator` but no `select`
  and no content should produce an empty string, not an error. The compiler
  is enforcing an XSLT 1.0 rule.

## Required Outcomes

### 1. Initial Mode in Test Harness

Add initial-mode support to the W3C test harness:

- Parse `<initial-mode>` from W3C catalog `<test>` elements in the generator
- Add `InitialMode` field to `generatedTest` struct (if not already present)
- Emit `InitialMode: "modename"` in generated test code
- In the test runner, call `xslt3.WithInitialMode(ctx, tc.InitialMode)` when set
- Add `WithInitialMode` option to `xslt3/options.go` if not already present
- Wire initial mode into `Transform()` to set `ec.currentMode` at startup

Affected tests: document-0801, document-0802, document-0803

### 2. xsl:value-of Separator

document-0307 outputs `truetrue` instead of `true true`. The issue is in
`execValueOf` — when multiple items are produced, the default separator
should be a single space per XSLT spec §11.3.

Check whether the `HasSeparator` flag is being set correctly when no explicit
`separator` attribute is present. The default should be `" "` for `select`
and `""` for content.

Affected tests: document-0307

### 3. xsl:value-of Empty Content

select-7502b: XSLT 3.0 allows `<xsl:value-of separator="..."/>` with no
`select` and no content — it produces an empty string. The compiler should
not raise XTSE0870 in this case.

Fix the compile-time check in `compileValueOf` to allow empty `xsl:value-of`
when other attributes (like `separator`) are present, or unconditionally
for XSLT 3.0 (version >= 3.0).

Affected tests: select-7502b

### 4. document-2402

Check the W3C catalog for this test case to understand what initial template
name is expected. If the generator needs to extract a different template name,
fix the extraction. If the stylesheet expects template "a" via an
`<initial-template name="a"/>` element, ensure the harness passes it.

Affected tests: document-2402

## Key Files

- `tools/xslt3gen/main.go` — initial-mode extraction
- `xslt3/w3c_helpers_test.go` — `w3cTest.InitialMode`, runner changes
- `xslt3/options.go` — `WithInitialMode` option
- `xslt3/execute.go` — initial mode dispatch in `Transform()`
- `xslt3/compile_instructions.go` — `compileValueOf` XTSE0870 check
- `xslt3/execute_instructions.go` — `execValueOf` separator default

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_document/(document-0307|document-0801|document-0802|document-0803|document-2402)$' -v -count=1 -p 1 -timeout 600s > .tmp/document-fixes.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_select/select-7502b$' -v -count=1 -p 1 -timeout 600s > .tmp/select-fixes.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_(document|select)$' -v -count=1 -p 1 -timeout 600s > .tmp/doc-select-all.txt 2>&1
```

## Acceptance

- document-0801/0802/0803 pass with initial-mode dispatch
- document-0307 outputs `true true` with space separator
- select-7502b no longer raises XTSE0870
- document + select failures drop from 7 to ≤2 (document-2401 blocked on package)
