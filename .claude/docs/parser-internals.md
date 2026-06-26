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

### SAX & Tree Building
- `sax` (sax.SAX2Handler) — callbacks (default: TreeBuilder)
- `doc *Document` — parsed document
- `elem *Element` — current element

### DTD & Entities
- `attsSpecial map[string]enum.AttributeType` — special attributes from DTD
- `attsDefault map[string][]*Attribute` — default attributes from DTD
- `inSubset int` — 0=not in subset, 1=internal, 2=external
- `replaceEntities bool` — expand entity refs (set by SubstituteEntities(true))
- `fsys fs.FS` — filesystem used to load external DTDs/entities; defaults to `internal/iofs.DenyAll{}` (refuses every open — safe-by-default), overridden via `Parser.FS()` (pass `helium.PermissiveFS()` / `internal/iofs.PermissiveRoot{}` to restore os.Open passthrough). Used by `TreeBuilder.ExternalSubset` and `TreeBuilder.ResolveEntity`. When a catalog resolves an identifier to a `file:` URI, `tree_builder.go`'s `catalogOpenName` converts it to a local path via `internal/iofs.FileURIToPath` before `fsys.Open` (mirroring the XInclude `file:` handling); non-file URIs and plain paths pass through unchanged. Entity sub-parsers (`parseExternalEntityPrivate`, `parseBalancedChunkInternal`) both seed their nested context through the shared `inheritNestedParserState` helper (`parser_entity_decl.go`), which copies the parent's `sax`, `treeBuilder`, `attsDefault`, config-derived policy (`options`, `loadsubset`, `replaceEntities`, `keepBlanks`, `pedantic`, `charBufferSize`, `maxExtDTDSize`, `maxNameLength`, `maxCMDepth`, `fsys`, `catalog`, `baseURI`), and — critically for depth enforcement — BOTH `maxElemDepth` (the limit) AND the parent's current `elemDepth`. (`maxNameLength`/`maxCMDepth` are the granular limit fields that replaced the old `XML_PARSE_HUGE` bit; they must be copied here too or a configured `MaxNameLength`/`MaxContentModelDepth` would not apply inside entity expansion.) Carrying the current `elemDepth` means element nesting that crosses an entity-expansion boundary keeps accumulating toward `MaxDepth` instead of restarting at 0: without it a single substituted element (`<!ENTITY e "<a/>">` used inside `<r>&e;</r>`) would wrongly pass `MaxDepth(1)` even though the literal `<r><a/></r>` is depth 2. This applies equally to external entity replacement text. The helper does NOT touch `doc`, the `external` flag, or the amplification counters (`sizeentcopy`/`inputSize`/`maxAmpl`); each caller sets those because their lifecycle differs (document swap, external flag, counter write-back on return). Otherwise these would all reset to zero-value defaults, e.g. `maxElemDepth=0` disabling the depth check and `fsys` falling back to the safe-by-default `DenyAll`.

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

**Strict decode of fixed-width Unicode encodings**: `internal/encoding` wraps the
UTF-16/UTF-32/UCS-2/UCS-4 decoders (`withStrictDecode`, `strict.go`) so that
malformed input the base decoder would silently replace with U+FFFD (e.g. an
unpaired surrogate or trailing odd byte) becomes a fatal `ErrInvalidEncodedChar`,
while a genuinely-encoded U+FFFD still decodes. The decoder reader's error is
remembered by `UTF8Cursor` as a sticky `Err()` (a clean `Done()` would otherwise
mask it as EOF); `parseDocument` surfaces it via `cursorDecodeErr()` at the
document-end gate.

## File Responsibilities

