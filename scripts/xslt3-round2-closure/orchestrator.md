# XSLT 3.0 Round 2 Closure Orchestrator

Coordinate subagents closing remaining round-2 blockers to an honest
`feat-xslt3` handoff.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

## Parent Worktree

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run focused baseline buckets from `feat-xslt3`.
Store logs in `.tmp/`.

- package / package-family bucket
- resolver / loading bucket
- evaluate bucket
- analyze-string bucket
- document / key / select bucket
- number / format-number bucket

## Subagent Definitions

| # | File | Branch | Area |
|---|------|--------|------|
| 1 | `01-package-system-no-escape.md` | `feat-xslt3-round2-package` | package root + package-family harness closure |
| 2 | `02-transform-resolvers-and-loading.md` | `feat-xslt3-round2-resolvers` | collection resolver + URI loading + merge/source-document loading |
| 3 | `03-dynamic-evaluation-and-assert-harness.md` | `feat-xslt3-round2-evaluate` | `xsl:evaluate` conformance + W3C namespace assertion fix |
| 4 | `04-analyze-string.md` | `feat-xslt3-round2-analyze-string` | `xsl:analyze-string` semantics |
| 5 | `05-key-document-select.md` | `feat-xslt3-round2-doc-key-select` | `document`, `key`, `select` correctness |
| 6 | `06-numbering-and-formatting.md` | `feat-xslt3-round2-numbering` | `xsl:number` + `format-number()` residuals |

## Launching

For each subagent:

```text
Read these two instruction files in order:
1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-round2-closure/00-shared-preamble.md
2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-round2-closure/NN-FEATURE.md
Follow instructions exactly.
```

## Merge Sequencing

Merges must be serialized. Recommended order:

1. `feat-xslt3-round2-package`
2. `feat-xslt3-round2-resolvers`
3. `feat-xslt3-round2-doc-key-select`
4. `feat-xslt3-round2-analyze-string`
5. `feat-xslt3-round2-numbering`
6. `feat-xslt3-round2-evaluate`

Reason:

- package blocker invalidates any XSLT 3.0 claim immediately
- resolver wiring unlocks multiple runtime buckets
- document/key/select fixes reduce shared infrastructure noise
- evaluate last benefits from resolver + function-environment fixes

## Conflict Points

- `xslt3/compile.go` — package root work
- `xslt3/options.go` — transform resolver surface
- `xslt3/execute.go` — XPath context wiring
- `xslt3/functions.go` — document + key + dynamic function environment
- `xslt3/execute_instructions.go` — analyze-string + evaluate + numbering
- `xslt3/execute_streaming.go` — merge + source-document loading
- `xslt3/w3c_helpers_test.go` — assertion namespace fix
- `tools/xslt3gen/main.go` — package-family extraction if still skipped

## Final Acceptance

Do NOT say `implements XSLT 3.0` until ALL are true:

1. `xsl:package` no longer rejected at entry point
2. package-family W3C cases no longer skip as `no stylesheet`
3. transform-time collection + URI loading are wired through `xslt3`
4. `xsl:evaluate` bucket no longer fails on known round-2 conformance gaps
5. W3C assertion helper no longer loses in-scope namespaces
6. analyze-string, document/key/select, number/format-number targeted buckets are green or have explicitly justified residuals

## If Full Closure Is Not Reached

Do NOT soften blocker language.

Use precise wording:

- `package system still incomplete`
- `conformance harness still skips package-family cases`
- `dynamic evaluation still non-conformant`
- `document/key/select semantics still incomplete`
