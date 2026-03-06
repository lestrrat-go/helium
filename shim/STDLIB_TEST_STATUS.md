# Stdlib encoding/xml Test Compatibility Status

379 pass, 53 skip, 0 fail. Skipped tests are grouped by feature gap below.

Files: `atom_stdlib_test.go`, `marshal_stdlib_test.go`, `read_stdlib_test.go`, `xml_stdlib_test.go`

Difficulty: **L** = low (isolated change, <30 min), **M** = medium (multiple touch-points, ~1-3 hrs), **H** = hard (new subsystem or architectural change, >3 hrs)

---

## ~~XML Declaration Handling~~ ✅

- [x] `TestUnmarshalFeedStdlib` — full Atom feed with XML declaration at start fails with "XML declaration in middle of document"
- [x] `TestUnmarshalWithoutNameTypeStdlib` — same XML declaration tolerance issue

## ~~uintptr Unmarshal~~ ✅

- [x] `TestAllScalarsStdlib` — unmarshaling into `uintptr` field not supported

## ~~NewTokenDecoder Idempotency~~ ✅

- [x] `TestNewTokenDecoderIdempotentStdlib` — `NewTokenDecoder` does not detect underlying `*Decoder`

## ~~ProcInst Target Validation~~ ✅

- [x] `TestProcInstEncodeTokenStdlib` — ProcInst `xml` target allowed after other tokens

## ~~EncodeToken Pointer Type Rejection~~ ✅

- [x] `TestSimpleUseOfEncodeTokenStdlib` — pointer types not rejected by `EncodeToken`

## ~~Invalid InnerXML Type Handling~~ ✅

- [x] `TestInvalidInnerXMLTypeStdlib` — error message for invalid innerxml type differs

## ~~[]byte / Array Marshaling~~ ✅

