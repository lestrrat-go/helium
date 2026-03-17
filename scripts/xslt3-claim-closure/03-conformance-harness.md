# Feature: W3C conformance harness closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-conformance-harness`

## Goal

Remove harness limitations that currently make conformance results incomplete
even when runtime code is correct.

## Current State

- Generator skips embedded-stylesheet tests as `no stylesheet`
- Runner skips any case with empty `StylesheetPath`
- `assert-result-document` unsupported
- `assert-serialization` unsupported
- Some source assets still missing in copied testdata

## Required Outcomes

### 1. Embedded Stylesheet Extraction

- Extend generator to materialize single-stylesheet cases from catalog content
- Copy extracted stylesheet into repo testdata
- Stop emitting `Skip: "no stylesheet"` for recoverable cases

### 2. Assertion Support

Implement runner support for:

- `assert-result-document`
- `assert-serialization`
- other currently downgraded assertion types if needed by this pack

Never convert supported assertions to `w3cAssertSkip()`.

### 3. Asset Copying

- Ensure transitive source assets required by `document()`, `xsl:source-document`, regex, and package tests are copied
- Fix known missing files discovered in review logs

### 4. Regeneration Discipline

- Modify `tools/xslt3gen/main.go`
- Regenerate `xslt3/w3c_*_gen_test.go`
- Do NOT hand-edit generated files

## Key Files

- `tools/xslt3gen/main.go`
- `xslt3/w3c_helpers_test.go`
- `xslt3/w3c_*_gen_test.go` via regeneration only
- `testdata/xslt30/testdata/`

## Verification

```bash
go run ./tools/xslt3gen > .tmp/xslt3gen.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_(result_document|source_document|package|use_package|evaluate)$' -v -count=1 -p 1 -timeout 600s > .tmp/harness-sensitive-buckets.txt 2>&1
```

## Acceptance

- Recoverable embedded-stylesheet tests no longer skip as `no stylesheet`
- Secondary-result and serialization assertions are checked
- Missing asset failures from review are eliminated or reduced to real implementation failures
