# Feature: package, assert, truthful capability reporting

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-package-assert-capability`

## Goal

Close claim blockers caused by unsupported core features being reported as
available.

## Current State

- `xsl:package` root rejected
- `xsl:use-package` family therefore blocked
- `xsl:assert` not compiled
- `element-available()` returns true for unsupported elements
- `system-property()` overclaims support

## Required Outcomes

### 1. Package Truth

Choose one path. Do NOT leave ambiguous:

- Implement package system enough for W3C package buckets to run
- OR stop advertising package-related support everywhere

If not implementing full package support now:

- `element-available('xsl:package')` → false
- Same for `use-package`, `accept`, `expose`, `override`
- Any package-root stylesheet must fail clearly, not look partially supported

### 2. `xsl:assert`

- Add compile support
- Add runtime behavior
- Raise correct dynamic errors on failed assertions
- Make `element-available('xsl:assert')` truthful

### 3. `system-property()`

Audit at least:

- `xsl:is-schema-aware`
- `xsl:supports-backwards-compatibility`
- `xsl:supports-serialization`
- version / product fields if tests depend on them

Return `yes` only when implementation + harness semantics justify it.

### 4. `function-available()` / `element-available()`

- Never return true for intentionally unsupported features
- Keep version gating coherent with actual compile/runtime support

## Key Files

- `xslt3/compile.go`
- `xslt3/compile_instructions.go`
- `xslt3/execute_instructions.go`
- `xslt3/functions.go`
- `xslt3/errors.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(package|use_package|accept|expose|override|assert|system_property)$' -v -count=1 -p 1 -timeout 600s > .tmp/package-assert-capability.txt 2>&1
```

## Acceptance

- No false positive from capability functions for unsupported core features
- `TestW3C_assert` passes
- Package-related tests either run correctly or are rejected honestly by capability checks, not by misleading partial support
