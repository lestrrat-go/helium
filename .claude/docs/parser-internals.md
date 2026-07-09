# Parser Internals

## Entry Points

- **`Parse(ctx, []byte)`** / **`ParseReader(ctx, io.Reader)`** — main entry points
- **`NewParser()`** — configurable parser; set options, SAX handler, catalog, baseURI, maxDepth, FS
- **`p := helium.NewParser(); pp := p.NewPushParser(ctx)`** — background push parser (parses incrementally as data arrives)
- **`ParseInNodeContext(ctx, node, []byte)`** — parse fragment in element context

Key files:
- `parser.go` — public parser builder/API
- `parserctx.go` — parser context/state, cursor stack, SAX dispatch, error handling
- `parser_document.go` — top-level document/prolog/epilogue flow
- `parser_element.go` — element/start-tag/end-tag/attribute/chardata parsing
- `parser_whitespace.go` — blank skipping and ignorable-whitespace classification
- `parser_xml_decl.go` + `parser_decl.go` — XML declaration and name/QName helpers
- `parser_dtd_subset.go` + `parser_dtd_element.go` + `parser_dtd_attr.go` — DTD subset/declaration parsing
- `parser_entity_decl.go` + `parser_entity_ref.go` — entity declaration and reference expansion
- `parser_encoding.go` — encoding detection/switching and low-level cursor helpers
- `parser_content.go` — comments, PI, CDATA, misc
- `tree_builder.go` — SAX→DOM

## Parse Pipeline

```
INPUT ([]byte or io.Reader)
  → ByteCursor on inputStack
  → detectEncoding() — BOM/pattern/EBCDIC scan
  → parseXMLDecl() — version, encoding, standalone
  → switchEncoding() — RuneCursor wrapping encoder
  → SetDocumentLocator SAX callback
  → StartDocument SAX → create Document
  → parseMisc() — comments, PIs before DOCTYPE
  → parseDocTypeDecl() + parseInternalSubset()
    → EntityDecl, AttlistDecl, ElementDecl, NotationDecl
    → ExternalSubset SAX → load external DTD
  → parseMisc() — between DOCTYPE and root
  → parseElement() [root + recursive content]
    → parseStartTag() → parseAttribute() → pushNS()
    → StartElementNS SAX → create Element
    → parseContent() [children: elements, text, comments, PIs, CDATA, refs]
    → parseEndTag() → EndElementNS SAX
  → parseMisc() — epilogue
  → EndDocument SAX
  → DTD validation (if ValidateDTD(true))
  → RETURN Document + error
```

## Parser Context (`parserCtx`)

Central state struct. Key fields:

### Input Management
- `inputTab` (inputStack) — LIFO stack of ByteCursor/RuneCursor. Entity expansion and external DTDs push new cursors.
- `getCursor()` — current cursor, auto-pops exhausted ones, caches the active cursor between parser calls

### Parser State Machine
States: `psStart`, `psContent`, `psPrologue`, `psEpilogue`, `psCDATA`, `psDTD`, `psEntityDecl`, `psAttributeValue`, `psComment`, `psStartTag`, `psEndTag`, `psSystemLiteral`, `psPublicLiteral`, `psEntityValue`, `psIgnore`, `psMisc`, `psPI`, `psEOF`

State affects parsing rules: e.g., external entity refs forbidden in `psAttributeValue`, parameter entity handling restricted in `psDTD`.

### Element/Namespace Stacks
- `nodeTab` (nodeStack) — element nesting stack
- `nsTab` (nsStack) — prefix→URI bindings; `Push(prefix, uri)`, `Lookup(prefix)`, `Pop(n)`
- `nsNrTab []int` — namespace count per element level (parallel to nodeTab); used to pop exact count on element close
- `spaceTab []int` — xml:space stack (-1=inherit, 0=default, 1=preserve)

### XML 1.1 Version-Gated Relaxations
`pctx.isXML11()` (`version == "1.1"`, from the XML declaration; absent decl = 1.0) gates two well-formedness relaxations; every non-1.1 document stays byte-identical:
- **Namespace prefix undeclaration** (`parser_element.go` `validatePrefixedNamespaceDecl`): `xmlns:pfx=""` is accepted only in XML 1.1 and pushes an empty-URI binding onto `nsTab` (undeclaring the prefix); XML 1.0 rejects it ("Empty XML namespace is not allowed"). The reserved `xml`/`xmlns` prefixes are checked before the empty-URI branch, so they can never be undeclared. Applies to both literal and DTD-defaulted namespace declarations. A default-namespace undeclaration (`xmlns=""`) is legal in both versions and is handled separately.
- **Control-character char references** (`parser_entity_ref.go` `parseCharRef`): a `&#N;`/`&#xN;` reference to a C0/C1 control character (all but U+0000) that the XML 1.0 Char production forbids is accepted in XML 1.1 via `isXML11CharValue` (`parser_content.go`). Only the char-reference value check is relaxed; literal control bytes and XML 1.0 documents are unchanged.

### SAX & Tree Building
- `sax` (sax.SAX2Handler) — callbacks (default: TreeBuilder)
- `doc *Document` — parsed document
- `elem *Element` — current element

### DTD & Entities
- `attsSpecial map[string]enum.AttributeType` — special attributes from DTD. `parseAttribute()` sets its value-normalize flag from this map (a non-CDATA tokenized type collapses whitespace), keyed by the element's full QName + the attribute's full QName, both exactly as written (prefix + local), so `<r p:id>` matches only an `<!ATTLIST r p:id …>` declaration and never the unprefixed `id`, and `<!ATTLIST p:r …>` applies to `<p:r>` and not `<r>` (parseStartTag threads the element QName into parseAttribute); `xml:id` additionally forces normalization unconditionally (it is implicitly `xs:ID` even with no DTD declaration, per the xml:id Recommendation §4 + XML §3.3.3 tokenized-type normalization), so its stored DOM value is trimmed and internal-space-collapsed and is what `GetElementByID`/`fn:id`/the XPath string-value resolve. This is a DELIBERATE XPath-3.1 / xml:id-§4 conformance divergence from libxml2, which normalizes `xml:id` only when it is DTD-declared ID and leaves undeclared-`xml:id` normalization as a documented open issue (it does not do it). It is verified to leave every libxml2-compat / c14n / serialization golden byte-identical (no parity fixture carries a normalizable-whitespace `xml:id`). A companion `attsSpecialExternal map[string]struct{}` records which entries were declared in external markup (libxml2's `XML_SPECIAL_EXTERNAL`), populated in `addSpecialAttribute` when `effectivelyExternal()` (in the external subset OR draining an external parameter entity — `inSubset == inExternalSubset || len(externalPEScopes) > 0`); combined with the transient `attrNormChanged` flag (set when the normalize path trims/collapses whitespace in `parseAttributeValueInternal`), `parseAttribute` records each external-driven normalization change on `doc.standaloneNormAttrs` for the post-parse Standalone Document Declaration VC (§2.9) — see validation-pipeline.md.
- `attsDefault map[string][]*Attribute` — default attributes from DTD
- `inSubset int` — 0=not in subset, 1=internal, 2=external
- `replaceEntities bool` — expand entity refs (set by SubstituteEntities(true))
- `fsys fs.FS` — filesystem used to load external DTDs/entities; defaults to `internal/iofs.DenyAll{}` (refuses every open — safe-by-default), overridden via `Parser.FS()` (pass `helium.PermissiveFS()` / `internal/iofs.PermissiveRoot{}` to restore os.Open passthrough). Used by `TreeBuilder.ExternalSubset` and `TreeBuilder.ResolveEntity`. When a catalog resolves an identifier to a `file:` URI, `tree_builder.go`'s `catalogOpenName` converts it to a local path via `internal/iofs.FileURIToPath` before `fsys.Open` (mirroring the XInclude `file:` handling); non-file URIs and plain paths pass through unchanged. Entity sub-parsers (`parseExternalEntityPrivate`, `parseBalancedChunkInternal`) both seed their nested context through the shared `inheritNestedParserState` helper (`parser_entity_decl.go`), which copies the parent's `sax`, `treeBuilder`, `attsDefault`, config-derived policy (`options`, `loadsubset`, `replaceEntities`, `keepBlanks`, `pedantic`, `charBufferSize`, `maxExtDTDSize`, `maxNameLength`, `maxCMDepth`, `maxNodeContent`, `fsys`, `catalog`, `baseURI`), and — critically for depth enforcement — BOTH `maxElemDepth` (the limit) AND the parent's current `elemDepth`. (`maxNameLength`/`maxCMDepth`/`maxNodeContent` are granular limit fields; they must be copied here too or a configured `MaxNameLength`/`MaxContentModelDepth`/`MaxNodeContentSize` would not apply inside entity expansion.) Carrying the current `elemDepth` means element nesting that crosses an entity-expansion boundary keeps accumulating toward `MaxDepth` instead of restarting at 0: without it a single substituted element (`<!ENTITY e "<a/>">` used inside `<r>&e;</r>`) would wrongly pass `MaxDepth(1)` even though the literal `<r><a/></r>` is depth 2. This applies equally to external entity replacement text. It ALSO carries the shared `ebcdicConsumed` pointer (the live consumed-byte counter over an EBCDIC `ParseReader` stream; `nil` off that path), because a nested entity sub-parse's `inputSize` is only the bounded sniff prefix — the amplification guard (`entityCheckLimits`) must divide by the real consumed-byte count or a large internal entity reached indirectly (`<!ENTITY wrap "&big;">` used in `<root>&wrap;</root>`) would be falsely rejected as amplification. The helper does NOT touch `doc`, the `external` flag, or the per-context amplification counters (`sizeentcopy`/`inputSize`/`maxAmpl`); each caller sets those because their lifecycle differs (document swap, external flag, counter write-back on return). Otherwise these would all reset to zero-value defaults, e.g. `maxElemDepth=0` disabling the depth check and `fsys` falling back to the safe-by-default `DenyAll`.

### Entity Amplification Guard
- `sizeentcopy int64` — cumulative entity expansion bytes
- `maxAmpl int` — max amplification factor (5 default, 0 with MaxEntityAmplification(-1))
- `inputSize int64` — original input size
- Rules: 1MB baseline before ratio check; 20 bytes fixed cost per entity ref

### Error Recovery
- `disableSAX bool` — suppress callbacks after fatal
- `recoverErr error` — first fatal error (RecoverOnError mode)
- `stopped bool` — StopParser() called

## Encoding Detection

Order in `detectEncoding()`:
1. UCS-4 BE/LE/2143/3412 — matches the encoded leading `<` byte patterns of a BOM-less document and PEEKS them (does not consume), unlike a real BOM which is consumed
2. EBCDIC (`0x4C 0x6F 0xA7 0x94` invariant prefix)
3. UTF-8 BOM (`0xEF 0xBB 0xBF`)
4. UTF-16 BOM (`0xFF 0xFE` LE, `0xFE 0xFF` BE)
5. UTF-16 by context (first 4 bytes pattern)
6. Default: ASCII/UTF-8

Special cases:
- **UTF-16**: switch encoding FIRST (XML decl is UTF-16 encoded), then parse decl
- **EBCDIC**: use invariant charset to extract encoding name, default IBM-037
- **ASCII-compatible**: parse XML decl at byte level, then switch

`switchEncoding()`: pop ByteCursor, create encoder, push RuneCursor.

**BOM vs declared encoding**: a real (consumed) byte-order mark at the document
start asserts the entity's encoding, recorded on `pctx.autoEncoding` (UTF-8 from
`EF BB BF`, UTF-16LE from `FF FE`, UTF-16BE from `FE FF`) in `detectEncoding`.
After the XML declaration is parsed and the encoding switched, `parseDocument`
calls `checkBOMEncodingConflict` (`parser_encoding.go`): a declared encoding that
is not in the BOM's allowed-alias set (`bomAllowedEncodings`, mirroring libxml2's
`xmlSetDeclaredEncoding` lists — UTF-8→{UTF-8,UTF8}, UTF-16BE→{UTF-16,UTF-16BE,
UTF16}, UTF-16LE→{UTF-16,UTF-16LE,UTF16}, case-insensitive) is a fatal
`ErrEncodingBOMMismatch` (XML §4.3.3; W3C not-wf `hst-lhs-007`/`hst-lhs-008`).
This is a byte-for-byte stricter stance than libxml2, which downgrades the
mismatch to a warning and continues. Only a consumed BOM sets `autoEncoding`, so
the byte-pattern (non-BOM) UTF-16/UCS-4 detection and a plain ASCII/UTF-8 `<?xml`
start that declares a single-byte encoding (e.g. `iso-8859-1`) are unaffected —
no over-rejection of a legitimately single-byte-encoded document.

