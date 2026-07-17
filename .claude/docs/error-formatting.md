# Error Formatting

All error formatting matches libxml2 output for golden test compatibility.

## Core Error Infrastructure

### Error Types (root package)

- **`ErrorLevel`** — `ErrorLevelNone`, `ErrorLevelWarning`, `ErrorLevelError`, `ErrorLevelFatal`
- **`ErrorDomain`** — `ErrorDomainParser`, `ErrorDomainNamespace`
- **`ErrorLeveler`** interface — optional interface for errors to report their `ErrorLevel()`; default is `ErrorLevelWarning`. `ErrorCollector`'s level filter (`errorAccumulator.Handle`) reads it via `errors.As`, so a wrapped leveler in the error chain is honored, not only a top-level one
- **`NewLeveledError(msg, level)`** — factory creating error implementing `ErrorLeveler`
- **`ErrExternalDTDTooLarge`** (`errors.go`) — sentinel returned from the `ExternalSubset` bounded read when a loaded external DTD subset exceeds the byte cap (`MaxExternalDTDBytes` or default `MaxExternalDTDSize`, 10 MiB). Enforced against actual bytes read, never the advisory `fs.FileInfo.Size()`, and checked before any read error; match with `errors.Is`
- **`ErrInvalidOperation`** (`errors.go`) — sentinel for a DOM mutation the tree state forbids; wrapped via `%w` into a descriptive message, matched with `errors.Is`. Returned by: `Document.SetDocumentElement` when `root` is not an element; and `node.DeclareNamespace(prefix, uri)` on an ELEMENT node when `prefix` is in use by the element's own name (`n.ns`) or a NON-empty-prefix attribute at a URI DIFFERENT from `uri` — a genuine prefix conflict, left unchanged (the conflict check is element-scoped: a non-element node never serializes `n.ns`, so it never returns this error; this holds whether or not an nsDefs entry already exists; an empty-prefix attribute is not counted, since it never uses the default namespace and the serializer skips it; a same-URI use is not a conflict, a same-URI redeclare is a no-op, and an unused-prefix redeclare collapses the slot, all returning nil; on that collapse `DeclareNamespace` installs a fresh `Namespace` while `AddNamespaceDecl` installs the caller's object, both leaving the replaced object unmodified; `AddNamespaceDecl` applies the same rule but is void, silently declining the conflict). These methods do not reconcile a conflict introduced afterward by `SetActiveNamespace`/`SetNs`; keeping one `xmlns:prefix` per element across all mutators is a serializer-level concern.
- **`ErrNodeContentTooLarge`** (`errors.go`) — sentinel returned when a single indivisible content run — a CDATA section (`parseCDataContent`), comment body (`parseComment`), processing-instruction body (`parsePI`), character-data run (`parseCharDataContent`), or attribute value (`parseAttributeValueInternal`) — exceeds the byte cap (`MaxNodeContentSize` or default `DefaultMaxNodeContentSize`, 10 MiB). The cap fires DURING accumulation (the loop scanners check `buf.Len()` each iteration; the char-data fast/fallback scanners pass a `maxBytes` budget into `ScanCharDataSlice`/`ScanCharDataInto`; the attribute-value fast path bounds `ScanSimpleAttrValue` with the same budget and re-checks the exact count, while the slow path routes every write through cap-enforcing `writeAttr*` helpers and decodes entity replacements through a cap-checking `attrEntitySink`) so the parse fails before the whole run is buffered. The SAME cap also bounds a single contiguous run of XML whitespace: `skipBlankRun` (`parser_whitespace.go`) scans blanks in 4 KiB chunks and trips this error once the run exceeds `blankRunLimit()` (= the resolved `maxNodeContent`), so an unbounded whitespace run cannot grow the cursor buffer. This covers the prolog/epilogue/inter-root blank skips (`skipBlanks`/`skipBlankBytes`) AND the blank skips inside the external DTD subset declaration loop and INCLUDE conditional sections (`parser_dtd_subset.go`, which call `skipBlankRun` directly to preserve `%pe;` expansion). `NewParser` applies the default (secure by default); `MaxNodeContentSize(-1)` resolves `maxNodeContent` to `0` and disables BOTH the node-content and the blank-run cap. The streaming SAX char-data path (`CharBufferSize>0`) is exempt — it is already chunked. Match with `errors.Is`
- **`ErrInvalidOutputVersion`** (`errors.go`) — writer sentinel raised (via `isValidXMLVersion`) when the effective output XML version is not a valid `VersionNum` `'1.' [0-9]+`. A non-empty `OutputVersion` override is validated at the `WriteTo` entry, ahead of the node-type branch, so BOTH a Document and a bare element/fragment fail on a malformed override; a Document's own version (`Document.SetVersion`, used when there is no override) is validated at the `writeDoc` entry. Both checks run BEFORE any output byte (ahead of the transcoding-encoder setup, whose deferred flush would emit a BOM), so a value carrying a quote (markup injection into the version pseudo-attribute) or a malformed/non-`1.x` value emits nothing. Match with `errors.Is`. See `packages.md` for the full serialization-parameter description
- **`ErrElementDeclNotFound`** (`errors.go`) — sentinel returned by `Document.IsMixedElement` when neither the internal nor the external subset has an element declaration for the given name (or the document has no internal subset). Match with `errors.Is`
- **`ErrUnsupportedOutputEncoding`** (`errors.go`) — writer sentinel; the `writeDoc`-entry label-well-formedness case rejects a non-empty effective encoding that is not a valid `EncName` (`xmlchar.IsValidEncName`) before any output byte (ahead of the transcoding encoder, so no BOM leaks; guarding the encoding pseudo-attribute against markup injection), layered before the separate US-ASCII/encoder-table transcoding rejects. Full description (all cases) in `packages.md`. Match with `errors.Is`
- **Writer structural-serialization sentinels** (`errors.go`) — `ErrWriterReservedElementName`, `ErrWriterReservedAttributeName`, `ErrWriterReservedNamespacePrefix`, `ErrWriterInvalidElementName`, `ErrWriterInvalidAttributeName`, `ErrWriterInvalidNamespacePrefix`, `ErrWriterInvalidComment`, `ErrWriterInvalidPITarget`, `ErrWriterInvalidPIContent`, `ErrWriterInvalidDTDNode`. Each flags a DOM node that cannot be serialized into well-formed XML (`writer.go` `checkElementName`/`checkAttributeName`/`checkNamespacePrefix` and the comment/PI guards; `writer_dtd.go` node-type switches). The original human-readable message is preserved and the sentinel appended via `fmt.Errorf("... : %w", …)`, so a caller can distinguish the failure class with `errors.Is`. The name/comment/PI guards route through the sticky-error `check()` so an earlier I/O error is not clobbered. Full list in `packages.md`
- **`ErrUnsupportedNormalizationForm`** (`errors.go`) — writer sentinel returned by `Writer.WriteTo` when `Writer.Normalization` was given a value outside `{"", "none", "NFC", "NFD", "NFKC", "NFKD"}`. `Normalization` stores the raw form and defers the check to `WriteTo` (both the Document and bare-element paths), which fails closed before any output byte rather than silently disabling normalization. Match with `errors.Is`

