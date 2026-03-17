# Shared Preamble — Read This First

Every subagent MUST follow these rules. No exceptions.

## Project Layout

- **Project root**: `/home/lestrrat/dev/src/github.com/lestrrat-go/helium`
- **Parent branch**: `feat-xslt3` (the branch you fork from and merge back to)
- **Parent worktree**: `.worktrees/feat-xslt3`
- **Your worktree**: `.worktrees/<your-branch>` (created by you in step 1)

## Pre-Read Rules

Before writing ANY code, read:

- `~/.claude/docs/shell.md` — one command per Bash call, no pipes, no `&&`
- `~/.claude/docs/go.md` — style, testing, `t.Context()`, `require` not `assert`
- `~/.claude/docs/git-operations.md` — worktrees, commit messages, prohibited flags
- `~/.claude/docs/commit-messages.md` — single line, imperative, lowercase, no attribution
- The project `CLAUDE.md` at the worktree root — pre-read doc table, cache maintenance

## Current Branch State

- `feat-xslt3` at commit `f626ae9f` (2026-03-15)
- 2165 pass / 747 fail / 481 skip across 46 categories
- Streaming infrastructure merged from `feat-streaming`
- `xsl:result-document` body redirects to secondary buffer (output discarded)
- `xsl:analyze-string` is a no-op stub (returns empty SequenceInst)
- `xsl:namespace-alias` is an ignored TODO
- `xsl:mode` partially implemented: `on-no-match`, `on-multiple-match`, conflict detection
- `xsl:iterate` partially implemented
- `xsl:merge` partially implemented (from streaming branch)
- `as` attribute type coercion NOT implemented on variables/functions/params

## Workflow: Fork → Fix → Verify → Report

### Step 1: Create Worktree

Each command below is a **separate Bash call**:

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

### Step 2: Fix Tests

Work through failures one by one:

1. Run the relevant test category to get current failures.
2. Pick the first failure.
3. Read the test stylesheet and source data.
4. Read the relevant source code.
5. Implement the fix.
6. `go build ./xslt3/` — must succeed with no errors.
7. Run the specific test — must pass.
8. Run the full category — must not regress.
9. `git add <specific files>` then `git commit -m '<message>'`.
10. Go to step 2.

**Redirect all command output to `.tmp/` files.** Never pipe output.

**One command per Bash call.** No `&&`, `||`, `;`, pipes.

**Commit after each logical batch.** Don't accumulate large uncommitted diffs.

### Step 3: Verify — Full Category Run

After all fixes, run your test categories one final time and record
pass/fail counts. Save to `.tmp/final-results.txt`.

### Step 4: Report

Do NOT merge yourself. Report your results back. The orchestrator
merges into `feat-xslt3` in a controlled sequence.

## Key Source Files

| File | Purpose |
|------|---------|
| `xslt3/compile.go` | Stylesheet compiler, top-level declarations |
| `xslt3/compile_instructions.go` | Instruction compiler |
| `xslt3/compile_patterns.go` | Pattern matching + priority |
| `xslt3/execute.go` | Runtime state, template application, built-in rules |
| `xslt3/execute_instructions.go` | Instruction execution |
| `xslt3/functions.go` | XSLT-specific functions |
| `xslt3/instruction.go` | Instruction AST types |
| `xslt3/stylesheet.go` | Compiled stylesheet structure |
| `xslt3/errors.go` | Error codes and constructors |
| `xslt3/output.go` | Serialization / output methods |
| `xslt3/keys.go` | Key table building and lookup |
| `xslt3/avt.go` | Attribute value template compilation and evaluation |
| `xslt3/streamability_analysis.go` | Update if adding new instruction types |
| `xpath3/` | XPath 3.1 evaluator (separate package) |
| `xslt3/w3c_helpers_test.go` | Test runner and assertion helpers |
| `xslt3/w3c_*_gen_test.go` | Generated test files — DO NOT EDIT |
| `tools/xslt3gen/main.go` | W3C test code generator — DO NOT MODIFY |

## Test Commands

Each command is a separate Bash call. Redirect output to `.tmp/`.

```bash
go test ./xslt3/ -run 'TestW3C_<category>/<test-name>$' -v -count=1 2>&1 > .tmp/<test>.txt
```

```bash
go test ./xslt3/ -run 'TestW3C_<category>$' -v -count=1 2>&1 > .tmp/<category>.txt
```

To count results, use the Grep tool on the output file (do NOT use `grep`
in Bash):

- Pattern `--- PASS: TestW3C_` with `output_mode: "count"`
- Pattern `--- FAIL: TestW3C_` with `output_mode: "count"`

## What NOT To Do

- Do NOT edit `w3c_*_gen_test.go` files.
- Do NOT modify `tools/xslt3gen/main.go`.
- Do NOT modify tests to make them pass — fix the implementation.
- Do NOT introduce panics — always return errors.
- Do NOT work in the root checkout or in another subagent's worktree.
- Do NOT use `git push`.
- Do NOT merge to `feat-xslt3` yourself.
