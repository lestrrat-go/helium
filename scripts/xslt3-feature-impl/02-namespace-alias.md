# Feature: xsl:namespace-alias

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-namespace-alias`

## Goal

Implement `xsl:namespace-alias` — 25 failing tests in `TestW3C_namespace_alias$`.

## XSLT 3.0 Specification

`xsl:namespace-alias` is a top-level declaration that maps one namespace to
another in the output. It allows stylesheets to generate XSLT output (or
other namespace-dependent output) by using alias prefixes in LREs.

```xml
<xsl:namespace-alias stylesheet-prefix="axsl" result-prefix="xsl"/>
```

This means: wherever `axsl:` appears in LREs in the stylesheet, replace it
with `xsl:` in the output. The namespace URI bound to `axsl` in the
stylesheet is replaced by the URI bound to `xsl`.

Special values:
- `stylesheet-prefix="#default"` — the default namespace in the stylesheet
- `result-prefix="#default"` — the default namespace in the output

## Current State

`compile.go` line 353: `xsl:namespace-alias` is an ignored TODO. All 25
tests fail because namespace aliasing never happens.

## Implementation Plan

### 1. Data Model

In `stylesheet.go`, add:

```go
type NamespaceAlias struct {
    StylesheetURI string // namespace URI used in stylesheet
    ResultURI     string // namespace URI to use in output
    ResultPrefix  string // preferred prefix for the result namespace
}
```

Add to `Stylesheet`:
```go
namespaceAliases []NamespaceAlias
```

### 2. Compile

In `compile.go`, replace the TODO with `compileNamespaceAlias(elem)`:
- Read `stylesheet-prefix` and `result-prefix` attributes
- Resolve `#default` to `""` (default namespace)
- Look up the actual namespace URIs for both prefixes
- Store in `stylesheet.namespaceAliases`

### 3. Apply During Serialization

The namespace aliasing must happen when LREs are compiled or when
elements are serialized. Two approaches:

**Approach A (compile-time):** When compiling `LiteralResultElement`,
check if its namespace matches any alias's `StylesheetURI` and replace
with `ResultURI`. Also check all literal attributes.

**Approach B (runtime):** When adding nodes to the result tree, check
if the element/attribute namespace matches an alias and remap.

**Recommended: Approach A** — it's simpler and more correct. Compile-time
aliasing means the instruction tree already has the correct namespaces.

### 4. Handle Namespace Declarations

When a LRE like `<axsl:template>` is aliased to `<xsl:template>`:
- The element's namespace URI changes
- The `xmlns:axsl="..."` declaration must become `xmlns:xsl="..."`
- Any namespace declarations on the LRE that match stylesheet-prefix
  need to be remapped to result-prefix

## Key Files

- `xslt3/compile.go` — parse `xsl:namespace-alias`, store aliases
- `xslt3/compile_instructions.go` — apply aliases in `compileLiteralResultElement()`
- `xslt3/stylesheet.go` — `NamespaceAlias` type, `namespaceAliases` field

## Conflict Avoidance

Changes are isolated to namespace-alias-specific code. The LRE compiler
changes are additions (checking aliases before outputting), not
restructuring.

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_namespace_alias$' -v -count=1 2>&1 > .tmp/namespace-alias-final.txt
```
