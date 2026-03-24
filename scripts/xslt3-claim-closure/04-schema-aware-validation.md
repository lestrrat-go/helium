# Feature: schema-aware truthfulness + validation buckets

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-schema-aware-validation`

## Goal

Make schema-aware claims truthful and close the failing schema-aware buckets
that were hit in review.

## Current State

- `system-property('xsl:is-schema-aware')` returns `yes`
- Generator still skips many schema-aware cases
- `type-available()` still wrong for some built-in / schema-aware cases
- `import-schema` and `validation` buckets still fail

## Required Outcomes

### 1. Truthful Schema-Aware Reporting

Choose one path. Do NOT mix them:

- Implement enough schema-aware behavior to justify `yes`
- OR return `no` / stop advertising support until feature is complete

### 2. `type-available()`

- Fix built-in type answers
- Fix imported-schema type lookup
- Respect XSD version assumptions already documented by repo

### 3. `xsl:import-schema`

- Verify file-backed + inline schema cases
- Fix remaining compile/runtime mismatches in test bucket

### 4. Validation Gaps

- Fix failing `validation` W3C cases in scope
- Keep generator skips only for truly unsupported full-schema-validation cases
- Remove stale skips when implementation now covers them

## Key Files

- `xslt3/compile.go`
- `xslt3/functions.go`
- `xslt3/execute.go`
- `xpath3/`
- `tools/xslt3gen/main.go`
- `tools/xslt3gen/schema_known_failures.txt`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(type_available|import_schema|validation)$' -v -count=1 -p 1 -timeout 600s > .tmp/schema-aware-validation.txt 2>&1
```

## Acceptance

- `type-available()` answers align with repo's XSD 1.1 stance
- `is-schema-aware` reporting matches actual support
- Remaining schema-aware skips are explicit, justified, and minimized
