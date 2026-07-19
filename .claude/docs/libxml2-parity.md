# libxml2 Parity Status

## Test Results

### libxml2-compat golden tests

| Package | Tests | Pass | Skip | Rate | Skip Reasons |
|---------|-------|------|------|------|--------------|
| Core XML (DOM) | 150+ | all | 0 | ~100% | — |
| Core XML (SAX2) | 150+ | all | 0 | ~100% | — |
| C14N | 73 | 73 | 0 | 100% | — |
| XSD | 226 | 225 | 1 | 99.6% | libxml2 IDC quirk with ref + attributeFormDefault |
| RELAX NG | 159 | 159 | 0 | 100% | — |
| Schematron | 42 | 42 | 0 | 100% | — |
| HTML SAX | 47 | 47 | 0 | 100% | — |
| HTML Serialization | 47 | 47 | 0 | 100% | — |
| HTML Errors | 5 | 4 | 1 | 80% | encoding-error.html: byte-level context window |

Test data: `testdata/libxml2-compat/` (golden files generated from libxml2's xmllint).

### W3C QT3 tests (XPath 3.1)

The QT3 (XPath/XQuery 3.1) conformance suite (generator, harness, ~22k generated
cases, and curated fixtures) lives in the sibling `../helium-w3c-tests` module
under `internal/suites/qt3` and `xpath3/`; run it against this module with a
local `go.work` (`go run ./cmd/w3ctest qt3`). Persistent per-case skips are
tracked in `helium-w3c-tests/expectations/qt3.json`; treat any checked-in counts
there as current-state only and regenerate to confirm.

### W3C XSLT 3.0 tests

The XSLT 3.0 conformance suite (generator, harness, ~13k generated cases, and
curated fixtures) lives in the sibling module `../helium-w3c-tests` under
`internal/suites/xslt30` and `xslt3/`; run it against this module with a local
`go.work` pointing at the Helium worktree (`go run ./cmd/w3ctest xslt30`).
Structural spec/feature/streaming skips are computed at generation time;
persistent per-case skips are tracked in
`helium-w3c-tests/expectations/xslt30.json`. Treat any checked-in counts there as
current-state only and regenerate to confirm.

### W3C XSD 1.1 tests

The heavyweight XSD 1.1 conformance harness lives in the sibling module
`../helium-w3c-tests` on branch `migrate-xsd11`; run it against this module with
a local `go.work` pointing at the Helium worktree. Treat checked-in counts as
current-state only: regenerate them from the current `migrate-xsd11` branch and
the Helium branch under test. Do not preserve old branch/SHA counts in this
section unless the heading explicitly marks them as historical. Persistent skips
are tracked in `helium-w3c-tests/expectations/xsd11.json` as XSD 1.1 conformance
gaps.

Current state (snapshot, regenerate to confirm): the `xsd11.json` skip list is
**empty** — the Basic XSD 1.1 W3C suite passes **967 / 0 failures / 0 skipped**
against `feat-xsd11` with `helium-w3c-tests` `migrate-xsd11`.

## Parser Limitations

Cross-element redundant namespace redeclarations and entity references in
single-quoted attributes parse correctly, and the C14N suite runs with no skips.
Same-element duplicate namespace declarations are a well-formedness error, as in
libxml2.

1. **External entity resolution** — limited; requires explicit config. `NewParser` is **secure by default**: `BlockXXE` is on, the `FS` is a deny-all FS, and network is off — so external loading is blocked even when `LoadExternalDTD(true)`/`SubstituteEntities(true)` are set, until `BlockXXE(false)` is also set AND an FS is supplied (`helium.PermissiveFS()` or a confined `fs.FS`). External subsets need `LoadExternalDTD(true)`, and inline expansion of parsed external entities needs `SubstituteEntities(true)`. External DTD subsets are read through a strict byte cap (`MaxExternalDTDSize`, 10 MiB default; overridable via `MaxExternalDTDBytes`) enforced against the actual bytes read — not any advisory `Stat` size — and a subset exceeding the cap is rejected with `ErrExternalDTDTooLarge`. The `relaxng.Compiler` mirrors this default: its `include`/`externalRef` fetch `FS` is deny-all by default (opt into host access with `Compiler.FS(helium.PermissiveFS())`), and each target is read under a per-resource byte cap (`relaxng.Compiler.MaxResourceBytes`, 10 MiB default).

## Feature Status

### Fully Implemented

| Feature | Notes |
|---------|-------|
| XML parsing | Well-formedness, namespace handling, encoding detection/transcoding |
| SAX2 interface | All callbacks: startElementNS, endElementNS, characters, cDataBlock, comment, PI, DTD events, entity/notation/element/attribute decls |
| DOM tree | All node types: Element, Attribute, Text, CDATA, Comment, PI, Document, DTD, Entity, EntityRef, Notation |
| Namespaces | Declarations, prefix resolution, namespace nodes |
| DTD | Internal subsets, external subsets (limited), entity/notation/element/attribute decls |
| Encoding | Auto-detection, BOM, UTF-8/16, ISO-8859-*, Windows-*, CJK, EBCDIC, UCS-4 |
| Tree ops | AppendChild, InsertBefore, RemoveChild, ReplaceChild, CopyNode, Walk |
| Attribute setters | `Element.SetAttribute`/`SetAttributeNS` store the value verbatim (literal), mirroring libxml2 `xmlSetProp`; `Element.SetParsedAttribute`/`SetParsedAttributeNS` parse the value for entity references, mirroring `xmlNewDocProp` (the parsed variant libxml2 itself flags as a mistake the API can't change) |
| XPath 1.0 | Full expression eval, all 13 axes, 27+ functions, custom function registration |
| C14N | All 3 modes (1.0, Exclusive 1.0, 1.1), comments, node-set, inclusive NS, xml:* inheritance |
| XSD | Complex/simple types, all compositors, facets, IDC (key/unique/keyref), substitution groups, import/include, xsi:type/nil |
| RELAX NG | All patterns, name classes, include/override, externalRef, parentRef, data types, interleave, backtracking |
| Schematron | Assert, report, let variables, name, value-of, old (ASCC) + new (ISO) syntax |
| XInclude | Recursive inclusion, fallback, marker nodes, base URI fixup, circular detection |
| Catalog | OASIS XML Catalog, public/system ID resolution, URI resolution, catalog chaining, URN support |
| XML Writer | Streaming output, namespace scopes, indentation, DTD internal subsets, self-close optimization |
| Serialization | Write, WriteString, Writer.WriteTo, formatted output, encoding handling |
| XPath 3.1 | Full expression eval, FLWOR, maps, arrays, inline functions, HOFs, 100+ built-in functions, VM compiler |
| XSLT 3.0 | Templates, apply/call, include/import, functions, keys, sort, number, for-each-group, streaming instructions, schema awareness, accumulator, merge |

### Partial / Limited

| Feature | What Works | Gap |
|---------|-----------|-----|
| HTML parsing | SAX + DOM, auto-close, void elements, entities, encoding | Structural element nesting not enforced, areBlanks heuristic simpler, attribute deduplication missing |
| encoding/xml shim | Marshal, Unmarshal, Encoder, Decoder, Token, struct tags | Strict-only, xmlns before regular attrs, InputOffset approximate, undeclared prefixes rejected. The helium parser is the single authority for the XML declaration — grammar, version and placement — and the three entry points (`Unmarshal`, reader-backed `Decoder`, TokenReader-backed `Decoder`) agree on its verdict. A document declaring a non-UTF-8 encoding (e.g. `UTF-16`, `ISO-8859-1`) is rejected unless a `Decoder.CharsetReader` is set (same rule as encoding/xml); shim applies this from helium's decoded encoding, so every entry point agrees even when the declaration is itself in a fixed-width Unicode encoding (UTF-16 / UCS-4) that the byte-level prolog scanner cannot read, while a fixed-width Unicode document declaring no encoding names none and is accepted. shim accepts the versions helium accepts, 1.0 AND 1.1 (helium implements XML 1.1), where encoding/xml rejects `version="1.1"`; a version outside the 1.x family (e.g. `2.0`) is rejected. `Unmarshal` and the reader-backed `Decoder` accept 1.1 directly; a TokenReader-backed `Decoder` accepts a 1.1 declaration once delivered as a token, but an `encoding/xml.Decoder` used as the TokenReader cannot deliver one (it rejects 1.1 during its own tokenization) — a limitation of encoding/xml, not shim. `version = "2.0"` (spaces around `=`) is rejected where encoding/xml accepts it, a declaration violating the XMLDecl grammar (`charset=`, missing/empty version, empty encoding, non-`yes`/`no` standalone, repeated or out-of-order pseudo-attributes) is rejected through every entry point where encoding/xml accepts it, a declaration anywhere but the very first thing in the document — at position 0, with only a byte-order mark allowed ahead of it (a declaration preceded by ANY leading whitespace, or by an earlier declaration, comment, PI or doctype, is rejected; encoding/xml tolerates the leading whitespace and reports a later `<?xml` as an ordinary ProcInst — this whitespace rejection is a divergence, coherent across all three entry points; whitespace ahead of the root ELEMENT with no declaration stays accepted), likewise rejected where encoding/xml accepts it as an ordinary ProcInst, the reserved `xml` PI target in any casing (`<?XML?>`, `<?Xml?>`, `<?xMl?>` — `PITarget` subtracts the name case-insensitively) rejected wherever it appears through `Unmarshal`, reader-backed `Decoder` and TokenReader-backed `Decoder` alike, while a longer xml-prefixed target (`<?xmlversion?>`, `<?xml-stylesheet?>`) stays a legal ordinary PI |
| XSD range comparison | decimal/integer via big.Rat; float/double; date/time/g-types; duration partial order | Non-ordered primitives rejected for range facets; NaN ordinary value comparison remains indeterminate |
| XSD validation mode | DOM-only | No SAX/streaming validation, no subtree validation |
| Push parser | Incremental parsing in background goroutine from blocking push stream | Blocking on chunk boundaries; semantics differ from libxml2 push APIs |

### Not Implemented

| Feature | libxml2 Equivalent | Notes |
|---------|-------------------|-------|
| Reader API | xmlTextReader | No pull-parser equivalent |
| Pattern API | xmlPattern | No compiled pattern matching |
| SAX/streaming validation | xmlSchemaSAXPlug | XSD/RELAX NG are DOM-only |
| Custom I/O callbacks | xmlIO | Uses io.Reader/io.Writer directly |
| Automata/Regexp | xmlAutomata, xmlRegexp | Go regexp replaces |
| Global state | xmlInitParser/xmlCleanupParser | Not needed in Go |
| Memory management | xmlMalloc/xmlFree | Go GC replaces |

## Parser Option Parity (Fluent API)

| Fluent Method | libxml2 Equivalent | Status | Notes |
|---------------|-------------------|--------|-------|
| RecoverOnError(bool) | XML_PARSE_RECOVER | ✅ | Recovery mode on errors |
| SubstituteEntities(bool) | XML_PARSE_NOENT | ✅ | Substitute entities |
| LoadExternalDTD(bool) | XML_PARSE_DTDLOAD | ✅ | Load external subsets |
| DefaultDTDAttributes(bool) | XML_PARSE_DTDATTR | ✅ | Default DTD attributes |
| ValidateDTD(bool) | XML_PARSE_DTDVALID | ✅ | Validate with DTD |
| SuppressErrors(bool) | XML_PARSE_NOERROR | ✅ | Suppress error reports |
| SuppressWarnings(bool) | XML_PARSE_NOWARNING | ✅ | Suppress warnings |
| PedanticErrors(bool) | XML_PARSE_PEDANTIC | ✅ | Pedantic error reporting |
| StripBlanks(bool) | XML_PARSE_NOBLANKS | ✅ | Remove blank nodes |
| XInclude(XIncludeProcessor) | XML_PARSE_XINCLUDE | ✅ | XInclude substitution; inject a configured `xinclude.Processor`, run over the tree during Parse (dependency-inversion seam — helium can't import xinclude) |
| AllowNetwork(bool) | XML_PARSE_NONET | ✅ | Inverted: false → forbid network. **Default false** (NONET set by NewParser). helium has no dedicated network loader; every external load (DTD subset, general/parameter entity) goes through the configured `fs.FS`. When false, the three `fs.FS.Open` sites in `tree_builder.go` (`ExternalSubset`, `ResolveEntity`) refuse a name whose URI scheme is `http`/`https`/`ftp` (case-insensitive) with `ErrNetworkAccessForbidden` before reaching the FS — defense-in-depth for a caller-supplied network-capable FS. A schemeless name, a `file:` scheme, or a bare path is not a network resource and loads as usual |
| CleanNamespaces(bool) | XML_PARSE_NSCLEAN | ✅ | Remove redundant NS decls |
| MergeCDATA(bool) | XML_PARSE_NOCDATA | ✅ | Merge CDATA as text |
| FixBaseURIs(bool) | XML_PARSE_NOBASEFIX | ✅ | Inverted: false → skip fixup |
| MaxNameLength(int) | (was XML_PARSE_HUGE) | ✅ | Per-limit knob: max name length (0=default 50000, <0=unlimited) |
| MaxEntityAmplification(int) | (was XML_PARSE_HUGE) | ✅ | Per-limit knob: max entity-amplification ratio (0=default 5, <0=ratio check off; 1 GiB hard ceiling always applies) |
| MaxContentModelDepth(int) | (was XML_PARSE_HUGE) | ✅ | Per-limit knob: max DTD content-model depth (0=default 128, <0=unlimited) |
| MaxNodeContentSize(int) | XML_MAX_TEXT_LENGTH (intent) | ✅ | Per-limit knob: max bytes of a single CDATA/comment/PI/char-data run or attribute value, AND of a single contiguous XML-whitespace (blank-skip) run (0=default `DefaultMaxNodeContentSize` 10 MiB, <0=unlimited — disables BOTH the node-content and the blank-run cap). Fires during accumulation; over-cap → `ErrNodeContentTooLarge`. Streaming SAX (`CharBufferSize>0`) char data is exempt (already chunked) |
| IgnoreEncoding(bool) | XML_PARSE_IGNORE_ENC | ✅ | Ignore encoding hint |
| BlockXXE(bool) | XML_PARSE_NOXXE | ✅ | Block XXE attacks. **Default true** (NOXXE set by NewParser; libxml2 defaults off) |
| SkipIDs(bool) | XML_PARSE_SKIP_IDS | ✅ | Skip ID interning |
| LenientXMLDecl(bool) | *(helium extension)* | ✅ | Relaxed XML decl attribute order |
| MaxExternalDTDBytes(int) | *(helium extension)* | ✅ | Byte cap for external DTD subset reads; `0` → `MaxExternalDTDSize` (10 MiB), negative disables the cap. Enforced against actual bytes read; over-cap → `ErrExternalDTDTooLarge` |
| *(dropped)* | XML_PARSE_NOUNZIP | no-op | No decompression support |
| *(dropped)* | XML_PARSE_NOSYSCATALOG | no-op | No global catalog |
| *(dropped)* | XML_PARSE_CATALOGPI | no-op | Not yet implemented |

## Known Issues

- `xinclude/issue733` — DOCTYPE not preserved after XInclude processing
- DTD element-content validity (VC: Element Valid), character-reference whitespace provenance: whitespace produced by a character reference does NOT match the S nonterminal, so it is not ignorable in element-only content (XML §3.2.1, errata 2e E15). With `SubstituteEntities(true)` the DOM text node is byte-identical to one from literal whitespace, so provenance is reconstructed by a parser flag: `parseReference` marks a `&#N;` delivery (`pctx.charDataFromCharRef`), `TreeBuilder.Characters` stamps the Text node's `fromCharRef` field, and `collectChildElements` (`valid.go`) treats a `fromCharRef` whitespace text node as character data (content-model mismatch). It survives general-entity expansion via the re-parse of the entity replacement text and `replayEntityNode`, distinguishing an entity value of `&#32;` (invalid, `rmt-e2e-15h`) from a literal-space entity value (valid, `rmt-e2e-15e`) or a `&#32;` expanded at DECLARATION time into a literal space (valid, `rmt-e2e-15f`). A direct character reference to a whitespace char in element-only content is likewise invalid (`rmt-e2e-15g`). Separately, a reference is content per XML production [43], so an EMPTY element containing one is invalid even when it expands to nothing (`rmt-e2e-15a`); `parseReference` sets `Element.contentHasReference` and `validateElementContent`'s EMPTY branch reads it. All flags are validity-only — invisible to serialization, C14N, XPath string-value, and node copy. A CDATA section is a distinct node type, so a CDATA section in element-only content — even empty or whitespace-only, never S — IS rejected (`sun/invalid empty`); and an NMTOKENS/IDREFS/ENTITIES token separator is exactly `#x20` (`splitNormalizedTokens`), so a character-reference whitespace char inside a tokenized attribute value is part of the token, making e.g. `abc&#9;xyz` a single invalid NMTOKEN (`rmt-e2e-20`).
- C14N relative namespace URI check uses heuristic (`!strings.Contains(uri, ":")`) not full URI parse
- HTML attribute deduplication: all kept (libxml2 keeps first)
- HTML areBlanks heuristic simpler than libxml2's

## Intentional Divergences

These are architectural choices, not bugs:

- Go error returns vs C integer return codes + xmlRaiseError
- Go GC vs malloc/free/reference counting
- Package splitting: single .c file → entire Go package
- Go interfaces for node types vs xmlNode.type enum switch
- No global state: explicit context passing
- Functional options (WithX()) vs bitmask flags
- XML push parser parses incrementally as chunks arrive; HTML push parser buffers through the initial ~1024-byte charset prescan, then streams only once a streamable encoding is settled (declared/detected charset=utf-8, or a non-UTF-8 head routed to Latin-1). An undeclared input that keeps proving valid UTF-8 stays undecided and buffers (a later non-UTF-8 byte would re-interpret the whole prefix as Latin-1/Windows-1252), but the undecided prefix is BOUNDED at the configured `MaxContentSize` (16 MiB default): each undecided read is capped to the remaining bound so the boundary is chunk-independent. A stream ending with valid UTF-8 at/below the cap is accepted (one-byte EOF probe); if the cap fills and more bytes still follow it fails closed with `ErrContentSizeExceeded` (`html/encoding_reader.go` deferredLatin1Reader).
- Namespace stack: frame-based visibleNSStack vs flat arrays
- `xml:id` value normalization: the parser applies tokenized-type (xs:ID) normalization — trim + internal-space collapse — to an `xml:id` attribute even when it is NOT DTD-declared, per the xml:id Recommendation §4 + XML §3.3.3 (so `GetElementByID`/`fn:id`/XPath string-value see the collapsed id). libxml2 normalizes `xml:id` only when it is DTD-declared ID and leaves undeclared-`xml:id` normalization as a documented open issue (it does not do it). This is a DELIBERATE XPath-3.1 / xml:id-§4 conformance choice; it is verified byte-identical on every libxml2-compat / c14n / serialization golden (no parity fixture carries a normalizable-whitespace `xml:id`).
- Node-builder colon validation: `Document.CreateElement(name)` and `Document.CreateAttribute(name, ...)` reject a colon in `name` and return an error (the caller supplies a namespaced element/attribute through `Document.CreateElementNS(localname, ns)` / `Element.SetAttributeNS(localname, ...)`). libxml2's `xmlNewNode`/`xmlNewProp` do no validation and let a colon sit in the single name field, producing an unbound prefix that serializes as namespace-ill-formed XML. This is the same namespace-aware strictness the W3C xml-conformance campaign classified as an intentional divergence (colon-in-Name). It changes no parsed-document output: the parser always splits a QName into prefix+local and sets the namespace separately, so every in-tree build/copy call site feeds a bare local name and the rejection never fires.
- Writer invalid-character default: the `Writer` REJECTS a syntactically valid character or numeric-reference target outside the target XML range (`ErrInvalidXMLChar` / SERE0006) by default, where libxml2 silently replaces it with U+FFFD. `Writer.RejectInvalidChars(false)` restores replacement — as `&#xFFFD;` under `EscapeNonASCII`/US-ASCII output and raw U+FFFD otherwise; reference-less contexts always use raw U+FFFD. Malformed numeric, named, and parameter-entity reference markup always fails with `ErrWriterInvalidName`; it is never repaired. Full DTD Names remain distinct from DOM QNames, entity and notation declarations require NCNames, and enumeration members are checked as distinct Nmtokens or Names before output. A character map is exempt because its replacement string is emitted verbatim (Serialization 3.1 §7). Parsed-document output and serialization goldens remain byte-identical; the divergence only affects malformed in-memory trees.
