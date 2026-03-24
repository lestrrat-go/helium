# Feature: source-document missing assets + validation semantics

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-source-doc-assets`

## Goal

Fix remaining source_document failures by extracting missing test assets
and implementing validation mode semantics.

## Current State

- 16 source_document failures remain
- 7 are missing test data files never extracted from W3C catalog
- 5 require schema-aware validation semantics (`validation="strip"` / `"lax"`)
- 1 requires accumulator integration
- Others are streamability analysis gaps

## Required Outcomes

### 1. Missing Test Data Extraction

Extend generator to extract these source-document test data files:

- `stream-205.xml`
- `stream-206.xml` (if separate from 205)
- `stream-210.xml`
- `stream-211a.xml`
- `stream-500.xml` (or equivalent)
- `non-stream-200` source data

These files are referenced by W3C stylesheets via relative URIs.
Check the W3C catalog `<environment>` elements for `<source>` entries
that point to these files.

If files exist in `testdata/xslt30/source/` (gitignored W3C source),
copy them to `testdata/xslt30/testdata/tests/insn/source-document/`.

### 2. Validation Mode Semantics

Implement `validation` attribute on `xsl:source-document`:

- `validation="strip"` — remove type annotations from loaded document
- `validation="lax"` — validate if schema is available, strip otherwise
- `validation="strict"` — validate against schema, error if no schema
- `validation="preserve"` — keep existing type annotations

For non-schema-aware mode (current default):
- `strip` is a no-op (already untyped)
- `lax` is a no-op
- `strict` should raise `XTTE1510` if no schema is imported
- `preserve` is a no-op

The key test pattern is:
```xpath
element instance of element(*, xs:untyped)+
```
This must return true when validation="strip" is in effect.

### 3. Generator Asset Extraction

If the generator doesn't currently copy source-document environment files:
- Add logic to `collectTransitiveDeps` or environment source handling
- Regenerate test files after asset extraction

## Key Files

- `tools/xslt3gen/main.go` — asset extraction for source-document environments
- `xslt3/compile_streaming.go` — validation attribute compilation
- `xslt3/execute_streaming.go` — validation mode at document load time
- `testdata/xslt30/testdata/tests/insn/source-document/` — test data directory

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_source_document$' -v -count=1 -p 1 -timeout 600s > .tmp/source-document.txt 2>&1
```

## Acceptance

- Missing test data files are extracted and available
- `validation="strip"` tests pass (`instance of element(*, xs:untyped)` returns true)
- source_document failures drop from 16 to ≤5