- `parserctx.go` owns the parser context, input/cursor stack, SAX callback dispatch, location/error reporting, and other shared parser state.
- `parser_document.go` owns the top-level parse pipeline (`parseDocument`, `parseContent`, recovery re-sync).
- `parser_element.go` owns recursive element parsing, start/end tags, attributes, and character data.
- `parser_xml_decl.go` owns byte/rune XML declaration parsing; `parser_decl.go` owns the lower-level declaration token/value helpers and name/QName parsing.
- `parser_dtd_*` files split DTD handling by declaration kind instead of keeping all markup parsing in one file.
- `parser_entity_decl.go` handles entity declaration bodies and balanced-chunk parsing; `parser_entity_ref.go` handles references, char refs, replay, and amplification checks.

## Entity Expansion

### Flow (`parseReference()`)

1. `parseEntityRef()` — resolve entity name
   - Check predefined (lt, gt, amp, apos, quot)
   - SAX `GetEntity()` callback
   - Document entity table lookup
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

- **External parsed entities** (`parseExternalEntityPrivate` in `parser_entity_decl.go`): capped at `externalEntityMaxBytes` (10 MiB); content over the cap is rejected with an "exceeds maximum size" error. The resolved input is closed once the bounded read completes.
- **External DTD subset** (`TreeBuilder.ExternalSubset` in `tree_builder.go`): read through `io.LimitReader` with cap `ctx.maxExtDTDSize` (defaults to `MaxExternalDTDSize`); the read is bounded one byte past the cap so a source that under-reports its size is still caught, and the cap is enforced authoritatively against the bytes actually read (Stat size is advisory only). The file is `Close()`d immediately after the bounded read, before the buffered DTD is parsed.
4. Deliver to SAX
   - `replaceEntities=true`: expand inline and replay parsed node children through SAX (`StartElementNS`/`EndElementNS`, `Characters`, `CDataBlock`, `Comment`, `PI`)
   - `replaceEntities=false`: fire Reference callback only

### Parameter Entity References (`parsePEReference()`)

When a `%name;` parameter-entity reference in the DTD subset resolves, `parsePEReference` (in `parser_dtd_subset.go`) decodes the PE replacement text via `decodeEntities(SubstituteBoth)` and then charges the PE's OWN replacement bytes against the amplification guard with `entityCheck(entity, len(entity.Content()))` BEFORE pushing the decoded text as new input via `pushInput`. It charges `len(entity.Content())` (the PE's stored replacement text), NOT `len(decodedContent)`. This matters because `decodeEntities(SubstituteBoth)` ALREADY charges every nested entity expansion it performs — general references such as `&g;` are left literal in a PE's stored value (only PE references are substituted at declaration time) and are expanded and charged here, as is any residual parameter reference — via its own `entityCheck` calls. `decodedContent` is the result AFTER those nested expansions, so charging its length would double-count the nested bytes and could falsely reject a legitimate DTD whose `%p;` expands mostly through a nested entity (e.g. `<!ENTITY g "...big...">` plus `<!ENTITY % p "<!-- &g; -->">`). Charging `entity.Content()` accounts only the direct bytes the PE itself contributes. Without this charge the PE's direct contribution would be free, letting a small DTD reference a large PE many times to drive unbounded expansion past the amplification limit. A `%name;` that resolves to nothing (not found, in a context where that is non-fatal) still calls `entityCheck(entity, 0)` to charge the per-reference fixed cost.

### Attribute Value Entities (`decodeEntities()`)

```
SubstitutionType: SubstituteNone(0), SubstituteRef(1), SubstitutePERef(2), SubstituteBoth(3)
```

Expands `&#NNN;`, `&#xHHH;`, `&name;`, `%name;` based on substitution type. Recursion capped at depth > 40.

