# XSLT 3.0 Claim Closure Orchestrator

Coordinate subagents closing remaining blockers to an honest XSLT 3.0 claim.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

## Parent Worktree

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run focused baseline buckets from `feat-xslt3`.
Store logs in `.tmp/`.

- spec blockers / gaps bucket
- schema-aware bucket
- document / key / select bucket
- analyze-string / iterate / merge / result-document bucket
- number / format-number bucket

## Subagent Definitions

| # | File | Branch | Area |
|---|------|--------|------|
| 1 | `01-package-assert-capability.md` | `feat-xslt3-package-assert-capability` | package system + truthful capability reporting |
| 2 | `02-dynamic-evaluation-hof.md` | `feat-xslt3-dynamic-eval-hof` | `xsl:evaluate`, `fn:transform()`, HOF blockers |
| 3 | `03-conformance-harness.md` | `feat-xslt3-conformance-harness` | generator extraction + assertion support |
| 4 | `04-schema-aware-validation.md` | `feat-xslt3-schema-aware-validation` | schema-aware truthfulness + import-schema / validation |
| 5 | `05-data-access-and-loading.md` | `feat-xslt3-data-access-loading` | namespace-alias + source/document/key/select |
| 6 | `06-control-flow-and-secondary-results.md` | `feat-xslt3-control-flow-secondary` | analyze-string + iterate + merge + result-document |
| 7 | `07-numbering-and-formatting.md` | `feat-xslt3-numbering-formatting` | `xsl:number` + `format-number()` |

## Launching

For each subagent:

```text
Read these two instruction files in order:
1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-claim-closure/00-shared-preamble.md
2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-claim-closure/NN-FEATURE.md
Follow instructions exactly.
```

## Merge Sequencing

Merges must be serialized. Recommended order:

1. `feat-xslt3-package-assert-capability`
2. `feat-xslt3-schema-aware-validation`
3. `feat-xslt3-data-access-loading`
4. `feat-xslt3-control-flow-secondary`
5. `feat-xslt3-numbering-formatting`
6. `feat-xslt3-dynamic-eval-hof`
7. `feat-xslt3-conformance-harness`

Reason:

- Early merges remove false capability claims
- Mid merges close runtime semantic gaps
- Harness last → regenerate once after implementation stabilizes

## Conflict Points

- `xslt3/functions.go` — touched by capability, schema, dynamic-eval, merge work
- `xslt3/compile.go` — touched by package + schema work
- `xslt3/compile_instructions.go` — touched by assert + control-flow work
- `xslt3/execute_instructions.go` — touched by control-flow + data-access work
- `tools/xslt3gen/main.go` — harness work only
- `xslt3/w3c_helpers_test.go` — harness work only

## Final Acceptance

Do NOT say “implements XSLT 3.0” until ALL are true:

1. No hard unsupported core feature from this pack remains falsely advertised
2. No major W3C category in this pack is skipped by generator-only limitations
3. Capability-reporting functions align with actual support
4. Result-document / serialization assertions are checked, not skipped
5. Package + dynamic-evaluation stance is explicit and test-backed

## If Full Closure Is Not Reached

Downgrade claim text. Use precise language:

- `partial XSLT 3.0 implementation`
- `passes runnable subset`
- `does not yet implement package system / dynamic evaluation / ...`
