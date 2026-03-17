# Feature: transform resolvers + URI loading closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-round2-resolvers`

## Goal

Close round-2 issues #2 + #3:

- transform context missing collection / uri-collection resolver wiring
- URI/document loading still too filesystem-specific

## Current State

- `xslt3` transform config lacks collection-resolver surface
- XPath collection support exists lower in `xpath3` but is not wired through transform execution
- `document()`, `doc()`, `xsl:source-document`, and `xsl:merge` load via direct filesystem joins/reads
- relative URI handling still produces wrong paths in some W3C cases

## Required Outcomes

### 1. Transform Resolver Surface

- add transform-time resolver hooks in `xslt3/options.go`
- wire them into `newXPathContext()` + other dynamic XPath contexts
- support both `fn:collection()` and `fn:uri-collection()`

### 2. Unified URI Loading

- stop duplicating ad-hoc file loading paths where practical
- use consistent base-URI resolution rules across:
  - `document()`
  - `doc()`
  - `xsl:source-document`
  - `xsl:merge`
  - dynamic evaluation paths that load external resources

### 3. Base URI Correctness

- fix doubled relative-path cases
- resolve against correct module/document base
- preserve document URL/base for downstream lookups

### 4. Test Harness Integration

- if W3C runner must provide resolver/resource-map setup for these buckets, add it
- do NOT accept `no collection resolver configured` as residual once resolver plumbing exists

## Key Files

- `xslt3/options.go`
- `xslt3/execute.go`
- `xslt3/functions.go`
- `xslt3/execute_streaming.go`
- `xslt3/w3c_helpers_test.go`
- `xpath3/context.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(merge|source_document|xsl_document|document)$' -v -count=1 -p 1 -timeout 600s > .tmp/resolvers-loading.txt 2>&1
```

## Acceptance

- targeted buckets no longer fail with missing collection resolver
- known doubled-path load failures are gone
- URI-loading behavior uses correct base URI semantics across affected instructions/functions
