# Feature: dynamic evaluation + higher-order blockers

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-dynamic-eval-hof`

## Goal

Remove major unsupported XSLT 3.0 / XPath 3.1 families currently skipped as
`dynamic_evaluation` or `higher_order_functions`.

## Current State

- Generator marks `dynamic_evaluation` unsupported
- Generated evaluate tests are skipped
- `fn:transform()` returns not implemented
- Review found HOF skips still present in W3C generated tests

## Required Outcomes

### 1. `xsl:evaluate`

- Add compile support
- Add runtime expression compilation/evaluation
- Build dynamic context from current XSLT runtime state
- Support namespace + variable injection required by W3C evaluate tests

### 2. `fn:transform()`

- Implement orchestration over stylesheet compile + transform
- Return correct map structure only after basic behavior is correct
- Remove not-implemented path

### 3. Higher-Order Dependencies

Audit skipped tests caused by HOF absence:

- function items
- dynamic function call
- functions passed in maps/arrays where required

Implement minimum XPath-side support needed by XSLT buckets in this repo.

### 4. Truthfulness

- Remove skip reasons only after implementation exists
- Keep generator feature flags aligned with reality

## Key Files

- `xslt3/compile_instructions.go`
- `xslt3/execute_instructions.go`
- `xslt3/functions.go`
- `xpath3/functions_misc.go`
- `xpath3/`
- `tools/xslt3gen/main.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_evaluate$' -v -count=1 -p 1 -timeout 600s > .tmp/dynamic-evaluate.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_(maps|for_each_group|lre)$' -v -count=1 -p 1 -timeout 600s > .tmp/hof-dependent-buckets.txt 2>&1
```

## Acceptance

- `fn:transform()` no longer returns not implemented
- Evaluate tests stop being generator-skipped for `dynamic_evaluation`
- HOF-dependent skips removed only where real support exists