`parseEntityValue()` stores the literal with general references left unexpanded, but first validates them (`validateEntityValueRefs` in `parser_entity_decl.go`): the value is PE-expanded (parameter entities and their char refs resolved) and the resulting lexical stream is scanned so a malformed general reference re-introduced through a PE (e.g. `%amp;broken` where `%amp;`→`&#38;`→`&` yields `&broken`) is rejected. Direct char refs in the literal are character data and never form a general reference with following text.

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
  - Bounded read: the DTD is read through `io.LimitReader(f, limit+1)` where `limit` is `ctx.maxExtDTDSize` (set by `Parser.MaxExternalDTDBytes`) or `MaxExternalDTDSize` (10 MiB) when unset/≤0. `fs.FileInfo.Size()` is advisory only — never used to accept or reject — because a valid `fs.FS` may stream, synthesize, under-report, or over-report size; the cap is enforced against actual bytes read. If `len(data) > limit` the load returns `ErrExternalDTDTooLarge`. The size cap is checked BEFORE the read error, so a reader that returns `n>0` with a non-EOF error on the cap-crossing read is still rejected; any other non-EOF read error (e.g. `io.ErrUnexpectedEOF`/transport failure on a partial read) is returned so a truncated subset is not silently accepted, while `io.EOF` is the normal terminator and is not an error. The declaration loop is driven by the shared `parserCtx.parseExternalSubsetDeclStep` helper (in `parser_dtd_subset.go`), used for BOTH the top-level external subset AND the body of an external `<![INCLUDE[ ... ]]>` conditional section so parameter-entity references expand identically in both. Each step does a blank-ONLY skip — NOT `skipBlanks`, whose `handlePEReference` consumes a `%pe;` reference without pushing its replacement text — then an explicit `parsePEReference` expansion (so a defaulting `<!ATTLIST>` supplied by a PE is applied, not skipped), or a `parseMarkupDecl`, or a nested `parseConditionalSections`. It snapshots the cursor position and returns `ErrDocTypeNotFinished` if a step neither advances nor errors (mirrors `parseInternalSubset`), preventing an infinite loop on malformed declarations like `<!BOGUS`; that guard error is raised via `ctx.error` WHILE the external DTD cursor and `baseURI` are still active (before the deferred cleanup restores them), so the reported location points at the external DTD source, not the main document's doctype line. Per-declaration `parseMarkupDecl` errors surface as the top-level parse error; conditional-section errors are tolerated only for the top-level subset (`tolerateCondError`), and propagate for nested INCLUDE bodies. Before and after the markup-decl/PE-reference handling the step pops any exhausted NESTED parameter-entity (or conditional-section) cursors via `popSpentExternalSubsetInputs` and reports `stop` once the enclosing content cursor (the pushed DTD cursor for the top-level subset, or the section's own cursor for an INCLUDE body) is exhausted, so a PE whose replacement text is (or ends with) only whitespace does not leave a Done() cursor on the stack that would break the loop and let the deferred cleanup pop the parent cursor, silently skipping declarations after the `%pe;` reference. The file is closed immediately after the bounded read, before the buffered DTD is parsed.
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

A background goroutine reads from the stream via `ParseReader` (`html/html.go`). Unlike the XML push parser, HTML parsing is **not** progressive from the first byte: it becomes progressive only AFTER an initial 1024-byte (or EOF) charset prescan AND only once a *streamable* encoding has been settled, and buffers until then. `ParseReader` builds a reader from `newParserFromReader`, whose encoding chain (`wrapReaderForHTML` in `html/encoding_reader.go`) first does this prescan: a manual read loop reading up to 1024 bytes until the buffer is full, EOF, or a read error fills a `head` buffer, which **blocks until 1024 bytes have been pushed OR the stream reaches EOF** (i.e. `Close`). Consequently an input smaller than 1024 bytes is fully buffered and only parses once `Close` is called, and a larger input only begins parsing progressively once those first 1024 bytes have arrived. After the prescan, the full reader is reconstructed with `io.MultiReader(bytes.NewReader(head), r)` so the sniffed bytes are not lost, and is wrapped with newline normalization plus one of three encoding readers. Streaming after the prescan applies only when the encoding is settled: a declared/detected `charset=utf-8` stream (the `utf8SanitizeReader`) and a head with genuine non-UTF-8 bytes routed to Latin-1/Windows-1252 (the `latin1Reader`) both stream incrementally, with the parser consuming data through its streaming cursor as chunks arrive (`push.stream.Read` returns whatever bytes are currently buffered rather than waiting for a full read or EOF). An **undeclared** input whose bytes keep proving valid UTF-8 is the exception: it routes to the `deferredLatin1Reader`, which withholds (emits nothing) all undecided-but-valid-UTF-8 bytes until either EOF — then flushes them as UTF-8 — or the first genuine non-UTF-8 byte forces the whole buffered prefix to be reinterpreted as Latin-1; so an undeclared valid-UTF-8 document buffers to `Close`/EOF rather than streaming after the prescan. The `push` package keeps both APIs symmetric, but only XML push parsing — which has no prescan — is progressive from the start.

