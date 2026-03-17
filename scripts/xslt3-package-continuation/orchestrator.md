# XSLT 3.0 Package System Continuation Orchestrator

Coordinate subagents closing remaining package-system gaps after round-2 delivered
basic `xsl:package` + `xsl:use-package` support.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

## Parent Worktree

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run package-family baseline from `feat-xslt3`.
Store logs in `.tmp/`.

```bash
go test ./xslt3/ -run 'TestW3C_(package|use_package|accept|expose|override)$' -v -count=1 -p 1 -timeout 600s > .tmp/pkg-baseline.txt 2>&1
```

Round-2 baseline: 73 pass / 251 fail / 2 skip out of 326 total.

## Subagent Definitions

| # | File | Branch | Area |
|---|------|--------|------|
| 1 | `01-visibility-enforcement.md` | `feat-xslt3-pkg-visibility` | accept + expose + visibility model |
| 2 | `02-override-replacement.md` | `feat-xslt3-pkg-override` | xsl:override + xsl:original |
| 3 | `03-version-and-chains.md` | `feat-xslt3-pkg-versions` | version matching + dependency chains |

## Launching

For each subagent:

```text
Read these two instruction files in order:
1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-package-continuation/00-shared-preamble.md
2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-package-continuation/NN-FEATURE.md
Follow instructions exactly.
```

## Merge Sequencing

Merges must be serialized:

1. `feat-xslt3-pkg-visibility`
2. `feat-xslt3-pkg-override`
3. `feat-xslt3-pkg-versions`

Reason:

- Visibility model must exist before override can enforce visibility constraints
- Override replacement must exist before version chains can test overridden packages
- Version matching is most independent but benefits from stable visibility + override

## Conflict Points

- `xslt3/compile.go` — touched by all three (mergePackageComponents, compileUsePackage)
- `xslt3/stylesheet.go` — touched by visibility + override (component metadata)
- `xslt3/execute.go` — touched by visibility (runtime visibility checks)
- `xslt3/compile_instructions.go` — touched by override (xsl:original)
- `xslt3/execute_instructions.go` — touched by override (xsl:original execution)

## Final Acceptance

Do NOT claim package system complete until ALL are true:

1. accept bucket passes or residuals are explicitly justified
2. expose bucket passes or residuals are explicitly justified
3. override bucket failures drop by at least 50%
4. use-package version-matching cases pass
5. No private component is accessible from a using stylesheet

## If Full Closure Is Not Reached

Use precise wording:

- `visibility enforcement still incomplete`
- `xsl:override component replacement still partial`
- `package version matching still missing`
- `package dependency chains still incomplete`
