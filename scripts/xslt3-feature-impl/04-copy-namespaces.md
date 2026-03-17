# Feature: Copy Namespace Scoping

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-copy-namespaces`

## Goal

Fix namespace handling in `xsl:copy` and `xsl:copy-of` — 42 failing tests
in `TestW3C_copy$`.

## Problem Summary

The `copy` category has the most diverse failures. They break down into:

1. **Namespace in-scope preservation** (~15 tests) — when copying elements,
   the in-scope namespace bindings should match the source. Extra bindings
   from the stylesheet or result tree context leak into copied elements.

2. **`copy-namespaces` attribute** (~10 tests) — `copy-namespaces="no"` on
   `xsl:copy` / `xsl:copy-of` should suppress namespace node copying.
   Expected errors XTDE0420 for conflicts.

3. **`inherit-namespaces` attribute** (~5 tests) — `inherit-namespaces="no"`
   prevents child elements from inheriting parent's namespace bindings.

4. **Expected errors** (~12 tests) — XTSE0260 (invalid copy), XTSE0090,
   XTDE0420 (namespace conflict), XTTE0945 (copy of namespace nodes).

## Current State

`xsl:copy` and `xsl:copy-of` are implemented but don't handle:
- `copy-namespaces` attribute (always copies all namespaces)
- `inherit-namespaces` attribute (always inherits)
- Namespace stripping on the copied element
- Namespace conflict detection

## Implementation Plan

### 1. Parse New Attributes

In `compile_instructions.go`, update `compileCopy` and `compileCopyOf`
to parse:
- `copy-namespaces` — "yes" (default) or "no"
- `inherit-namespaces` — "yes" (default) or "no"

Add these fields to `CopyInst` and `CopyOfInst` in `instruction.go`.

### 2. Implement copy-namespaces="no"

When `copy-namespaces="no"`:
- Don't copy namespace nodes from the source element
- Only preserve namespace bindings that are actually used by the
  element's name and attribute names

### 3. Implement inherit-namespaces="no"

When `inherit-namespaces="no"` on `xsl:copy` / `xsl:element`:
- Child elements should not inherit namespace bindings from the parent
  in the result tree

### 4. Namespace Conflict Detection

When `xsl:copy` copies an element and the body adds attributes or
namespace bindings, check for conflicts:
- XTDE0420: attribute with namespace that conflicts with existing binding
- XTDE0430: attribute with duplicate expanded name

### 5. Fix In-Scope Namespace Leakage

The main issue: when `CopyNode()` copies an element from the source
tree into the result tree, the result tree's ambient namespace
declarations leak onto the copied element. Fix by:
- Explicitly setting only the source element's namespace declarations
  on the copied element
- Not inheriting the result tree parent's namespace declarations

## Key Files

- `xslt3/compile_instructions.go` — parse copy-namespaces, inherit-namespaces
- `xslt3/execute_instructions.go` — `execCopy()`, `execCopyOf()` namespace handling
- `xslt3/instruction.go` — add fields to `CopyInst`, `CopyOfInst`
- `helium` package — `CopyNode()` may need a namespace-aware variant

## Strategy

Start with the `copy-namespaces="no"` tests since they have the clearest
pass/fail signal. Then fix namespace leakage. Finally add error detection.

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_copy$' -v -count=1 2>&1 > .tmp/copy-final.txt
```
