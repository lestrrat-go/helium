# helium command

Unified CLI entrypoint. Wrapper file: `cmd/helium/main.go`. Implementation package: `internal/cli/heliumcmd`.

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

## Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | `ExitOK` | Success |
| 1 | `ExitErr` | Generic error / usage error |
| 3 | `ExitValidation` | Validation failed |
| 4 | `ExitReadFile` | File/stdin read error |
| 5 | `ExitSchemaComp` | Schema compilation error |
| 10 | `ExitXPath` | XPath compile/evaluation error |

Multiple XML inputs → highest exit code wins.

## Input Rules

- `helium lint` → file args if present, else stdin when piped
- `helium xpath` → first positional arg always expression, XML from file args or stdin when none
- `helium xsd validate` → first positional arg schema path, XML from file args or stdin when none
- `helium relaxng validate` → first positional arg schema path, XML from file args or stdin when none
- `helium schematron validate` → first positional arg schema path, XML from file args or stdin when none
- TTY + missing required XML input → usage + `ExitErr`

## `helium lint`

Primary file: `internal/cli/heliumcmd/lint.go`

### Processing Pipeline

```
1. READ      — os.ReadFile() / io.ReadAll(os.Stdin)
2. PARSE     — parser.Parse() with parseOptions
3. XINCLUDE  — xinclude.Process() if --xinclude
4. SCHEMA    — xsd.CompileFile() + xsd.Validate() if --schema
5. DTD       — parser/DTD validation result if --valid
6. XPATH     — xpath.Evaluate() if --xpath
7. OUTPUT    — C14N or helium.Writer unless --noout
```

### Flag Groups

| Group | Flags |
|-------|-------|
| Parser | `--recover`, `--noent`, `--loaddtd`, `--dtdattr`, `--valid`, `--nowarning`, `--pedantic`, `--noblanks`, `--nsclean`, `--nocdata`, `--nonet`, `--huge`, `--noenc`, `--noxincludenode`, `--nofixup-base-uris` |
| Features | `--xinclude`, `--schema FILE`, `--xpath EXPR`, `--catalogs`, `--nocatalogs`, `--path DIRS` |
| Output | `--noout`, `--format`, `--pretty N`, `--encode ENC`, `--output FILE`, `--c14n`, `--c14n11`, `--exc-c14n`, `--dropdtd` |
| Behavior | `--quiet`, `--timing`, `--repeat N`, `--version` |

### Cascades

- `--dtdattr` → also sets `--loaddtd`
- `--valid` → also sets `--loaddtd`
- `--xpath EXPR` → also sets `--noout`
- `--pretty N>=1` → also sets `--format`

### Output Modes

- C14N mode → `--c14n` / `--c14n11` / `--exc-c14n`
- XPath mode → type-aware result printing
- Standard dump → `helium.NewWriter()` with format/dropdtd options

## `helium xpath`

Primary file: `internal/cli/heliumcmd/xpath.go`

- Usage: `helium xpath [--engine 1|3] EXPR [XMLfiles ...]`
- Default engine: `3`
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
- Schema compiled once with `xsd.CompileFile()`
- Each XML input parsed with `helium.NewParser()` + validated with `xsd.Validate()`

## `helium relaxng validate`

Primary file: `internal/cli/heliumcmd/relaxng_validate.go`

- Usage: `helium relaxng validate [--timing] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- Grammar compiled once with `relaxng.CompileFile()`
- Each XML input parsed with `helium.NewParser()` + validated with `relaxng.Validate()`

## `helium schematron validate`

Primary file: `internal/cli/heliumcmd/schematron_validate.go`

- Usage: `helium schematron validate [--timing] SCHEMA [XMLfiles ...]`
- Schema path mandatory positional arg
- Schema compiled once with `schematron.NewCompiler().SchemaFilename(path).CompileFile(ctx, path)`
- Each XML input parsed with `helium.NewParser()` + validated with `schematron.NewValidator(schema).Filename(name).Validate(ctx, doc)`
- Validation passes `.Filename(input.name)` so error output names the current XML source

