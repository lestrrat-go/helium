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

- A temp file (`.helium-out-*`) is created (via `os.OpenFile` with `O_CREATE|O_EXCL`, mode `0666` so the kernel applies umask) in the SAME directory as the target; output is written there, and `os.Rename`d onto the target ONLY after all inputs are processed successfully.
- When the target is a symlink, the rename target is the resolved real file (`os.Lstat` + `resolveSymlinkTarget`, which follows the chain with `os.Readlink` so even a DANGLING link resolves to its would-be target rather than failing like `filepath.EvalSymlinks`), so output is written THROUGH the link (matching `os.Create`) instead of replacing the link with a regular file.
- If the resolved target already exists, `newPendingOutput` requires it to be a regular, writable file (probed via `os.Stat` + `os.OpenFile(O_WRONLY)`) before proceeding; a read-only (`0444`) or non-regular target is rejected with `ExitErr` and left untouched, since `os.Rename` would otherwise replace it regardless of mode and exit 0.
- This closes a truncate-before-read hole: `os.Create` on the target would truncate it up front, destroying a resource the same path is read from LATER — e.g. a DTD/entity resolved via `--path` during validation (lint), or a stylesheet read at transform time via `fn:transform(map{'stylesheet-location':...})` through the retained `URIResolver` (xslt).
- On any non-OK exit code the temp file is removed (`Cleanup`) and the target is left untouched.
- A failed commit (flush/close/rename) folds `ExitErr` into the exit status, so an incomplete write is never reported as success.
- The pre-flight same-file rejection (`checkOutputCollision`) is kept as a fast/friendly error for the obvious `--output X X` case, but the temp+rename is what actually protects later-resolved reads.
- stdout output and `--noout` are unaffected (no temp file is created).
- Output file mode: the temp is created `0666` so the kernel applies the process umask, which already matches `os.Create` for a NEW destination (no chmod). For an EXISTING destination `Commit` chmods the temp to the current mode (`os.Stat` + `chmod`) before the rename. umask is never read in-process (the old `syscall.Umask(0)` read-modify-write was racy and has been removed).

### Flag Groups

| Group | Flags |
|-------|-------|
| Parser | `--recover`, `--noent`, `--loaddtd`, `--dtdattr`, `--valid`, `--nowarning`, `--pedantic`, `--noblanks`, `--nsclean`, `--nocdata`, `--nonet`, `--huge`, `--noenc`, `--noxincludenode`, `--nofixup-base-uris` |
| Features | `--xinclude`, `--schema FILE`, `--xpath EXPR`, `--catalogs`, `--nocatalogs`, `--path DIRS` |
| Output | `--noout`, `--format`, `--pretty N`, `--encode ENC`, `--output FILE`, `--c14n`, `--c14n11`, `--exc-c14n`, `--dropdtd` |
| Behavior | `--quiet`, `--timing`, `--repeat N`, `--max-input-bytes N`, `--max-depth N`, `--version` |

### Cascades

- `--dtdattr` → also sets `--loaddtd`
- `--valid` → also sets `--loaddtd`
- `--xpath EXPR` → also sets `--noout`
- `--pretty N>=1` → also sets `--format`
- `--loaddtd` / `--dtdattr` / `--valid` / `--noent` → external-loading opt-in: each lifts the parser's default `BlockXXE` block and installs a permissive FS (or the `--path` search FS) so the requested external DTD/entity actually loads. Bare `lint` is safe-by-default (`NewParser` blocks external loading and uses a deny-all FS), matching the library.
- `--huge` → lifts the tunable parser limits for trusted input: `MaxNameLength(-1)`, `MaxEntityAmplification(-1)`, `MaxContentModelDepth(-1)`, `MaxNodeContentSize(-1)` (disables the 10 MiB cap on both single-construct CDATA/comment/PI/char-data/attribute-value runs and contiguous XML-whitespace blank-skip runs), and `MaxDepth(0)`. An explicit `--max-depth` still wins (applied after `--huge`).
- `--xinclude` → `xinclude.NewProcessor()` now denies all FS access by default (safe-by-default, matching the library), so the CLI installs a permissive resolver (`Resolver(NewFSResolver(...))`) backed by the same permissive root — or the `--path` search FS — used by the parser, preserving the historical behavior of reading includes off disk.

### Output / Input Safety