### DOM operation sentinels (root package, `errors.go`)

`ErrNilNode`, `ErrInvalidOperation`, and `ErrCyclicNode` back the guarded tree API and are all matchable via `errors.Is`:

- **`ErrNilNode`** — a nil or typed-nil node (including Go's interface nil trap, e.g. the typed-nil `*Element` `Document.DocumentElement()` returns for a rootless doc) reached `AddChild`/`AddSibling`/`Replace`/`Walk`/`CopyNode`/`ParseInNodeContext`/`SetDocumentElement`.
- **`ErrInvalidOperation`** — an unsupported structural op: an empty `Replace()` (matching `Document.Replace`), a non-attribute sibling/replacement of a property attribute, or duplicate replacement operands.
- **`ErrCyclicNode`** — a `wouldCreateCycle` rejection in `AddChild`/`AddSibling`/`Replace` (inserting a node into itself or a descendant, or replacing a node with an ancestor).

The `ErrInvalidOperation`/`ErrCyclicNode` mutation sites wrap a descriptive message via `%w`, so the human text is preserved while `errors.Is` still matches.

### ErrParseError (root package, `errors.go`)

```
type ErrParseError struct {
    Column, LineNumber int
    Domain             ErrorDomain
    Err                error
    File               string       // baseURI
    Level              ErrorLevel
    Line               string       // source text context
}
```

**Format** (`FormatError()`):
```
FILE:LINE: DOMAIN SEVERITY : MESSAGE
CONTEXT_LINE
     ^
```

Domain maps: `ErrorDomainNamespace` → `"namespace"`, others → `"parser"`.
Severity maps: `ErrorLevelWarning` → `"warning"`, others → `"error"`.

### ErrorHandler (root package, `errorhandler.go`)

```
type ErrorHandler interface {
    Handle(context.Context, error)
}
```

Implementations:
- **`NilErrorHandler`** — discards all errors
- **`ErrorCollector`** — accumulates into slice via `Sink[error]`, filterable by level

**Ownership & lifecycle**: a handler is retained by reference on the component it is set on (root `Parser`; `xinclude` `Processor`; `xsd`/`relaxng`/`schematron` compilers and validators; the `catalog` `Loader`) and shared across every operation run on that configured value. `xslt3` has no `ErrorHandler` of its own — it drives the `xsd` compiler's handler internally. These are immutable-value builders, so setting a handler returns a new value and leaves the original unchanged; there is no in-place replacement. A nil handler is normalized to `NilErrorHandler` (discard) at use time — never a panic. Which errors reach the handler is component-specific: the root `Parser` consults it ONLY during DTD validation (`ValidateDTD`) — well-formedness/namespace errors surface solely as `Parse`'s returned error, not through the handler; the `xinclude` `Processor` delivers non-fatal XInclude warnings (`ErrorLevelWarning`) during `Process`/`ProcessTree`. **Close ownership differs by component.** When a handler also implements `io.Closer`, the root `Parser` (after each DTD-validating `Parse`) and the `xsd`/`relaxng`/`schematron` compilers and validators (after each `Compile`/`Validate`) close it once by calling `closeHandler` at the end of that operation, so a `Closer` handler should not be shared across those operations. The `catalog` `Loader` and the `xinclude` `Processor` do NOT close the handler — they retain it (the `Loader` for lazy delegate/`nextCatalog` loads) and the caller owns its lifecycle, closing it once the resulting value is no longer in use.

## Package-Specific Error Formatting

### XSD (`xsd/errors.go`)

| Function | Format |
|----------|--------|
| `validityError()` | `{file}:{line}: Schemas validity error : Element '{elem}': {msg}\n` |
| `validityErrorAttr()` | `{file}:{line}: Schemas validity error : Element '{elem}', attribute '{attr}': {msg}\n` |
| `schemaParserError()` | `{file}:{line}: element {local}: Schemas parser error : Element '{xsdNS}{xsdElem}': {msg}\n` |
| `schemaParserErrorAttr()` | `{file}:{line}: element {local}: Schemas parser error : Element '{xsdNS}{xsdElem}', attribute '{attr}': {msg}\n` |
| `schemaParserWarning()` | `{file}:{line}: element {local}: Schemas parser warning : Element '{xsdNS}{xsdElem}': {msg}\n` |
| `schemaComponentError()` | `{file}:{line}: element {local}: Schemas parser error : {component}: {msg}\n` |
| `schemaElemDeclError()` | `{file}:{line}: element element: Schemas parser error : element decl. '{name}': {msg}\n` |
| `schemaElemDeclErrorAttr()` | `{file}:{line}: element element: Schemas parser error : element decl. '{name}', attribute '{attr}': {msg}\n` |

### RELAX NG (`relaxng/errors.go`)

| Function | Format |
|----------|--------|
| `validityError()` | `{file}:{line}: element {name}: Relax-NG validity error : {msg}\n` |
| `bareValidityError()` | `Relax-NG validity error : {msg}\n` |
| `rngParserError()` | `Relax-NG parser error : {msg}\n` |
| `rngParserErrorAt()` | `{file}:{line}: element {elem}: Relax-NG parser error : {msg}\n` |
| `formatXMLParseError()` | Reconstructs parser error with context line + caret from `ErrParseError` |

### Schematron (`schematron/errors.go`)

| Function | Format |
|----------|--------|
| `schematronError()` | `{file}:{line}: element {elem}: schematron error : {path} line {line}: {msg}\n` |

Note: line number appears twice (prefix and message suffix).

### HTML (in `html/libxml2_compat_test.go`)

`formatHTMLErrors()` produces:
```
./test/HTML/{filename}:{line}: HTML parser error : {message}
{source context up to 80 chars}
{spaces + ^ at error column}
```

Context extraction matches libxml2's `xmlParserInputGetWindow`: skip-eol, walk back 80, forward 80, cap caret.

#### HTML sentinel errors (`html/sax.go`)

- **`ErrContentSizeExceeded`** (exported) — returned from `parse()` when an over-cap construct blows a hard cap. Wrapped with a descriptive prefix via `fmt.Errorf("... : %w", ErrContentSizeExceeded)`; callers match with `errors.Is`. The error reaches `parse()`'s caller via one of two surfacing paths, both checked at the top of (and after) the main loop: a sub-parser sets `parser.fatalErr` for an in-band content/structural overrun, OR — on the streaming `ParseReader`/push path — the over-cap deferred-encoding reader sets a sticky `capErr` whose bytes propagate up the cursor as `p.cur.Err()`. Returned in these cases:
  - A comment, bogus comment, or PI (`parseComment`/`parseBogusComment`/`parsePI`) that exceeds `MaxContentSize` before reaching its terminator. These map to a single indivisible SAX event / DOM node, so they cannot be chunked — a hard cap, not a soft chunk size. (Sets `parser.fatalErr`.)
  - An over-cap UNRESOLVED named-reference literal in normal data-state text OR the RCDATA path (`parseCharRefBounded`/`parseSaturatedCharRefLiteral`): ANY `"&"`-prefixed run that does not resolve to a known entity or legacy prefix, once the literal bytes it would emit (`"&"` + name + optional `";"`) exceed the cap — whether short, semicolon-terminated, or unbounded. Also an over-cap SATURATED ambiguous legacy-prefix run (`&amp` + a tail overflowing the 32-byte lookahead): it is consumed into a cap-bounded spool and hard-fails (emitting nothing) if the run exceeds the cap before its end is reached, because the resolve-vs-literal decision can only be made at the run's end. A SHORT resolvable reference whose run fits the cap is exempt: it is resolved to its value and never charged. (Sets `parser.fatalErr`.)
  - An over-cap leading-whitespace deferral (`deferPendingWS`) or an over-cap attribute value (`parseQuotedAttrValue`/`parseUnquotedAttrValue`, enforced per byte and covering `&`-led entity and `&#`-led numeric runs) before its significance/terminator is known. (Sets `parser.fatalErr`.)
  - An over-cap indivisible STRUCTURAL token scan — a tag name (`parseName`), end-tag name, attribute name, PUBLIC/SYSTEM DOCTYPE literal (`parseQuotedString`), or intra-tag whitespace run (`skipWhitespace`). Bounded by the `scanTokenLimit` STRUCTURAL cap, NOT `MaxContentSize`: it is FLOORED at the 16 MiB default (so a tiny `MaxContentSize` never rejects ordinary names like `script`) and grows only when `MaxContentSize` is raised above the floor. `parseStartTag`/`parseEndTag`/`parseDoctype` check `fatalErr` after EACH scanner (skipWhitespace/parseName/parseQuotedString) so an over-cap run on a streaming reader stalled at the boundary surfaces the fatal promptly — returning before any text-fallback drain — instead of issuing another blocking read. (Sets `parser.fatalErr`.)
  - An UNDECLARED-charset stream (`ParseReader`/push only) whose bytes stay valid UTF-8 past `MaxContentSize` (16 MiB default) without ever revealing their encoding (`html/encoding_reader.go` `deferredLatin1Reader.decide`): the encoding decision cannot be made within the memory bound — a later non-UTF-8 byte would flip the whole document to Windows-1252, EOF would keep it UTF-8 — so the reader fails closed rather than committing. This is NOT routed through `parser.fatalErr`; the reader stores a sticky `capErr` (`fmt.Errorf("... %w", ErrContentSizeExceeded)`) returned from its `Read`, which surfaces as `p.cur.Err()`. A declared-charset stream (incl. declared Latin-1, which `Parse([]byte)` and `ParseReader` decode identically) or one that settles below the cap is unaffected.
- **`ErrHandlerUnspecified`** (exported) — returned by a `SAXCallbacks` method whose handler is unset (see SAX-callback error routing below; filtered by `handleSAXErr`, never fatal).

#### HTML SAX-callback error routing (`html/parser.go`)

When a SAX callback (e.g. `InternalSubset`, `StartElement`, `Characters`) returns a non-nil error other than `ErrHandlerUnspecified`, `handleSAXErr` forwards it through one of two paths:

- Default (`Parser.Strict(false)`): wrapped as a warning and delivered to the SAX `Warning(err)` slot (via `emitWarning`, gated by `cfg.noWarning`). The parser continues — HTML's libxml2-style tolerance.
- `Parser.Strict(true)`: captured in `parser.fatalSAXErr` and returned from `parse()` after the parser reaches a stable state. `ErrHandlerUnspecified` is filtered in both modes.

This is distinct from `emitError` (parser-detected malformed input → `Error(err)` slot, gated by `cfg.noError`), which has been the existing convention for tokenization/structure errors.

### XSLT 3.0 (`xslt3/errors.go`)

**`XSLTError`** — structured error with W3C XSLT error code:

```
type XSLTError struct {
    Code    string
    Message string
    Cause   error
    Value   interface{}  // xsl:message body for $err:value in xsl:catch
}
```

**Format** (`Error()`): `CODE: MESSAGE` (or just `MESSAGE` when Code is empty).

Unwraps to `Cause` via `Unwrap()`. When `Cause` is an `errors.Join`, `errors.Is`
matches any joined sentinel (see `dynamicErrorCause`).

| Constructor | Usage |
|-------------|-------|
| `staticError(code, fmt, args...)` | Compile-time XSLT errors (XTSE*) |
| `dynamicError(code, fmt, args...)` | Runtime XSLT errors (XTDE*, XTTE*, etc.); `Cause = ErrDynamicError` |
| `dynamicErrorCause(code, cause, fmt, args...)` | Like `dynamicError` but `Cause = errors.Join(ErrDynamicError, cause)` so a distinguishable sentinel (e.g. `ErrResourceTooLarge`) stays observable via `errors.Is` |

**Sentinel errors** (exported):
- `ErrStaticError`, `ErrDynamicError`, `ErrCircularRef`, `ErrNoTemplate`, `ErrTerminated`, `ErrInvalidOutput`, `ErrResourceTooLarge`

**Internal sentinels**:
- `errNilStylesheet` — returned by convenience wrappers (`Transform`, `TransformString`, `TransformToWriter`) when `*Stylesheet` is nil; prevents nil-pointer panic

**Error code checker**: `isXSLTError(err, code) → bool` — unwraps and matches `XSLTError.Code`.

### Shim (`shim/compat_errors.go`)

`convertParseError()` maps helium `ErrParseError` → `encoding/xml.SyntaxError`:
- `"invalid name start char"` → `"expected element name after <"`
- Namespace errors mapped to stdlib phrasing

## Error Accumulation Pattern

### XSD

All validation errors flow through `ErrorHandler.Handle()`. No `strings.Builder` accumulation.

- `Compile()` / `CompileFile()` return `(nil, ErrCompilationFailed)` when the schema has one or more fatal diagnostics (`compileSchema` converts `c.errorCount > 0` into the sentinel after linking); the individual diagnostics are still delivered to the `ErrorHandler`. A well-formed schema returns `(schema, nil)`. This prevents callers from validating against an invalid schema. `xslt3` schema-awareness maps the sentinel to `XTSE0220`.
- `Validate()` returns `ErrValidationFailed` when the document is invalid; individual errors go to `ErrorHandler`
- `Validate()` returns `ErrNilSchema` (Validator has no compiled schema) or `ErrNilDocument` (doc is nil) before touching the document, instead of panicking. Handler setup runs first and `closeHandler()` is deferred, so a closable `ErrorHandler` is closed on every exit path (including the nil guards); a nil `ctx` is normalized to `context.Background()` at entry
- Errors sent to the handler are `*xsd.ValidationError` (extractable via `errors.As`) wrapped with an `ErrorLeveler` for transport. `ValidationError` fields: `Filename`, `Line`, `Element`, `AttributeName` (empty for element-level errors), `Message`.
- `reportValidityError` / `reportValidityErrorAttr` on `validationContext` check `suppressDepth > 0` to suppress errors during union member trials

### RelaxNG

`Validate()` returns `ErrValidationFailed` sentinel on failure. Individual errors buffered internally during validation (for backtracking), flushed to `ErrorHandler` at the end.

### Schematron

`Validate()` returns `ErrValidationFailed` sentinel on failure. Individual `*ValidationError` errors go to `ErrorHandler`. `Quiet()` suppresses error delivery to the handler.

### DTD

`Parse()` with `ValidateDTD(true)` returns `ErrDTDValidationFailed` sentinel on failure. Individual errors go to the parser's `ErrorHandler` as `*DTDValidationError` (`valid.go`), delivered by `validCtx.addf`. `DTDValidationError{Message, Level}` implements `ErrorLeveler` returning `ErrorLevelError`, so a level-filtered `ErrorCollector` classifies each diagnostic at its true severity (not the plain-error warning default); callers recover it via `errors.As`. `.Error()` returns `Message`, which is byte-identical to the old `fmt.Errorf` text (golden/error-string parity).

DTD-construction sites expose matchable sentinels (`errors.go`): `ErrNoInternalSubset` (`Document.InternalSubset()` when the document has no internal subset), `ErrDuplicateDeclaration` (a second `AddElementDecl`/`AddNotation`/`AddAttributeDecl` for an already-declared element, notation, or `(element, attribute)` pair — wrapped via `%w` into a message naming the kind and name), and `ErrInvalidArgument` (a public builder given an out-of-range enum argument — e.g. an `AddAttributeDecl` attribute type or default kind that is not a defined enum value — or a colon in an `AddNotation` notation name, the one name-grammar rule these caller-trusting builders enforce because a notation name is an XML NCName and the parser rejects a colon-bearing `<!NOTATION>` name). Match with `errors.Is`.

XSD, RelaxNG, Schematron, and DTD all use sentinel error + ErrorHandler pattern.

### XSD Validation Error Helpers (`xsd/validate.go`)

- `reportValidityError(file, line, elemName, msg)` — sends to ErrorHandler (suppressed when `suppressDepth > 0`)
- `reportValidityErrorAttr(file, line, elemName, attrName, msg)` — sends to ErrorHandler (suppressed when `suppressDepth > 0`)

### XSD Internal Types (`xsd/validate.go`)

- `validationErrors` — synchronous `ErrorHandler` that appends `err.Error()` to `[]string`; used by `ValidateElement`

### TypeDef Validation Methods (`xsd/validate.go`)

- `(*TypeDef).Validate(ctx context.Context, value string, nsMap map[string]string) error` — validates a lexical value against a simple type; uses `NilErrorHandler` (pass/fail only)
- `(*TypeDef).ValidateElement(ctx context.Context, elem *helium.Element, schema *Schema) error` — validates an element's content against the type; uses internal `validationErrors` collector for error messages

## Compilation vs Validation Errors

- **Compilation errors** — reported via `ErrorHandler.Handle(ctx, err)` during `Compile()`
- **Validation errors** — reported via `ErrorHandler.Handle(ctx, err)` during `Validate()`
- Both types partitioned in tests via `partitionCompileErrors()` (split by `ErrorLevelFatal`)
