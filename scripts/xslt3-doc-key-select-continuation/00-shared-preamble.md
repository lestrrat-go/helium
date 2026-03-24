# Shared Preamble — XSLT 3.0 Doc/Key/Select Continuation

Every subagent MUST follow these rules. No exceptions.

## Project Layout

- **Project root**: `/home/lestrrat/dev/src/github.com/lestrrat-go/helium`
- **Parent branch**: `feat-xslt3`
- **Parent worktree**: `.worktrees/feat-xslt3`
- **Your worktree**: `.worktrees/<your-branch>`

## Pre-Read Rules

Before writing ANY code, read:

- `~/.claude/docs/shell.md`
- `~/.claude/docs/go.md`
- `~/.claude/docs/git-operations.md`
- `~/.claude/docs/commit-messages.md`
- `CLAUDE.md` at worktree root

## Baseline Context

After round-2, the doc/key/select buckets have these remaining failures (23 total):

### document failures (6)

| Test | Error |
|------|-------|
| document-0801/0802/0803 | `document()` returns filenames not content — test needs initial-mode |
| document-0307 | whitespace: `truetrue` vs `true true` (missing separator) |
| document-2401 | requires `xsl:package` (blocked) |
| document-2402 | initial template "a" not found |

### key failures (13)

| Test | Error | Root Cause |
|------|-------|-----------|
| key-073, key-074, key-075 | AVT error: context item absent / type mismatch | `for-each` over non-node items, AVT needs atomized string context |
| key-076 | `<out/>` empty | key lookup returns nothing for multi-key match |
| key-034 | `number=""` for all | `xsl:number` inside `for-each` with `key()` grouping |
| key-047 | xsl:number select empty | key interaction with `xsl:number` |
| key-055 | wrong Muenchian grouping output | `generate-id()` inconsistency across key lookups |
| key-066 | missing footnote entries | key with multiple `use` expressions or complex match |
| key-069 | `<out/>` empty | typed key comparison (integer vs string) |
| key-087, key-090 | `<out/>` empty | namespace node keys — `helium.Walk` doesn't visit namespace nodes |
| key-093 | wrong output | composite key (`composite="yes"`) not implemented |
| key-096 | type error: sequence length > 1 | composite key with `tokenize()` |

### select failures (1)

| Test | Error |
|------|-------|
| select-7502b | XTSE0870: xsl:value-of must have select or content |

## Key Files

| File | Purpose |
|------|---------|
| `xslt3/keys.go` | key table construction + lookup |
| `xslt3/functions.go` | `key()`, `document()`, `generate-id()` |
| `xslt3/execute_instructions.go` | `xsl:number`, `xsl:value-of`, `xsl:for-each` |
| `xslt3/compile_instructions.go` | instruction compilation |
| `xslt3/execute.go` | template dispatch, mode handling |
| `xslt3/w3c_helpers_test.go` | test runner |
| `tools/xslt3gen/main.go` | test generator |

## Known Hang Warning

There is a known hang in `fn:distinct-values`. Do NOT run the full test suite. Use targeted `-run` patterns only.

## Workflow: Fork → Fix → Verify → Report

### Step 1: Create Worktree

Each command below is a separate Bash call:

```bash
cd /home/lestrrat/dev/src/github.com/lestrrat-go/helium
```
```bash
git worktree add .worktrees/<your-branch> -b <your-branch> feat-xslt3
```
```bash
cd .worktrees/<your-branch>
```
```bash
mkdir -p .tmp
```

### Step 2: Fix Target Area

1. Run focused test subset into `.tmp/`
2. Read failing stylesheet + source assets
3. Read relevant source
4. Implement root-cause fix
5. Re-run exact failing test
6. Re-run category
7. Commit logical batch

### Step 3: Verify + Report

- Run final verification into `.tmp/final-results.txt`
- Do NOT merge yourself
- Report branch, files touched, tests run, residual failures

## Command Rules

- Redirect all command output to `.tmp/`
- One command per Bash call
- No pipes, `&&`, `||`, `;`

## What NOT To Do

- Do NOT work in root checkout
- Do NOT push or merge
- Do NOT run the full test suite
- Do NOT delete tests to make a category pass
