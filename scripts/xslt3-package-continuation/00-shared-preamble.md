# Shared Preamble — XSLT 3.0 Package System Continuation

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

Round-2 package work delivered:

- `xsl:package` root accepted at compile entry
- `xsl:use-package` resolves and merges package components via `PackageResolver`
- Package metadata (name, version, declared-modes) parsed into `Stylesheet`
- Generator extracts `<package>` elements from W3C catalog
- 73/326 package-family tests passing

Remaining failures (251) break down into:

| Gap | Affected categories | Approx failures |
|-----|---------------------|---:|
| Visibility enforcement (public/private/final/abstract) | accept, expose, override | ~100 |
| `xsl:override` component replacement | override | ~80 |
| Package version matching (`package-version="*"`, ranges) | use-package | ~30 |
| Initial mode/template visibility errors (XTDE0040/0045) | package, use-package | ~20 |
| Package-to-package dependency chains | use-package | ~20 |

## Key Files

| File | Purpose |
|------|---------|
| `xslt3/compile.go` | package root compile, `compileUsePackage`, `mergePackageComponents` |
| `xslt3/stylesheet.go` | package fields, `usedPackages`, visibility metadata |
| `xslt3/options.go` | `PackageResolver` interface |
| `xslt3/execute.go` | runtime state, template/function dispatch |
| `xslt3/functions.go` | function-available, element-available |
| `xslt3/w3c_helpers_test.go` | `w3cPackageResolver`, test harness |
| `tools/xslt3gen/main.go` | package element extraction |

## Workflow: Fork -> Fix -> Verify -> Report

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

For each failure batch:

1. Run focused test subset into `.tmp/` — use `-run 'TestW3C_<category>/<test-name>'` for individual tests
2. Read failing stylesheet + source assets
3. Read relevant source
4. Implement root-cause fix
5. Re-run exact failing test
6. Re-run category
7. Commit logical batch

### Step 3: Verify

- Run final category verification into `.tmp/final-results.txt`
- Record pass/fail/skip counts
- Call out which failures are spec-justified vs. still implementation gaps

### Step 4: Report

- Do NOT merge yourself
- Report branch, files touched, tests run, residual failures
- Explicitly state which package scenarios now pass vs. which remain blocked

## Known Hang Warning

There is a known hang in `fn:distinct-values` during `xsl:value-of` for template `d-003`. Avoid running the full test suite. Use targeted `-run` patterns only.

## Command Rules

- Redirect all command output to `.tmp/`
- One command per Bash call
- No pipes, `&&`, `||`, `;`

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
- Do NOT run the full test suite (hang risk)
- Do NOT delete tests to make category pass
