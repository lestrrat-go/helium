# Feature: control flow, regex, secondary results

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-control-flow-secondary`

## Goal

Close major remaining semantic failures in already-implemented XSLT 3.0
instruction families.

## Current State

Review found failures in:

- `analyze_string`
- `iterate`
- `merge`
- `result_document`

Signals included:

- wrong regex-group / zero-length semantics
- iterate scoping + variable visibility bugs
- merge comparison / variable / grouping failures
- result-document partially passing but still hidden behind unsupported assertions

## Required Outcomes

### 1. `xsl:analyze-string`

- Fix residual regex semantics
- Fix expected error behavior
- Fix asset-loading cases used by analyze-string tests

### 2. `xsl:iterate`

- Fix parameter scope / visibility bugs
- Fix `xsl:break`, `xsl:next-iteration`, `xsl:on-completion`
- Verify no undefined-variable regressions remain

### 3. `xsl:merge`

- Fix merge-key type coercion and comparison
- Fix current-merge-group / key behavior
- Fix mixed-source scenarios failing in review

### 4. `xsl:result-document`

- Fix remaining runtime semantics
- Keep secondary result ownership separate from harness assertion support
- Verify no false passes hidden by skipped assertions

## Key Files

- `xslt3/compile_instructions.go`
- `xslt3/compile_streaming.go`
- `xslt3/execute_instructions.go`
- `xslt3/execute_streaming.go`
- `xslt3/functions.go`
- `xslt3/output.go`
- `xpath3/`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(analyze_string|iterate|merge|result_document)$' -v -count=1 -p 1 -timeout 600s > .tmp/control-flow-secondary.txt 2>&1
```

## Acceptance

- Buckets pass without relying on harness skips
- Review-log failures like undefined iterate vars or merge type mismatches are gone