**Strict decode of fixed-width Unicode encodings**: `internal/encoding` wraps the
UTF-16/UTF-32/UCS-2/UCS-4 decoders (`withStrictDecode`, `strict.go`) so that
malformed input the base decoder would silently replace with U+FFFD (e.g. an
unpaired surrogate or trailing odd byte) becomes a fatal `ErrInvalidEncodedChar`,
while a genuinely-encoded U+FFFD still decodes. The decoder reader's error is
remembered by `UTF8Cursor` as a sticky `Err()` (a clean `Done()` would otherwise
mask it as EOF); `parseDocument` surfaces it via `cursorDecodeErr()` at the
document-end gate.

**Strict US-ASCII decode**: US-ASCII is a 7-bit encoding. It is deliberately
excluded from `IsUTF8`'s direct-byte-cursor fast path and routed through `Load`,
whose `asciiEncoding` decoder (`ascii.go`) rejects any byte >= 0x80 with
`ErrInvalidASCII`. A document declaring `encoding="US-ASCII"` that contains a
byte >= 0x80 (even a valid UTF-8 multibyte sequence) is therefore a fatal decode
error rather than being silently passed through.

## File Responsibilities

- `parserctx.go` owns the parser context, input/cursor stack, SAX callback dispatch, location/error reporting, and other shared parser state.
- `parser_document.go` owns the top-level parse pipeline (`parseDocument`, `parseContent`, recovery re-sync).
- `parser_element.go` owns recursive element parsing, start/end tags, attributes, and character data. `parseStartTag` enforces XML §3.1 P40/P44 required inter-attribute whitespace: after EVERY attribute — the default-namespace, prefixed-namespace, and regular-attribute branches alike — the next character must close the tag (`>` or `/>`) or be whitespace; a NameStartChar with no intervening `S` is a fatal `ErrSpaceRequired` (mirrors libxml2's uniform `next_attr` check; W3C not-wf `attlist10`/`attlist11`, `o-p40fail1`/`o-p44fail4`, `not-wf-sa-186`).
- `parser_xml_decl.go` owns byte/rune XML declaration parsing; `parser_decl.go` owns the lower-level declaration token/value helpers and name/QName parsing. Both `parseXMLDecl` (byte) and `parseXMLDeclFromCursor` (rune) enforce EncodingDecl [80]/EncName [81]: a `parseEncodingDecl(FromCursor)` error is fatal UNLESS it is an `AttrNotFoundError` (the `encoding` keyword is wholly absent, which falls through to the optional StandaloneDecl). A present-but-malformed encoding pseudo-attribute — missing `=`, missing opening quote, or an invalid/empty EncName — is therefore a fatal well-formedness error rather than being silently dropped (W3C `ibm-not-wf-P80-ibm80n03`). EncName [81] (`[A-Za-z] ([A-Za-z0-9._] | '-')*`) is validated on BOTH cursors — `parseEncodingName` (byte) and `parseEncodingDeclFromCursor` (rune, used for a UTF-16 XMLDecl/TextDecl) share the `isEncNameStart`/`isEncNameChar` predicates — so an empty or grammar-invalid EncName is rejected regardless of the source encoding. The external-entity TextDecl paths (`parseTextDecl` byte / `parseTextDeclFromCursor` rune) treat ANY encoding-decl error as fatal since a TextDecl's EncodingDecl is mandatory.
- `parser_dtd_*` files split DTD handling by declaration kind instead of keeping all markup parsing in one file. `parseExternalID` (`parser_dtd_attr.go`) returns a `found bool` reporting whether a SYSTEM/PUBLIC production was present, distinguishing an empty-but-present literal (`SYSTEM ""`, found=true) from a wholly absent ExternalID (found=false) — mirroring libxml2's NULL-vs-non-NULL URI/literal pointers. `parseNotationDecl` requires it: NotationDecl [82] makes `(ExternalID | PublicID)` mandatory, so `!found` is a fatal `ErrNotationExternalIDRequired` (W3C `ibm-not-wf-P82-ibm82n03`; a byte-for-byte stricter stance than libxml2, which accepts a bare `<!NOTATION n>`).
- `parser_entity_decl.go` handles entity declaration bodies and balanced-chunk parsing; `parser_entity_ref.go` handles references, char refs, replay, and amplification checks. `parseEntityDecl` requires an EntityDef [73] / PEDef [74] body: when neither an EntityValue (a quoted literal) nor an ExternalID (`parseExternalID` found=true) is present, the declaration is a fatal `ErrValueRequired` (W3C `o-p73fail4`), matching libxml2's `(URI == NULL) && (literal == NULL)` check for both general and parameter entities. An external parameter/general entity is registered (its `EntityDecl` SAX callback fires) whenever the ExternalID was present (found), INCLUDING a valid empty SystemLiteral (`<!ENTITY % pe SYSTEM "">`) — registration is driven off `found`, not `literal != ""`, so an empty-literal PE still appears via `GetParameterEntity`.

## Entity Expansion

### Flow (`parseReference()`)

1. `parseEntityRef()` — resolve entity name
   - Check predefined (lt, gt, amp, apos, quot)
   - SAX `GetEntity()` callback
   - Document entity table lookup

**WFC: Entity Declared under standalone="yes"** (XML §4.1 / §2.9). `getEntity` (`parser_entity_ref.go`, a port of libxml2's `xmlSAX2GetEntity`) enforces that a `standalone="yes"` document must not reference a general entity declared ONLY in the external subset — a standalone document asserts that no external markup declarations affect its content. `Document.GetEntity` HIDES external-subset declarations while `standalone == StandaloneExplicitYes`, so `getEntity` first looks the name up with standalone active (internal subset only); on a miss it retries with standalone temporarily disabled. That retry SUCCEEDING is exactly the violation: for a reference in the document body (`inSubset == 0`) it fires a fatal `ErrNotStandalone` ("document marked standalone but requires external subset", libxml2's `XML_ERR_NOT_STANDALONE`) while still returning the resolved entity (so the caller surfaces a precise error). A genuinely undeclared name (miss on both lookups) returns `(nil, nil)` and the caller applies the ordinary "Entity Declared" WFC via `handleUndeclaredEntity`. Both document-reference resolvers route through this: the cursor path (`parseEntityRef`) and the string-buffer path (`parseStringEntityRef`) fall back to `getEntity` precisely because the `TreeBuilder.GetEntity` SAX callback returned nil (its `doc.GetEntity` is standalone-filtered). References WITHIN the DTD (`inSubset != 0`, e.g. an entity-declaration `SetOrig` lookup) and references INSIDE the external subset (`inSubset == 2`, which resolves against the whole DTD without flagging) are not flagged. A `standalone="no"` document referencing an externally-declared entity, and a `standalone="yes"` document referencing an internally-declared entity, both parse unchanged (W3C not-wf `ibm-not-wf-P32-ibm32n09`, `ibm-not-wf-P68-ibm68n06`, `not-wf-sa03`).
2. `entityCheck(ent, size)` — amplification guard
   - Baseline: 1MB free
   - Fixed cost: 20 bytes per ref (charged once per reference)
   - Max amplification: 5× input (disabled with MaxEntityAmplification(-1))
   - Already-checked entities use cached `expandedSize`
   - Raw-byte variant `entityCheckBytes(size)` charges expansion bytes WITHOUT the fixed cost, used for external content already charged the fixed cost by `parseReference`
3. Parse entity content if needed (`parseBalancedChunkInternal`, or `parseExternalEntityPrivate` for external entities)
   - Recursively parse entity text
   - Seed in-scope namespaces from the surrounding element before parsing
   - Fill `ent.firstChild` (parsed nodes)
   - Mark `ent.checked = 2`, cache `ent.expandedSize`
   - External entities: content is read through a bounded `io.LimitReader` (cap `externalEntityMaxBytes`, 10 MiB in parserctx.go) and the read bytes are charged to `sizeentcopy` via `entityCheckBytes` (raw bytes only, NOT `entityCheck`): `parseReference` already paid the fixed per-reference cost (`entityFixedCost`) for this reference, so charging it again here would double-count it. Repeated references to a near-cap external entity still trip the amplification guard; the running counters propagate into and back out of the nested parse context. `ent.expandedSize` caches the external (plus nested) size for subsequent references.

### External Input Bounds & Lifetime

All bytes pulled from the filesystem (`ctx.fsys`) are byte-capped and the opened `fs.File` is closed promptly, never held open for the lifetime of the parse:

- **External parsed entities** (`parseExternalEntityPrivate` in `parser_entity_decl.go`): capped at `externalEntityMaxBytes` (10 MiB); content over the cap is rejected with an "exceeds maximum size" error. The resolved input is closed once the bounded read completes. An external parsed general entity's replacement text may begin with an OPTIONAL leading TextDecl (`'<?xml' VersionInfo? EncodingDecl S? '?>'` — VersionInfo OPTIONAL, EncodingDecl REQUIRED, NO StandaloneDecl per XML §4.3.1). It is consumed and the body decoded per its declared encoding by `decodeExternalPEContent` (the same helper the external DTD subset and external parameter entities use) BEFORE the nested parse, so the sub-parser is handed post-TextDecl UTF-8 content rather than a version-less declaration its `parseXMLDecl` would reject for a missing version, or a leading `<?xml` its `parseContent` would reject as a PI (target may not be `xml`). The TextDecl grammar is enforced strictly by `parseTextDecl` (ASCII-compatible content) or `parseTextDeclFromCursor` (fixed-width content, below) — a version-only declaration (missing the mandatory encoding), a standalone-bearing one, or any other out-of-grammar form is a fatal error (W3C not-wf `encoding07`/`not-wf-ext-sa-002`). `decodeExternalPEContent` also decodes UTF-16 / UCS-4 external content: when the bytes begin with a byte-order mark or the encoded shape of a leading `<` (`fixedWidthUnicodeEncoding` in `parser_encoding.go`), `decodeFixedWidthExternalContent` switches the sub-cursor to the detected fixed-width encoding and consumes any leading TextDecl — itself in that encoding, so invisible to a byte-level `<?xml` scan — on the decoded rune cursor via `parseTextDeclFromCursor`, returning the UTF-8 body. A resource with no TextDecl (e.g. a UTF-16 external DTD subset opening on a comment) is still decoded from the BOM (W3C `ext02`, `valid-ext-sa-008`, japanese `weekly-*`).
- **External DTD subset** (`TreeBuilder.ExternalSubset` in `tree_builder.go`): read through `io.LimitReader` with cap `ctx.maxExtDTDSize` (defaults to `MaxExternalDTDSize`); the read is bounded one byte past the cap so a source that under-reports its size is still caught, and the cap is enforced authoritatively against the bytes actually read (Stat size is advisory only). The file is `Close()`d immediately after the bounded read, before the buffered DTD is parsed.
4. Deliver to SAX
   - `replaceEntities=true`: expand inline and replay parsed node children through SAX (`StartElementNS`/`EndElementNS`, `Characters`, `CDataBlock`, `Comment`, `PI`)
   - `replaceEntities=false`: fire Reference callback only

### Parameter Entity References (`parsePEReference()`)

When a `%name;` parameter-entity reference in the DTD subset resolves, `parsePEReference(ctx, pad)` (in `parser_dtd_subset.go`) decodes the PE replacement text via `decodeEntities(SubstituteBoth)` and then charges the PE's OWN replacement bytes against the amplification guard with `entityCheck(entity, len(entity.Content()))` BEFORE pushing the decoded text as new input via `pushInput`. It charges `len(entity.Content())` (the PE's stored replacement text), NOT `len(decodedContent)`. When `pad` is true the pushed replacement is enlarged by one leading and one trailing space (`padPEContent`) per XML §4.4.8 "Included as PE"; the between-declaration callers pass `pad=false` (a PE standing on its own is already whitespace-separated), while the in-markup expansion path (`skipBlanksPE`, below) passes `pad=true`. This matters because `decodeEntities(SubstituteBoth)` ALREADY charges every nested entity expansion it performs — general references such as `&g;` are left literal in a PE's stored value (only PE references are substituted at declaration time) and are expanded and charged here, as is any residual parameter reference — via its own `entityCheck` calls. `decodedContent` is the result AFTER those nested expansions, so charging its length would double-count the nested bytes and could falsely reject a legitimate DTD whose `%p;` expands mostly through a nested entity (e.g. `<!ENTITY g "...big...">` plus `<!ENTITY % p "<!-- &g; -->">`). Charging `entity.Content()` accounts only the direct bytes the PE itself contributes. Without this charge the PE's direct contribution would be free, letting a small DTD reference a large PE many times to drive unbounded expansion past the amplification limit. A `%name;` that resolves to nothing (not found, in a context where that is non-fatal) still calls `entityCheck(entity, 0)` to charge the per-reference fixed cost.

