# Missing Features for stdlib encoding/xml Compatibility

Current status: 381 pass, 49 skip, 0 fail.

Each section below is a self-contained feature description. They are ordered by
estimated impact (tests unlocked) and feasibility. Each includes: what the
feature does, which files to change, which tests to verify, concrete examples of
current vs expected behavior, and implementation guidance.

Worktree: `.worktrees/feat-encoding-xml-compat/`
Branch: `feat-encoding-xml-compat`

---

## 1. Path Field Merging (marshal + unmarshal)

**Tests unlocked**: ~27 (marshal 49-64, unmarshal 52-54, 56, 61-62, 64, 110-112, TestUnmarshalPathsStdlib)
**Difficulty**: High
**Files**: `shim/marshal.go`, `shim/unmarshal.go`

### Problem

Struct fields with path tags like `xml:"A>B"` and `xml:"A>C"` share the wrapper
element `<A>`. Stdlib groups them under a single `<A>` wrapper; the shim emits
each path independently, producing duplicate `<A>` wrappers.

### Current behavior (marshal)

```go
type T struct {
    B string `xml:"A>B"`
    C string `xml:"A>C"`
}
// shim produces:   <T><A><B>1</B></A><A><C>2</C></A></T>
// stdlib produces:  <T><A><B>1</B><C>2</C></A></T>
```

### Current behavior (unmarshal)

```xml
<T><A><B>1</B><C>2</C></A></T>
```
The shim's `findPath` function in `unmarshal.go:1145` walks `children` (direct
child elements) looking for an element matching `path[0]`, then descends into it
for `path[1:]`. It uses a flat consumed-set keyed by child index. When `B` is
found under `<A>` at child index 0, that `<A>` is marked consumed. Then `C`
(also under the same `<A>`) cannot find an unconsumed `<A>`, so it fails.

For slices (`xml:"A>B"` where B maps to `[]string`), the shim only captures the
first `<B>` inside the first unconsumed `<A>`, missing subsequent `<B>` elements
inside the same `<A>` (tests 110-112).

### Implementation guidance

**Marshal** (`shim/marshal.go`):

Replace the per-field `marshalPathField` call in the content-encoding loop
(line ~220) with a grouping pass:

