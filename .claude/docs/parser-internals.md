# Parser Internals

## Entry Points

- **`Parse(ctx, []byte)`** / **`ParseReader(ctx, io.Reader)`** — main entry points
- **`NewParser()`** — configurable parser; set options, SAX handler, catalog, baseURI, maxDepth
- **`p := helium.NewParser(); pp := p.NewPushParser(ctx)`** — background push parser (parses incrementally as data arrives)
- **`ParseInNodeContext(ctx, node, []byte)`** — parse fragment in element context

Key files: `parser.go` (API), `parserctx.go` (state machine), `tree.go` (SAX→DOM)

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
  → DTD validation (if ParseDTDValid)
  → RETURN Document + error
```

## Parser Context (`parserCtx`)

Central state struct. Key fields:

### Input Management
- `inputTab` (inputStack) — LIFO stack of ByteCursor/RuneCursor. Entity expansion and external DTDs push new cursors.
- `getCursor()` — current cursor, auto-pops exhausted ones

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
- `replaceEntities bool` — expand entity refs (set by ParseNoEnt)

### Entity Amplification Guard
- `sizeentcopy int64` — cumulative entity expansion bytes
- `maxAmpl int` — max amplification factor (5 default, 0 with ParseHuge)
- `inputSize int64` — original input size
- Rules: 1MB baseline before ratio check; 20 bytes fixed cost per entity ref

### Error Recovery
- `disableSAX bool` — suppress callbacks after fatal
- `recoverErr error` — first fatal error (ParseRecover mode)
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

## Entity Expansion

### Flow (`parseReference()`)

1. `parseEntityRef()` — resolve entity name
   - Check predefined (lt, gt, amp, apos, quot)
   - SAX `GetEntity()` callback
   - Document entity table lookup
2. `entityCheck(ent, size, replacement)` — amplification guard
   - Baseline: 1MB free
   - Fixed cost: 20 bytes per ref
   - Max amplification: 5× input (disabled with ParseHuge)
   - Already-checked entities use cached `expandedSize`
3. Parse entity content if needed (`parseBalancedChunkInternal`)
   - Recursively parse entity text
   - Seed in-scope namespaces from the surrounding element before parsing
   - Fill `ent.firstChild` (parsed nodes)
   - Mark `ent.checked = 2`, cache `ent.expandedSize`
4. Deliver to SAX
   - `replaceEntities=true`: expand inline and replay parsed node children through SAX (`StartElementNS`/`EndElementNS`, `Characters`, `CDataBlock`, `Comment`, `PI`)
   - `replaceEntities=false`: fire Reference callback only

### Attribute Value Entities (`decodeEntities()`)

```
SubstitutionType: SubstituteNone(0), SubstituteRef(1), SubstitutePERef(2), SubstituteBoth(3)
```

Expands `&#NNN;`, `&#xHHH;`, `&name;`, `%name;` based on substitution type. Recursion capped at depth > 40.

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
- `GetEntity`/`GetParameterEntity` → lookup in document entity table

### Parent Selection
1. In DTD subset → add to DTD
2. No current element → add to document
3. Current is Element → add as child

## Push Parser

`PushParser` uses background goroutine + `pushStream` (thread-safe concurrent buffer):

```
pushStream:
  Read(p []byte)  — blocks until bytes available or closed
  Write(p []byte) — non-blocking append + signal
  Close()         — mark closed, signal waiters
```

Usage: `p := helium.NewParser(); pp := p.NewPushParser(ctx)` → `pp.Push(chunk)` → `doc, err := pp.Close()`

Parser runs in background goroutine, reading from pushStream via `ParseReader`. Processes tokens incrementally as data is pushed — the stream's `Read` blocks until bytes are available, so the parser advances as each chunk arrives.

## Character Buffering

`deliverCharacters()` splits data into chunks respecting UTF-8 boundaries:
- Walk back from chunk boundary to find UTF-8 rune start
- Deliver chunks via SAX Characters callback

Controlled by `Parser.SetCharBufferSize(size)`.

## Attribute Default Application

After parsing start tag:
1. Look up DTD defaults for element
2. Apply default `xmlns="..."` first (namespace in scope)
3. Apply default `xmlns:prefix="..."` next
4. Apply remaining defaults (skip if explicit attr exists)

## Recovery Mode (ParseRecover)

On error in `parseContent()`:
1. Save error in `recoverErr`
2. Set `disableSAX=true`
3. `skipToRecoverPoint()` → advance to next `<`
4. Continue parsing
5. Return partial document + saved error

## Early Termination

`StopParser(ctx)` → set `stopped=true`, `instate=psEOF`. Returns parsed document so far, nil error.

## Key ParseOption Effects

| Flag | Effect |
|------|--------|
| ParseNoBlanks | keepBlanks=false (discard ignorable whitespace) |
| ParseNoEnt | replaceEntities=true (expand entities inline; external parsed entities are replayed as full SAX node subtrees) |
| ParseDTDLoad | loadsubset.Set(DetectIDs) (load external DTD; external subset system IDs resolve relative to the DTD base URI) |
| ParseDTDAttr | loadsubset.Set(CompleteAttrs) (apply default attrs) |
| ParseDTDValid | validate content models after parse |
| ParseHuge | maxAmpl=0 (disable amplification checks) |
| ParseNoCDATA | deliver CDATA as Characters (not CDataBlock) |
| ParseRecover | error recovery (continue on errors) |
| ParseIgnoreEnc | don't use XML decl encoding |
| ParseNoXXE | reject external entity loads |
| ParseSkipIDs | don't register ID attributes |
