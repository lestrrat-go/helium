# XSLT3 Feature Implementation Orchestrator

You are coordinating subagents implementing major XSLT 3.0 features in
parallel. Each subagent works in its own worktree branched off `feat-xslt3`.

## Project Root

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium`

The `feat-xslt3` worktree lives at:

`/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Baseline

Before launching subagents, run the full test suite from `feat-xslt3` to
establish baseline counts. Record in `.tmp/suite-pre-features.txt`.

Current approximate state: 2165 pass / 747 fail / 481 skip.

## Subagent Definitions

| # | File | Branch | Area | ~Fail |
|---|------|--------|------|-------|
| 1 | `01-analyze-string.md` | `feat-xslt3-analyze-string` | `xsl:analyze-string` | 55 |
| 2 | `02-namespace-alias.md` | `feat-xslt3-namespace-alias` | `xsl:namespace-alias` | 25 |
| 3 | `03-result-document.md` | `feat-xslt3-result-document` | `xsl:result-document` | 14 |
| 4 | `04-copy-namespaces.md` | `feat-xslt3-copy-namespaces` | Copy namespace scoping | 42 |
| 5 | `05-type-coercion.md` | `feat-xslt3-type-coercion` | `as` attribute type system | ~50 |
| 6 | `06-iterate-merge.md` | `feat-xslt3-iterate-merge` | `xsl:iterate` + `xsl:merge` | 85 |

## Launching

For each subagent:

```
prompt: |
  Read these two instruction files in order:
  1. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-feature-impl/00-shared-preamble.md
  2. /home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3/scripts/xslt3-feature-impl/NN-FEATURE.md
  Follow the instructions exactly.
```

Launch all 6 concurrently.

## Merge Sequencing

Merges must be serialized. Recommended order (least conflict first):

1. `feat-xslt3-namespace-alias` — isolated to compile.go namespace handling
2. `feat-xslt3-analyze-string` — new instruction, minimal overlap
3. `feat-xslt3-result-document` — extends existing stub
4. `feat-xslt3-copy-namespaces` — touches copy execution
5. `feat-xslt3-iterate-merge` — touches execute_instructions.go broadly
6. `feat-xslt3-type-coercion` — touches variable/param execution broadly

### Conflict Points

- `instruction.go` — all subagents adding types. Additive, easy to merge.
- `execute_instructions.go` — iterate/merge and copy-namespaces both edit
  this file but in different functions.
- `compile_instructions.go` — analyze-string and copy-namespaces both edit
  this file but in different functions.
- `streamability_analysis.go` — any subagent adding instruction types must
  update the switches.

## After All Subagents Complete

1. Merge each branch in sequence
2. Run the full test suite
3. Report combined results

```bash
go test ./xslt3/ -run 'TestW3C_(choose|copy|sort|number|variable|call_template|apply_templates|import|element|strip_space|param|message|attribute|lre|construct_node|output|include|template|for_each|for_each_group|sequence|try|document|key|expand_text|avt|mode|match|select|tunnel|xpath_default_namespace|format_number|format_date|format_date_en|expression|axes|predicate|path|nodetest|attribute_set|analyze_string|namespace_alias|result_document|iterate|merge|source_document|xsl_document|system_property|type_available)$' -v -count=1 -p 1 -timeout 600s 2>&1 > .tmp/suite-post-features.txt
```

## Features NOT Covered Here

These remain blocked and are not assigned to any subagent:

- **xsl:package** — major spec feature, needs design doc first
- **Schema-aware processing** — needs XSD type system integration
- **Higher-order functions** — needs XPath 3.1 HOF support
- **Dynamic evaluation** — `xsl:evaluate`, `fn:transform`
- **use-when** — compile-time conditional inclusion
- **Backwards compatibility** — XSLT 1.0 mode
