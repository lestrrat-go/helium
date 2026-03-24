# Feature: package system closure — no escape hatch

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-package`

## Goal

Close round-2 issue #1 for real.

## Why This File Is Strict

Previous run appears to have taken partial steps without closing root cause.
That is NOT acceptable here.

This task is NOT complete if any of these remain true:

- `xsl:package` root still rejected at compile entry
- package-family tests still skip as `no stylesheet`
- package-family support is hidden only by capability-reporting changes
- review symptom disappears but package execution path still does not exist

## Current State

- compile entry rejects `xsl:package`
- package-family generated tests still contain empty `StylesheetPath`
- runner still skips empty `StylesheetPath`
- claim cannot honestly advance while package system remains absent

## Required Outcomes

### 1. Compiler Entry

- `xsl:package` root must no longer fail immediately at compile entry
- top-level package metadata must be parsed into stylesheet/package model
- imports / use-package relationships must have explicit runtime/compile representation

### 2. Package-Family Test Execution

- `package`, `use-package`, `accept`, `expose`, `override` W3C buckets must no longer skip as `no stylesheet`
- if generator cannot materialize needed package assets today → extend generator
- if runner cannot compile multi-module package layout today → extend runner/compiler

### 3. No Cosmetic Closure

- Do NOT stop at `element-available()` / `system-property()` edits
- Do NOT stop at better error message
- Do NOT stop at generator extraction if compiled package still fails immediately

### 4. Honest End State

Only two acceptable end states:

- package-family buckets run meaningfully with real package support
- OR you hit a precise blocker after implementing substantial package-path code and report exact blocking gap with failing tests

`Not acceptable`:

- package root still rejected
- package tests still skipped
- doc claims adjusted without runtime closure

## Key Files

- `xslt3/compile.go`
- `xslt3/compile_instructions.go`
- `xslt3/instruction.go`
- `xslt3/execute.go`
- `xslt3/functions.go`
- `tools/xslt3gen/main.go`
- `xslt3/w3c_helpers_test.go`
- `xslt3/w3c_decl_gen_test.go` via regeneration only

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(package|use_package|accept|expose|override)$' -v -count=1 -p 1 -timeout 600s > .tmp/package-system.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_(package|use_package|accept|expose|override)/' -v -count=1 -p 1 -timeout 600s > .tmp/package-system-subtests.txt 2>&1
```

## Acceptance

- compile entry no longer rejects `xsl:package`
- package-family tests are runnable, not skipped as `no stylesheet`
- package support is backed by code path, not by wording change
- final report explicitly states which package scenarios now pass