`ParseFile` (XML in `parser.go`, HTML in `html/html.go`) opens the file and delegates to `ParseReader` rather than reading the whole file with `os.ReadFile`, so parser limits and context cancellation apply during the read. The XML side sets the absolute path as both base URI and document URL; the HTML side sets it as the document URL.

**XML EBCDIC over a reader**: EBCDIC is not ASCII-compatible and its detection/decode (`parser_document.go`) replays the original raw bytes from `pctx.rawInput`, which the streaming reader path does not retain. `ParseReader` first checks `ctx.Err()` BEFORE touching the reader so an already-cancelled context returns the context error without ever entering a (possibly blocking, non-context-aware) `Read` — preserving the "cancelled before any blocking read" contract. It then reads the EBCDIC sniff head via a small loop (NOT `bufio.Peek`) into a 512-byte scratch buffer, re-checking `ctx.Err()` between reads so a slow reader delivering the prefix piecemeal is interrupted as soon as the parser regains control. The scratch buffer is larger than the 4-byte invariant prefix (`0x4C 0x6F 0xA7 0x94`) so a reader returning more bytes than requested in one `Read` is captured in full rather than truncated. When the prefix matches it reads the whole input and routes through `Parse([]byte)` (which sets `rawInput`), so an EBCDIC document parses identically via `ParseReader`/`ParseFile` and `Parse([]byte)`. Otherwise the captured head bytes are prepended to the remaining stream so the common (non-EBCDIC) case streams unchanged; a non-EOF error returned alongside the head bytes is preserved via a trailing `readerReturningErr` (the underlying reader is NOT read again after it errored), so it surfaces once the buffered head drains. `io.EOF` is the normal terminator and is not re-delivered.

HTML encoding detection over a stream (`wrapReaderForHTML`) peeks the first 1024 bytes for a declared charset. The peeked head is classified with `headHasGenuineInvalidUTF8` rather than `utf8.Valid`: if byte 1024 splits an otherwise-valid multibyte rune, the trailing rune is merely incomplete (detected via `isIncompleteTrailingRune`, a valid lead byte followed only by valid continuation bytes but fewer than the lead requires) and is NOT treated as non-UTF-8 — otherwise a valid UTF-8 document whose rune straddles the sniff boundary would be misclassified as Latin-1 and corrupted (`é`→`Ã©`). Only a genuine invalid sequence routes straight to the Latin-1 path; an incomplete trailing rune falls through to the deferred reader, whose `decide()` re-reads more bytes (or EOF) to settle the encoding. When the head is valid UTF-8 with no `charset=utf-8` declaration, a `deferredLatin1Reader` buffers undecided raw bytes (emitting nothing) while they stay valid UTF-8, until either EOF is reached with everything valid — then the buffer is emitted unchanged as UTF-8 — or the first genuine non-UTF-8 byte appears, at which point the ENTIRE buffered prefix plus the remainder is reinterpreted as Latin-1/Windows-1252 via `latin1ToUTF8`. This matches the whole-document `[]byte` path, which decides the encoding for the whole document at once (a document that is not valid UTF-8 as a whole is reinterpreted as Windows-1252 in its entirety, including leading bytes that formed valid UTF-8 sequences). A fully-valid-UTF-8 undeclared document buffers to EOF before flushing (the `[]byte` path likewise holds the whole document in memory). The chosen encoding name (`ISO-8859-1` if `charset=iso-8859-1` was declared, else `Windows-1252`) is reported lazily after parsing via `parser.finalEncoding()`.

