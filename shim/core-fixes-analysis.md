# Core Fixes Needed to Unblock TestRawTokenStdlib / TestTokenStdlib

## 1. Namespace Prefix Redeclaration Rejection -- FIXED

Added `nsStack.LookupInTopN()` to check only the current element's bindings
for true duplicates. Ancestor bindings are valid prefix shadowing, not
duplicates. `ParseNsClean` still checks the full stack for redundancy.

Commit: `af10c84`

## 2. CharData Splitting Per Entity/Character Reference -- FIXED

Added shim-level merging in `decoder.go`. Consecutive non-CDATA CharData
events are coalesced into a single token. CDATA sections are kept separate
to match stdlib behavior (stdlib treats CDATA as a distinct token boundary).

## 3. Raw Token Prefix Handling -- FIXED

Raw tokens now put the namespace prefix in `Name.Space` instead of
concatenating it with the local name (`Name{Space: "tag", Local: "name"}`
instead of `Name{Local: "tag:name"}`).

## Remaining Blockers

### A. Attribute vs xmlns Source Order

SAX reports `namespaces` and `attrs` as separate lists. The shim always
prepends xmlns declarations before regular attributes. But stdlib preserves
source order -- e.g., `<outer foo:attr="value" xmlns:tag="ns4">` has the
regular attr first. Without source position info from SAX, the interleaved
order cannot be reconstructed.

This blocks both `TestRawTokenStdlib` and `TestTokenStdlib`.

### B. InputOffset Alignment

`advancePosition()` estimates byte sizes from expanded token content, but
the raw source bytes differ (entity references are shorter/longer than their
expansions). Accurate offset tracking would require the SAX parser to report
byte positions per callback, or a parallel byte-level scanner.

This blocks `TestRawTokenStdlib` (which checks `InputOffset()` per token).
`TestTokenStdlib` does not check offsets.
