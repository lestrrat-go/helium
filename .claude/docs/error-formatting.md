# Error Formatting

All error formatting matches libxml2 output for golden test compatibility.

## Core Error Infrastructure

### Error Types (root package)

- **`ErrorLevel`** — `ErrorLevelNone`, `ErrorLevelWarning`, `ErrorLevelError`, `ErrorLevelFatal`
- **`ErrorDomain`** — `ErrorDomainParser`, `ErrorDomainNamespace`
- **`ErrorLeveler`** interface — optional interface for errors to report their `ErrorLevel()`; default is `ErrorLevelWarning`
- **`NewLeveledError(msg, level)`** — factory creating error implementing `ErrorLeveler`

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

Unwraps to `Cause` via `Unwrap()`.

| Constructor | Usage |
|-------------|-------|
| `staticError(code, fmt, args...)` | Compile-time XSLT errors (XTSE*) |
| `dynamicError(code, fmt, args...)` | Runtime XSLT errors (XTDE*, XTTE*, etc.) |

**Sentinel errors** (exported):
- `ErrStaticError`, `ErrDynamicError`, `ErrCircularRef`, `ErrNoTemplate`, `ErrTerminated`, `ErrInvalidOutput`

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

- `Validate()` returns `ErrValidationFailed` on failure; individual errors go to `ErrorHandler`
- `reportValidityError` / `reportValidityErrorAttr` on `validationContext` check `suppressDepth > 0` to suppress errors during union member trials

### RelaxNG / Schematron

Errors written to `strings.Builder` AND sent to `ErrorHandler.Handle()` if configured.

### ValidateError (relaxng, schematron only)

```
type ValidateError struct {
    Output string             // full libxml2-compatible formatted output
    Errors []ValidationError  // structured per-error details (xsd, relaxng)
}
func (e *ValidateError) Error() string { return e.Output }
```

XSD uses `ErrValidationFailed` sentinel instead. Individual errors go to ErrorHandler.

### XSD Validation Error Helpers (`xsd/validate_context.go`)

- `reportValidityError(file, line, elemName, msg)` — sends to ErrorHandler (suppressed when `suppressDepth > 0`)
- `reportValidityErrorAttr(file, line, elemName, attrName, msg)` — sends to ErrorHandler (suppressed when `suppressDepth > 0`)

### XSD Internal Types (`xsd/validate_context.go`)

- `validationErrors` — synchronous `ErrorHandler` that appends `err.Error()` to `[]string`; used by `ValidateElement`

### TypeDef Validation Methods (`xsd/typedef_validate.go`)

- `(*TypeDef).Validate(value string, nsMap map[string]string) error` — validates a lexical value against a simple type; uses `NilErrorHandler` (pass/fail only)
- `(*TypeDef).ValidateElement(elem *helium.Element, schema *Schema) error` — validates an element's content against the type; uses internal `validationErrors` collector for error messages

## Compilation vs Validation Errors

- **Compilation errors** — reported via `ErrorHandler.Handle(ctx, err)` during `Compile()`
- **Validation errors** — reported via `ErrorHandler.Handle(ctx, err)` during `Validate()`
- Both types partitioned in tests via `partitionCompileErrors()` (split by `ErrorLevelFatal`)
