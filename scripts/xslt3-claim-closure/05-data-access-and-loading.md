# Feature: data access, loading, namespace alias

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-data-access-loading`

## Goal

Close buckets where document loading, key lookup, select semantics, or
namespace aliasing still fail.

## Current State

Review found failures in:

- `namespace_alias`
- `source_document`
- `xsl_document`
- `document`
- `key`
- `select`

Signals included:

- wrong namespace-alias output
- missing asset loads
- incorrect `key()` counts
- incorrect `select` static/dynamic errors

## Required Outcomes

### 1. Namespace Alias

- Fix result-namespace rewriting semantics
- Verify aliasing affects result tree, not stylesheet parsing

### 2. Source / Document Loading

- Fix URI resolution + asset availability
- Align `document()` and `xsl:source-document` behavior with copied test assets
- Ensure `xsl:document` behavior matches W3C expectations in this repo

### 3. `key()`

- Fix index construction / lookup semantics in failing W3C cases
- Verify multi-document + context interactions

### 4. `select`

- Fix missing static errors
- Fix empty or wrong output in failing select buckets

## Key Files

- `xslt3/compile.go`
- `xslt3/compile_streaming.go`
- `xslt3/execute.go`
- `xslt3/execute_streaming.go`
- `xslt3/functions.go`
- `xslt3/keys.go`
- `xslt3/output.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(namespace_alias|source_document|xsl_document|document|key|select)$' -v -count=1 -p 1 -timeout 600s > .tmp/data-access-loading.txt 2>&1
```

## Acceptance

- Bucket failures shift from asset/load bugs to real residual semantics only
- `namespace_alias` bucket passes
- `document`, `key`, `select` buckets no longer fail on issues seen in review logs
