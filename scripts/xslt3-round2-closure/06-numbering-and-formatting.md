# Feature: numbering + formatting closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-numbering`

## Goal

Close round-2 issue #8:

- `xsl:number` + `format-number()` still fail language/morphology buckets

## Current State

- targeted review still shows wrong German cardinal/ordinal forms
- some gender/case-specific ordinal outputs are still wrong
- targeted bucket still contains missing expected static/dynamic errors

## Required Outcomes

### 1. Word-Form Correctness

- fix German cardinal spacing/morphology where review still mismatches
- fix German ordinal morphology, including irregular forms like `dritte`
- verify compound-number ordinals such as `zweihunderterste`

### 2. Language / Ordinal Hints

- respect `lang` normalization
- respect ordinal hints such as masculine/feminine/neuter when required by tests
- do NOT collapse distinct ordinal forms to one generic suffix

### 3. Error Semantics

- fix remaining `XTSE0020`, `XTSE0870`, and other bucket-specific error mismatches
- verify compile-time vs. runtime error boundary

### 4. `format-number()` Stability

- keep grouping, separator, digit-system, and format-token behavior correct while fixing word forms

## Key Files

- `xslt3/execute_instructions.go`
- `xslt3/number_words.go`
- `xslt3/compile_instructions.go`
- `xslt3/functions.go` if decimal-format interactions are involved

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(number|format_number)$' -v -count=1 -p 1 -timeout 600s > .tmp/numbering-formatting.txt 2>&1
```

## Acceptance

- targeted `number` + `format-number` buckets no longer fail on review-known morphology/output issues
- final report names any still-failing language-specific cases explicitly
