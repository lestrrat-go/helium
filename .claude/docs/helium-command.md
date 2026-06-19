# helium command

Unified CLI entrypoint. Wrapper file: `cmd/helium/main.go`. Implementation package: `internal/cli/heliumcmd`.

User-facing command docs live in `cmd/helium/README.md`.

## Entry API

- `heliumcmd.Execute(ctx, args)` is package entrypoint
- `heliumcmd.WithIO(ctx, stdin, stdout, stderr)` injects stdio for tests/examples
- `heliumcmd.WithStdinTTY(ctx, bool)` overrides stdin TTY detection for tests/examples
- When no CLI-specific values exist on `ctx`, defaults are `os.Stdin`, `os.Stdout`, `os.Stderr`, + TTY detection from `os.Stdin`

## Command Tree

| Command | Purpose |
|---------|---------|
| `helium lint` | Parse/lint XML with xmllint-style flags |
| `helium xpath` | Evaluate XPath expression against XML input |
| `helium xsd validate` | Validate XML against XSD schema |
| `helium relaxng validate` | Validate XML against RELAX NG schema |
| `helium schematron validate` | Validate XML against Schematron schema |
| `helium xslt` | Transform XML with XSLT 3.0 stylesheet |

## Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | `ExitOK` | Success |
| 1 | `ExitErr` | Generic error / usage error |
| 3 | `ExitValidation` | Validation failed |
| 4 | `ExitReadFile` | File/stdin read error |
| 5 | `ExitSchemaComp` | Schema compilation error |
| 10 | `ExitXPath` | XPath compile/evaluation error |
| 11 | `ExitXSLT` | XSLT compile/transform error |

Multiple XML inputs → highest exit code wins.

## Input Rules

- `helium lint` → file args if present, else stdin when piped
- `helium xpath` → first positional arg always expression, XML from file args or stdin when none
- `helium xsd validate` → first positional arg schema path, XML from file args or stdin when none
- `helium relaxng validate` → first positional arg schema path, XML from file args or stdin when none
- `helium schematron validate` → first positional arg schema path, XML from file args or stdin when none
- `helium xslt` → first positional arg stylesheet path, XML from file args or stdin when none
- TTY + missing required XML input → usage + `ExitErr`

## `helium lint`

Primary file: `internal/cli/heliumcmd/lint.go`

### Processing Pipeline

```
1. READ      — os.ReadFile() / io.ReadAll(os.Stdin)
2. PARSE     — parser.Parse() with parseOptions
3. XINCLUDE  — xinclude.Process() if --xinclude
4. SCHEMA    — xsd.NewCompiler().CompileFile() + xsd.NewValidator().Validate() if --schema
5. DTD       — parser/DTD validation result if --valid
6. XPATH     — xpath.Evaluate() if --xpath
7. OUTPUT    — C14N or helium.Writer unless --noout
```

### `--output FILE` safety (lint and xslt)

File output (`--output`/`-o`, not stdout and not `--noout`) is written through a write-to-temp-then-atomic-rename scheme (`pendingOutput` in `safety.go`):

- A temp file (`.helium-out-*`) is created in the SAME directory as the target; output is written there, and `os.Rename`d onto the target ONLY after all inputs are processed successfully.
- This closes a truncate-before-read hole: `os.Create` on the target would truncate it up front, destroying a resource the same path is read from LATER — e.g. a DTD/entity resolved via `--path` during validation (lint), or a stylesheet read at transform time via `fn:transform(map{'stylesheet-location':...})` through the retained `URIResolver` (xslt).
- On any non-OK exit code the temp file is removed (`Cleanup`) and the target is left untouched.
- A failed commit (flush/close/rename) folds `ExitErr` into the exit status, so an incomplete write is never reported as success.
- The pre-flight same-file rejection (`checkOutputCollision`) is kept as a fast/friendly error for the obvious `--output X X` case, but the temp+rename is what actually protects later-resolved reads.
- stdout output and `--noout` are unaffected (no temp file is created).

### Flag Groups

| Group | Flags |
|-------|-------|
| Parser | `--recover`, `--noent`, `--loaddtd`, `--dtdattr`, `--valid`, `--nowarning`, `--pedantic`, `--noblanks`, `--nsclean`, `--nocdata`, `--nonet`, `--huge`, `--noenc`, `--noxincludenode`, `--nofixup-base-uris` |
| Features | `--xinclude`, `--schema FILE`, `--xpath EXPR`, `--catalogs`, `--nocatalogs`, `--path DIRS` |
| Output | `--noout`, `--format`, `--pretty N`, `--encode ENC`, `--output FILE`, `--c14n`, `--c14n11`, `--exc-c14n`, `--dropdtd` |
| Behavior | `--quiet`, `--timing`, `--repeat N`, `--max-input-bytes N`, `--version` |

### Cascades

- `--dtdattr` → also sets `--loaddtd`
- `--valid` → also sets `--loaddtd`
- `--xpath EXPR` → also sets `--noout`
- `--pretty N>=1` → also sets `--format`