## Character Buffering

`deliverCharacters()` splits data into chunks respecting UTF-8 boundaries:
- Walk back from chunk boundary to find UTF-8 rune start
- If a single rune is wider than the buffer size (e.g. `CharBufferSize(1)` over multibyte text), the backward walk reaches offset 0; rather than split the rune into invalid UTF-8 fragments, it walks FORWARD to the next rune boundary and delivers that one rune whole (over budget)
- Deliver chunks via SAX Characters callback

Controlled by `Parser.CharBufferSize(size)`.

For the UTF-8 cursor fast path, character-data scanners normally continue across reader chunk boundaries before classifying the text run. This preserves CRLF normalization and prevents whitespace-only content from being split into mixed `Characters` / `IgnorableWhitespace` events at buffer edges.

**Bounded char-data scan for streaming SAX consumers.** When `CharBufferSize(size)` is set (`size > 0`) AND no DOM is being built (a custom SAX handler replaced the default `TreeBuilder`, so `pctx.treeBuilder == nil`), `parseCharDataContent` delegates to `parseCharDataChunkedSAX`, which scans and delivers a delimiter-free run in `size`-byte chunks instead of materializing the whole run first. This bounds BOTH the reusable `charBuf` and the `UTF8Cursor`'s internal read buffer (the single-shot scanner never advances `bufpos` mid-run, so `fillBuffer` would otherwise grow it to the full run length). `UTF8Cursor.ScanCharDataSlice(dst, maxBytes)` takes a byte budget: `maxBytes > 0` stops the scan on a UTF-8 boundary once that many input bytes are consumed (a lone rune wider than `maxBytes` is still returned whole to guarantee progress); `maxBytes <= 0` is the unbounded default. Context cancellation is checked between chunks. Blank-vs-text classification must match the single-shot path, which classifies the WHOLE run as one unit (`<root>  text</root>` is character data, leading blanks included; `<root>   </root>` is ignorable whitespace) — so the chunked path must NOT emit any `IgnorableWhitespace` event until the whole run is proven blank (an early per-chunk `IgnorableWhitespace` that a later non-blank byte contradicts cannot be taken back). Two cases keep this bounded without the whole-run lookahead `areBlanksBytes` needs (it peeks for the end-of-run delimiter only the final chunk can see): contextual eligibility (`whitespaceContextIgnorable` — `xml:space`, node stack, mixed-content model) is decided once; when it is false (whitespace non-ignorable here) the run is character data regardless of bytes and is streamed in fixed-size chunks via `streamCharDataChunks` (covers the unbounded-text DoS in those contexts). When it is true the leading blank run is accumulated while every byte seen is whitespace; the first non-blank byte proves character data and flushes the accumulated prefix plus the rest as `Characters` (then streams the tail), while a run that stays blank to its end is delivered as `IgnorableWhitespace` — EXCEPT an over-budget blank run. Because the buffered blank prefix would otherwise grow without bound, a still-blank prefix that exceeds the pending-whitespace budget (`blankBudget`, floored at `minPendingBlankBytes`) is, as a **documented policy**, reclassified and delivered as `Characters` rather than `IgnorableWhitespace`, keeping memory bounded; only abnormally large pure-blank runs hit this — realistic indentation is far below the budget and stays `IgnorableWhitespace`. The realistic huge run (non-blank text) commits to `Characters` on its first chunk and never accumulates; a pathological multi-megabyte run of pure whitespace is bounded by this policy, because it cannot be classified as ignorable without seeing its end. The DOM/`TreeBuilder` path stays single-shot because it classifies the whole run to drive whitespace stripping, and the DOM holds the full text node anyway. Misplaced `]]>`, control-char, and EOF handling are unchanged: the chunked loop yields control at the delimiter and the next `parseContent` dispatch re-enters char-data parsing, surfacing the same error as the single-shot path.

