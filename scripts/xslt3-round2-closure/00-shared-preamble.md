# Shared Preamble — XSLT 3.0 Round 2 Closure

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

Round-2 review in `.worktrees/feat-xslt3` found these remaining blockers:

- `xsl:package` still rejected at compile entry
- package-family W3C cases still skipped as `no stylesheet`
- transform context still lacks collection / uri-collection resolver wiring
- document / merge / source-document loading still too filesystem-specific
- `xsl:evaluate` still misses required conformance behavior
- W3C XPath assertion helper still loses in-scope namespaces
- `xsl:analyze-string` still has semantic + error-code mismatches
- `document`, `key`, `select`, `number`, `format-number` still fail targeted buckets

## Hard Rules

### No Papering Over

- NEVER claim closure by changing claim text only
- NEVER call issue fixed if tests merely skip instead of run
- NEVER replace unsupported behavior with broader capability masking unless task explicitly says to downgrade claim
- NEVER stop after making review symptom disappear if root cause remains

### Package Item Is Special

- Previous run failed here
- Do NOT treat `element-available()` changes as package fix
- Do NOT treat `Skip: "no stylesheet"` as package fix
- Do NOT hand off package task while `xsl:package` root still rejected at compile entry
- If blocked, report exact blocker with file + line + failing test names. Do NOT claim success

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
4. Implement root-cause fix
5. Re-run exact failing test
6. Re-run category
7. Commit logical batch

### Step 3: Verify

- Run final category verification into `.tmp/final-results.txt`
- Record pass/fail/skip counts
- Call out which skips are spec-justified vs. still implementation gaps

### Step 4: Report

- Do NOT merge yourself
- Report branch, files touched, tests run, residual failures
- Explicitly state whether task removed implementation gap vs. only exposed a deeper blocker

## Generator / Harness Rules

- `tools/xslt3gen/main.go` MAY be modified
- Generated `xslt3/w3c_*_gen_test.go` MAY be regenerated
- NEVER hand-edit generated test files. Regenerate them
- If generator changes asset extraction → regenerate + verify copied files exist
- If assertion support changes → add runner support before unskipping tests
- If task depends on in-scope namespaces in assertions → fix helper, not expectations

## Command Rules

- Redirect all command output to `.tmp/`
- One command per Bash call
- No pipes, `&&`, `||`, `;`

## Key Files

| File | Purpose |
|------|---------|
| `xslt3/compile.go` | stylesheet root + top-level compile |
| `xslt3/compile_instructions.go` | instruction compile |
| `xslt3/compile_streaming.go` | merge / source-document compile |
| `xslt3/execute.go` | runtime state + XPath context wiring |
| `xslt3/execute_instructions.go` | instruction execution |
| `xslt3/execute_streaming.go` | merge / source-document runtime |
| `xslt3/functions.go` | XSLT functions + document/key helpers |
| `xslt3/keys.go` | key table construction |
| `xslt3/number_words.go` | language-specific numbering words |
| `xslt3/options.go` | transform configuration surface |
| `xslt3/w3c_helpers_test.go` | W3C runner + assertion support |
| `tools/xslt3gen/main.go` | W3C generator |
| `xpath3/` | collection resolver + dynamic XPath dependencies |

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
- Do NOT delete tests to make category pass
- Do NOT keep false success wording after discovering residual blocker