1. Before iterating bindings, collect all path-tagged element bindings.
2. Group them by shared prefix. Two bindings share a prefix if
   `path[:n]` matches for some n < min(len(pathA), len(pathB))`.
3. Sort groups in binding-order (first occurrence).
4. For each group, emit the shared wrapper element(s) once, then marshal
   each field's remaining path suffix inside. Non-path fields are emitted
   between groups at their original position.

The key data structure is a prefix tree (trie) of path segments. Each leaf
holds the field binding. To emit: DFS-walk the trie, emitting
`StartElement` on enter and `EndElement` on leave.

Example grouping for fields `A>B`, `A>C`, `D`:
```
group 1: wrapper [A], leaves [B, C]
group 2: no wrapper, leaf [D]
```

The omitempty wrapper logic (lines 169-184) must integrate: emit wrapper
even when all leaves are omitempty-empty.

**Unmarshal** (`shim/unmarshal.go`):

Replace `findPath` (line 1145) with a path-aware matching strategy:

1. When a binding has `path = ["A", "B"]`, first find an unconsumed child
   `<A>`. But do NOT consume it (mark it in `consumed`) yet — multiple
   bindings may share the same `<A>`.
2. Track "claimed" wrapper elements: a wrapper `<A>` is fully consumed only
   after all bindings that reference it have been processed.
3. For slice bindings under a path (`xml:"A>B"` → `[]string`), iterate ALL
   `<B>` children inside each `<A>`, not just the first one.

Suggested approach: build a map from wrapper-element-name to the set of
bindings that share it. Process wrapper elements once, dispatching their
children to the appropriate bindings.

**Namespace path tags** (tests 61-63):

Tags like `xml:"space name>child"` put a namespace on the first path
segment. The `parseFieldBinding` function (line 1024) currently splits on
`>` before parsing namespace. It should:
1. Split on `>`
2. For the first segment, parse `"space name"` via `parseTagNameSpec`
3. Propagate the namespace to the wrapper `StartElement`

### Test verification

```sh
# After implementation, remove these from marshalSkipSet/unmarshalSkipSet:
# Marshal: 49-64
# Unmarshal: 52-54, 56, 61-62, 64, 110-112
go test -run 'TestMarshalStdlib/(49|5[0-9]|6[0-4])$' -v
go test -run 'TestUnmarshalStdlib/(5[2-4]|56|6[12]|64|11[012])$' -v
go test -run 'TestUnmarshalPathsStdlib' -v
```

---

## 2. ~~Namespace-Aware Element/Attribute Matching (unmarshal)~~ (DONE)

**Status**: Implemented. `buildElementFromTokens`/`populateElement` now
propagate namespace URIs to elements (via `SetActiveNamespace`) and attributes
(via `SetAttributeNS`). SAX emitter tracks in-scope namespace bindings for
resolving prefixed attributes from ancestor declarations.
TestUnmarshalNSStdlib and TestUnmarshalNSAttrStdlib pass.

~~**Tests unlocked**: 2 (TestUnmarshalNSStdlib, TestUnmarshalNSAttrStdlib)~~
~~**Difficulty**: Medium-High~~
**Files**: `shim/decoder.go`

### Problem

Struct field tags can specify namespace URIs: `xml:"http://example.com/ name"`.
During unmarshal, the shim must match incoming elements/attributes by resolved
namespace URI, not by prefix. The current `Unmarshal` path uses
`helium.Parse` which resolves namespaces in the DOM, so `elem.URI()` returns the
correct namespace. The `matchElementByTag` function (line 1192) already checks
`elem.URI()` against the tag's space when `hasSpace` is true. However, the
`Decoder.Decode` path builds elements via `buildElementFromTokens` (line 489)
which does NOT preserve namespace URIs on child elements or attributes.

### Current behavior

```go
type Tables struct {
    HTable string `xml:"http://www.w3.org/TR/html4/ table"`
    FTable string `xml:"http://www.w3schools.com/furniture table"`
}
```
Given `<Tables><table xmlns="http://www.w3.org/TR/html4/">hello</table>...</Tables>`:
- Via `Unmarshal()`: element `URI()` returns the namespace → should work
- Via `Decoder.Decode()`: `buildElementFromTokens` creates elements with
  `CreateElement(localName)` and `SetAttribute(localName, value)` — namespace
  information from `StartElement.Name.Space` is not propagated to the helium
  element, so `elem.URI()` returns `""` and matching fails.

For attributes (TestUnmarshalNSAttrStdlib), the same issue: `lookupAttr`
(line 1226) checks `attr.URI()` but the built elements don't have URI info.

### Implementation guidance

In `decoder.go`, `buildElementFromTokens` (line 489) and `populateElement`
(line 522):

1. When creating child elements from `StartElement` tokens with
   `v.Name.Space != ""`, set the element's namespace. Use helium's
   `CreateNamespace` + `SetAttributeNS` or find the appropriate API to set
   the element's namespace URI.
2. For attributes with `attr.Name.Space != ""`, use the namespace-aware
   attribute creation API instead of plain `SetAttribute`.

Additionally, `lookupAttr` in `unmarshal.go` (line 1226) must handle the
`Unmarshal()` path correctly. The helium DOM from `helium.Parse` resolves
prefixed attributes: `h:table` with `xmlns:h="..."` → `attr.URI()` returns
the namespace. Verify this works for both paths.

The `DefaultSpace` decoder field must also be applied: when
`d.DefaultSpace` is set, elements without an explicit namespace get that
default. `applyDefaultSpace` (decoder.go:401) already handles tokens, but
verify it flows through to built elements.

### Test verification

```sh
# Remove t.Skip from TestUnmarshalNSStdlib and TestUnmarshalNSAttrStdlib
go test -run 'TestUnmarshalNSStdlib' -v
go test -run 'TestUnmarshalNSAttrStdlib' -v
```

---

## 3. Namespace Prefix Allocation (marshal) ✅ DONE

**Tests unlocked**: 1 (TestMarshalNSAttrStdlib)
**Files**: `shim/namespace.go`, `shim/encoder.go`

### Solution

Rewrote `nsStack.allocPrefix` to derive prefixes from namespace URIs matching
stdlib's `createAttrPrefix` algorithm: last path segment, xml-prefix guard,
collision resolution with `_N` suffixes. The XML namespace
(`http://www.w3.org/XML/1998/namespace`) returns `"xml"` with no declaration.