- `--output FILE` refers to the same file as an XML input or the `--schema` → rejected (`would overwrite input/schema`, `ExitErr`) before any file is truncated. Same-file detection uses absolute-path equality plus `os.SameFile` (catches `./` prefixes and symlinks).
- `--output FILE` combined with `--noout` → rejected (`--output cannot be combined with --noout`). Exception: `--xpath` (which also sets `--noout` internally) still writes its result, so it is allowed.
- The output file is closed explicitly after processing; a close error is folded into the exit status (`ExitErr`).
- `--max-input-bytes N` caps the bytes read per input (file or stdin); default `DefaultMaxInputBytes` (100 MiB). `0` disables the cap. Exceeding it fails with `input exceeds maximum size` and `ExitReadFile`.
- `--max-depth N` caps element nesting depth; default `256` (the `NewParser` default), `0` = unlimited. Exceeding it fails the parse (`exceeded max depth`). `--max-depth` (and `--huge`) apply to the **document being processed** (the linted/validated/transformed instance). Schema and stylesheet **compilation** (XSD/RELAX NG/Schematron/XSLT, including nested include/import/module loads) parses with the default parser limits; exposing those compiler-internal limits is intentionally out of scope here and left as a follow-up.
- `--quiet` suppresses informational output: timing messages are silenced and parser/validator warnings are suppressed.
- `--path DIRS` (colon-separated) is wired into DTD/entity resolution: a `pathSearchFS` falls back to each listed directory (by base name) when the default loader cannot open a referenced resource.

### Output Modes

- C14N mode → `--c14n` / `--c14n11` / `--exc-c14n`
- XPath mode → type-aware result printing
- Standard dump → `helium.NewWriter()` with format/dropdtd options

### `--encode ENC`

- Validated at parse time against `internal/encoding.Load`; an unrecognized encoding name is rejected with `--encode: unsupported encoding` and `ExitErr` (no silent fallback).
- US-ASCII and its aliases (`ascii`, `ANSI_X3.4-1968`, `csASCII`, detected via `internal/encoding.IsASCII`) are rejected with the same `--encode: unsupported encoding` message: `Load`'s strict ASCII encoder delegates to UTF-8, which would emit raw UTF-8 bytes for non-ASCII characters while declaring US-ASCII.
- Cannot be combined with `--xpath`: the XPath path serializes node values without re-encoding, so the combination is rejected at parse time with `--encode cannot be combined with --xpath` and `ExitErr`.
- Applied to the standard dump path via `doc.SetEncoding`, so the serializer loads the matching encoder and emits the matching encoding declaration.
- Ignored for C14N modes, which are always UTF-8 per the C14N spec.

## `helium xpath`

Primary file: `internal/cli/heliumcmd/xpath.go`

- Usage: `helium xpath [--engine 1|3] [--max-input-bytes N] [--max-depth N] EXPR [XMLfiles ...]`
- Default engine: `3`
- `--max-input-bytes N` caps bytes read per input (default 100 MiB; `0` = unlimited)
- `--max-depth N` caps element nesting depth (default `256`, `0` = unlimited); when absent, the `NewParser` default is left untouched
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