## UTF-8 Parser Fast Paths

The parser has several ASCII/UTF-8 fast paths that bypass more general rune-by-rune logic when the input shape is already known:
- `parseQName()` first tries `UTF8Cursor.ScanQNameBytes()` for common ASCII QNames and falls back to the older NCName/Name parser path for non-ASCII or malformed cases
- `parseNCName()` uses `UTF8Cursor.ScanNCNameBytes()` on ASCII input
- `parseAttributeValueInternal()` uses `UTF8Cursor.ScanSimpleAttrValue()` for non-normalized attribute values with no entities or special whitespace

These fast paths all intern scanned names before advancing the cursor, because cursor advancement may compact the buffer and invalidate borrowed byte slices.

When a UTF-8 fast path has already proven the consumed bytes contain no newlines, it now advances with `AdvanceFast()` instead of the full newline-counting `Advance()` path. This is currently used for:
- `ConsumeString()` token consumption in the UTF-8 cursor
- ASCII QName scans in `parseQName()`
- ASCII NCName scans in `parseNCName()`
- simple attribute value scans in `parseAttributeValueInternal()`

## Name Interning

`intern.go` seeds a global map from `internal/lexicon.WellKnownNames`, but the hot path now cheap-checks `(first byte, length)` before probing that map. That avoids paying the global map lookup for most document-local names that can never match a lexicon constant, while preserving the existing global-name-first behavior on actual candidates.

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
  - `parseCharacters`/`parseRCDATAContent` cap the plain-text run at the limit, then call `clampTextChunkToRune` to back the byte index off `isUTF8Continuation` bytes to the last whole-rune boundary; a lone rune larger than the cap is extended (not split) so a partial rune is never emitted. `parseCharacters` returns after each capped chunk and the main `parse()` loop re-enters it for the remainder. (Before this fix, normal data-state text was the one unbounded path: a long delimiter-free run was buffered whole.)

### Hard cap (indivisible content): comments / bogus comments / PIs

`parseComment`, `parseBogusComment`, and `parsePI` map to a single indivisible SAX event / DOM node — chunking would corrupt the document (the remainder leaks as stray text). They enforce `contentLimit()` as a HARD cap: exceeding it before the terminator sets `parser.fatalErr` to a wrapped `ErrContentSizeExceeded` and aborts. The check is strict-greater (`n >= limit` only after the terminator check at offset `n`), so exactly `limit` content bytes followed by a terminator is accepted. `HasByteAt` distinguishes EOF from a real `NUL` (both `PeekAt` 0) so a NUL counts as content and cannot bypass the cap.

### fatalErr / ErrContentSizeExceeded surfacing

