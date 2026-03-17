# Feature: document + key + select closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-doc-key-select`

## Goal

Close round-2 issue #7:

- `document`, `key`, `select` buckets still fail targeted review

## Current State

- `document()` / `doc()` still produce empty outputs or wrong AVT/type failures in some W3C cases
- `key()` still returns wrong or extra values in targeted bucket
- some tests still succeed/fail for wrong reason because lookup/scoping semantics are incomplete

## Required Outcomes

### 1. `document()` / `doc()` Correctness

- fix remaining empty-result cases
- fix AVT/type/context interactions seen in targeted failures
- ensure dedup + document-order semantics remain correct

### 2. `key()` Correctness

- verify root selection semantics, including 3rd-arg subtree behavior
- verify key-table construction for `use=` and sequence-valued results
- verify dedup + ordering semantics
- do NOT accept over-broad matches as residual

### 3. `select` Bucket Residuals

- fix known wrong-success / wrong-empty-output cases in targeted select bucket
- ensure expected static/dynamic errors fire when spec requires

## Key Files

- `xslt3/functions.go`
- `xslt3/keys.go`
- `xslt3/execute_instructions.go`
- `xslt3/compile.go`
- `xslt3/compile_patterns.go` if pattern/select interaction is implicated

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(document|key|select)$' -v -count=1 -p 1 -timeout 600s > .tmp/document-key-select.txt 2>&1
```

## Acceptance

- targeted `document`, `key`, `select` buckets no longer fail on review-known wrong outputs
- report any remaining failures with exact test names + why they still fail
