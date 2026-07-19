# Parser Internals

Index to the XML/HTML parse pipeline. Each concern points at the file/function
that owns it — the deep invariants (DoS caps, cancellation, spec citations) live
as doc comments AT those functions; read the code for the mechanics.

## Entry Points

- **`Parse(ctx, []byte)`** / **`ParseReader(ctx, io.Reader)`** — main entry points
- **`NewParser()`** — configurable parser; set options, SAX handler, catalog, baseURI, maxDepth, FS
- **`p := helium.NewParser(); pp := p.NewPushParser(ctx)`** — background push parser (incremental)
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
  → finalize(): XInclude substitution (if XInclude(proc) injected) then DTD validation (if ValidateDTD(true))
  → RETURN Document + error
```

## Parser Context (`parserCtx`)

Central state struct (`parserctx.go`). Key fields:

### Input Management
- `inputTab` (inputStack) — LIFO stack of ByteCursor/RuneCursor; entity expansion + external DTDs push new cursors
- `getCursor()` — current cursor; auto-pops exhausted ones, caches the active cursor between calls

### Parser State Machine
States: `psStart`, `psContent`, `psPrologue`, `psEpilogue`, `psCDATA`, `psDTD`, `psEntityDecl`, `psAttributeValue`, `psComment`, `psStartTag`, `psEndTag`, `psSystemLiteral`, `psPublicLiteral`, `psEntityValue`, `psIgnore`, `psMisc`, `psPI`, `psEOF`. State affects parsing rules (e.g. external entity refs forbidden in `psAttributeValue`; PE handling restricted in `psDTD`).

### Element/Namespace Stacks
- `nodeTab` (nodeStack) — element nesting stack
- `nsTab` (nsStack) — prefix→URI bindings; `Push`/`Lookup`/`Pop(n)`
- `nsNrTab []int` — namespace count per element level (pop exact count on close)
- `spaceTab []int` — xml:space stack (-1=inherit, 0=default, 1=preserve)

### XML 1.1 Version-Gated Relaxations
`pctx.isXML11()` gates two WF relaxations; every non-1.1 document stays byte-identical:
- Namespace prefix undeclaration (`xmlns:pfx=""`) — `parser_element.go` `validatePrefixedNamespaceDecl`
- Control-character char references — `parser_entity_ref.go` `parseCharRef` / `isXML11CharValue` (`parser_content.go`)

### SAX & Tree Building
- `sax` (sax.SAX2Handler) — callbacks (default: TreeBuilder; `Parser.SAXHandler(nil)` restores that default, so this is never nil at parse time)
- `doc *Document` — parsed document; `elem *Element` — current element

### DTD & Entities
- `attsSpecial` / `attsSpecialExternal` — DTD special-attribute normalization keying + external-markup provenance (§2.9 standalone VC); see `addSpecialAttribute` / `parseAttribute` (`parser_element.go`). `xml:id` unconditional normalization is a deliberate XPath-3.1/xml:id-§4 divergence from libxml2
- `attsDefault` — DTD default attributes
- `inSubset int` — 0=none, 1=internal, 2=external
- `replaceEntities bool` — expand entity refs (SubstituteEntities(true))
- `fsys fs.FS` — filesystem for external DTDs/entities; defaults to `internal/iofs.DenyAll{}` (safe-by-default), overridden via `Parser.FS()`; `catalogOpenName` (`tree_builder.go`) maps `file:` URIs to local paths. `openExternalResource` (`tree_builder.go`) opens through `fsys`: it tries the raw resolved name first (a `file:` URI/absolute path verbatim, so `iofs.PermissiveRoot`/os.Open and a file-URI-keyed FS are unchanged), and on an `fs.ErrInvalid` rejection — returned by `os.DirFS`/`os.Root.FS`/`fs.Sub`, never by PermissiveRoot/DenyAll — retries with the name made relative to the **fixed top-level document base**'s directory (`documentBaseURI`, captured once at parse start via `parserCtx.init` and carried through nested sub-parses by `inheritNestedParserState`, NOT the moving per-resource `baseURI`), via `catalogOpenName`→`baseRelativeFSName`, so a confined FS rooted at the document's directory resolves a SYSTEM id — including a nested resource in a subdirectory (relativized against the document root). The retry fires **only for an originally-relative reference**: `openExternalResource` takes a `retryEligible` bool gated by `systemIDRetryEligible` on the SYSTEM id **as declared** (before URI resolution / catalog mapping) — ineligible when the id is an absolute path, carries a URI scheme, **or has a colon anywhere in its first path segment** (the part before the first `/`/`\`; this catches a one-letter RFC 3986 scheme like `x:opaque` and a bare drive letter, which `HasURIScheme`'s 2-char minimum does not) — so an originally-absolute, `file:`-URI, or otherwise scheme-carrying SYSTEM id is never retried, even one naming an in-root file whose base-relative form would be a valid `fs.ValidPath` (`filepath.Rel` would re-relativize it, so eligibility, NOT the retry name's path-shape, enforces the promise). Eligibility is passed directly by `ExternalSubset`; for entity resolution the two entity loaders (`parseExternalEntityPrivate`, `loadExternalParameterEntityContent`) set `parserCtx.extRefRelative` from the entity's declared `systemID` around the `ResolveEntity` call. The supported confined-FS document base is an **absolute path or `file:` URI** (as `ParseFile` sets); a relative document base is out of scope — `BuildURI` yields a valid-but-absent relative path that fails with `fs.ErrNotExist` (not `fs.ErrInvalid`), so the retry never fires (`helium.DirFS(root)` — `internal/iofs.ConfinedDir`, which serves an in-root absolute name directly — is the general fix for an FS rooted elsewhere than the document directory). The retry name is a validated `fs.ValidPath` (leading `/` or surviving `..` disqualifies it → blocks `../`/absolute path escape; does NOT confine symlinks — `os.DirFS` follows an in-root symlink out of root, only `os.Root.FS` is symlink-safe). `openExternalResource` is the single network-guard enforcement point: `networkAccessForbidden` runs on the primary AND the retry name (returning `ErrNetworkAccessForbidden`), so a base-relative retry that becomes a network-scheme name is refused too
- Entity sub-parsers seed nested context via `inheritNestedParserState` (`parser_entity_decl.go`) — copies policy + running resource-accounting (depth, `ebcdicConsumed`); see its doc comment

### Entity Amplification Guard
- `sizeentcopy` (cumulative expansion bytes), `maxAmpl` (5 default; 0 with MaxEntityAmplification(-1)), `inputSize`
- 1 MiB baseline before ratio check; 20-byte fixed cost per entity ref; see `entityCheck` (`parser_entity_ref.go`)

### Error Recovery
- `disableSAX` — suppress callbacks after fatal; `recoverErr` — first fatal (RecoverOnError); `stopped` — StopParser()

## Encoding Detection

`detectEncoding()` order: UCS-4 BE/LE/2143/3412 (PEEK, not consume) → EBCDIC invariant prefix → UTF-8 BOM → UTF-16 BOM → UTF-16 by context → default ASCII/UTF-8. `switchEncoding()` pops the ByteCursor, pushes a RuneCursor wrapping the encoder. UTF-16 switches encoding before parsing the decl; EBCDIC extracts the name from the invariant charset (default IBM-037); ASCII-compatible parses the decl at byte level then switches.

- BOM vs declared encoding — `checkBOMEncodingConflict` (`parser_encoding.go`): a declared name resolving to a different Unicode family than a consumed BOM is fatal `ErrEncodingBOMMismatch` (§4.3.3; stricter than libxml2). Reads `pctx.declaredEncoding` (recorded unconditionally at the leaf EncName parsers), so it fires under `IgnoreEncoding`/`LenientXMLDecl`.
- Strict fixed-width decode — `withStrictDecode` (`internal/encoding/strict.go`): malformed UTF-16/32/UCS-2/4 → fatal `ErrInvalidEncodedChar`; surfaced via `UTF8Cursor.Err()` at the document-end gate.
- Strict US-ASCII decode — `asciiEncoding` (`internal/encoding/ascii.go`): any byte ≥ 0x80 → fatal `ErrInvalidASCII`.

## File Responsibilities

Each file's functions carry the spec citations (XML/Namespaces clauses, W3C test IDs) and libxml2-parity notes in their doc comments; below is the ownership map.

- `parserctx.go` — parser context, input/cursor stack, SAX dispatch, location/error reporting (`errorAtLevel` blank-run/cursor-error preference)
- `parser_document.go` — top-level pipeline (`parseDocument`, `parseContent`, recovery re-sync)
- `parser_element.go` — recursive element parsing; `parseStartTag` enforces P40/P44 inter-attribute whitespace + Namespaces §3 prefix-binding/reserved-URI WFCs (`validateDefaultNamespaceDecl`)
- `parser_xml_decl.go` / `parser_decl.go` — XML/Text declaration parsing; EncodingDecl [80]/EncName [81] enforcement (`parseEncodingName`, `parseEncodingDeclFromCursor`); name/QName helpers
- `parser_dtd_*` — DTD handling by declaration kind; `parseExternalID` found-bool (empty-vs-absent ExternalID); NotationDecl [82] / EntityDef [73] mandatory-body enforcement
- `parser_entity_decl.go` — entity declaration bodies, balanced-chunk parsing
- `parser_entity_ref.go` — references, char refs, replay, amplification checks

## Entity Expansion

Flow: `parseReference()` → `parseEntityRef()` → `entityCheck()` → parse content (`parseBalancedChunkInternal` / `parseExternalEntityPrivate`) → deliver to SAX (expand + replay node children when `replaceEntities`, else fire `Reference`). Each invariant lives at its function:

- Amplification guard — `entityCheck` / `entityCheckBytes` (`parser_entity_ref.go`); external content read through `io.LimitReader` (`externalEntityMaxBytes` 10 MiB) charged raw-only to avoid double-counting the fixed cost
- WFC Entity Declared under standalone="yes" (§4.1/§2.9) — `getEntity` / `undeclaredEntityValidityError` (`parser_entity_ref.go`)
- Balanced replacement text (WFC §4.3.2) — `parseBalancedChunkInternal` → `ErrEntityNotWellBalanced`
- External entity/DTD TextDecl handling + version check (§4.3.1/§4.3.4) — `parseExternalEntityPrivate` (`parser_entity_decl.go`), `decodeExternalPEContent` / `decodeFixedWidthExternalContent`, `checkEntityVersion`
- Document XMLDecl VersionNum constraint (§2.8) — `checkDocumentVersion` (`parser_xml_decl.go`), applied on all four document-declaration paths (`parseXMLDecl`, `parseXMLDeclFromCursor`, and both `LenientXMLDecl` variants — that option relaxes pseudo-attribute ORDER only), never on a TextDecl. `versionNumLen` (`parser_decl.go`) is the single definition of the VersionNum grammar and accepts the looser `[0-9] '.' [0-9]+` (a faithful port of libxml2 `xmlParseVersionNum`); BOTH version parsers scan through it — the byte path (`parseVersionNum`) and the rune-cursor path used for UTF-16/UCS-4 (`parseVersionInfoFromCursor`) — so the two accept exactly the same VersionNum language. The remaining constraint is enforced separately, mirroring libxml2 `xmlParseXMLDecl`: `1.0`/`1.1` pass silently; another `1.x` warns (`XML_WAR_UNKNOWN_VERSION`) and parsing continues with the declared version retained; anything outside the 1.x family is fatal `ErrUnsupportedXMLVersion` (`XML_ERR_UNKNOWN_VERSION`). libxml2 warns on `1.1` too — helium implements XML 1.1 (`isXML11`), so it stays silent there. Both halves are load-bearing for the roundtrip invariant — every version the parser accepts serializes through the `Writer`, whose `'1.' [0-9]+` check (`isValidXMLVersion`) is stricter than the grammar: the family check alone would pass a non-VersionNum like `1.x` (it has the `1.` prefix) and the grammar alone would pass `2.0`. A path enforcing only one of the two reopens the accept-then-fail-to-serialize gap
- Character-reference provenance (element-content validity, §3.2.1 E15) — `charDataFromCharRef` flag in `parseReference`; `fromCharRef` node field (see `node-types.md`)
- Parameter entity refs — `parsePEReference` (`parser_dtd_subset.go`): charges the PE's OWN replacement bytes (not post-expansion), `padPEContent` §4.4.8; external PE load via `loadExternalParameterEntityContent` (XXE-gated, base-URI-scoped, active-PE recursion guard)
- PE references inside/adjacent to markup decls (external subset) — `skipBlanksPE` (`parser_whitespace.go`) + `dtdRefetch`; boundary violations → `ErrEntityBoundary` (VC Proper Declaration/Group/PE Nesting)
- WFC PEs in Internal Subset (§2.8) — `expandEntityValueForRefCheck` `%` branch → `ErrPEReferenceInInternalSubset`
- Attribute-value entity WFCs (No External Ref / No `<` / Entity Declared) — `checkEntityInAttValue` / `lookupGeneralEntity`, memoized via `entWFCValidated`/`entWFCChecked`; DTD defaults re-scanned by `validateAttributeDefaultsWFC`
- `decodeEntities()` — SubstitutionType None(0)/Ref(1)/PERef(2)/Both(3); recursion capped at depth > 40

## Tree Builder (SAX→DOM)

`TreeBuilder` (`tree_builder.go`) implements `sax.SAX2Handler`, mapping callbacks to DOM nodes:

- `StartDocument` → create Document
- `StartElementNS` → create Element, declare namespaces, add attributes, register IDs, append to parent
- `EndElementNS` → pop element, restore parent
- `Characters` → AppendText (merges adjacent text)
- `CDataBlock` → CDATASection; `Comment` → Comment; `ProcessingInstruction` → PI
- `InternalSubset` → internal DTD
- `ExternalSubset` → load decision resolved ONCE from three independent intents (matching libxml2): load iff `parseDTDValid` (ValidateDTD) OR `DetectIDs` (LoadExternalDTD → parseDTDLoad) OR `CompleteAttrs` (DefaultDTDAttributes → parseDTDAttr) — so call order never changes whether the subset loads. The `LoadExternalDTD`/`DefaultDTDAttributes`/`ValidateDTD` setters each touch only their own option bit. A requested-but-failed `fsys.Open` (the gate passed, so loading WAS requested) emits a non-fatal `warning` (naming the resolved URI, gated by `parseNoWarning`) instead of a silent drop, then continues; under validation the missing content model surfaces downstream as `ErrDTDValidationFailed`. Then bounded read (`maxExtDTDSize`, `io.LimitReader`), baseURI scoping, TextDecl decode, PE-expanding declaration loop (`parseExternalSubsetDeclStep`), conditional sections — see the function's doc comments
- `GetEntity`/`GetParameterEntity` → document entity table lookup

Parent selection: DTD subset → add to DTD; no current element → add to document; else → append as child.

DOM fast path: when the default parser builds a DOM, start-tag attribute/ID/child linking bypasses generic duplicate-checking setters where parser invariants already guarantee the `xmlAddChild` preconditions. Public DOM shape is preserved.

## Hardening / Resource-Bound Pointers

Each cap is enforced DURING accumulation (fail-closed before the whole run buffers) and its rationale (DoS intent, spec, strict-greater edges, cancellation) lives at the function. `NewParser` applies secure defaults; the negative-sentinel option disables the cap for trusted input.

- **Node-content cap** (`MaxNodeContentSize`, 10 MiB; `resolveLimit`/`nodeContentTooLong`) — caps a single indivisible run: CDATA/comment/PI (`parseCDataContent`/`parsePI`/`parseComment`, `parser_content.go`), DTD quoted literals (`scanQuotedLiteral`, `parser_dtd_attr.go`), char data (`parseCharDataContent` via `nodeContentScanBudget` + `ScanCharDataSlice`/`ScanCharDataInto`), attribute values (`parseAttributeValueInternal` fast path + `writeAttr*`/`attrEntitySink` slow path). Over-cap → `ErrNodeContentTooLarge`. Entity sub-parses inherit the cap via `inheritNestedParserState`; the streaming-SAX char-data path is exempt (already chunked).
- **Blank-run cap** (`skipBlankRun`/`blankRunLimit`, `parser_whitespace.go`) — the same cap bounds a contiguous whitespace run in 4 KiB chunks; sticky `blankRunErr`, preferred by `errorAtLevel`. DTD subset/INCLUDE loops call `skipBlankRun` directly (not `skipBlanks`, which consumes `%pe;` unexpanded).
- **Character buffering** (`deliverCharacters`, `CharBufferSize`) — UTF-8-boundary-respecting chunking; bounded streaming-SAX char-data path `parseCharDataChunkedSAX` (no DOM built) with a documented over-budget blank-run reclassification policy.
- **UTF-8 fast paths** — `parseQName`/`parseNCName`/`parseAttributeValueInternal` try `ScanQNameBytes`/`ScanNCNameBytes`/`ScanSimpleAttrValue`, intern before advancing (advance may compact the cursor buffer, invalidating borrowed slices), and use `AdvanceFast()` when the run is proven newline-free. See `internal/strcursor/utf8cursor.go`.
- **Name interning** (`intern.go`) — global lexicon seed with a `(first byte, length)` cheap-check before the map probe.
- **Entity-amplification / external bounds** — see Entity Expansion above.

## Context Cancellation (parse abort)

`context.Canceled`/`context.DeadlineExceeded` BYPASS recovery (even under `RecoverOnError`), return a nil document + the context error (never a partial tree), and do NOT invoke the SAX `Error` handler. Checked BETWEEN reads/parse steps — an in-progress `io.Reader.Read` cannot be interrupted generically (`Parse([]byte)` has no blocking read; `ParseReader` needs a context-honoring reader; the push parser's stream wait is a `sync.Cond` unblocked by a watcher goroutine on `ctx.Done()`). Bounded scanners (blank scan, char-data, char-ref) re-check `ctx.Err()` and disambiguate exhaustion from a sticky cursor read error via `HasByteAt`/`Err()` for push-cancel safety. See `parseDocument`, `skipBlankRun`, and the `push` package.

## Push Parser

Both push parsers use the `push` package (`push.Parser[T]`): a background goroutine reading a thread-safe stream via `ParseReader`; `pp.Push(chunk)` / `doc, err := pp.Close()`.

- **XML** (`parser.go` + `push/`) — progressive from the first byte; stream `Read` returns available bytes without waiting to fill, context-aware wait (see Context Cancellation)
- **HTML** (`html/html.go` + `push/`) — progressive only AFTER the 1024-byte (or EOF) charset prescan (`wrapReaderForHTML`, `html/encoding_reader.go`) and once a streamable encoding is settled; an undeclared valid-UTF-8 stream defers to EOF/Close (`deferredLatin1Reader`, bounded by `contentLimit()`, fail-closed at the cap)
- `ParseFile` (XML/HTML) opens the file and delegates to `ParseReader` so limits + cancellation apply during the read
- **EBCDIC over a reader** (`parser_document.go` `encEBCDIC` branch) — non-ASCII-compatible; the name is recovered by `encoding.ExtractEBCDICEncoding` from a bounded sniff prefix, then STREAM-decoded (not buffered whole); `countingReader`/`ebcdicConsumed` feeds the amplification guard the real consumed-byte count. See `ParseReader` doc comments.

## HTML Content Bounding (`html/parser.go`, `html/options.go`)

`Parser.MaxContentSize` (`parseConfig.maxContentSize`, effective via `contentLimit()`, 16 MiB default). Two meanings; rationale at each function:

- **Soft cap (chunkable)**: normal data-state text (`parseCharacters`), raw-text (`parseRawContent`), RCDATA (`parseRCDATAContent`), plaintext (`parsePlaintext`) — chunk boundaries never split a rune (`clampTextChunkToRune`, `peekRuneToken`); a section over the cap still parses, in multiple chunks
- **Hard cap (indivisible)**: comments/bogus comments/PIs (`parseComment`/`parseBogusComment`/`parsePI`) → `fatalErr` wrapping `ErrContentSizeExceeded`
- **Hard cap (tag-level tokens)**: `parseName`/`parseAttrName`/`skipWhitespace` bound `PeekAt` growth against `scanTokenLimit()` (floored at 16 MiB so a small `MaxContentSize` never rejects `script`)
- **Leading-whitespace deferral** — `emitCharacters` + `pendingWS`/`deferPendingWS`: significance (`StripBlanks`) + implied-`<body>` insertion are decided at the first non-whitespace byte; `pendingWS` bounded by `contentLimit()`, fail-closed
- **Char-ref bounding** — `parseCharRefBounded` (shared by normal data + RCDATA): fixed 32-byte name lookahead; unresolved-literal / ambiguous-legacy-prefix work charged against the cap; `consumeNumericCharRefBounded` (saturating digit run), `parseSaturatedCharRefLiteral` (bounded spool, no emit before the resolve-vs-literal-vs-error decision)
- **Surfacing** — `parser.fatalErr` (in-band overruns) and `p.cur.Err()` (deferred-reader `capErr`) both match `errors.Is(err, ErrContentSizeExceeded)`; `parseWhileMaxErr` disambiguates exhaustion from a read error via `HasByteAt`. Every bounded scanner checks `ctx.Err()` between steps and unwinds without emitting a partial node.

## Attribute Default Application

After parsing a start tag (`parser_element.go`): (1) DTD lookup; (2) apply default `xmlns` first — only if the default namespace was NOT explicitly declared on the same tag; (3) apply default `xmlns:prefix` next — only if that prefix (incl. reserved `xml`) was NOT explicitly declared; (4) remaining defaults (skip if an explicit attr exists). Explicit namespace declarations always win over ATTLIST defaults.

## Recovery Mode / Early Termination

- **RecoverOnError** — on a recoverable error in `parseContent()`: save `recoverErr`, `disableSAX=true`, `skipToRecoverPoint()` (advance to next `<`), continue, return partial document + saved error. Applies to genuine parse errors only — NOT context cancellation (above).
- **StopParser(ctx)** — `stopped=true`, `instate=psEOF`; returns the parsed-so-far document + nil error (partial document, unlike cancellation's nil document + context error).

## Key Parser Fluent Method Effects

| Method | Effect |
|--------|--------|
| StripBlanks(true) | keepBlanks=false (discard ignorable whitespace) |
| SubstituteEntities(true) | replaceEntities=true (expand entities inline; external parsed entities replayed as full SAX node subtrees) |
| LoadExternalDTD(true) | loadsubset.Set(DetectIDs) (load external DTD; system IDs resolve relative to the DTD base URI) |
| DefaultDTDAttributes(true) | loadsubset.Set(CompleteAttrs) (apply default attrs) |
| ValidateDTD(true) | validate content models after parse |
| MaxEntityAmplification(-1) | maxAmpl=0 (disable amplification ratio check; 1 GiB hard ceiling still applies) |
| MaxNameLength(-1) / MaxContentModelDepth(-1) | disable the name-length / DTD content-model-depth caps |
| MaxNodeContentSize(-1) | disable the node-content + blank-run caps |
| MaxDepth(-1) | maxElemDepth=0 (disable the element-nesting cap; MaxDepth(0) selects the 256 default) |
| MergeCDATA(true) | deliver CDATA as Characters (not CDataBlock) |
| RecoverOnError(true) | error recovery (continue on errors) |
| IgnoreEncoding(true) | don't use XML decl encoding |
| BlockXXE(true) | reject external entity loads |
| SkipIDs(true) | don't register ID attributes |
