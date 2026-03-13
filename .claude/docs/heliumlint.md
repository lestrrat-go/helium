# helium lint

Unified CLI lint subcommand. Matches previous lint behavior + xmllint-style flags.

Primary entrypoint: `cmd/helium/main.go`
Lint implementation: `cmd/helium/lint.go`

## Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | `ExitOK` | Success |
| 1 | `ExitErr` | Generic error |
| 3 | `ExitValidation` | Validation failed (DTD/Schema) |
| 4 | `ExitReadFile` | File read error |
| 5 | `ExitSchemaComp` | Schema compilation error |
| 10 | `ExitXPath` | XPath evaluation error |

Multiple files: highest exit code wins.

## Processing Pipeline

```
1. READ   — os.ReadFile(filename) or io.ReadAll(os.Stdin)
2. PARSE  — parser.Parse() with parseOptions (repeated cfg.repeat times)
3. XINCLUDE — xinclude.Process() if --xinclude
4. SCHEMA — xsd.CompileFile(ctx, ...) + xsd.Validate() if --schema
5. DTD    — check DTD validation result if --valid
6. XPATH  — xpath.Evaluate() if --xpath (bypasses output)
7. OUTPUT — C14N mode OR standard dump (unless --noout)
```

## Flags

### Parser Options

| Flag | ParseOption | Notes |
|------|-------------|-------|
| `--recover` | ParseRecover | Output partial tree on broken XML |
| `--noent` | ParseNoEnt | Substitute entity references |
| `--loaddtd` | ParseDTDLoad | Fetch external DTD |
| `--dtdattr` | ParseDTDAttr + ParseDTDLoad | Auto-sets loaddtd |
| `--valid` | ParseDTDValid + ParseDTDLoad | Auto-sets loaddtd |
| `--nowarning` | ParseNoWarning | |
| `--pedantic` | ParsePedantic | |
| `--noblanks` | ParseNoBlanks | |
| `--nsclean` | ParseNsClean | |
| `--nocdata` | ParseNoCDATA | |
| `--nonet` | ParseNoNet | |
| `--huge` | ParseHuge | |
| `--noenc` | ParseIgnoreEnc | |
| `--noxincludenode` | ParseNoXIncNode | |
| `--nofixup-base-uris` | ParseNoBaseFix | |

### Feature Flags

| Flag | Effect |
|------|--------|
| `--xinclude` | Enable XInclude processing (also sets ParseXInclude) |
| `--schema FILE` | Validate against XSD schema |
| `--xpath EXPR` | Evaluate XPath (implies --noout) |
| `--catalogs` | Use catalogs from `$XML_CATALOG_FILES` env var |
| `--nocatalogs` | Override --catalogs |
| `--path DIRS` | Search path for DTD/entities (colon-separated) |

### Output Flags

| Flag | Effect |
|------|--------|
| `--noout` | Suppress output |
| `--format` | Reformat with 2-space indent |
| `--pretty N` | 0=none, 1=format, 2=format+attrs. ≥1 auto-sets --format |
| `--encode ENC` | Output encoding (parsed but not fully integrated) |
| `--output FILE` | Save to file |
| `--c14n` | C14N 1.0 with comments |
| `--c14n11` | C14N 1.1 with comments |
| `--exc-c14n` | Exclusive C14N with comments |
| `--dropdtd` | Remove DOCTYPE from output |

### Behavioral Flags

| Flag | Effect |
|------|--------|
| `--quiet` | Suppress non-error output |
| `--timing` | Print timing to stderr |
| `--repeat N` | Parse N times (benchmarking, default 1) |
| `--version` | Display helium version |

## Flag Cascades

- `--dtdattr` → auto-sets `--loaddtd`
- `--valid` → auto-sets `--loaddtd`
- `--xpath EXPR` → auto-sets `--noout`
- `--pretty N≥1` → auto-sets `--format`

## Output Modes (mutually exclusive)

1. **C14N** (if --c14n/--c14n11/--exc-c14n) — canonical form, always with comments. Bypasses --format/--dropdtd/--encode.
2. **XPath** (if --xpath) — type-specific output: nodes one-per-line, boolean, number, or string. Attributes as ` name="value"`, namespaces as ` xmlns:prefix="uri"`.
3. **Standard dump** (default) — uses --format, --dropdtd options via `helium.NewWriter()`

## Catalog Integration

- `--catalogs` reads `$XML_CATALOG_FILES` (space-separated paths)
- Loads first successful catalog, logs errors for others
- Passed to parser via `parser.SetCatalog(cat)`

## Input Handling

- File args: `helium lint file1.xml file2.xml` — each processed sequentially
- Stdin: detected when piped (not TTY) and no file args
- TTY with no args: shows usage, exits with code 1

## Command Layout

- `helium lint ...` — supported form
- sibling command: `helium xsd validate ...`

## Differences from xmllint

- `--encode` parsed but not fully integrated into output
- Catalog via helium API (not libxml2's global catalog system)
- Different error messages (helium parser vs libxml2)
- Pure Go implementation (performance characteristics differ)
