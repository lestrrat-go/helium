# XSLT 3.0 Resolvers & Loading Continuation Orchestrator

Coordinate subagents closing remaining resolver/loading gaps after round-2
delivered collection resolver wiring and base URI fixes.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

## Parent Worktree

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run resolver/loading baseline from `feat-xslt3`.
Store logs in `.tmp/`.

```bash
go test ./xslt3/ -run 'TestW3C_(merge|source_document|document|xsl_document)$' -v -count=1 -p 1 -timeout 600s > .tmp/resolvers-baseline.txt 2>&1
```

Round-2 baseline: source_document 16 fail, merge 44 fail, document 6 fail, xsl_document 6 fail.

## Subagent Definitions

| # | File | Branch | Area |
|---|------|--------|------|
| 1 | `01-merge-collection-iteration.md` | `feat-xslt3-merge-collection` | merge collection iteration + error semantics |
| 2 | `02-source-document-assets-and-validation.md` | `feat-xslt3-source-doc-assets` | missing test data + validation modes |
| 3 | `03-document-harness-and-xsl-document.md` | `feat-xslt3-doc-harness` | initial-mode harness + content model |

## Launching

For each subagent:

```text
Read these two instruction files in order:
1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-resolvers-continuation/00-shared-preamble.md
2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-resolvers-continuation/NN-FEATURE.md
Follow instructions exactly.
```

## Merge Sequencing

Merges can be done in any order — these three areas are mostly independent:

1. `feat-xslt3-merge-collection` (merge execution)
2. `feat-xslt3-source-doc-assets` (source-document assets + validation)
3. `feat-xslt3-doc-harness` (document harness + xsl:document)

Only conflict point: `xslt3/w3c_helpers_test.go` may be touched by #3 (initial-mode).

## Conflict Points

- `xslt3/execute_streaming.go` — touched by #1 (merge) and #2 (source-document)
- `xslt3/compile_streaming.go` — touched by #1 and #2
- `xslt3/w3c_helpers_test.go` — touched by #3 (initial-mode field)
- `tools/xslt3gen/main.go` — touched by #2 (asset extraction) and #3 (initial-mode)

## Final Acceptance

Do NOT claim resolver/loading complete until ALL are true:

1. Collection-based merge tests process all documents in collection
2. Missing source-document test data files are extracted
3. document-0801/0802/0803 pass with initial-mode support
4. Combined failures across all 4 categories drop by at least 30

## If Full Closure Is Not Reached

Use precise wording:

- `merge collection iteration still incomplete`
- `source-document validation semantics still partial`
- `initial-mode harness support still missing`
- `xsl:document content model still incomplete`