- Usage: `helium xsd validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- `--max-input-bytes N` caps bytes read per XML input (file or stdin) via `readInput`/`readInputFile`; default `DefaultMaxInputBytes` (100 MiB), `0` = unlimited; over-cap fails with `ExitReadFile`
- `--max-depth N` caps element nesting depth (default `256`, `0` = unlimited); when absent, the `NewParser` default is left untouched; over-cap fails the parse (`ExitErr`)
- Schema compiled once with `xsd.NewCompiler().Label(schema).ErrorHandler(...).CompileFile(ctx, schema)`; a `compileErrorHandler` streams compilation diagnostics (file/line/detail) to stderr and records whether any FATAL diagnostic was seen
- The xsd compiler may return a non-nil schema with a nil error for a malformed schema; the CLI folds that into a failure (`errSchemaCompilation`) when the handler saw a fatal diagnostic, so it never validates against a bad schema. Compilation failure → `ExitSchemaComp`
- Each XML input parsed with `helium.NewParser()` (file inputs get `.BaseURI(name)`) + validated with `xsd.NewValidator(schema).ErrorHandler(...).Validate(ctx, doc)`, diagnostics streamed to stderr via a `writerErrorHandler`

## `helium relaxng validate`

Primary file: `internal/cli/heliumcmd/relaxng_validate.go`

- Usage: `helium relaxng validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- `--max-input-bytes N` caps bytes read per XML input (file or stdin) via `readInput`/`readInputFile`; default `DefaultMaxInputBytes` (100 MiB), `0` = unlimited; over-cap fails with `ExitReadFile`
- `--max-depth N` caps element nesting depth (default `256`, `0` = unlimited); when absent, the `NewParser` default is left untouched; over-cap fails the parse (`ExitErr`)
- Grammar compiled once with `relaxng.NewCompiler().FS(helium.PermissiveFS()).Label(schema).ErrorHandler(...).CompileFile(ctx, schema)`; `FS(helium.PermissiveFS())` opts back into host-filesystem loading for `include`/`externalRef` (the compiler's FS now defaults to deny-all), mirroring `lint`/`xslt`; a `compileErrorHandler` streams compilation diagnostics (file/line/detail) to stderr and records whether any FATAL diagnostic was seen
- The RELAX NG compiler may return a non-nil grammar with a nil error (a poisoned `notAllowed` grammar) on a fatal diagnostic; the CLI folds that into a failure (`errSchemaCompilation`) when the handler saw a fatal diagnostic, so it never validates against a bad grammar. Compilation failure → `ExitSchemaComp`
- Each XML input parsed with `helium.NewParser()` (file inputs get `.BaseURI(name)`) + validated with `relaxng.NewValidator(grammar).Label(name).ErrorHandler(...).Validate(ctx, doc)`, diagnostics streamed to stderr via a `writerErrorHandler`

## `helium schematron validate`

Primary file: `internal/cli/heliumcmd/schematron_validate.go`

- Usage: `helium schematron validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- `--max-input-bytes N` caps bytes read per XML input (file or stdin) via `readInput`/`readInputFile`; default `DefaultMaxInputBytes` (100 MiB), `0` = unlimited; over-cap fails with `ExitReadFile`
- `--max-depth N` caps element nesting depth (default `256`, `0` = unlimited); when absent, the `NewParser` default is left untouched; over-cap fails the parse (`ExitErr`)
- Schema compiled once with `schematron.NewCompiler().Label(path).CompileFile(ctx, path)`
- Each XML input parsed with `helium.NewParser()` + validated with `schematron.NewValidator(schema).Label(name).Validate(ctx, doc)`
- Validation passes `.Label(input.name)` so error output names the current XML source

## `helium xslt`

Primary file: `internal/cli/heliumcmd/xslt.go`

- Usage: `helium xslt [options] STYLESHEET [XMLfiles ...]`
- Stylesheet path mandatory positional arg
- Stylesheet parsed with the secure `helium.NewParser()` default (no external DTD/entity loading, deny-all FS) — a hostile stylesheet cannot read local files via a SYSTEM entity. `SubstituteEntities(true)` IS enabled by default so the stylesheet's INTERNAL-subset general entities expand (xslt3 only compiles text/CDATA in sequence constructors, so an unexpanded `EntityRefNode` would silently drop the value); `BlockXXE`/`LoadExternalDTD(false)` still block external content, mirroring xslt3's own secure parser (`xslt3/xslt3.go`). External loading is opt-in via `--noent` (substitutes external entities) and/or `--loaddtd` (`LoadExternalDTD`), mirroring `helium lint`'s flag names. `--loaddtd` on its own loads the external DTD subset WITHOUT substituting its general entities (the internal-substitution default is suppressed for `--loaddtd`-without-`--noent`). When opted in, the parser's `BlockXXE` is lifted and the FS is `confinedDirFS` rooted at the **stylesheet's own directory** (not a raw permissive root), so an attacker-controlled SYSTEM identifier (`/etc/passwd`, `../../secret`) still cannot exfiltrate files outside that directory. Compiled once with `xslt3.NewCompiler().URIResolver(fileResolver{}).Compile()`
- A filesystem `URIResolver` is installed so local `xsl:include`/`xsl:import` modules load (the compiler default-denies module loading without one)
- `fileResolver.Resolve` accepts plain relative/absolute paths AND `file:` URIs (`localFilePath` in `safety.go`): a `file:` URI is parsed, only an empty or `localhost` host is accepted, the path is percent-decoded (and de-slashed before a Windows drive letter); any other scheme (`http`/`https`/...) is rejected so the resolver never reaches the network. A bare Windows drive path (`C:\...`) is not mistaken for a scheme.
- Each XML input parsed with `helium.NewParser()`, transformed with `ss.Transform(doc).WriteTo(ctx, out)`
- Flags: `--output FILE` / `-o FILE`, `--param NAME VAL` (XPath), `--stringparam NAME VAL`, `--noout`, `--noent`, `--loaddtd`, `--timing`, `--max-input-bytes N`, `--max-depth N`, `--version`
- `--noent` / `--loaddtd` → stylesheet-parser external-loading opt-in (off by default): each lifts the parser's default `BlockXXE` and installs `confinedDirFS` (stylesheet directory only). They affect ONLY the stylesheet parse; the source-document parser is always the plain secure `NewParser()` default
- `--max-depth N` caps element nesting depth (default `256`, `0` = unlimited) and applies to BOTH the stylesheet parser and the source-document parser; when absent, the `NewParser` default is left untouched
- Parameters passed via `inv.GlobalParameters()`
- Same output safety as `helium lint`: `--output` is rejected when it matches an input or the stylesheet, or when combined with `--noout`; close errors fold into the exit status; inputs are read under the `--max-input-bytes` cap
