# Feature: merge collection iteration + error semantics

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-merge-collection`

## Goal

Fix `xsl:merge` to correctly iterate over collection-sourced document sequences
and raise required merge-specific errors.

## Current State

- Collection resolver is wired and returns document sequences
- But `for-each-item` over collection documents only processes the first document
- Many merge tests expect specific dynamic/static errors that are not raised
- Sort/merge output is wrong in several multi-source cases

## Required Outcomes

### 1. Collection Iteration

- `xsl:merge-source` with `select="collection('...')"` must iterate over ALL
  documents in the collection, not just the first
- Each document in the collection is a separate merge input
- Merge keys must be evaluated per-document per-item

### 2. Multi-Document Merge

- Fix merge algorithm to correctly interleave items from multiple source documents
- Preserve stable sort order within each source
- `current-merge-group()` and `current-merge-key()` must reflect the correct
  group across all sources

### 3. Expected Error Semantics

Fix these merge-specific errors:

- `XTDE2210`: merge input not sorted according to declared merge key
- `XTDE3490`: invalid merge-source attributes at runtime
- `XTTE1020`: type error in merge key comparison
- `XTTE2230`: merge key type mismatch
- `XTSE3195`: invalid merge-source configuration at compile time
- `XTSE0020`: invalid attribute on merge element
- `XTDE3362`: duplicate merge-source names

### 4. Sort Key Correctness

- Verify sort key evaluation uses correct collation
- Verify ascending/descending order is respected
- Verify data-type="number" vs. default string comparison

## Key Files

- `xslt3/execute_streaming.go` — merge execution, `executeMerge`, `loadMergeDocument`
- `xslt3/compile_streaming.go` — merge compilation, validation
- `xslt3/functions.go` — `current-merge-group()`, `current-merge-key()`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_merge$' -v -count=1 -p 1 -timeout 600s > .tmp/merge.txt 2>&1
```

## Acceptance

- Collection-based merge tests (001, 039, 085) process all documents
- Expected error tests raise correct error codes
- Multi-source merge output matches W3C expectations
- merge bucket failures drop from 44 to ≤20