### Output / Input Safety

- `--output FILE` refers to the same file as an XML input or the `--schema` → rejected (`would overwrite input/schema`, `ExitErr`) before any file is truncated. Same-file detection uses absolute-path equality plus `os.SameFile` (catches `./` prefixes and symlinks).
- `--output FILE` combined with `--noout` → rejected (`--output cannot be combined with --noout`). Exception: `--xpath` (which also sets `--noout` internally) still writes its result, so it is allowed.
- The output file is closed explicitly after processing; a close error is folded into the exit status (`ExitErr`).
- `--max-input-bytes N` caps the bytes read per input (file or stdin); default `DefaultMaxInputBytes` (100 MiB). `0` disables the cap. Exceeding it fails with `input exceeds maximum size` and `ExitReadFile`.
- `--quiet` suppresses informational output: timing messages are silenced and parser/validator warnings are suppressed.
- `--path DIRS` (colon-separated) is wired into DTD/entity resolution: a `pathSearchFS` falls back to each listed directory (by base name) when the default loader cannot open a referenced resource.

### Output Modes

- C14N mode → `--c14n` / `--c14n11` / `--exc-c14n`
- XPath mode → type-aware result printing
- Standard dump → `helium.NewWriter()` with format/dropdtd options

### `--encode ENC`

- Validated at parse time against `internal/encoding.Load`; an unrecognized encoding name is rejected with `--encode: unsupported encoding` and `ExitErr` (no silent fallback).
- US-ASCII and its aliases (`ascii`, `ANSI_X3.4-1968`, `csASCII`, detected via `internal/encoding.IsASCII`) are rejected with the same `--encode: unsupported encoding` message: `Load` maps them to the UTF-8 encoder, which would emit raw UTF-8 bytes for non-ASCII characters while declaring US-ASCII.
- Cannot be combined with `--xpath`: the XPath path serializes node values without re-encoding, so the combination is rejected at parse time with `--encode cannot be combined with --xpath` and `ExitErr`.
- Applied to the standard dump path via `doc.SetEncoding`, so the serializer loads the matching encoder and emits the matching encoding declaration.
- Ignored for C14N modes, which are always UTF-8 per the C14N spec.

## `helium xpath`

Primary file: `internal/cli/heliumcmd/xpath.go`

- Usage: `helium xpath [--engine 1|3] [--max-input-bytes N] EXPR [XMLfiles ...]`
- Default engine: `3`
- `--max-input-bytes N` caps bytes read per input (default 100 MiB; `0` = unlimited)
- `EXPR` mandatory + non-empty
- Engine `1` → `xpath1`
- Engine `3` → `xpath3`
- Output:
  - nodes → one per line via shared node printer
  - booleans → `true` / `false`
  - numbers → `%g`
  - strings / atomic values → one line per item

## `helium xsd validate`

Primary file: `internal/cli/heliumcmd/xsd_validate.go`

- Usage: `helium xsd validate [--timing] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- Schema compiled once with `xsd.NewCompiler().CompileFile()`
- Each XML input parsed with `helium.NewParser()` + validated with `xsd.NewValidator(schema).Validate()`

## `helium relaxng validate`

Primary file: `internal/cli/heliumcmd/relaxng_validate.go`

- Usage: `helium relaxng validate [--timing] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- Grammar compiled once with `relaxng.NewCompiler().CompileFile()`
- Each XML input parsed with `helium.NewParser()` + validated with `relaxng.NewValidator(grammar).Validate()`

## `helium schematron validate`

Primary file: `internal/cli/heliumcmd/schematron_validate.go`

- Usage: `helium schematron validate [--timing] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- Schema compiled once with `schematron.NewCompiler().Label(path).CompileFile(ctx, path)`
- Each XML input parsed with `helium.NewParser()` + validated with `schematron.NewValidator(schema).Label(name).Validate(ctx, doc)`
- Validation passes `.Label(input.name)` so error output names the current XML source

## `helium xslt`

Primary file: `internal/cli/heliumcmd/xslt.go`

- Usage: `helium xslt [options] STYLESHEET [XMLfiles ...]`
- Stylesheet path mandatory positional arg
- Stylesheet parsed with `helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true)`, compiled once with `xslt3.NewCompiler().URIResolver(fileResolver{}).Compile()`
- A filesystem `URIResolver` is installed so local `xsl:include`/`xsl:import` modules load (the compiler default-denies module loading without one)
- Each XML input parsed with `helium.NewParser()`, transformed with `ss.Transform(doc).WriteTo(ctx, out)`
- Flags: `--output FILE` / `-o FILE`, `--param NAME VAL` (XPath), `--stringparam NAME VAL`, `--noout`, `--timing`, `--max-input-bytes N`, `--version`
- Parameters passed via `inv.GlobalParameters()`
- Same output safety as `helium lint`: `--output` is rejected when it matches an input or the stylesheet, or when combined with `--noout`; close errors fold into the exit status; inputs are read under the `--max-input-bytes` cap
