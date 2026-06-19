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
- `fsys fs.FS` — filesystem used to load external DTDs/entities; defaults to `internal/iofs.PermissiveRoot{}` (passthrough to os.Open), overridden via `Parser.FS()`. Used by `TreeBuilder.ExternalSubset` and `TreeBuilder.ResolveEntity`. Entity sub-parsers (`parseExternalEntityPrivate`, `parseBalancedChunkInternal`) inherit the parent's `fsys`, `catalog`, and `baseURI` so nested external entities stay confined to the same sandbox/policy as the top-level parse (they would otherwise default `fsys` back to `PermissiveRoot`).

### Entity Amplification Guard
- `sizeentcopy int64` — cumulative entity expansion bytes
- `maxAmpl int` — max amplification factor (5 default, 0 with RelaxLimits(true))
- `inputSize int64` — original input size
- Rules: 1MB baseline before ratio check; 20 bytes fixed cost per entity ref

### Error Recovery
- `disableSAX bool` — suppress callbacks after fatal
- `recoverErr error` — first fatal error (RecoverOnError mode)
- `stopped bool` — StopParser() called

## Encoding Detection

Order in `detectEncoding()`:
1. UCS-4 BE/LE/2143/3412 (4-byte BOM patterns)
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
   - Max amplification: 5× input (disabled with RelaxLimits(true))
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
  - Bounded read: the DTD is read through `io.LimitReader(f, limit+1)` where `limit` is `ctx.maxExtDTDSize` (set by `Parser.MaxExternalDTDBytes`) or `MaxExternalDTDSize` (10 MiB) when unset/≤0. `fs.FileInfo.Size()` is advisory only — never used to accept or reject — because a valid `fs.FS` may stream, synthesize, under-report, or over-report size; the cap is enforced against actual bytes read. If `len(data) > limit` the load returns `ErrExternalDTDTooLarge`. The size cap is checked BEFORE the read error, so a reader that returns `n>0` with a non-EOF error on the cap-crossing read is still rejected; any other (non-cap) read error is silently ignored. The file is closed immediately after the bounded read, before the buffered DTD is parsed.
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

Parser runs in a background goroutine reading from the stream via `ParseReader`. Tokens are processed incrementally as data is pushed — the stream's `Read` blocks only while the buffer is empty and the stream is neither closed nor its context cancelled, then returns whatever bytes are currently available (up to `len(p)`), so the parser advances as each chunk arrives without waiting to fill a full read buffer. A context watcher goroutine wakes a blocked `Read` on cancellation (returning the ctx error) and exits when the stream is closed.

**HTML PushParser** (`html/html.go` + `push/`): `p.NewPushParser(ctx)` → `pp.Push(chunk)` → `doc, err := pp.Close()`

A background goroutine reads from the stream via `ParseReader` (`html/html.go`). Unlike the XML push parser, HTML parsing is **not** progressive from the first byte: it becomes progressive only AFTER an initial 1024-byte (or EOF) charset prescan, and buffers until then. `ParseReader` builds a reader from `newParserFromReader`, whose encoding chain (`wrapReaderForHTML` in `html/encoding_reader.go`) first does this prescan: `io.ReadFull(r, head)` reads up to 1024 bytes into a `head` buffer, which **blocks until 1024 bytes have been pushed OR the stream reaches EOF** (i.e. `Close`). Consequently an input smaller than 1024 bytes is fully buffered and only parses once `Close` is called, and a larger input only begins parsing progressively once those first 1024 bytes have arrived. After the prescan, the full reader is reconstructed with `io.MultiReader(bytes.NewReader(head), r)` so the sniffed bytes are not lost, and is wrapped with newline normalization plus either a Latin-1→UTF-8 converter or a UTF-8 sanitizer; the parser then consumes the remaining data through its streaming cursor incrementally as chunks arrive (`push.stream.Read` returns whatever bytes are currently buffered rather than waiting for a full read or EOF). The `push` package keeps both APIs symmetric, but only XML push parsing — which has no prescan — is progressive from the start.

## Character Buffering

`deliverCharacters()` splits data into chunks respecting UTF-8 boundaries:
- Walk back from chunk boundary to find UTF-8 rune start
- Deliver chunks via SAX Characters callback

Controlled by `Parser.SetCharBufferSize(size)`.

For the UTF-8 cursor fast path, character-data scanners now continue across reader chunk boundaries before classifying the text run. This preserves CRLF normalization and prevents whitespace-only content from being split into mixed `Characters` / `IgnorableWhitespace` events at buffer edges.

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
2. Apply default `xmlns="..."` first (namespace in scope)
3. Apply default `xmlns:prefix="..."` next
4. Apply remaining defaults (skip if explicit attr exists)

## Recovery Mode (RecoverOnError)

On error in `parseContent()`:
1. Save error in `recoverErr`
2. Set `disableSAX=true`
3. `skipToRecoverPoint()` → advance to next `<`
4. Continue parsing
5. Return partial document + saved error

## Early Termination

`StopParser(ctx)` → set `stopped=true`, `instate=psEOF`. Returns parsed document so far, nil error.

## Key Parser Fluent Method Effects

| Method | Effect |
|--------|--------|
| StripBlanks(true) | keepBlanks=false (discard ignorable whitespace) |
| SubstituteEntities(true) | replaceEntities=true (expand entities inline; external parsed entities are replayed as full SAX node subtrees) |
| LoadExternalDTD(true) | loadsubset.Set(DetectIDs) (load external DTD; external subset system IDs resolve relative to the DTD base URI) |
| DefaultDTDAttributes(true) | loadsubset.Set(CompleteAttrs) (apply default attrs) |
| ValidateDTD(true) | validate content models after parse |
| RelaxLimits(true) | maxAmpl=0 (disable amplification checks) |
| MergeCDATA(true) | deliver CDATA as Characters (not CDataBlock) |
| RecoverOnError(true) | error recovery (continue on errors) |
| IgnoreEncoding(true) | don't use XML decl encoding |
| BlockXXE(true) | reject external entity loads |
| SkipIDs(true) | don't register ID attributes |
