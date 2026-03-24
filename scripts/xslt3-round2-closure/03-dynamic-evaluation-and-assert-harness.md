# Feature: dynamic evaluation + W3C assertion namespace closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-evaluate`

## Goal

Close round-2 issues #4 + #5:

- `xsl:evaluate` still non-conformant
- W3C XPath assertion helper still loses in-scope namespaces

## Current State

- `xsl:evaluate` compile + runtime path exists
- targeted review still shows wrong success/error behavior
- dynamic environment still misses some function-resolution / context semantics
- W3C helper still raises false `XPST0081` failures for prefixed result-tree assertions

## Required Outcomes

### 1. `xsl:evaluate` Conformance

- fix known `XTDE3160` cases
- fix known `XTTE3165` cases
- ensure empty / invalid dynamic expression handling matches spec
- ensure namespace-context, xpath-default-namespace, base-uri, with-params, and context-item semantics are correct

### 2. Dynamic Function Environment

- dynamic expressions must see correct built-in + user-defined function environment
- investigate current unknown-function failures from review:
  - `parse#1`
  - `function1#0`
  - `abs#1`
  - `system-id#0`
- do NOT treat these as acceptable residuals without precise root cause

### 3. Ordering / Output Correctness

- fix known ordering mismatch in dynamic evaluation output
- verify map/param iteration behavior where review saw reversed order

### 4. W3C Assertion Namespace Fix

- helper must gather in-scope namespaces, not only local declarations on root
- eliminate false `XPST0081` assertion failures from prefixed checks

## Key Files

- `xslt3/compile_instructions.go`
- `xslt3/execute_instructions.go`
- `xslt3/execute.go`
- `xslt3/functions.go`
- `xslt3/w3c_helpers_test.go`
- `xpath3/` function registry files if needed

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_evaluate$' -v -count=1 -p 1 -timeout 600s > .tmp/evaluate.txt 2>&1
```

## Acceptance

- targeted evaluate bucket no longer fails on known round-2 conformance gaps
- prefixed W3C XPath assertions no longer fail because helper lost namespaces
- report any remaining evaluate failures by exact error code + root cause