- [x] **Marshal**: `[]byte` field via fmt instead of string (#16, #18)
- [x] **Marshal**: `[N]byte` array via fmt (#19)
- [x] **Marshal**: `[]int` slice not expanded to separate elements (#21)
- [x] **Marshal**: `[N]int` array not expanded (#22)

## ~~[]byte Nil vs Empty Slice (unmarshal)~~ ✅

- [x] **Unmarshal**: `[]byte` field initialized as empty slice instead of nil (#37)

## ~~Comment Trailing Dash Padding~~ ✅

- [x] **Marshal**: comment ending with `-` not padded with space (#36)

## ~~CharData Control Character Escaping~~ ✅

- [x] **Marshal**: newline in chardata not escaped to `&#xA;` (#94)

## ~~CDATA `]]>` Splitting~~ ✅

- [x] **Marshal**: CDATA `]]>` splitting not implemented (#99-102)

## ~~Empty Path Wrapper Element~~ ✅

- [x] **Marshal**: empty path wrapper element not emitted for nil/empty slices (#103)

## ~~XML Declaration Encoding Validation~~ ✅

- [x] `TestIssue12417Stdlib` — XML declaration encoding attribute parsing

## Invalid Element Name Error Messages [L-M]

SAX parser produces different error messages for malformed names like `<a:te:st>`. Fix: map SAX error strings to stdlib's exact phrasing or add pre-validation.

- [ ] `TestIssue20396Stdlib` — error messages for invalid element names differ

## ~~Invalid XML Name Validation (marshal)~~ ✅

- [x] `TestInvalidXMLNameStdlib` — invalid XML name validation not implemented

## ~~Encoder Close State~~ ✅

- [x] `TestCloseStdlib` — encoder state after `Close` not fully implemented

## ~~Bad Comment Type Error~~ ✅

- [x] **Marshal**: bad comment type error not returned (#115, #119)

## Nil Interface Omission in Special Fields [M] (partially done)

- [x] **Marshal**: nil interface chardata not omitted (#130)
- [x] **Marshal**: nil interface cdata not omitted (#141)
- [x] **Marshal**: nil interface innerxml not omitted (#151)
- [x] **Marshal**: nil interface element not omitted (#162)
- [x] **Marshal**: nil interface in any field not omitted (#181)
- [ ] **Marshal**: nil interface in path field not omitted (#55)
- [ ] **Marshal**: nil pointer/interface omission with path merging (#57-60)

## ~~`any`-Tagged Field Element Naming~~ ✅

- [x] **Marshal**: any-tagged field uses type name instead of field name (#176-177, #183-184)
- [x] **Marshal**: any-tagged interface field uses empty element name (#179)
- [x] **Unmarshal**: direct any field not populated (#183, #191)

## ~~`[]xml.Attr` any,attr Support~~ ✅

- [x] **Marshal**: `[]xml.Attr` with `any,attr` tag not supported (#75-77)
- [x] **Unmarshal**: `any,attr` captures all attrs instead of only unmatched ones (#76)

## ~~xml.Name Field as Element Content~~ ✅

- [x] **Marshal**: xml.Name field as element content not handled (#70, #72)
- [x] **Unmarshal**: xml.Name field as element content not handled (#70-71)

## ~~XMLName Precedence~~ ✅

- [x] **Marshal**: XMLName precedence (value vs tag) differs (#68)
- [x] **Marshal**: embedded XMLName precedence differs (inner overrides outer) (#107)
- [x] **Unmarshal**: XMLName tag precedence differs (#69)
- [x] **Unmarshal**: embedded struct XMLName populated when should remain zero (#106, #108-109)
- [x] **Unmarshal**: outer element name mismatch for named embedded struct (#107)

## Embedded Struct Edge Cases (partially done)

Embedded field shadowing (path-tagged field vs plain field) and nil embedded pointer preservation differ from stdlib. Cross-depth shadowing and nil pointer preservation are now fixed. Remaining issue: path merging (#64 depends on Path Field Merging).

- [ ] **Marshal**: embedded field path conflict resolution differs (#64) — needs path merging
- [x] **Marshal**: embedded struct omitempty handling differs (#65)
- [ ] **Unmarshal**: embedded field path conflict resolution (#64) — needs path merging
- [x] **Unmarshal**: embedded struct pointer allocated when should remain nil (#65)

## ~~Indirect Pointer Handling (unmarshal)~~ ✅

- [x] **Unmarshal**: indirect comment pointer allocated when no comment present (#115-116)
- [x] **Unmarshal**: indirect innerxml pointer allocated when no content (#147-148)
- [x] **Unmarshal**: indirect any pointer allocated when no content (#176, #178, #185, #187)

## InnerXML Serialization Format [M]

The shim's innerxml capture serializes empty elements as self-closed (`<T1/>`) while stdlib preserves the original `<T1></T1>` form. Fix: adjust the innerxml capture to preserve the original serialization of empty elements.

- [ ] **Unmarshal**: innerxml captures self-closed tags instead of empty-element form (#154, #156)

## ~~TextMarshaler for time.Time~~ ✅

- [x] **Marshal**: time.Time TextMarshaler not invoked (#45)

## ~~Generic Type Name Brackets~~ ✅

- [x] **Marshal**: generic type name includes brackets (#47)

## ~~Interface Value Element Naming~~ ✅

- [x] **Marshal**: interface value defaultStart produces empty element name (#23)

## ~~Struct Pointer Marshal in any Slice~~ ✅

- [x] `TestStructPointerMarshalStdlib` — struct pointer marshal formatting

## ~~Marshal Error Handling~~ ✅

- [x] `TestMarshalErrorsStdlib` — chan type, comment `--`, attr path errors

## ~~Write Error Propagation~~ ✅

- [x] `TestMarshalWriteErrorsStdlib` — write error propagation
- [x] `TestMarshalWriteIOErrorsStdlib` — IO error propagation

## ~~Empty Element Value for Numeric Types~~ ✅

- [x] `TestUnmarshalEmptyValuesStdlib` — empty element value parsing for numeric types

## ~~Tag Path Conflict Detection~~ ✅

- [x] `TestUnmarshalBadPathsStdlib` — tag path conflict detection

## ~~interface{} Field Support (unmarshal)~~ ✅ (partial)

Stdlib leaves `interface{}` fields nil for comment/innerxml/element/omitempty/any bindings and returns "cannot unmarshal into interface {}" for chardata/cdata bindings. The shim now matches this behavior.

- [x] **Unmarshal**: interface{} comment fields (#118-120)
- [x] **Unmarshal**: interface{} chardata fields (#127-130)
- [x] **Unmarshal**: interface{} cdata fields (#138-141)
- [x] **Unmarshal**: interface{} innerxml fields (#150-152)
- [x] **Unmarshal**: interface{} element fields (#161)
- [x] **Unmarshal**: interface{} omitempty fields (#171)
- [x] **Unmarshal**: interface{} any fields (#180-182, #188-190)
- [x] `TestUnmarshalIntoInterfaceStdlib` — unmarshal into pre-populated `interface{}` field

## CharsetReader Support [M]

`Decoder.CharsetReader` callback is not wired up. Fix: intercept XML declaration encoding attribute, call `CharsetReader` to wrap the byte stream.

- [ ] `TestRawTokenAltEncodingStdlib` — CharsetReader callback
- [ ] `TestRawTokenAltEncodingNoConverterStdlib` — error when no CharsetReader and non-UTF-8 encoding

## ~~Decoder Nil Token Handling~~ ✅

- [x] `TestDecodeNilTokenStdlib` — error type for nil token reader

## ~~Decoder EarlyEOF Handling~~ ✅

- [x] `TestDecodeEOFStdlib` — earlyEOF / error type handling

## SyntaxError Line Number [M]

`SyntaxError.Line` reports the wrong line number. The SAX parser reports line positions differently from stdlib. Fix: reconcile SAX error positions with stdlib semantics.

- [ ] `TestSyntaxErrorLineNumStdlib` — SyntaxError line tracking

## Disallowed Character Detection [M]

The SAX parser produces different error messages for illegal XML characters (control codes, surrogates, invalid entity refs). Fix: map SAX error strings to stdlib's exact wording for each class of violation.

- [ ] `TestDisallowedCharactersStdlib` — disallowed character detection

## TokenReader Infinite Recursion Guard [M]

A `TokenReader` that always returns `StartElement` causes infinite recursion in `populateElement`. Fix: add a depth/repetition limit in the token-consumption loop.

- [ ] `TestDecodeIntrinsicStdlib` — `TokenReader` always returning `StartElement` causes infinite recursion

## Namespace Prefix Allocation (marshal) [M]

The shim's prefix assignment for namespaced attributes uses different naming conventions than stdlib (which derives prefix from the last URI path segment, deduplicates with `_1` suffixes). Fix: align the prefix allocation algorithm.

- [ ] `TestMarshalNSAttrStdlib` — namespace prefix allocation

## Empty Namespace Override [M-H]

`xmlns=""` (explicitly clearing the default namespace) is not fully propagated through unmarshal/re-marshal. Fix: both the SAX namespace stack and marshal namespace-emission logic need to handle the empty-namespace case.

- [ ] `TestIssue7113Stdlib` — empty namespace override (`xmlns=""`)

## Parse Error Messages [M-H]

Specific malformed inputs produce error messages that don't match stdlib's exact substrings (e.g. `"unexpected end element </foo>"`, `"unsupported version \"1.1\""`). Fix: map SAX errors to stdlib phrasing; add specific pre-processing hooks for version check and UTF-8 errors.

- [ ] `TestParseErrorsStdlib` — parse error messages and detection

## InputPos Line/Column Tracking [M-H]

`dec.InputPos()` does not return the same `(line, col)` as stdlib after each token. The SAX parser provides positions but with different granularity (end-of-token vs start-of-next-token). Fix: accurate per-token byte-position bookkeeping mapping back to line/column.

- [ ] `TestInputPosStdlib` — InputPos line/column tracking

## Path Field Merging (marshal + unmarshal) [H]

Stdlib merges fields sharing a path prefix (e.g. `xml:"Items>item"` and `xml:"Items>item1"` share the `<Items>` wrapper). The shim emits/parses each path independently. Fix: during marshal, group path fields by shared ancestor and emit a single wrapper; during unmarshal, maintain a path-descent state machine that buffers descendant elements.

- [ ] **Marshal**: path field merging (#49-60, #103)
- [ ] **Marshal**: namespace in path tags (#61-63)
- [ ] **Unmarshal**: path field matching (#52-54, #56)
- [ ] **Unmarshal**: namespace-aware path fields (#61-62)
- [ ] **Unmarshal**: nested path slice only captures one element (#110-112)
- [ ] `TestUnmarshalPathsStdlib` — full path-based field matching

## EncodeToken Validation + NS Prefix Allocation [H]

Comprehensive `EncodeToken` test covering namespace prefix allocation, error detection (no name, mismatched tags, invalid directives), and prefix collision resolution. Fix: align the entire prefix-allocation algorithm and all error-checking paths with stdlib.

- [ ] `TestEncodeTokenStdlib` — EncodeToken validation and namespace prefix allocation

## Namespace-Aware Element/Attribute Matching (unmarshal) [H]

Struct fields tagged with namespace URIs (e.g. `xml:"http://www.w3.org/TR/html4/ table"`) must match against the resolved namespace of incoming elements/attributes. Fix: carry namespace context from the token stream and match the space portion of field tags.

- [ ] `TestUnmarshalNSStdlib` — namespace-aware element matching
- [ ] `TestUnmarshalNSAttrStdlib` — namespace-aware attribute matching

## RawToken Namespace Preservation [H]

`RawToken()` should preserve `xmlns:*` and `xmlns` attributes on `StartElement.Attr` with prefix in `Name.Space`. The SAX parser resolves namespaces before the shim can intercept them. Fix: intercept SAX callbacks at a lower level or run a parallel raw-byte parser.

- [ ] `TestRawTokenStdlib` — RawToken namespace handling

## Cooked Token xmlns Attribute Preservation [H]

`Token()` (namespace-aware) should still include `xmlns:*` and bare `xmlns` attributes in `StartElement.Attr`. Same SAX root cause as RawToken. Fix: capture namespace declaration events from SAX and reinsert them as attributes.

- [ ] `TestTokenUnmarshalStdlib` — cooked token namespace handling

## Directive / DOCTYPE Token Emission [H]

The SAX parser never emits `Directive` tokens for DOCTYPE/entity declarations. The internal DTD subset is consumed silently by libxml2. Fix: would require custom pre-parsing or a different event hook; libxml2's SAX does not expose these as accessible events.

- [ ] `TestDirectivesStdlib` — Directive tokens not emitted
- [ ] `TestDirectivesWithCommentsStdlib` — Directive tokens with embedded comments not emitted

## Round-Trip with Non-Strict Tokens [H]

Depends on both non-strict mode (trailing colon in attr name) and Directive token support. Fix: requires both features implemented.

- [ ] `TestRoundTripStdlib` — round-trip with trailing colon and directives

## Parser Depth Limit [H]

The helium SAX parser has no recursion depth limit. 17M nested `<a>` tags cause a fatal stack overflow. Fix: add a depth counter to the SAX parser or shim callbacks; requires changes to the underlying helium parser.

- [ ] `TestCVE202230633Stdlib` — stack overflow on deeply nested input

## Non-Strict / AutoClose Mode [H]

Not planned — the shim targets strict XML only. Would require replacing the SAX parser with a lenient tokenizer.

- [ ] `TestNonStrictRawTokenStdlib` — `Strict = false`
- [ ] `TestUnquotedAttrsStdlib` — `Strict = false`, unquoted attribute values
- [ ] `TestValuelessAttrsStdlib` — `Strict = false`, HTML-style valueless attributes
- [ ] `TestHTMLAutoCloseStdlib` — `AutoClose = HTMLAutoClose`, synthesized end elements
