# Shared Preamble — XSLT 3.0 Claim Closure

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
- `~/.claude/docs/agent-instructions.md`
- `CLAUDE.md` at worktree root
- `docs/xslt3-major-spec-gaps-handoff.md`
- `docs/feat-xslt3-remaining-work-handoff.md`
- `.claude/docs/xslt3-schema-aware.md`
- `.claude/docs/streaming-handoff.md`

## Review Baseline

Review in `.worktrees/feat-xslt3` found these claim blockers:

- `xsl:package` rejected at compile entry
- `xsl:assert` unimplemented in instruction compiler
- `xsl:evaluate` unsupported
- `fn:transform()` returns not implemented
- `element-available()` + `system-property()` overclaim support
- W3C generator skips embedded-stylesheet cases as `no stylesheet`
- W3C generator skips `assert-result-document` / `assert-serialization`
- Major implemented families still fail targeted buckets

Targeted failing buckets from review:

- `type_available`, `system_property`, `namespace_alias`
- `analyze_string`, `iterate`, `merge`, `source_document`, `result_document`
- `import_schema`, `validation`
- `document`, `key`, `select`
- `number`, `format_number`

## Workflow: Fork → Fix → Verify → Report

### Step 1: Create Worktree

Each command below is separate Bash call:

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

For each failure batch:

1. Run focused test category into `.tmp/`
2. Read failing stylesheet + source assets
3. Read relevant source
4. Implement fix
5. Re-run exact failing test
6. Re-run category
7. Commit logical batch

### Step 3: Verify

- Run final category verification into `.tmp/final-results.txt`
- Record pass/fail/skip counts

### Step 4: Report

- Do NOT merge yourself
- Report branch, files touched, tests run, residual failures

## Generator / Harness Rules

- `tools/xslt3gen/main.go` MAY be modified in this instruction pack
- Generated `xslt3/w3c_*_gen_test.go` MAY be regenerated
- NEVER hand-edit generated test files. Regenerate them
- If generator changes asset extraction → regenerate + verify copied files exist
- If assertion support changes → add runner support before unskipping tests

## Command Rules

- Redirect all command output to `.tmp/`
- One command per Bash call
- No pipes, `&&`, `||`, `;`

## Key Files

| File | Purpose |
|------|---------|
| `xslt3/compile.go` | stylesheet root + top-level compile |
| `xslt3/compile_instructions.go` | instruction compile |
| `xslt3/compile_streaming.go` | iterate / merge / source-document compile |
| `xslt3/execute.go` | runtime state + capability wiring |
| `xslt3/execute_instructions.go` | instruction execution |
| `xslt3/execute_streaming.go` | iterate / merge / source-document runtime |
| `xslt3/functions.go` | XSLT functions + capability reporting |
| `xslt3/instruction.go` | instruction AST |
| `xslt3/output.go` | serialization + result-document |
| `xslt3/w3c_helpers_test.go` | W3C runner + assertion support |
| `tools/xslt3gen/main.go` | W3C generator |
| `xpath3/functions_misc.go` | `fn:transform()` |
| `xpath3/` | HOF / dynamic eval dependencies |

## Verification Commands

```bash
go test ./xslt3/ -run 'TestW3C_<category>$' -v -count=1 -p 1 -timeout 600s > .tmp/<category>.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_<category>/<test-name>$' -v -count=1 -p 1 -timeout 600s > .tmp/<test>.txt 2>&1
```

## What NOT To Do

- Do NOT work in root checkout
- Do NOT push
- Do NOT merge to `feat-xslt3` yourself
- Do NOT change expected behavior by deleting tests
- Do NOT keep false capability claims after discovering unsupported behavior