**External parameter entities** (`enum.ExternalParameterEntity`): the replacement text is the content of the referenced external resource, not inline text, so `parsePEReference` does NOT route it through `decodeEntities` (which is for inline internal-PE text). It loads the bytes via `loadExternalParameterEntityContent` (XXE-gated by `parseNoXXE`, resolved through SAX `ResolveEntity`, byte-capped at `externalEntityMaxBytes`, closed at the read boundary, cached on the entity), charges them with `entityCheck`, then pushes them as RAW input for the surrounding declaration loop to parse. When external loading is disabled (secure default, or the resolver declines) the load returns empty and nothing is pushed — behavior is unchanged from a non-loading parser. `loadExternalParameterEntityContent` returns `([]byte, uri string, error)`: the `uri` is the resource actually opened (`sax.ParseInput.URI()`, falling back to the entity's declared URI). Both the bytes (`Entity.content`) AND this resolved URI (`Entity.resolvedURI`) are cached on the entity on first load; a later reference returns the cached pair — NOT `e.URI()` — so cached bytes always parse against the SAME base the first load used (critical when the first load went through a catalog/custom resolver whose URI differs from the entity's declared system ID, e.g. a `%pe;` first loaded via an entity value then referenced top-level). An external parsed entity's OPTIONAL leading TextDecl (`<?xml … encoding=…?>`) is consumed and the body decoded per its declared encoding by `decodeExternalPEContent` INSIDE `loadExternalParameterEntityContent`, BEFORE the post-TextDecl bytes are cached on the entity — so EVERY consumer (the top-level `%pe;` loop AND the entity-value expansion path) gets the same stripped/decoded bytes regardless of reference order, and the declaration loop is not handed a `<?xml` it would reject as a PI. The TextDecl grammar is enforced strictly by `parseTextDecl` (`parser_xml_decl.go`): `TextDecl ::= '<?xml' VersionInfo? EncodingDecl S? '?>'` — VersionInfo OPTIONAL, EncodingDecl REQUIRED, NO StandaloneDecl. A version-only declaration, a standalone-bearing one, or a missing encoding is REJECTED. The external-PE input is then pushed via `pushExternalPEInput(cursor, uri, ent)`, which (a) calls `pushInputWithBaseURI(cursor, uri)` to scope `pctx.baseURI` to the PE's OWN resolved URI while that cursor is on the input stack (the previous `baseURI` is recorded in `pctx.baseURIScopes` and restored by `popInput` when that exact cursor is popped — LIFO), and (b) marks the entity active (`activeExternalPECount`/`externalPEScopes`) so a self/mutually recursive external PE — whose pushed input references itself while still being drained — is rejected by the `externalPEActive` guard in `parsePEReference` BEFORE another load/push, instead of pushing cursors until the entity-amplification ceiling trips (internal PEs are guarded separately by the decode-depth cap). The active mark is cleared by `popInput` when the cursor is popped, exactly like the baseURI scope. The baseURI scoping makes a relative system ID in a declaration INSIDE the external PE (e.g. `<!ENTITY e SYSTEM "leaf.ent">` inside `sub/pe.ent`) resolve against the PE's location (`sub/leaf.ent`) via `TreeBuilder.EntityDecl`, not against the containing DTD. An empty `uri` is treated as "no override" (pushed normally). The same external-PE replacement text is also reached when a `%pe;` reference appears inside an entity value or the entity-value reference syntax check: both `decodeEntitiesInternal` and `expandEntityValueForRefCheck` (in `parser_entity_decl.go`) obtain a PE's replacement text through the shared `parameterEntityReplacement` chokepoint, which loads external-PE content the same way (and identically order-independent) instead of seeing an empty `Content()` until a top-level `%pe;` happens to cache it first.

`handlePEReference` (`parser_entity_ref.go`), called by `skipBlanks` when a `%` follows a blank run, treats a `%` immediately followed by whitespace (or end of input) as the parameter-entity DECLARATION marker of `<!ENTITY % name ...>` — never a PE reference, which is `%name;` with a NameStartChar right after `%` — and leaves it for the declaration parser in EVERY parser state. This is state-independent because the first declaration of an external subset runs before `instate` becomes `psDTD` (`parseMarkupDecl` sets `psDTD` only after a declaration), so a `<!ENTITY % ...>` PE declaration as that first declaration must not be mis-parsed as a reference.

