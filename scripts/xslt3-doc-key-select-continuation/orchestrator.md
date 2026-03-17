# XSLT 3.0 Doc/Key/Select Continuation Orchestrator

Coordinate subagents closing remaining document/key/select gaps.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

## Parent Worktree

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run baseline from `feat-xslt3`.
Store logs in `.tmp/`.

```bash
go test ./xslt3/ -run 'TestW3C_(document|key|select)$' -v -count=1 -p 1 -timeout 600s > .tmp/dks-baseline.txt 2>&1
```

Post round-2 baseline: document 6 fail, key 13 fail, select 1 fail = 20 total.

## Subagent Definitions

| # | File | Branch | Area |
|---|------|--------|------|
| 1 | `01-key-typed-and-composite.md` | `feat-xslt3-key-typed-composite` | typed comparison + composite keys + namespace keys |
| 2 | `02-key-context-and-numbering.md` | `feat-xslt3-key-context` | for-each context + xsl:number + multi-doc key |
| 3 | `03-document-and-select.md` | `feat-xslt3-doc-select` | initial-mode harness + separator + select |

## Launching

For each subagent:

```text
Read these two instruction files in order:
1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-doc-key-select-continuation/00-shared-preamble.md
2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-doc-key-select-continuation/NN-FEATURE.md
Follow instructions exactly.
```

## Merge Sequencing

1. `feat-xslt3-key-typed-composite` — foundational key infrastructure
2. `feat-xslt3-key-context` — builds on key infrastructure
3. `feat-xslt3-doc-select` — independent (harness + compiler)

## Conflict Points

- `xslt3/keys.go` — touched by #1 and #2
- `xslt3/functions.go` — touched by #1 and #2
- `xslt3/execute_instructions.go` — touched by #2 and #3
- `xslt3/compile_instructions.go` — touched by #3
- `tools/xslt3gen/main.go` — touched by #3
- `xslt3/w3c_helpers_test.go` — touched by #3

## Final Acceptance

Do NOT claim doc/key/select complete until ALL are true:

1. Typed key comparison works (key-069 passes)
2. Composite keys work (key-093 passes)
3. for-each over key results handles non-element context items
4. document-0801/0802/0803 pass with initial-mode
5. select-7502b no longer errors
6. Combined failures drop from 20 to ≤6

## If Full Closure Is Not Reached

Use precise wording:

- `typed key comparison still incomplete`
- `composite key support still missing`
- `initial-mode harness still not wired`
- `xsl:number in key context still broken`
