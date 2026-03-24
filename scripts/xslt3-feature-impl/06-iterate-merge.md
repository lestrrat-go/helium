# Feature: xsl:iterate and xsl:merge Fixes

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-iterate-merge`

## Goal

Fix remaining failures in `xsl:iterate` (23 tests) and `xsl:merge` (62 tests).
Both have partial implementations from the streaming branch merge.

## Current State

### xsl:iterate

`IterateInst`, `BreakInst`, `NextIterationInst` types exist in
`instruction.go`. The compiler and executor partially handle them.

Remaining issues:
- `xsl:on-completion` may not produce output correctly
- `xsl:break` with `select` may not return the right value
- `xsl:next-iteration` param passing may have scoping bugs
- Some tests reference documents via `fn:doc()` that need test data

### xsl:merge

`MergeInst`, `MergeSource`, `MergeKey` types exist. The streaming branch
added merge infrastructure. But many merge tests fail because:

- Non-streaming merge (no `streamable="yes"`) may not work correctly
- `merge-source` with `for-each-source` needs document loading
- `merge-key` ordering (ascending/descending) may not be applied
- `current-merge-group()` / `current-merge-key()` functions may be
  incomplete

## Strategy

Read `.claude/docs/streaming-handoff.md` for context on what the
streaming branch implemented.

### For iterate:
1. Run `TestW3C_iterate$` and categorize failures
2. Start with assert-xml failures (wrong output) — debug each one
3. Fix `xsl:on-completion` and `xsl:break` semantics
4. Fix `xsl:next-iteration` parameter passing

### For merge:
1. Run `TestW3C_merge$` and categorize failures
2. Focus on non-streaming merge tests first
3. Fix `merge-source` document loading
4. Fix `merge-key` comparison and ordering

## Key Files

- `xslt3/instruction.go` — `IterateInst`, `MergeInst` types
- `xslt3/compile_instructions.go` — `compileIterate()`, `compileMerge()`
- `xslt3/execute_instructions.go` — `execIterate()`, `execMerge()`
- `xslt3/functions.go` — `current-merge-group()`, `current-merge-key()`
- `.claude/docs/streaming-handoff.md` — streaming implementation notes

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_iterate$' -v -count=1 2>&1 > .tmp/iterate-final.txt
```

```bash
go test ./xslt3/ -run 'TestW3C_merge$' -v -count=1 2>&1 > .tmp/merge-final.txt
```
