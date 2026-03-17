# Feature: key context item + xsl:number interaction

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-key-context`

## Goal

Fix key-related failures where `xsl:for-each` over key results fails due to
context item type mismatches and `xsl:number` interaction bugs.

## Current State

- key-073/074/075: `for-each select="key(...)"` iterates over non-node items,
  then AVT evaluation inside the loop fails with "context item is absent" or
  type mismatch because AVTs try to use the context node
- key-034: `xsl:number` inside `for-each` with key-based grouping produces
  `number=""` for all items
- key-047: `xsl:number select` returns empty sequence in key context
- key-076: key lookup returns nothing for multi-document key match

## Required Outcomes

### 1. Context Item in for-each

When `xsl:for-each select="key('k', value)"` iterates over results, each item
becomes the context item. If the key returns attribute nodes or text nodes,
the context item is that node, not nil.

The bug is likely in how `execForEach` sets up the context — it may assume
element nodes only. Fix to handle any node type as context.

For AVTs evaluated inside `for-each`: the AVT evaluation calls
`newXPathContext(ec.contextNode)` which may be nil if the context item is an
attribute. Ensure attribute/text/PI nodes are valid context nodes.

Affected tests: key-073, key-074, key-075

### 2. xsl:number in Key Grouping

key-034 uses Muenchian grouping with `xsl:number` to count within groups.
The `number=""` output suggests `xsl:number level="single"` fails to find
matching ancestors/siblings when the current node is from a `key()` result set.

Check that `xsl:number` respects the original document position of nodes
returned by `key()`, not the position in the key result sequence.

Affected tests: key-034, key-047

### 3. Multi-Document Key Lookup

key-076 uses `key()` with a third argument (document node) to look up keys
in a specific document loaded via `document()`. The result is empty.

Check that the key table is built for the target document, not just the
source document. The `key()` function's third argument specifies the
document to search in — the key index must be built on demand for that document.

Affected tests: key-076

## Key Files

- `xslt3/execute_instructions.go` — `execForEach`, `execNumber`, `execValueOf`
- `xslt3/execute.go` — context node handling
- `xslt3/functions.go` — `key()` third argument handling
- `xslt3/keys.go` — per-document key table building

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_key/(key-034|key-047|key-073|key-074|key-075|key-076)$' -v -count=1 -p 1 -timeout 600s > .tmp/key-context.txt 2>&1
```

## Acceptance

- for-each over key results handles non-element context items
- xsl:number works correctly inside key-based grouping loops
- key() with third argument searches the specified document
- These 6 key tests pass or have explicitly justified residuals