**PE references INSIDE / ADJACENT to a markup declaration** (external subset). In the external subset a parameter-entity reference is recognized and included ANYWHERE a markup declaration occurs — not only at blank-adjacent positions between declarations — so a `%e;` may supply (or sit adjacent to) part of a declaration: `<!ATTLIST doc a1 CDATA%e;>`, `<!ATTLIST%e;a1 …>`, `<!ATTLIST %e; "v1">`, `<!ATTLIST head %common.att;>`, `<!ELEMENT head (%mix;)*>`. This mirrors libxml2's input-level `xmlSkipBlankCharsPE`. Every DTD declaration parser threads the mandatory "S" (and any PE that supplies a token) through `skipBlanksPE` (`parser_whitespace.go`) instead of a raw `isBlankByte` check, re-fetching the cursor via `dtdRefetch` after each skip: `parseAttributeListDecl`, `parseElementDecl` and the content-model parsers, plus the enumeration/notation type lists (`parseEnumerationType`/`parseNotationType`), `#FIXED` default values (`parseDefaultDecl`), `parseExternalID` (SYSTEM/PUBLIC literal separators), `parseNotationDecl`, and `parseEntityDecl` (a PE may supply an internal entity's value literal, e.g. `<!ENTITY greet %pub;>`). A PE reference is NEVER recognized inside a quoted literal (an entity value, SYSTEM/PubidLiteral, or default AttValue): those scanners treat `%` as a literal character, and `skipBlanksPE` only runs at the blank position BEFORE the opening quote, so a PE that SUPPLIES a literal is expanded while a `%` INSIDE one is not. In the external subset (`pctx.external`) `skipBlanksPE` loops: skip a bounded blank run on the top cursor; expand a `%name;` reference via `parsePEReference(ctx, pad=true)` (its §4.4.8 padding provides the separating "S"); and, when the current cursor is a pushed PE input that is spent, pop it to resume in the enclosing input — a crossed boundary counts as a consumed separator. It never pops BELOW `pctx.dtdInputFloor` (the external subset's own base DTD cursor, set in `TreeBuilder.ExternalSubset`), so it cannot drop into the main document input. Outside the external subset it delegates to `skipBlanks` so the INTERNAL subset is byte-identical: a PE must NOT supply part of a markup declaration there (WFC: PEs in Internal Subset). Because expanding/popping changes the top cursor, the declaration parsers re-fetch the cursor after each skip via `dtdRefetch(cur)` (`parserctx.go`), which re-fetches `getCursor()` only in the external subset; in the internal subset it returns the caller's cursor unchanged so an exhausted PE input is NOT auto-popped across the declaration boundary (the resulting stall surfaces the boundary violation as an error, as before). A markup declaration / content-model group that crosses a PE boundary — a closing `>` or `)` supplied by a different entity than the one that opened the `<!ATTLIST`/`<!ELEMENT` or the group's `(` — is a boundary violation (VC: Proper Declaration/Group/PE Nesting) reported as a fatal `ErrEntityBoundary`, the same condition `<!ELEMENT>` already enforces (W3C invalid E14, invalid/002, ibm P49/P50/P51). Content-model groups compare the input at the matching `)` against the input captured at the `(` (`openInput`), threaded through `parseElementMixedContentDecl`/`parseElementChildrenContentDeclPriv`.

An EXTERNAL parameter entity (`enum.ExternalParameterEntity`, e.g. `<!ENTITY % pe SYSTEM "pe.ent">`) carries no inline replacement text, so `parsePEReference` does NOT decode `entity.Content()` (empty); it instead loads the referenced external resource via `loadExternalParameterEntityContent` (in `parser_dtd_subset.go`) and pushes the RAW loaded bytes as new input, so the surrounding DTD declaration loop parses them — the same mechanism the external subset body uses, and unlike the internal-PE path it does NOT run `decodeEntities` (the loaded resource is a DTD fragment whose own references are resolved lexically during declaration parsing). The load resolves through the SAX `ResolveEntity` callback (the same path general external entities use), is byte-capped at `externalEntityMaxBytes` (10 MiB), closes the input at the read boundary, and caches the result on the entity for repeat references. It honors the secure-default gating: when XXE loading is disabled (`parseNoXXE`) or the resolver declines to open the resource, nothing is loaded and no input is pushed (behavior unchanged). The loaded bytes are charged against the amplification guard via `entityCheck(entity, len(content))` like the internal PE path. A leading TextDecl is stripped (and the body decoded) at the shared load/cache chokepoint, before both the charge and the cache (see above), so the charged/cached/pushed bytes are the post-TextDecl content; a self/mutually recursive external PE is rejected by the active-PE guard before re-pushing, and the cached resolved URI (not `e.URI()`) is reused for the baseURI scope on repeat references.

### Attribute Value Entities (`decodeEntities()`)

```
SubstitutionType: SubstituteNone(0), SubstituteRef(1), SubstitutePERef(2), SubstituteBoth(3)
```

Expands `&#NNN;`, `&#xHHH;`, `&name;`, `%name;` based on substitution type. Recursion capped at depth > 40.

`parseEntityValue()` stores the literal with general references left unexpanded, but first validates them (`validateEntityValueRefs` in `parser_entity_decl.go`): the value is PE-expanded (parameter entities and their char refs resolved) and the resulting lexical stream is scanned so a malformed general reference re-introduced through a PE (e.g. `%amp;broken` where `%amp;`→`&#38;`→`&` yields `&broken`) is rejected. Direct char refs in the literal are character data and never form a general reference with following text.

**WFC: PEs in Internal Subset** (XML §2.8). The PE-expansion that both `validateEntityValueRefs`→`expandEntityValueForRefCheck` and the real `decodeEntities(SubstitutePERef)` perform over an EntityValue is only permitted when the parser is effectively external. A RESOLVED parameter-entity reference (`%name;`) occurring inside an EntityValue literal is a fatal `ErrPEReferenceInInternalSubset` ("PEReferences forbidden in internal subset") UNLESS `effectivelyExternal()` — i.e. the reference is in the external subset OR is being read from an external parameter entity (libxml2's `PARSER_EXTERNAL` gate in `xmlExpandPEsInEntityValue`). The check lives at the `%` branch of `expandEntityValueForRefCheck`, which runs first, so an internal-subset entity value carrying a PE reference is rejected before any general-reference scan or real expansion. This forbids `%e;` WITHIN a markup declaration in the internal subset while leaving a PE reference BETWEEN declarations (handled by `parsePEReference` in the declaration loop) and the same construct in the external subset unaffected (W3C not-wf-sa-160/162, ibm-not-wf-P29-ibm29n04, ibm-not-wf-P69-ibm69n06/07 — the P69 recursive-PE cycle is unreachable because the PE reference inside the entity declaration is rejected first). An UNDECLARED PE reference (unresolved) is not flagged here, matching libxml2's early return.

Under `SubstituteEntities(false)`, a general entity referenced from an attribute value is additionally checked against the XML 1.0 attribute-value WFCs on its TRANSITIVE replacement text — "No External Entity References", "No < in Attribute Values", and the nested "Entity Declared" WFC — by `parseAttributeValueInternal` (`parser_element.go`). This is a port of libxml2's `xmlCheckEntityInAttValue`/`xmlLookupGeneralEntity`: `checkEntityInAttValue` walks the entity's replacement text and resolves each nested `&name;` reference through `lookupGeneralEntity` (`parser_entity_ref.go`), which resolves SAX getEntity FIRST then the document entity table — so the walk works for a DOM-building parse AND for a pure SAX-event parse whose custom handler (replacing the tree builder, leaving `pctx.doc` absent and the document table empty) answers `GetEntity`. It classifies the entity as `attrWFCExternal`/`attrWFCUnparsed`/`attrWFCLessThan`/`attrWFCNone` (`entity.go`); an undefined nested entity is routed to `handleUndeclaredEntity` (the "Entity Declared" WFC, W3C not-wf-sa-077: fatal `Entity '…' not defined` unless an external subset/PE reference makes it non-fatal). The walk uses an EXPLICIT work stack rather than native recursion (so a long acyclic internal-entity chain cannot grow the Go call stack), with a visited-set guarding reference cycles and bounding the walk to the distinct declared entities. The DIRECT case (the referenced entity itself external/unparsed, or its own content directly containing `<`) is rejected earlier by `parseEntityRef`; predefined entities (`&lt;` etc.) are never flagged.

The result is MEMOIZED on each walked internal entity via WFC flags (`entWFCValidated`/`entWFCChecked`, mirroring libxml2's `XML_ENT_VALIDATED`/`XML_ENT_CHECKED`): the caller runs `checkEntityInAttValue` only when the entity's flags don't already include the target set, so a repeated reference — or a nested entity shared across walks — skips the re-walk and does NOT re-emit the `getEntity` callbacks its nested lookups make (this is what keeps helium's SAX `getEntity`/`ResolveEntity` emission byte-identical to libxml2's, e.g. `test/att11.sax2`). The memoization is INDEPENDENT of `ent.checked` (the general-content check), so an entity first expanded in element content is still validated under the stricter attribute rules. The target set is context-gated on `pctx.inSubset`: in body content (`notInSubset`) an entity is fully trusted (`entWFCChecked|entWFCValidated`); inside the DTD subset a nested entity may be forward-declared, so it is only provisionally `entWFCValidated`. A DTD **attribute default value** is parsed while the DTD is still being read, so `validateAttributeDefaultsWFC` (`parser_element.go`, run from `parseDocument` once the internal+external subset are fully parsed) re-scans every stored default value via `checkAttrValueStringWFC` with the body target set — re-walking any entity a forward reference left provisionally validated — enforcing the same WFCs on the default-value DECLARATION (W3C rmt-e3e-12) regardless of whether any element uses the default.

## Tree Builder (SAX→DOM)

`TreeBuilder` implements `sax.SAX2Handler`, mapping callbacks to DOM nodes:

- `StartDocument` → create Document
- `StartElementNS` → create Element, declare namespaces, add attributes, register IDs, append to parent
- `EndElementNS` → pop element, restore parent
- `Characters` → AppendText (merges adjacent text)
- `CDataBlock` → create CDATASection
- `Comment` → create Comment (added to DTD if inSubset)
- `ProcessingInstruction` → create PI
- `InternalSubset` → create internal DTD
- `ExternalSubset` → load external DTD, parse declarations
  - Temporarily switches parser `baseURI` to the resolved DTD path while parsing the subset so entity system IDs resolve relative to the DTD file
  - An OPTIONAL leading TextDecl (`'<?xml' VersionInfo? EncodingDecl S? '?>'`) is consumed and any declared encoding honored by routing the read DTD bytes through `decodeExternalPEContent` (the same helper external parameter/general entities use) BEFORE the declaration loop, which would otherwise reject the `<?xml` as a processing instruction whose target may not be `xml`.
  - Bounded read: the DTD is read through `io.LimitReader(f, limit+1)` where `limit` is `ctx.maxExtDTDSize` (set by `Parser.MaxExternalDTDBytes`) or `MaxExternalDTDSize` (10 MiB) when unset/≤0. `fs.FileInfo.Size()` is advisory only — never used to accept or reject — because a valid `fs.FS` may stream, synthesize, under-report, or over-report size; the cap is enforced against actual bytes read. If `len(data) > limit` the load returns `ErrExternalDTDTooLarge`. The size cap is checked BEFORE the read error, so a reader that returns `n>0` with a non-EOF error on the cap-crossing read is still rejected; any other non-EOF read error (e.g. `io.ErrUnexpectedEOF`/transport failure on a partial read) is returned so a truncated subset is not silently accepted, while `io.EOF` is the normal terminator and is not an error. The declaration loop is driven by the shared `parserCtx.parseExternalSubsetDeclStep` helper (in `parser_dtd_subset.go`), used for BOTH the top-level external subset AND the body of an external `<![INCLUDE[ ... ]]>` conditional section so parameter-entity references expand identically in both. Each step does a blank-ONLY skip — NOT `skipBlanks`, whose `handlePEReference` consumes a `%pe;` reference without pushing its replacement text — then an explicit `parsePEReference` expansion (so a defaulting `<!ATTLIST>` supplied by a PE is applied, not skipped), or a `parseMarkupDecl`, or a nested `parseConditionalSections`. It snapshots the cursor position and returns `ErrDocTypeNotFinished` if a step neither advances nor errors (mirrors `parseInternalSubset`), preventing an infinite loop on malformed declarations like `<!BOGUS`; that guard error is raised via `ctx.error` WHILE the external DTD cursor and `baseURI` are still active (before the deferred cleanup restores them), so the reported location points at the external DTD source, not the main document's doctype line. Per-declaration `parseMarkupDecl` errors surface as the top-level parse error. Conditional-section errors: a malformed or miscased INCLUDE/IGNORE keyword (`ErrConditionalSectionKeyword`) is a FATAL well-formedness error (XML §3.4 P62/P63 — the keyword is case-sensitive and mandatory) and ALWAYS propagates, in both the top-level subset and nested INCLUDE bodies (W3C `ibm-not-wf-P61/P62/P63`, `o-p61fail1`/`o-p62fail1`/`o-p63fail1`, `cond01`/`cond02`); only the unterminated-section sentinel (`ErrConditionalSectionNotFinished`) is tolerated at the top level (`tolerateCondError`), mirroring the best-effort handling of a truncated/streaming external subset, and it too propagates for nested INCLUDE bodies (the INCLUDE-body caller passes `tolerateCondError=false`). The blank skip immediately after `<![` is a blank-ONLY `skipBlankRun` — NOT `skipBlanks` — so a `%pe;` that SUPPLIES the INCLUDE/IGNORE keyword (`<![ %e;` with `%e; -> "INCLUDE["`) survives for the explicit `parsePEReference` to expand rather than being consumed unexpanded (which would drop the keyword and the whole section, including a defaulting `<!ATTLIST>` in the body). Once the keyword and its `[` are consumed, the spent keyword-supplying PE cursor is popped (`popSpentExternalSubsetInputs`) to the section's own cursor BEFORE the INCLUDE-body floor (`baseLen`) is captured, so popping it later cannot drop the stack below the floor and skip the body. The conditional-section tolerance NEVER masks a resource-limit violation: `parseConditionalSections` checks `pctx.blankRunErr` after each header skip (after `<![`, after a `%pe;`, after the INCLUDE/IGNORE keyword) and returns it immediately, so an over-cap whitespace run in a conditional-section HEADER fails closed at the source instead of returning a generic `ErrConditionalSectionKeyword`/`ErrConditionalSectionNotFinished` sentinel; the `tolerateCondError` branch additionally re-checks `blankRunErr` before suppressing the tolerated `ErrConditionalSectionNotFinished` sentinel, so a tripped blank-run guard propagates as a real fatal error rather than being downgraded to "stop parsing the subset". (The IGNORE-section BODY scan advances one byte at a time — no `skipBlanks`, bounded memory — so it has no blank-run-cap gap.) Before and after the markup-decl/PE-reference handling the step pops any exhausted NESTED parameter-entity (or conditional-section) cursors via `popSpentExternalSubsetInputs` and reports `stop` once the enclosing content cursor (the pushed DTD cursor for the top-level subset, or the section's own cursor for an INCLUDE body) is exhausted, so a PE whose replacement text is (or ends with) only whitespace does not leave a Done() cursor on the stack that would break the loop and let the deferred cleanup pop the parent cursor, silently skipping declarations after the `%pe;` reference. The file is closed immediately after the bounded read, before the buffered DTD is parsed.
- `GetEntity`/`GetParameterEntity` → lookup in document entity table

### Parent Selection
1. In DTD subset → add to DTD
2. No current element → add to document
3. Current is Element → add as child

### DOM Fast Path

When the default parser is building a DOM with the internal fast path enabled:
- start-tag handling appends parser-created attributes directly in parse order instead of routing through the generic duplicate-checking setters
- common-case ID/type propagation happens inline during start-tag processing
- character data and fresh child nodes are linked directly into the current parent when parser invariants already guarantee the normal `xmlAddChild` preconditions

These shortcuts preserve the public DOM shape; they only avoid generic API work that the parser has already proven unnecessary.

## Push Parser

Both XML and HTML push parsers use the `push` package (`push.Parser[T]`), which manages a background goroutine fed by a thread-safe concurrent stream with blocking Read and non-blocking Write.

**XML PushParser** (`parser.go` + `push/`): `p.NewPushParser(ctx)` → `pp.Push(chunk)` → `doc, err := pp.Close()`

Parser runs in a background goroutine reading from the stream via `ParseReader`. Tokens are processed incrementally as data is pushed — the stream's `Read` blocks only while the buffer is empty and the stream is neither closed nor its context cancelled, then returns whatever bytes are currently available (up to `len(p)`), so the parser advances as each chunk arrives without waiting to fill a full read buffer. The stream wait is context-aware: a context watcher goroutine wakes a blocked `Read` on cancellation of the context passed to `New`/`NewPushParser`, and the `Read` returns the context error so the background parse aborts even while waiting for more pushed data (see Context Cancellation below); the watcher exits when the stream is closed.

**HTML PushParser** (`html/html.go` + `push/`): `p.NewPushParser(ctx)` → `pp.Push(chunk)` → `doc, err := pp.Close()`

A background goroutine reads from the stream via `ParseReader` (`html/html.go`). Unlike the XML push parser, HTML parsing is **not** progressive from the first byte: it becomes progressive only AFTER an initial 1024-byte (or EOF) charset prescan AND only once a *streamable* encoding has been settled, and buffers until then. `ParseReader` builds a reader from `newParserFromReader`, whose encoding chain (`wrapReaderForHTML` in `html/encoding_reader.go`) first does this prescan: a manual read loop reading up to 1024 bytes until the buffer is full, EOF, or a read error fills a `head` buffer, which **blocks until 1024 bytes have been pushed OR the stream reaches EOF** (i.e. `Close`). Consequently an input smaller than 1024 bytes is fully buffered and only parses once `Close` is called, and a larger input only begins parsing progressively once those first 1024 bytes have arrived. After the prescan, the full reader is reconstructed with `io.MultiReader(bytes.NewReader(head), r)` so the sniffed bytes are not lost, and is wrapped with newline normalization plus one of three encoding readers. Streaming after the prescan applies only when the encoding is settled: a declared/detected `charset=utf-8` stream (the `utf8SanitizeReader`) and a head with genuine non-UTF-8 bytes routed to Latin-1/Windows-1252 (the `latin1Reader`) both stream incrementally, with the parser consuming data through its streaming cursor as chunks arrive (`push.stream.Read` returns whatever bytes are currently buffered rather than waiting for a full read or EOF). An **undeclared** input whose bytes keep proving valid UTF-8 is the exception: it routes to the `deferredLatin1Reader`, which withholds (emits nothing) all undecided-but-valid-UTF-8 bytes until either EOF — then flushes them as UTF-8 — or the first genuine non-UTF-8 byte forces the whole buffered prefix to be reinterpreted as Latin-1; so an undeclared valid-UTF-8 document buffers to `Close`/EOF rather than streaming after the prescan. This withholding is bounded at the parser's configured content limit (`parseConfig.contentLimit()`, 16 MiB by default — `wrapReaderForHTML` is passed this value and threads it into `newDeferredLatin1Reader`'s `maxBuffer`) and **fails closed** at the cap: see the bounded-decision note in the encoding-detection section below. The `push` package keeps both APIs symmetric, but only XML push parsing — which has no prescan — is progressive from the start.

`ParseFile` (XML in `parser.go`, HTML in `html/html.go`) opens the file and delegates to `ParseReader` rather than reading the whole file with `os.ReadFile`, so parser limits and context cancellation apply during the read. The XML side sets the absolute path as both base URI and document URL; the HTML side sets it as the document URL.

**XML EBCDIC over a reader**: EBCDIC is not ASCII-compatible, so its XML declaration cannot be parsed at byte level; the encoding name is recovered by `encoding.ExtractEBCDICEncoding`, which scans the invariant-translated declaration in the first ~200 bytes only. `ParseReader` first checks `ctx.Err()` BEFORE touching the reader so an already-cancelled context returns the context error without ever entering a (possibly blocking, non-context-aware) `Read` — preserving the "cancelled before any blocking read" contract. It then reads the EBCDIC sniff head via a small loop (NOT `bufio.Peek`) into a 512-byte scratch buffer, re-checking `ctx.Err()` between reads so a slow reader delivering the prefix piecemeal is interrupted as soon as the parser regains control. The scratch buffer is larger than the 4-byte invariant prefix (`0x4C 0x6F 0xA7 0x94`) so a reader returning more bytes than requested in one `Read` is captured in full rather than truncated. Both sniff loops (the initial invariant-prefix read AND the prefix-extension read) treat a transient `(0, nil)` read as "wait for more", NOT "stop sniffing": a single empty read is legal (a slow producer may return no data and no error while it waits), and stopping on it would leave the prefix too short — below the 4-byte invariant for detection, or below the encoding declaration for `ExtractEBCDICEncoding` — making the parser silently default to IBM-037 and never re-switch to the declared (e.g. CP1141) EBCDIC variant. Each loop retries up to `maxSniffZeroProgressReads` (100, `parser_encoding.go`) CONSECUTIVE empty reads (mirroring the cursor fill loops' `maxZeroProgressReads`/`io.ErrNoProgress` guard), resetting the counter whenever any bytes arrive and failing fast with `io.ErrNoProgress` once the bound is hit, so a `(0, nil)`-forever reader cannot hang the parser. When the prefix matches, the head is extended (ctx-checked) up to a BOUNDED `ebcdicEncodingSniffMax` (256 bytes, `parser_encoding.go`) — enough for `ExtractEBCDICEncoding`, and the retry never grows the prefix past that cap (no unbounded buffering) — and then `head + remainder` is STREAM-decoded through the normal cursor pipeline, NOT buffered whole. The streaming `parserCtx` is marked `ebcdicStream=true` with `rawInput` set to that bounded prefix (used only for `ExtractEBCDICEncoding`). Because `rawInput` is only that prefix, `init` cannot seed `inputSize` with the real document size (it would otherwise be ~256/512 bytes); the reconstructed `head+remainder` stream is wrapped in a `countingReader` (`pctx.ebcdicConsumed`, `parserctx.go`) so the entity-amplification guard (`entityCheckLimits`) divides `sizeentcopy` by the larger of `inputSize` and the real bytes consumed so far rather than the prefix length — otherwise a legitimate large internal entity referenced exactly once (which `Parse([]byte)`, where `inputSize` is the full slice length, accepts) would FALSELY trip "maximum entity amplification factor exceeded" over `ParseReader`. The consumed count never exceeds the source's true byte length, so it is a safe lower bound; `ParseFile` (known size) additionally seeds `inputSize` directly. The `encEBCDIC` branch in `parser_document.go` therefore SKIPS the byte-slice path's cursor reset (`popInput`/`pushInput(NewByteCursor(bytes.NewReader(rawInput)))`) and instead decodes the live prefix+remainder cursor in place — detection only PEEKS, so the byte cursor still sits at the document start. Resident memory is bounded by the parser's incremental per-node content caps (`MaxNodeContentSize`, applied during char-data/CDATA/comment/PI scanning) exactly as on the non-EBCDIC path, so a large finite EBCDIC document parses under the SAME per-node limits `Parse([]byte)` applies (no total-document cap, full parity), while a hostile never-ending EBCDIC stream is bounded by those caps (e.g. an unterminated whitespace run inside `<root>` trips `ErrNodeContentTooLarge`) instead of being read whole into memory before parsing begins. The captured head bytes are prepended to the remaining stream so the common (non-EBCDIC) case streams unchanged; a non-EOF error returned alongside the head bytes is preserved via a trailing `readerReturningErr` (the underlying reader is NOT read again after it errored), so it surfaces once the buffered head drains. `io.EOF` is the normal terminator and is not re-delivered.

HTML encoding detection over a stream (`wrapReaderForHTML`) peeks the first 1024 bytes for a declared charset. The peeked head is classified with `headHasGenuineInvalidUTF8` rather than `utf8.Valid`: if byte 1024 splits an otherwise-valid multibyte rune, the trailing rune is merely incomplete (detected via `isIncompleteTrailingRune`, a valid lead byte followed only by valid continuation bytes but fewer than the lead requires) and is NOT treated as non-UTF-8 — otherwise a valid UTF-8 document whose rune straddles the sniff boundary would be misclassified as Latin-1 and corrupted (`é`→`Ã©`). Only a genuine invalid sequence routes straight to the Latin-1 path; an incomplete trailing rune falls through to the deferred reader, whose `decide()` re-reads more bytes (or EOF) to settle the encoding. When the head is valid UTF-8 with no `charset=utf-8` declaration, a `deferredLatin1Reader` buffers undecided raw bytes (emitting nothing) while they stay valid UTF-8, until either EOF is reached with everything valid — then the buffer is emitted unchanged as UTF-8 — or the first genuine non-UTF-8 byte appears, at which point the ENTIRE buffered prefix plus the remainder is reinterpreted as Latin-1/Windows-1252 via `latin1ToUTF8`. This matches the whole-document `[]byte` path (`newParser`), which decides the encoding for the whole document at once: a declared `charset=iso-8859-1` is honored BEFORE the `utf8.Valid` check (so a declared-Latin1 document whose bytes happen to be valid UTF-8 still decodes as Latin-1, matching the streaming path's immediate `latin1Reader` — without this hoist `Parse([]byte)` and `ParseReader` would diverge on such input), and otherwise a document that is not valid UTF-8 as a whole is reinterpreted as Windows-1252 in its entirety, including leading bytes that formed valid UTF-8 sequences. **Bounded-decision semantics (fail closed):** the deferred reader's withholding is capped at the parser's configured content limit (`maxBuffer`, defaulting to `parseConfig.contentLimit()` = 16 MiB). If no non-UTF-8 byte appears within that many bytes the reader cannot buffer the (possibly endless) stream whole without an unbounded-memory DoS, AND it still cannot make the exact encoding decision — a later high byte would flip the WHOLE document to Latin-1 (matching the `[]byte` path), while EOF-while-valid would keep it UTF-8. Rather than commit to one interpretation and risk silently mis-decoding later bytes (diverging from `Parse([]byte)`), `decide()` **fails closed**: it sets a sticky `capErr` (`fmt.Errorf(... %w, ErrContentSizeExceeded)`) that the reader's `Read` surfaces, emitting NO output. This keeps the memory bound and preserves parity (no irreversible mis-decoded SAX/DOM output). A real document that declares or stays in one encoding settles far below the cap and is unaffected; a legitimate multi-megabyte (e.g. ~1.1 MiB) ASCII/UTF-8 document under the content limit parses normally, and only a pathological undeclared stream that stays valid UTF-8 past the content limit is rejected. A genuine high byte landing exactly at the cap still flips the whole buffer to Windows-1252 — the switch fires while scanning, before the fail-closed cap check — so a valid Latin-1 document whose first high byte sits at the cap decodes correctly rather than being rejected. A fully-valid-UTF-8 undeclared document under the cap buffers to EOF before flushing (the `[]byte` path likewise holds the whole document in memory). The chosen encoding name (`ISO-8859-1` if `charset=iso-8859-1` was declared, else `Windows-1252`) is reported lazily after parsing via `parser.finalEncoding()`.

## Character Buffering

`deliverCharacters()` splits data into chunks respecting UTF-8 boundaries:
- Walk back from chunk boundary to find UTF-8 rune start
- If a single rune is wider than the buffer size (e.g. `CharBufferSize(1)` over multibyte text), the backward walk reaches offset 0; rather than split the rune into invalid UTF-8 fragments, it walks FORWARD to the next rune boundary and delivers that one rune whole (over budget)
- Deliver chunks via SAX Characters callback

Controlled by `Parser.CharBufferSize(size)`.

For the UTF-8 cursor fast path, character-data scanners normally continue across reader chunk boundaries before classifying the text run. This preserves CRLF normalization and prevents whitespace-only content from being split into mixed `Characters` / `IgnorableWhitespace` events at buffer edges.

**Bounded char-data scan for streaming SAX consumers.** When `CharBufferSize(size)` is set (`size > 0`) AND no DOM is being built (a custom SAX handler replaced the default `TreeBuilder`, so `pctx.treeBuilder == nil`), `parseCharDataContent` delegates to `parseCharDataChunkedSAX`, which scans and delivers a delimiter-free run in `size`-byte chunks instead of materializing the whole run first. This bounds BOTH the reusable `charBuf` and the `UTF8Cursor`'s internal read buffer (the single-shot scanner never advances `bufpos` mid-run, so `fillBuffer` would otherwise grow it to the full run length). `UTF8Cursor.ScanCharDataSlice(dst, maxBytes)` takes a byte budget: `maxBytes > 0` stops the scan on a UTF-8 boundary once that many input bytes are consumed (a lone rune wider than `maxBytes` is still returned whole to guarantee progress); `maxBytes <= 0` is the unbounded default. Context cancellation is checked between chunks. Blank-vs-text classification must match the single-shot path, which classifies the WHOLE run as one unit (`<root>  text</root>` is character data, leading blanks included; `<root>   </root>` is ignorable whitespace) — so the chunked path must NOT emit any `IgnorableWhitespace` event until the whole run is proven blank (an early per-chunk `IgnorableWhitespace` that a later non-blank byte contradicts cannot be taken back). Two cases keep this bounded without the whole-run lookahead `areBlanksBytes` needs (it peeks for the end-of-run delimiter only the final chunk can see): contextual eligibility (`whitespaceContextIgnorable` — `xml:space`, node stack, mixed-content model) is decided once; when it is false (whitespace non-ignorable here) the run is character data regardless of bytes and is streamed in fixed-size chunks via `streamCharDataChunks` (covers the unbounded-text DoS in those contexts). When it is true the leading blank run is accumulated while every byte seen is whitespace; the first non-blank byte proves character data and flushes the accumulated prefix plus the rest as `Characters` (then streams the tail), while a run that stays blank to its end is delivered as `IgnorableWhitespace` — EXCEPT an over-budget blank run. Because the buffered blank prefix would otherwise grow without bound, a still-blank prefix that exceeds the pending-whitespace budget (`blankBudget`, floored at `minPendingBlankBytes`) is, as a **documented policy**, reclassified and delivered as `Characters` rather than `IgnorableWhitespace`, keeping memory bounded; only abnormally large pure-blank runs hit this — realistic indentation is far below the budget and stays `IgnorableWhitespace`. The realistic huge run (non-blank text) commits to `Characters` on its first chunk and never accumulates; a pathological multi-megabyte run of pure whitespace is bounded by this policy, because it cannot be classified as ignorable without seeing its end. The DOM/`TreeBuilder` path stays single-shot because it classifies the whole run to drive whitespace stripping, and the DOM holds the full text node anyway. Misplaced `]]>`, control-char, and EOF handling are unchanged: the chunked loop yields control at the delimiter and the next `parseContent` dispatch re-enters char-data parsing, surfacing the same error as the single-shot path.

## Indivisible-Content Cap (`MaxNodeContentSize`)

A CDATA section, comment body, processing-instruction body, character-data
run, attribute value, internal-DTD entity-value literal, and SYSTEM/PUBLIC
external-ID literal each map to a single SAX event / DOM node / stored value and
cannot be chunked, so a giant one on untrusted input is a memory-amplification DoS.
`parserCtx.maxNodeContent`
caps the byte size of any single such run (`Parser.MaxNodeContentSize(int)`; `0` =
`DefaultMaxNodeContentSize` 10 MiB, applied by `NewParser`; negative = unlimited;
resolved via `resolveLimit`). Over-cap fails the parse with
`ErrNodeContentTooLarge` (`errors.Is`-matchable). The cap is enforced DURING
accumulation, before the whole run is buffered — mirroring the html package's
hard content cap:

- `parseCDataContent` / `parsePI` / `parseComment` (`parser_content.go`) check
  `buf.Len()` at the top of their scan loop and bail the moment the accumulated
  output exceeds the cap (strict-greater: exactly the cap is accepted). This also
  bounds `cur.PeekAt(off)` growth (and thus the cursor buffer).
- `parseEntityValueInternal` (`parser_entity_decl.go`) and
  `parseSystemLiteral` / `parsePubidLiteral` (`parser_dtd_attr.go`) — all three
  quoted-literal scanners reachable from an internal DTD (which parses by
  default) — share `scanQuotedLiteral` (`parser_dtd_attr.go`). It advances the
  cursor in fixed-size `literalScanChunk` (4096-byte) chunks instead of peeking an
  ever-growing `off` then a single `Advance`, checks `ctx` between chunks, and
  enforces the cap on `buf.Len()` after each chunk (strict-greater). This bounds
  BOTH the output buffer and the cursor's PeekAt buffer to ~cap+one chunk, so an
  unbounded entity value / system / public literal — including an unterminated one
  over an EBCDIC `ParseReader` stream (which streams through the normal cursor
  pipeline) — fails closed with `ErrNodeContentTooLarge` before any per-node cap
  could be bypassed, making the `parser.go` "EBCDIC streams bounded by parser
  caps" claim true for these DTD-reachable paths. `HasByteAt` distinguishes a
  clean unterminated-literal EOF (returned for the caller's closing-quote check)
  from a cursor read error such as a push-stream cancel (surfaced). A non-literal
  byte ends the scan WITHOUT error. The
  SYSTEM/PUBLIC call sites in `parseExternalID` route the scan error through
  `externalIDLiteralError`, which preserves `ErrNodeContentTooLarge` and
  parse-abort errors verbatim (so `errors.Is` keeps matching) instead of masking
  them with the generic "system URI required" / "public ID required" message.
- `parseCharDataContent` (`parser_element.go`) bounds the scan itself: it passes
  `nodeContentScanBudget()` (= `maxNodeContent + utf8.UTFMax`, or 0 when
  unlimited) as the `maxBytes` argument to `UTF8Cursor.ScanCharDataSlice` (fast
  path, checked via the returned consumed-byte count `i`) and to
  `Cursor.ScanCharDataInto` (non-UTF8 fallback, checked via `buf.Len()`). The
  `+utf8.UTFMax` slack guarantees a run longer than the cap is detected even when
  a multi-byte rune straddles the boundary, so a truncated run is never
  delivered. `ScanCharDataInto` takes a `maxBytes int` parameter (output-byte
  budget) across all three cursor implementations and the interface.
- `parseAttributeValueInternal` (`parser_element.go`) caps attribute values
  both ways. The `ScanSimpleAttrValue` fast path takes `nodeContentScanBudget()`
  (= `maxNodeContent + utf8.UTFMax`) as a `maxBytes` argument, bounding the
  cursor buffer instead of materializing the whole value. Because the budget
  runs `utf8.UTFMax` past the cap, a successful scan can return a `nBytes`
  slightly over the cap; the fast path therefore re-checks
  `nodeContentTooLong(nBytes)` (strict-greater) BEFORE `AdvanceFast` and returns
  `ErrNodeContentTooLarge` directly, so a `cap+1..cap+UTFMax`-byte value is
  rejected and not accepted by the fast path. A value the scan cannot settle
  within the budget returns `nBytes == 0` and falls back to the slow
  accumulating path, which routes EVERY write into its buffer through the
  bounded helpers `writeAttrString`/`writeAttrByte`/`writeAttrRune`
  (`parserctx.go`). Each helper checks the would-be length
  (`nodeContentTooLong(b.Len() + len(write))`, strict-greater) BEFORE the copy
  and returns `ErrNodeContentTooLarge` if it would exceed. This bounds all slow-
  path write paths uniformly — literal text, predefined-entity replacement,
  char-ref output, AND the non-substituted general-entity branch
  (`"&"`+`ent.name`+`";"`), whose long entity name under `MaxNameLength(-1)`
  would otherwise be copied unbounded in a single loop iteration before the next
  cap check.
- The `SubstituteEntities`/forced-namespace entity-replacement branch does NOT
  first materialize the decoded replacement and then copy it:
  `decodeEntitiesToSink` (`parser_entity_decl.go`)
  streams every output byte through an `entityDecodeSink`. The attribute path
  uses an `attrEntitySink` that normalizes attribute whitespace (TAB/CR/LF ->
  space) and writes each byte via `writeAttrByte`, so an over-cap expansion
  (`<r a="&big;"/>` with SubstituteEntities, or `xmlns:x="&big;"`) fails DURING
  decode the instant the running total would exceed the remaining attr-buffer
  budget — never building the full expansion first.
  `decodeEntitiesInternal` (the string-returning decode used everywhere else)
  shares the same core via an `entityStringSink`; a nested expansion's
  contributed size for `entityCheck` amplification accounting is the sink's
  `count()` delta across the recursive call, so
  amplification behavior matches the string-returning path.
- Entity-expansion sub-parses inherit the cap via `inheritNestedParserState`.
- The bounded streaming SAX char-data path (`parseCharDataChunkedSAX`, used when
  `CharBufferSize > 0` and no DOM is built) is EXEMPT from the char-data cap: it
  already delivers in bounded chunks, so there is no unbounded buffer to guard.
  Its CDATA/comment/PI runs are still capped by the loop scanners above.

### Blank-Run Bounding (`skipBlanks`)

Whitespace OUTSIDE any element — the prolog (between the XML declaration and the
root), between prolog `Misc` nodes, and the epilogue — is consumed by
`skipBlanks`/`skipBlankBytes` (`parser_whitespace.go`), NOT by the char-data
scanners, so the node-content cap above does not apply to it. An unbounded
whitespace run there is its own memory-amplification DoS over a streaming reader
(e.g. an XML declaration followed by infinite EBCDIC/ASCII spaces). The blank
skip therefore advances in fixed-size chunks (`blankScanChunk`, 4096) via the
shared `skipBlankRun(ctx, cur)` helper — scanning at most one chunk ahead with
`PeekAt` then `Advance`-ing it, so the cursor buffer stays bounded instead of
growing with the run — and checks `ctx.Err()` between chunks. The total run is
capped by `blankRunLimit()` (= the resolved `maxNodeContent`): a positive value
caps the run, and `0` — the unlimited sentinel `resolveLimit` produces from
`MaxNodeContentSize(-1)` — disables the blank-run cap exactly as it disables the
node-content cap. `NewParser` applies `DefaultMaxNodeContentSize` (10 MiB), so a
blank run is bounded by default and only an explicit `MaxNodeContentSize(-1)`
lifts it (trusted input only). An over-cap run returns `ErrNodeContentTooLarge`;
the `skipBlanks`/`skipBlankBytes` wrappers record it on the sticky
`parserCtx.blankRunErr` (`errors.Is`-matchable) and stop. Once `blankRunErr` is
set, `skipBlanks`/`skipBlankBytes` short-circuit (consume no more whitespace) so
no loop can spin advancing over the unbounded tail. `parseMisc`
(prolog/epilogue loop) and `parseDocument` (the pre-root skip) surface
`blankRunErr` via `ctx.error`. The DTD external-subset declaration loop and
INCLUDE conditional sections (`parser_dtd_subset.go`) call `skipBlankRun`
DIRECTLY — not `skipBlanks`, whose `handlePEReference` would consume a `%pe;`
reference without expanding its replacement text — and return its
`(bool, error)` result so an over-cap blank run there also fails with
`ErrNodeContentTooLarge`. The XML-declaration path terminates on its own
error/non-progress guard because the bounded cursor stops advancing.

**Read-error disambiguation in the blank scan (push-cancel safety).** When a
chunk scan stops short of `blankScanChunk`, a `PeekAt` of 0 at the stop position
is ambiguous: a genuine non-blank byte (possibly a real NUL), a clean EOF, OR a
recorded read failure — most importantly a push-stream `Read` that returned
`context.Canceled` when cancellation unblocked its pending wait, which the cursor
stores as a sticky `Err()`. `skipBlankRun` therefore consults `HasByteAt(0)`
(present byte vs. exhausted buffer) and, when no byte is present, surfaces
`cur.Err()` (then any pending `ctx.Err()`) through `blankRunErr` so the cursor's
read failure propagates as cancellation instead of letting a caller synthesize a
syntax error (e.g. `parseXMLDecl`'s "blank needed after '<?xml'"); `errorAtLevel`
prefers `blankRunErr` over the synthesized error. The `HasByteAt` guard is
essential: a reader may return its final bytes together with a non-EOF error, and
those buffered bytes must still be parsed before the error surfaces — so the
error is withheld while real input remains at the scan position. Both
`*strcursor.ByteCursor` (XML-declaration/DTD byte path) and the `strcursor.Cursor`
interface (post-`switchEncoding` rune path) expose `Err()` and `HasByteAt(int)`
for this; the local `blankScanner` interface in `parser_whitespace.go` requires
both.

Not every read failure flows through the blank scanner, though, so `errorAtLevel`
ALSO generalizes the same preference: when the error it is asked to report is not
already a parse-abort and `blankRunErr` is unset, it prefers a pending `ctx.Err()`
and then the active cursor's sticky read error (`cursorDecodeErr()`) over the
synthesized syntax error. This mirrors the `ctx.Err()`/`cursorDecodeErr()` gate
`parseDocument` applies at the document-end boundary, and closes the masking case
where a read failure (push-stream `context.Canceled`) lands right after `<?xml`
BEFORE its required trailing blank is read: `looksLikeXMLDecl` cannot confirm the
declaration (the sixth byte never arrives, `PeekAt(5)` is 0), so `<?xml` is
reparsed as a processing instruction whose reserved `xml` target synthesizes "XML
declaration allowed only at the start of the document" — which would otherwise
mask the real `context.Canceled`. The blank scanner is never reached on that path,
so `blankRunErr` stays nil and only the central `errorAtLevel` preference surfaces
the cancellation.

## UTF-8 Parser Fast Paths

The parser has several ASCII/UTF-8 fast paths that bypass more general rune-by-rune logic when the input shape is already known:
- `parseQName()` first tries `UTF8Cursor.ScanQNameBytes()` for common ASCII QNames and falls back to the older NCName/Name parser path for non-ASCII or malformed cases
- `parseNCName()` uses `UTF8Cursor.ScanNCNameBytes()` on ASCII input
- `parseAttributeValueInternal()` uses `UTF8Cursor.ScanSimpleAttrValue()` for non-normalized attribute values with no entities or special whitespace

These fast paths all intern scanned names before advancing the cursor, because cursor advancement may compact the buffer and invalidate borrowed byte slices.

When a UTF-8 fast path has already proven the consumed bytes contain no newlines, it advances with `AdvanceFast()` instead of the full newline-counting `Advance()` path. This is used for:
- `ConsumeString()` token consumption in the UTF-8 cursor
- ASCII QName scans in `parseQName()`
- ASCII NCName scans in `parseNCName()`
- simple attribute value scans in `parseAttributeValueInternal()`

## Name Interning

`intern.go` seeds a global map from `internal/lexicon.WellKnownNames`, but the hot path cheap-checks `(first byte, length)` before probing that map. That avoids paying the global map lookup for most document-local names that can never match a lexicon constant, while preserving the global-name-first behavior on actual candidates.

## Attribute Default Application

After parsing start tag:
1. Look up DTD defaults for element
2. Apply default `xmlns="..."` first (namespace in scope) — only if the default
   namespace was NOT explicitly declared on the same start tag
3. Apply default `xmlns:prefix="..."` next — only if that prefix (including the
   reserved `xml` prefix) was NOT explicitly declared on the same start tag
4. Apply remaining defaults (skip if explicit attr exists)

Explicit namespace declarations on the start tag always win over DTD ATTLIST
defaults; a defaulted `xmlns`/`xmlns:prefix` is applied only when the prefix is
otherwise undeclared on that element.

## Recovery Mode (RecoverOnError)

On a recoverable parse error in `parseContent()`:
1. Save error in `recoverErr`
2. Set `disableSAX=true`
3. `skipToRecoverPoint()` → advance to next `<`
4. Continue parsing
5. Return partial document + saved error

Recovery applies only to genuine parse errors (malformed content). It does NOT
apply to context cancellation — see Context Cancellation below.

## Context Cancellation (parse abort)

Context cancellation is distinct from recovery and from `StopParser`. When the
parse context is cancelled or its deadline is exceeded, the parser aborts:

- `context.Canceled` / `context.DeadlineExceeded` BYPASS recovery entirely, even
  when `RecoverOnError(true)` is set — a cancelled parse is not a recoverable
  parse error.
- Parse returns a **nil document** and the **context error** (matchable with
  `errors.Is(err, context.Canceled)` / `context.DeadlineExceeded`). It never
  returns a partial tree.
- The SAX `Error` handler is NOT invoked: a clean cancellation must not look like
  a malformed document to the handler.
- The cancellation is observed both in the normal content loop and inside the
  recovery / `skipToRecoverPoint()` skip path, so an in-progress recovery scan
  also aborts promptly.

### Cancellation boundary (between reads vs. blocked Read)

Cancellation is checked BETWEEN read operations and parse steps — before each
cursor refill and between content-loop iterations. The parser does not (and
cannot generically) interrupt a read already in progress:

- **`Parse([]byte, ...)`**: reads from an in-memory byte slice, so there is no
  blocking read. Cancellation is always observed promptly.
- **`ParseReader(io.Reader, ...)`**: cancellation is observed as soon as the
  parser regains control between reads. A reader already blocked inside its own
  `Read` cannot be unblocked generically — Go provides no way to cancel a Read in
  progress. Such a read is only interruptible if the reader itself honors the
  context or a deadline (e.g. sets a read deadline when `ctx.Done()` fires, or
  returns an error from `Read` on cancellation). To make a slow/never-returning
  reader cancellable, wrap it so its `Read` observes `ctx`, or read the bytes
  yourself and call `Parse`.
- **Push parser** (`push` package): the parser-owned stream wait IS a sync.Cond,
  not an arbitrary `io.Reader.Read`, so it is unblocked on cancellation. A watcher
  goroutine broadcasts on the cond when `ctx.Done()` fires; `stream.Read` then
  returns the context error instead of blocking for more pushed data. The watcher
  exits when the stream is closed so it does not outlive a parse on a context that
  is never cancelled.

## Early Termination

`StopParser(ctx)` → set `stopped=true`, `instate=psEOF`. Returns the parsed
document so far and a nil error (partial document, as opposed to context
cancellation which returns a nil document and the context error).

## HTML Raw-Text / RCDATA Bounding (`html/parser.go`, `html/options.go`)

The HTML parser bounds the streaming scanner's working set with `Parser.MaxContentSize` (config field `parseConfig.maxContentSize`; effective value via `contentLimit()`, which substitutes `defaultMaxContentSize` = 16 MiB when ≤ 0). The cap has two distinct meanings depending on whether the construct is chunkable.

### Soft cap (chunkable content): normal data-state text / raw-text / RCDATA / plaintext

Normal data-state character data (`parseCharacters`), raw-text (`script`/`style`/`iframe`/`xmp`, `parseRawContent`), RCDATA (`title`/`textarea`, `parseRCDATAContent`), and plaintext (`parsePlaintext`) are delivered to SAX in chunks **targeting** `contentLimit()` bytes — the scanner never buffers a whole gigantic or unterminated section. The cap is a SOFT target, not a hard limit: a section larger than the cap still parses successfully, just in multiple chunks.

- **UTF-8-aware chunk boundaries**: a chunk boundary never splits a multi-byte rune.
  - `parseRawContent`/`parsePlaintext` accumulate whole tokens into a `bytes.Buffer` and flush (clone-then-Reset) before a token would push the buffer past the cap; a whole rune is read via `peekRuneToken` and appended as one indivisible token, so a rune (or complete `U+FFFD`) is never split. A single token larger than the cap is emitted whole as its own chunk.
  - `parseCharacters`/`parseRCDATAContent` cap the plain-text run at the limit, then call `clampTextChunkToRune` to back the byte index off `isUTF8Continuation` bytes to the last whole-rune boundary; a lone rune larger than the cap is extended (not split) so a partial rune is never emitted. `parseCharacters` returns after each capped chunk and the main `parse()` loop re-enters it for the remainder.

### Hard cap (indivisible content): comments / bogus comments / PIs

`parseComment`, `parseBogusComment`, and `parsePI` map to a single indivisible SAX event / DOM node — chunking would corrupt the document (the remainder leaks as stray text). They enforce `contentLimit()` as a HARD cap: exceeding it before the terminator sets `parser.fatalErr` to a wrapped `ErrContentSizeExceeded` and aborts. The check is strict-greater (`n >= limit` only after the terminator check at offset `n`), so exactly `limit` content bytes followed by a terminator is accepted. `HasByteAt` distinguishes EOF from a real `NUL` (both `PeekAt` 0) so a NUL counts as content and cannot bypass the cap.

### Hard cap (indivisible tag-level tokens): names + intra-tag whitespace

`parseName` (tag names), `parseAttrName` (attribute names), and `skipWhitespace` (intra-tag whitespace runs) scan with a growing `PeekAt(n)` lookahead before advancing. An enormous unterminated name or a multi-megabyte whitespace run would otherwise grow the cursor buffer without bound (a DoS). Each loop enforces a HARD cap: the check `if n >= limit` sits AFTER the terminator/non-matching-byte break and BEFORE `n++`, so exactly `limit` token bytes are accepted and the `limit+1`-th sets `parser.fatalErr` to a wrapped `ErrContentSizeExceeded` (`parseName`/`parseAttrName` return `""`; `skipWhitespace` breaks and advances what it scanned). The main `parse()` loop surfaces `fatalErr` at the top of its next iteration.

The cap is `parseConfig.scanTokenLimit()` (`html/options.go`), NOT `contentLimit()`: these tokens are not chunkable content, and `MaxContentSize` is a content-chunking granularity knob callers legitimately set very small (e.g. `1`) for fine-grained streaming — binding tag names to it would reject ordinary names like `script` under `MaxContentSize(1)`. `scanTokenLimit()` is therefore floored at `defaultMaxContentSize` (16 MiB) and only grows when `MaxContentSize` is raised above it, so realistic markup is never rejected while the unbounded-`PeekAt` DoS stays bounded. (Attribute *values*, by contrast, are bounded directly by `contentLimit()` in `parseQuotedAttrValue`/`parseUnquotedAttrValue`.)

### fatalErr / ErrContentSizeExceeded surfacing

`parser.fatalErr` is set by a sub-parser on an unrecoverable condition (over-cap indivisible construct, or an over-cap unresolved char-ref literal). `parseCharRefBounded` is shared by the normal data state (`parseCharacters`) and RCDATA (`parseRCDATAContent`), so the over-cap unresolved char-ref fatal covers normal data-state text too, not just RCDATA. The main `parse()` loop checks `p.fatalErr` at the top of each iteration AND after the loop, returning it (wrapping `ErrContentSizeExceeded`, matchable with `errors.Is`). `parseRCDATAContent` returns immediately after `parseCharRefBounded` sets `fatalErr` so the main loop surfaces it rather than running on; `parseCharacters` calls it as its last action and returns, so the main loop surfaces the error on the next iteration. `fatalErr` is not the only carrier of `ErrContentSizeExceeded`: on the streaming `ParseReader`/push path the deferred-encoding reader's over-cap fail-closed (`deferredLatin1Reader.decide`, see the encoding-detection section) does NOT touch `fatalErr` — it stores a sticky `capErr` (a wrapped `ErrContentSizeExceeded`) returned from the reader's `Read`, which propagates up the cursor and is surfaced by the main loop's per-iteration `p.cur.Err()` check (line below). So `errors.Is(err, ErrContentSizeExceeded)` matches BOTH the in-band content overruns (via `fatalErr`) and the undeclared-valid-UTF-8-over-cap encoding case (via `p.cur.Err()`).

### Leading-whitespace deferral (significance + implied-`<body>` insertion)

Two normal data-state decisions can only be made once a run's first non-whitespace byte is seen, and `MaxContentSize` chunking would otherwise commit a leading whitespace prefix too early:

1. **Whitespace-significance** (`noBlanks`/StripBlanks): a run is stripped only when EVERY byte is whitespace, so a leading whitespace prefix must not be suppressed on its own.
2. **Implied-`<body>` insertion**: a run containing non-whitespace triggers `htmlStartCharData` (opening the implied `<body>`); emitting the leading whitespace before that runs would land it under `<html>` while the following text lands under `<body>`, splitting one logical run across two parents.

Both are centralized in the single chokepoint `emitCharacters` via the `parser.pendingWS` buffer. A still-undecided leading whitespace prefix is **deferred** into `pendingWS` (helper `deferPendingWS`) instead of being emitted; char-data tokens (`&` entity, real `NUL`→`U+FFFD`, lone non-markup `<`) are folded into the SAME run and do not flush it. When the first non-whitespace byte arrives, `emitCharacters` sets `curTextRunSignificant`, calls `flushPendingWS` (after the caller has established the insertion target), then emits the byte. When the run ends all-whitespace — the main `parse()` loop dispatches a real markup tag, or EOF — `flushPendingWSRunEnd` strips it under `noBlanks`, drops it before the root element, or emits it under the current element. Whitespace in `<head>` (target already correct), before the root element (ignorable), or inside raw-text/RCDATA elements (always kept) is committed directly and never enters the buffer.

`pendingWS` is bounded by `contentLimit()`: a leading whitespace prefix that reaches the cap before any non-whitespace byte establishes significance HARD-FAILS in `deferPendingWS` with a wrapped `ErrContentSizeExceeded` (set on `fatalErr`) rather than buffering unbounded — the same over-cap policy indivisible constructs use. This applies under both `StripBlanks` and the default (a 16 MiB+ contiguous whitespace prefix before any text is pathological).

Run significance (`parser.curTextRunSignificant`) is SET in `emitCharacters` on **any** non-whitespace emit — plain text chunk, resolved or unresolved char-ref / numeric-ref output, `U+FFFD`, or a lone literal `<` — and RESET only when the main loop dispatches a real markup tag. A char-ref and a lone `<` belong to the same run and never reset it, so significance established by entity output or a literal `<` keeps the run's later whitespace chunks from being suppressed.

### Context cancellation in the bounded scanners

Every bounded scanner checks `ctx.Err()` between steps so a cancelled parse aborts promptly without first draining a (possibly unbounded/unterminated) run:

- `parseComment`/`parseBogusComment`/`parsePI` loop on `ctx.Err() == nil`; on mid-scan cancellation they abort WITHOUT emitting — publishing the bytes scanned so far would leak a truncated indivisible node as stray text.
- `parseRawContent`/`parseRCDATAContent`/`parsePlaintext` check `ctx.Err()` per loop iteration (raw-text/plaintext flush the buffered chunk first; RCDATA returns).
- The char-ref helpers (`consumeNumericCharRefBounded`, `parseSaturatedCharRefLiteral`) check `ctx.Err()` between bounded chunks and unwind without emitting a partial entity.
- **Exhaustion vs. read-error disambiguation in the low-level scan helper (push-cancel safety).** `parseWhileMaxErr` returns `(chunk, err)`. A short chunk is ambiguous: `PeekAt` returns 0 BOTH at true EOF and when `fillBuffer` recorded a non-EOF read error (the push stream's blocking wait returns `context.Canceled` on cancel, which the cursor stores as a sticky `Err()`), so length alone cannot tell "run ended" from "read failed". `parseWhileMaxErr` uses `HasByteAt` to detect a short scan and then consults `p.cur.Err()`, returning a non-nil err ONLY when the scan stopped short because of a recorded read error (a genuine non-matching byte / clean EOF / full-limit fill returns nil). Every bounded char-ref caller — `parseSaturatedCharRefLiteral` (the saturated spool), `consumeNumericCharRefBounded` (digit run), and the named-entity scan in `parseCharRefBounded` — checks this err (and re-checks `ctx.Err()` / `p.cur.Err()` before the `Peek()` that settles a trailing `;`, since that peek may refill at a buffer boundary) BEFORE concluding "run ended" or emitting. On a cancel/read error they return WITHOUT any `Characters`/`CDataBlock`/partial resolution, letting the main loop surface the error. This closes the class where a cancelled push parse could emit a partial saturated-run resolution before returning `context.Canceled`.
- The main loop also surfaces `ctx.Err()` and a sticky `p.cur.Err()` (e.g. a cancelled push-stream wait) per iteration.

### Unresolved RCDATA named references charged against the cap

In RCDATA, `&` is handled by `parseCharRefBounded`, which mirrors `parseCharRef`'s exact resolution decisions while keeping memory bounded by two independent budgets:

- **Entity-name resolution** uses a FIXED `maxEntityNameLen`-byte (32) lookahead, a constant independent of `MaxContentSize` — every known entity (≤ 31 chars) and legacy prefix (≤ 6 chars) fits, so a SHORT resolvable reference whose run fits the cap is always decided in that window and never rejected for being a small name (`&amp;` resolves under `MaxContentSize(2)`).
- **The cap governs the LITERAL text emitted for an UNRESOLVED run AND the work spent settling an AMBIGUOUS legacy-prefix run.** Numeric digit runs are consumed in fixed-size chunks with overflow saturation (`consumeNumericCharRefBounded`), never buffered whole. A run that genuinely exceeds the cap before its outcome is decided sets `fatalErr` (`ErrContentSizeExceeded`) and emits NOTHING. The charged length is `"&"` + name (+ `";"` when consumed) — the exact bytes that would be emitted.

The legacy-prefix-resolves-vs-literal decision mirrors `parseCharRef`, but the bounded scanner adds a memory/work bound that changes the over-cap saturated case:

- A `;`-terminated name is NOT legacy-prefix-resolved (the prefix loop in `resolveNamedEntity` is gated on `!hasSemicolon`); an over-long `;`-terminated unknown name is emitted literally and charged against the cap.
- A no-`;` run that fits the cap resolves only its longest legacy prefix (which lies within the 32-byte head) and emits the resolution + the head's leftover followed by the tail as ordinary text.
- **`parseSaturatedCharRefLiteral` handles the saturated case (the alphanumeric run overflowed the 32-byte lookahead) with a BOUNDED SPOOL.** A saturated `&amp` + alphanumeric tail is AMBIGUOUS until the run ends: a trailing `;` makes it an over-long unknown LITERAL (no legacy resolution), its absence legacy-resolves "amp" and emits the tail. Settling that requires reaching the run's end, so nothing may be emitted before the decision. The function CONSUMES the tail in chunks bounded by both the cap and a small constant (`saturatedCharRefChunk`, 4096) into a spool, tracking the would-be literal length. If the run exceeds the cap BEFORE its end is reached it hard-fails with `ErrContentSizeExceeded` and emits nothing — it never reintroduces an unbounded non-consuming `PeekAt` lookahead. Consequently an over-cap saturated no-`;` legacy-prefix run HARD-FAILS rather than streaming an unbounded tail; only a within-cap saturated run legacy-resolves. This keeps all three invariants: peak retained memory ≤ `MaxContentSize`, NO SAX emission before the resolve-vs-literal-vs-error decision, and bytes-read work bounded (no draining of an unbounded tail). `ctx` is checked between bounded chunks.

## Key Parser Fluent Method Effects

| Method | Effect |
|--------|--------|
| StripBlanks(true) | keepBlanks=false (discard ignorable whitespace) |
| SubstituteEntities(true) | replaceEntities=true (expand entities inline; external parsed entities are replayed as full SAX node subtrees) |
| LoadExternalDTD(true) | loadsubset.Set(DetectIDs) (load external DTD; external subset system IDs resolve relative to the DTD base URI) |
| DefaultDTDAttributes(true) | loadsubset.Set(CompleteAttrs) (apply default attrs) |
| ValidateDTD(true) | validate content models after parse |
| MaxEntityAmplification(-1) | maxAmpl=0 (disable amplification ratio check; 1 GiB hard ceiling still applies) |
| MaxNameLength(-1) / MaxContentModelDepth(-1) | disable the name-length / DTD content-model-depth caps |
| MergeCDATA(true) | deliver CDATA as Characters (not CDataBlock) |
| RecoverOnError(true) | error recovery (continue on errors) |
| IgnoreEncoding(true) | don't use XML decl encoding |
| BlockXXE(true) | reject external entity loads |
| SkipIDs(true) | don't register ID attributes |
