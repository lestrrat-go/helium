# Feature: analyze-string semantic closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-analyze-string`

## Goal

Close round-2 issue #6:

- `xsl:analyze-string` still has semantic + error-code mismatches

## Current State

- instruction compiles + runs
- targeted review still shows wrong `XTDE1150` behavior
- matching/non-matching output still differs from W3C expectations in some cases
- context-item handling inside substring bodies is still suspect

## Required Outcomes

### 1. Error-Code Correctness

- zero-length-match behavior must match XSLT version rules
- invalid regex / flags must raise correct mapped errors
- do NOT leave review-known `XTDE1150` mismatches

### 2. Body Execution Semantics

- matching-substring body must see correct `regex-group()` values
- non-matching-substring body must see correct context item
- position/last semantics over produced segments must be correct

### 3. Output Correctness

- fix review cases that currently emit source text instead of transformed substring output
- preserve expected segment order exactly

## Key Files

- `xslt3/compile_instructions.go`
- `xslt3/execute_instructions.go`
- `xslt3/functions.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_analyze_string$' -v -count=1 -p 1 -timeout 600s > .tmp/analyze-string.txt 2>&1
```

## Acceptance

- targeted analyze-string bucket passes or residual failures are fully explained with exact root cause
- no remaining review-known `XTDE1150` mismatch
