# Error Formatting

All error formatting matches libxml2 output for golden test compatibility.

## Core Error Infrastructure

### Error Types (root package)

- **`ErrorLevel`** — `ErrorLevelNone`, `ErrorLevelWarning`, `ErrorLevelError`, `ErrorLevelFatal`
- **`ErrorDomain`** — `ErrorDomainParser`, `ErrorDomainNamespace`
- **`ErrorLeveler`** interface — optional interface for errors to report their `ErrorLevel()`; default is `ErrorLevelWarning`
- **`NewLeveledError(msg, level)`** — factory creating error implementing `ErrorLeveler`
- **`ErrExternalDTDTooLarge`** (`errors.go`) — sentinel returned from the `ExternalSubset` bounded read when a loaded external DTD subset exceeds the byte cap (`MaxExternalDTDBytes` or default `MaxExternalDTDSize`, 10 MiB). Enforced against actual bytes read, never the advisory `fs.FileInfo.Size()`, and checked before any read error; match with `errors.Is`

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

- **`ErrContentSizeExceeded`** (exported) — returned from `parse()` (via `parser.fatalErr`) when a single content section blows the `Parser.MaxContentSize` hard cap. Wrapped with a descriptive prefix via `fmt.Errorf("... : %w", ErrContentSizeExceeded)`; callers match with `errors.Is`. Returned in two cases:
  - A comment, bogus comment, or PI (`parseComment`/`parseBogusComment`/`parsePI`) that exceeds `MaxContentSize` before reaching its terminator. These map to a single indivisible SAX event / DOM node, so they cannot be chunked — a hard cap, not a soft chunk size.
  - An over-cap UNRESOLVED RCDATA named-reference literal (`parseCharRefBounded`/`parseSaturatedCharRefLiteral`): ANY `"&"`-prefixed run that does not resolve to a known entity or legacy prefix, once the literal bytes it would emit (`"&"` + name + optional `";"`) exceed the cap — whether short, semicolon-terminated, or unbounded. Also an over-cap SATURATED ambiguous legacy-prefix run (`&amp` + a tail overflowing the 32-byte lookahead): it is consumed into a cap-bounded spool and hard-fails (emitting nothing) if the run exceeds the cap before its end is reached, because the resolve-vs-literal decision can only be made at the run's end. A SHORT resolvable reference whose run fits the cap is exempt: it is resolved to its value and never charged.
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

`Parse()` with `ValidateDTD(true)` returns `ErrDTDValidationFailed` sentinel on failure. Individual errors go to the parser's `ErrorHandler`.

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
