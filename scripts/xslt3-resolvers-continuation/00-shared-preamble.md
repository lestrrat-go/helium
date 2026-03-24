# Shared Preamble — XSLT 3.0 Resolvers & Loading Continuation

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

Round-2 resolvers work delivered:

- Collection/uri-collection resolver wired via `WithCollectionResolver` API
- Base URI resolution fixed for `xsl:source-document` and `xsl:merge` (xml:base-aware)
- Doubled-path bug in merge document loading fixed
- Missing `books.xml` test data for source-document tests copied

Post round-2 counts:

| Category | Pass | Fail | Skip |
|----------|------|------|------|
| source_document | ~64 | 16 | varies |
| merge | 54 | 44 | 1 |
| document | varies | 6 | varies |
| xsl_document | varies | 6 | varies |

### source_document remaining failures (16)

- 5 validation semantics (stream-011/013/014, non-stream-011/013): `element instance of element(*, xs:untyped)+` requires schema-aware typing
- 7 missing test data (stream-200/201/202/204/205/206/210/211/500, non-stream-200): files never extracted from W3C catalog
- 1 accumulator behavior (stream-208): requires accumulator integration
- remaining: streamability analysis gaps

### merge remaining failures (44)

- Collection-based merge tests (merge-001/039/085): `for-each-item` over collection documents only processes one document
- Many expected-error tests (XTDE2210, XTDE3490, XTTE1020, etc.) where processor doesn't raise required errors
- Several incorrect sort/merge output cases

### document remaining failures (6)

- document-0801/0802/0803: need initial-mode support in test harness
- document-2401: requires xsl:package
- document-2402: wrong initial template
- document-0307: whitespace in output

### xsl_document remaining failures (6)

- use-when not implemented (xsl-document-0606)
- comment content construction (xsl-document-0601)
- other content-model issues

## Key Files

| File | Purpose |
|------|---------|
| `xslt3/execute_streaming.go` | merge + source-document runtime |
| `xslt3/compile_streaming.go` | merge + source-document compile |
| `xslt3/functions.go` | document/doc loading, URI resolution helpers |
| `xslt3/options.go` | transform config, collection resolver |
| `xslt3/execute.go` | XPath context wiring |
| `xslt3/w3c_helpers_test.go` | collection resolver harness, test runner |
| `tools/xslt3gen/main.go` | test asset extraction |

## Known Hang Warning

Avoid running the full test suite. Use targeted `-run` patterns only.

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