Changed the encoder to interleave `xmlns:` declarations with attributes
(each declaration immediately before the first attribute that uses it),
matching stdlib's output ordering.

---

## 4. EncodeToken Validation and Namespace Prefix Allocation

**Tests unlocked**: 1 (TestEncodeTokenStdlib)
**Difficulty**: High
**Files**: `shim/encoder.go`, `shim/namespace.go`

### Problem

`EncodeToken` has multiple compliance gaps with stdlib:

1. **No-name rejection**: `EncodeToken(StartElement{})` with empty name should
   error. Currently it writes `<>`.
2. **Mismatched end element**: `EncodeToken(EndElement{Name: Name{Local: "a"}})`
   when the open tag was `<b>` should error with specific message.
3. **Invalid directive detection**: Some directive content should be rejected.
4. **Pointer type rejection**: `EncodeToken` should reject pointer types passed
   as tokens (e.g., `*StartElement`).
5. **Namespace prefix allocation**: Uses `ns1`/`ns2` instead of URI-derived
   prefixes (same as feature #3 above — fix #3 first).
6. **Prefix collision resolution**: Multiple attributes with different namespace
   URIs mapping to the same derived prefix need `_1` suffix deduplication.

### Implementation guidance

In `encoder.go`:

1. `writeStartElement` (line 81): Add validation at the top:
   ```go
   if se.Name.Local == "" {
       return fmt.Errorf("xml: start tag with no name")
   }
   ```

2. `writeEndElement` (line 171): Validate the end element matches the current
   open tag:
   ```go
   expected := enc.tags[len(enc.tags)-1]
   if ee.Name != expected {
       return fmt.Errorf("xml: end tag </%s> does not match start tag <%s>",
           ee.Name.Local, expected.Local)
   }
   ```
   Note: stdlib compares both Local and Space.

3. `EncodeToken` (line 43): Add type assertion check for pointer types before
   the switch:
   ```go
   if reflect.TypeOf(t).Kind() == reflect.Pointer {
       return fmt.Errorf("xml: EncodeToken of invalid token type")
   }
   ```
   (Import `reflect` for this.)

4. `writeDirective` (line 254): Add validation for balanced quotes and
   angle brackets within the directive content.

### Test verification

```sh
# Remove t.Skip from TestEncodeTokenStdlib
go test -run 'TestEncodeTokenStdlib' -v
```

Review `encodeTokenTestsStdlib` (in marshal_stdlib_test.go, search for
`var encodeTokenTestsStdlib`) to see all subtests and their expected
errors/output.

---

## 5. RawToken Namespace Preservation

**Tests unlocked**: 1 (TestRawTokenStdlib)
**Difficulty**: High
**Files**: `shim/decoder.go`

### Problem

`RawToken()` must preserve `xmlns:*` and `xmlns` attributes on `StartElement.Attr`.
Stdlib's `RawToken` returns raw prefix form (`foo:bar` not URI-resolved) AND
includes `xmlns:foo="..."` and `xmlns="..."` as regular attributes.

The shim's SAX parser resolves namespaces and does not emit namespace
declarations as attributes. The `OnStartElementNS` callback receives
`namespaces []sax.Namespace` separately from `attrs []sax.Attribute`, but
the shim only processes `attrs` — it ignores `namespaces` for both the
cooked and raw token variants.

### Expected behavior

Input: `<body xmlns:foo="ns1" xmlns="ns2">`

RawToken should return:
```go
StartElement{
    Name: Name{Local: "body"},
    Attr: []Attr{
        {Name: Name{Space: "xmlns", Local: "foo"}, Value: "ns1"},
        {Name: Name{Local: "xmlns"}, Value: "ns2"},
        // ... other regular attributes
    },
}
```

Currently the shim returns:
```go
StartElement{
    Name: Name{Local: "body"},
    Attr: []Attr{
        // only regular attributes, no xmlns declarations
    },
}
```

### Implementation guidance

In `decoder.go`, `OnStartElementNS` callback (line 97):

1. For the raw token variant (`rawSE`), convert `namespaces` to attributes:
   ```go
   for _, ns := range namespaces {
       if ns.Prefix() == "" {
           rawSE.Attr = append(rawSE.Attr, Attr{
               Name: Name{Local: "xmlns"}, Value: ns.URI(),
           })
       } else {
           rawSE.Attr = append(rawSE.Attr, Attr{
               Name: Name{Space: "xmlns", Local: ns.Prefix()}, Value: ns.URI(),
           })
       }
   }
   ```
2. Insert these BEFORE regular attributes to match stdlib ordering.
3. For raw element names, the prefix format (`prefix:local`) is already
   handled (line 113-116).

Note: `RawToken` also needs to emit `Directive` tokens for DOCTYPEs (see
feature #9). TestRawTokenStdlib expects both namespace attrs AND a DOCTYPE
`Directive` token. You may need to fix both to pass the test.

### Test verification

```sh
# Remove t.Skip from TestRawTokenStdlib
go test -run 'TestRawTokenStdlib' -v
```

---

## 6. Cooked Token xmlns Attribute Preservation

**Tests unlocked**: 1 (TestTokenUnmarshalerStdlib)
**Difficulty**: High (same root cause as #5)
**Files**: `shim/decoder.go`

### Problem

`Token()` (namespace-resolved) should ALSO include `xmlns:*` and `xmlns`
attributes in `StartElement.Attr`, but with namespace-resolved form. Stdlib's
`Token` returns:
```go
Attr{Name: Name{Space: "http://www.w3.org/2000/xmlns/", Local: "foo"}, Value: "ns1"}
```
for `xmlns:foo="ns1"`.

### Implementation guidance

Same approach as #5 but for the cooked token (`se`):

In `OnStartElementNS`, also add namespace declarations to `se.Attr`:
```go
for _, ns := range namespaces {
    if ns.Prefix() == "" {
        se.Attr = append(se.Attr, Attr{
            Name: Name{Space: "xmlns", Local: "xmlns"}, Value: ns.URI(),
        })
    } else {
        se.Attr = append(se.Attr, Attr{
            Name: Name{Space: "http://www.w3.org/2000/xmlns/", Local: ns.Prefix()}, Value: ns.URI(),
        })
    }
}
```

Check the exact `cookedTokensStdlib` in `xml_stdlib_test.go` to see the precise
`Name.Space` values stdlib uses for xmlns attributes.

### Test verification

```sh
# Remove t.Skip from TestTokenUnmarshalerStdlib
go test -run 'TestTokenUnmarshalerStdlib' -v
# Also: TestTokenStdlib (cooked token test)
go test -run 'TestTokenStdlib' -v
```

---

## 7. Empty Namespace Override

**Tests unlocked**: 1 (TestIssue7113Stdlib)
**Difficulty**: Medium-High
**Files**: `shim/unmarshal.go`, `shim/marshal.go`

### Problem

`xmlns=""` explicitly clears the default namespace. When unmarshaling:
```xml
<A xmlns="b"><C xmlns=""></C><d></d></A>
```
- `A.XMLName.Space` should be `"b"`
- `C.XMLName.Space` should be `""` (cleared by `xmlns=""`)
- `d` (no xmlns override) should inherit `"b"`

When re-marshaling, the output must preserve `xmlns=""` on `C`.

### Current behavior

The helium DOM from `helium.Parse` handles `xmlns=""` correctly at the DOM level
(elements inside `xmlns=""` get empty URI). The issue is in the marshal path:
when `XMLName.Space` is empty but the parent has a namespace, the encoder must
emit `xmlns=""` to explicitly clear the default namespace.

### Implementation guidance

**Unmarshal**: Likely works already via helium DOM. Test by removing the skip
and checking if unmarshal passes independently.

**Marshal** (`shim/encoder.go`):

In `writeStartElement` (line 81), after determining the element's namespace:
1. If the element's namespace is `""` but the parent scope has a default
   namespace binding, emit `xmlns=""` to clear it.
2. Track the current default namespace in the nsStack. The `resolve` method
   currently only resolves URI→prefix; add a `defaultURI()` method that
   returns the URI bound to prefix `""`.

### Test verification

```sh
# Remove t.Skip from TestIssue7113Stdlib
go test -run 'TestIssue7113Stdlib' -v
```

---

## 8. CharsetReader Support

**Tests unlocked**: 2 (TestRawTokenAltEncodingStdlib, TestRawTokenAltEncodingNoConverterStdlib)
**Difficulty**: Medium
**Files**: `shim/decoder.go`

### Problem

`Decoder.CharsetReader` is declared but not wired up. When the XML declaration
specifies a non-UTF-8 encoding (e.g., `<?xml version="1.0" encoding="x-testing-uppercase"?>`),
the decoder should:

1. If `CharsetReader` is non-nil, call it to get a converting reader, then
   feed that reader to the SAX parser instead of the original.
2. If `CharsetReader` is nil and encoding is not UTF-8, return an error.

### Current behavior

The SAX parser receives the raw reader. The `checkProcInstEncoding` method
(decoder.go:361) validates the encoding attribute but doesn't actually rewrap
the reader — it only checks after the fact. By then, the SAX parser has already
started reading the unconverted bytes.

### Implementation guidance

The challenge is that the SAX parser is started in a goroutine from
`startSAXEmitter` (line 78) which receives the original reader. The encoding
declaration is inside the XML data, so you need to:

**Option A**: Pre-scan the first few bytes of the reader for the XML declaration
before starting the SAX parser. If a non-UTF-8 encoding is found:
- If `CharsetReader` is set, wrap the reader
- If not, return an error immediately
Then pass the (potentially wrapped) reader to the SAX parser.

**Option B**: Use helium's SAX parser option to handle encoding. Check if
helium's parser has an encoding option or callback.

For Option A, in `newDecoderFromReader` (line 55):

```go
func newDecoderFromReader(r io.Reader) (*Decoder, error) {
    // Pre-read enough to detect XML declaration
    // Look for <?xml ... encoding="..." ?>
    // If non-UTF-8 encoding found and CharsetReader is nil → error
    // If CharsetReader is set → wrap reader
    // Pass (potentially wrapped) reader to startSAXEmitter
}
```

Problem: `CharsetReader` is set after `NewDecoder` returns. The stdlib pattern
is:
```go
d := NewDecoder(r)
d.CharsetReader = myFunc
d.Token() // charset reader used here on first token read
```

So the charset wrapping must happen lazily, on the first `Token()` call. This
means `startSAXEmitter` should be deferred until the first `readToken` call.
Refactor:

1. Store the reader in the Decoder struct
2. On first `readToken`, scan for encoding, apply CharsetReader if needed,
   then start the SAX emitter goroutine

### Test verification

```sh
# Remove t.Skip from TestRawTokenAltEncodingStdlib and TestRawTokenAltEncodingNoConverterStdlib
go test -run 'TestRawTokenAltEncoding' -v
```

Note: TestRawTokenAltEncodingStdlib also uses `testRawTokenStdlib` which checks
`InputOffset`. This may require accurate offset tracking as well.

---

## 9. Directive / DOCTYPE Token Emission ✅ DONE

**Tests unlocked**: 2 (TestNestedDirectivesStdlib, TestDirectivesWithCommentsStdlib)
**Files**: `shim/directive.go` (new), `shim/decoder.go`

### Solution

Implemented a prolog pre-scanner (`scanProlog` in `directive.go`) that tokenizes
the entire XML prolog before handing input to the SAX parser. The SAX parser does
not emit any prolog tokens (ProcInst, CharData whitespace, Directive), so the
pre-scanner handles them all. The full input is replayed to SAX for semantic
processing (entity resolution, default attributes, etc.).

The directive scanning algorithm mirrors stdlib's `rawToken` method:
- Tracks quote state, nested `<! >` depth, and `[...]` internal subsets
- Handles `<!-- -->` comments inside directives (replaced with spaces)
- Uses a `goto handleB`-equivalent pattern for failed comment checks

For prolog-only inputs (no root element), the SAX parser is not started.

---

## 10. Error Message Alignment

**Tests unlocked**: ~5 (TestIssue20396Stdlib, TestSyntaxErrorLineNumStdlib, TestDisallowedCharactersStdlib, TestParseErrorsStdlib, TestSyntaxStdlib)
**Difficulty**: Medium
**Files**: `shim/compat_errors.go`, `shim/decoder.go`

### Problem

The SAX parser produces error messages with different phrasing than stdlib.
Tests check error messages via `strings.Contains` or exact equality.

### Specific mismatches

**TestIssue20396Stdlib** (read_stdlib_test.go:1115):
Input like `<a:te:st xmlns:a="abcd"/>` should produce:
`"XML syntax error on line 1: expected element name after <"`
The SAX parser produces a different error string.

**TestSyntaxErrorLineNumStdlib** (xml_stdlib_test.go:761):
The `SyntaxError.Line` field reports wrong line numbers. SAX parser positions
differ from stdlib's (end-of-token vs start-of-token).

**TestDisallowedCharactersStdlib** (xml_stdlib_test.go:825):
Illegal XML characters (control codes, surrogates) produce different error
messages. Stdlib says things like `"illegal character code U+0000"`.

**TestParseErrorsStdlib** (xml_stdlib_test.go:1329):
Multiple specific error substrings expected, e.g.:
- `"unexpected end element </foo>"`
- `"unsupported version \"1.1\""`

**TestSyntaxStdlib** (xml_stdlib_test.go:498):
Various malformed inputs must produce `*SyntaxError`. Some inputs may not
trigger errors from the SAX parser at all.

### Implementation guidance

In `compat_errors.go`, expand `convertParseError`:

1. Parse the helium error message string
2. Map known patterns to stdlib's exact error phrasing
3. For line numbers, the `pe.LineNumber` from helium may differ — check if
   helium reports end-of-token position vs stdlib's start-of-next-token

For pre-validation errors (like unsupported XML version), add checks in
`decoder.go` before or during SAX parsing.

Example mapping:
```go
// helium: "xmlParseStartTag: invalid element name"
// stdlib: "expected element name after <"
```

Build a mapping table and apply it in `convertParseError`. For each test,
run it without skip, capture the actual error, and map it.

### Test verification

```sh
go test -run 'TestIssue20396Stdlib' -v
go test -run 'TestSyntaxErrorLineNumStdlib' -v
go test -run 'TestDisallowedCharactersStdlib' -v
go test -run 'TestParseErrorsStdlib' -v
go test -run 'TestSyntaxStdlib' -v
```

---

## 11. ~~InputPos Line/Column Tracking~~ ✅ DONE

**Status**: Fixed. The CDATA SAX callback was firing before consuming the
`]]>` delimiter, causing the document locator to report the wrong position.
Refactored `parseCDSect()` to consume `]]>` before the SAX callback, matching
libxml2 behavior. `TestInputLinePosStdlib` now passes.

---

## 12. InnerXML Serialization Format

**Tests unlocked**: 1 (unmarshal test 156 is blocked by same-root issue as 154)
**Difficulty**: Medium
**Files**: `shim/unmarshal.go`

### Problem

The `innerXML` function (unmarshal.go:1098) serializes child nodes via
`helium.NewWriter`. The writer self-closes empty elements (`<T1/>`) while
stdlib preserves the original form (`<T1></T1>` stays as-is, `<hi/>` stays
as-is). The DOM does not track whether an element was originally self-closed.

### Current behavior

Input: `<Outer><T1></T1><hi/><T2></T2></Outer>`
- shim innerxml: `<T1/><hi/><T2/>`  (all self-closed)
- stdlib innerxml: `<T1></T1><hi/><T2></T2>`  (original form preserved)

### Why it's hard

The helium DOM represents both `<T1></T1>` and `<T1/>` as identical empty
element nodes. The original serialization form is lost.

### Implementation guidance

The only correct fix is to capture the raw input bytes instead of
re-serializing from the DOM. This requires:

1. Thread the raw `[]byte` input through `Unmarshal` → `decodeElementInto`
2. Determine byte offsets for each element's inner content region (between
   the end of the start tag and the start of the end tag)
3. For innerxml bindings, slice the raw bytes instead of re-serializing

This is complex because helium nodes don't carry byte offsets. Possible
approaches:
- Add byte offset tracking to helium's DOM nodes (requires changes to the
  helium core library)
- Build a parallel byte-offset index by scanning the raw input for element
  boundaries after parsing (fragile)
- Accept the limitation and document which tests remain skipped

### Test verification

```sh
# Unmarshal tests 154 and 156 in marshalSkipSet
go test -run 'TestUnmarshalStdlib/15[46]$' -v
```

---

## 13. TokenReader Infinite Recursion Guard

**Tests unlocked**: 1 (TestDecodeIntrinsicStdlib — not yet in test files, needs to be added)
**Difficulty**: Medium
**Files**: `shim/decoder.go`

### Problem

A `TokenReader` that always returns `StartElement` without returning
`EndElement` causes infinite recursion in `populateElement` (decoder.go:522).
Stdlib has a depth limit to prevent stack overflow.

### Implementation guidance

Add a depth counter to `populateElement`:

```go
const maxNestingDepth = 10000 // match stdlib's limit

func (d *Decoder) populateElement(doc *helium.Document, parent *helium.Element, name Name) error {
    d.depth++
    if d.depth > maxNestingDepth {
        return errors.New("xml: exceeded max depth")
    }
    defer func() { d.depth-- }()
    // ... existing code
}
```

Note: the `depth` field on the Encoder struct tracks open tags for the
encoder. Add a separate `nestDepth` field to Decoder for this purpose.

### Test verification

Add `TestDecodeIntrinsicStdlib` to `xml_stdlib_test.go` (copy from stdlib's
`xml_test.go` TestDecodeIntrinsic). Then:
```sh
go test -run 'TestDecodeIntrinsicStdlib' -v -timeout 10s
```

---

## 14. ~~Parser Depth Limit~~ (DONE)

**Status**: Implemented. `Parser.SetMaxDepth()` added to core parser;
shim sets `maxParseDepth = 100_000`. TestCVE202230633Stdlib passes.

---

## 15. Non-Strict / AutoClose Mode (not planned)

**Tests unlocked**: 3 (TestNonStrictRawTokenStdlib, TestUnquotedAttrsStdlib, TestValuelessAttrsStdlib) + 1 (TestHTMLAutoCloseStdlib)
**Difficulty**: Very High
**Files**: would need a lenient tokenizer

These features require `Decoder.Strict = false` which enables lenient parsing
(unquoted attributes, valueless attributes, unknown entities as literal text).
This would require replacing or supplementing the libxml2 SAX parser with a
lenient tokenizer. Not planned for implementation.

---

## 16. Round-Trip with Non-Strict Tokens

**Tests unlocked**: 1 (TestRoundTripStdlib)
**Difficulty**: High (depends on #9 and #15)

Requires both Directive token support (feature #9) and non-strict mode
(feature #15, trailing colon in attribute names). Blocked until both are
implemented.

---

## Summary: Test Impact by Feature

| # | Feature | Tests | Feasibility |
|---|---------|-------|-------------|
| 1 | Path Field Merging | ~27 | High effort, high impact |
| 2 | ~~NS Element/Attr Matching~~ | ~~2~~ | **DONE** |
| 3 | NS Prefix Allocation | 1 | ✅ Done |
| 4 | EncodeToken Validation | 1 | High effort |
| 5 | RawToken NS Preservation | 1 | Medium effort (but needs #9 too) |
| 6 | Cooked Token xmlns | 1 | Medium effort (same as #5) |
| 7 | Empty NS Override | 1 | Medium effort |
| 8 | CharsetReader | 2 | Medium effort |
| 9 | Directive Tokens | 2 | ✅ Done |
| 10 | Error Messages | ~5 | Medium effort (tedious) |
| 11 | InputPos Tracking | 1 | Medium effort |
| 12 | InnerXML Format | 2 | Hard (needs core changes) |
| 13 | TokenReader Depth | 1 | Low effort |
| 14 | ~~Parser Depth Limit~~ | ~~1~~ | **DONE** |
| 15 | Non-Strict Mode | 4 | Not planned |
| 16 | Round-Trip | 1 | Blocked on #9, #15 |