`parser.fatalErr` is set by a sub-parser on an unrecoverable condition (over-cap indivisible construct, or an over-cap unresolved char-ref literal). `parseCharRefBounded` is shared by the normal data state (`parseCharacters`) and RCDATA (`parseRCDATAContent`), so the over-cap unresolved char-ref fatal covers normal data-state text too, not just RCDATA. The main `parse()` loop checks `p.fatalErr` at the top of each iteration AND after the loop, returning it (wrapping `ErrContentSizeExceeded`, matchable with `errors.Is`). `parseRCDATAContent` returns immediately after `parseCharRefBounded` sets `fatalErr` so the main loop surfaces it rather than running on; `parseCharacters` calls it as its last action and returns, so the main loop surfaces the error on the next iteration.

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
- **Exhaustion vs. read-error disambiguation in the low-level scan helper (push-cancel safety).** `parseWhileMax` was removed and folded into `parseWhileMaxErr`, which returns `(chunk, err)`. A short chunk is ambiguous: `PeekAt` returns 0 BOTH at true EOF and when `fillBuffer` recorded a non-EOF read error (the push stream's blocking wait returns `context.Canceled` on cancel, which the cursor stores as a sticky `Err()`), so length alone cannot tell "run ended" from "read failed". `parseWhileMaxErr` uses `HasByteAt` to detect a short scan and then consults `p.cur.Err()`, returning a non-nil err ONLY when the scan stopped short because of a recorded read error (a genuine non-matching byte / clean EOF / full-limit fill returns nil). Every bounded char-ref caller — `parseSaturatedCharRefLiteral` (the saturated spool), `consumeNumericCharRefBounded` (digit run), and the named-entity scan in `parseCharRefBounded` — checks this err (and re-checks `ctx.Err()` / `p.cur.Err()` before the `Peek()` that settles a trailing `;`, since that peek may refill at a buffer boundary) BEFORE concluding "run ended" or emitting. On a cancel/read error they return WITHOUT any `Characters`/`CDataBlock`/partial resolution, letting the main loop surface the error. This closes the class where a cancelled push parse could emit a partial saturated-run resolution before returning `context.Canceled`.
- The main loop also surfaces `ctx.Err()` and a sticky `p.cur.Err()` (e.g. a cancelled push-stream wait) per iteration.

### Unresolved RCDATA named references charged against the cap

In RCDATA, `&` is handled by `parseCharRefBounded`, which mirrors `parseCharRef`'s exact resolution decisions while keeping memory bounded by two independent budgets:

- **Entity-name resolution** uses a FIXED `maxEntityNameLen`-byte (32) lookahead, a constant independent of `MaxContentSize` — every known entity (≤ 31 chars) and legacy prefix (≤ 6 chars) fits, so a SHORT resolvable reference whose run fits the cap is always decided in that window and never rejected for being a small name (`&amp;` resolves under `MaxContentSize(2)`).
- **The cap governs the LITERAL text emitted for an UNRESOLVED run AND the work spent settling an AMBIGUOUS legacy-prefix run.** Numeric digit runs are consumed in fixed-size chunks with overflow saturation (`consumeNumericCharRefBounded`), never buffered whole. A run that genuinely exceeds the cap before its outcome is decided sets `fatalErr` (`ErrContentSizeExceeded`) and emits NOTHING. The charged length is `"&"` + name (+ `";"` when consumed) — the exact bytes that would be emitted.

The legacy-prefix-resolves-vs-literal decision mirrors `parseCharRef`, but the bounded scanner adds a memory/work bound that changes the over-cap saturated case:

- A `;`-terminated name is NOT legacy-prefix-resolved (the prefix loop in `resolveNamedEntity` is gated on `!hasSemicolon`); an over-long `;`-terminated unknown name is emitted literally and charged against the cap.
- A no-`;` run that fits the cap resolves only its longest legacy prefix (which lies within the 32-byte head) and emits the resolution + the head's leftover followed by the tail as ordinary text.
- **`parseSaturatedCharRefLiteral` handles the saturated case (the alphanumeric run overflowed the 32-byte lookahead) with a BOUNDED SPOOL.** A saturated `&amp` + alphanumeric tail is AMBIGUOUS until the run ends: a trailing `;` makes it an over-long unknown LITERAL (no legacy resolution), its absence legacy-resolves "amp" and emits the tail. Settling that requires reaching the run's end, so nothing may be emitted before the decision. The function CONSUMES the tail in chunks bounded by both the cap and a small constant (`saturatedCharRefChunk`, 4096) into a spool, tracking the would-be literal length. If the run exceeds the cap BEFORE its end is reached it hard-fails with `ErrContentSizeExceeded` and emits nothing — it never reintroduces an unbounded non-consuming `PeekAt` lookahead (the round-19/21 memory bound). Consequently an over-cap saturated no-`;` legacy-prefix run HARD-FAILS rather than streaming an unbounded tail; only a within-cap saturated run legacy-resolves. This keeps all three invariants: peak retained memory ≤ `MaxContentSize`, NO SAX emission before the resolve-vs-literal-vs-error decision, and bytes-read work bounded (no draining of an unbounded tail). `ctx` is checked between bounded chunks.

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
