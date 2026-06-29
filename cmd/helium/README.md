# helium CLI

The `helium` executable provides command-line access to parsing, validation,
querying, and XSLT transforms.

Wrapper entrypoint: `cmd/helium/main.go`

Implementation package: `internal/cli/heliumcmd`

| Command | Purpose |
|---------|---------|
| `helium lint` | Parse and lint XML documents |
| `helium xpath` | Evaluate XPath expressions against XML input |
| `helium xslt` | Transform XML with XSLT 3.0 stylesheets |
| `helium relaxng validate` | Validate XML documents against a RELAX NG schema |
| `helium schematron validate` | Validate XML documents against a Schematron schema |
| `helium xsd validate` | Validate XML documents against an XML Schema |

## `helium lint`

```text
helium lint [options] [XMLfiles ...]
```

Parses and lints XML documents with xmllint-style options. It can also apply
XInclude processing, schema validation, XPath checks, and canonicalization.
Run `helium lint` with no arguments to print the authoritative usage text
(`showUsage` in `internal/cli/heliumcmd/lint.go`).

| Flag | Description |
|------|-------------|
| `--version` | Display the version of the XML library used |
| `--recover` | Output what was parsable on broken XML documents |
| `--noent` | Substitute entity references by their value |
| `--loaddtd` | Fetch external DTD |
| `--dtdattr` | `--loaddtd` + populate tree with inherited attributes |
| `--valid` | Validate the document with the DTD |
| `--nowarning` | Do not emit warnings from parser/validator |
| `--pedantic` | Enable pedantic error reporting |
| `--noblanks` | Drop (ignorable) blank spaces |
| `--nsclean` | Remove redundant namespace declarations |
| `--nocdata` | Replace CDATA sections by equivalent text nodes |
| `--nonet` | Refuse to fetch DTDs or entities over network |
| `--huge` | Lift the tunable parser limits (element depth, name length, DTD content-model depth, entity-amplification ratio). The absolute 1 GB entity-expansion ceiling is always retained as a billion-laughs backstop |
| `--noenc` | Ignore any encoding specified inside the document |
| `--noxincludenode` | Do not generate XInclude START/END nodes |
| `--nofixup-base-uris` | Do not fix up `xml:base` URIs in XInclude |
| `--noout` | Do not print the result tree |
| `--format` | Reformat/reindent the output |
| `--pretty LEVEL` | Pretty-print the output (0=none; any level >=1 enables plain `--format` reindenting — there is no distinct higher-level behavior) |
| `--encode ENCODING` | Output in the given encoding |
| `--output FILE` | Save to a given file |
| `--c14n` | Save in W3C canonical format v1.0 (with comments) |
| `--c14n11` | Save in W3C canonical format v1.1 (with comments) |
| `--exc-c14n` | Save in W3C exclusive canonical format (with comments) |
| `--xinclude` | Do XInclude processing |
| `--schema FILE` | Validate against the WXS (XML Schema) schema |
| `--xpath EXPR` | Evaluate the XPath expression, implies `--noout` |
| `--catalogs` | Use catalogs from `$XML_CATALOG_FILES` |
| `--nocatalogs` | Do not use any catalogs |
| `--path DIRS` | Set search path for DTD/entities (colon-separated) |
| `--quiet` | Suppress non-error output |
| `--timing` | Print timing information to stderr |
| `--dropdtd` | Remove the DOCTYPE of the result |
| `--repeat N` | Parse N times for benchmarking |
| `--max-input-bytes N` | Cap bytes read per input (`0` = unlimited) |
| `--max-depth N` | Cap element nesting depth (default `256`, `0` = unlimited) |

The parser is **secure by default** (see the library's Security section): bare
`helium lint` blocks external entity/DTD loading and exposes no filesystem. The
loading flags `--loaddtd`, `--dtdattr`, `--valid`, and `--noent` are an explicit
opt-in — each one lifts that block and installs a permissive filesystem (or the
`--path` search path) so the requested external DTD/entity is actually loaded.

`--max-depth` and `--huge` govern the **document being processed**. Schema and
stylesheet **compilation**
(XSD / RELAX NG / Schematron / XSLT, including nested include/import/module loads)
parses with the default parser limits; tuning those compiler-internal limits is
a separate follow-up, not exposed here.

## `helium xpath`

```text
helium xpath [--engine 1|3] [--max-input-bytes N] [--max-depth N] EXPR [XMLfiles ...]
```

Evaluates an XPath expression against XML input. Engine `3` is the default;
engine `1` selects the XPath 1.0 implementation. `--max-input-bytes` caps the
bytes read per XML input (default 100 MiB; `0` = unlimited). `--max-depth` caps
element nesting depth (default 256; `0` = unlimited).

## `helium xslt`

```text
helium xslt [options] STYLESHEET [XMLfiles ...]
```

Applies an XSLT 3.0 stylesheet to one or more XML documents.

## `helium relaxng validate`

```text
helium relaxng validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]
```

Compiles a RELAX NG schema once, then validates each input XML document
against it. `--max-input-bytes` caps the bytes read per input (default 100 MiB;
`0` = unlimited). `--max-depth` caps element nesting depth (default 256;
`0` = unlimited).

## `helium schematron validate`

```text
helium schematron validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]
```

Compiles a Schematron schema once, then validates each input XML document
against it. `--max-input-bytes` caps the bytes read per input (default 100 MiB;
`0` = unlimited). `--max-depth` caps element nesting depth (default 256;
`0` = unlimited).

## `helium xsd validate`

```text
helium xsd validate [--timing] [--max-input-bytes N] [--max-depth N] SCHEMA [XMLfiles ...]
```

Compiles an XML Schema once, then validates each input XML document against
it. `--max-input-bytes` caps the bytes read per input (default 100 MiB;
`0` = unlimited). `--max-depth` caps element nesting depth (default 256;
`0` = unlimited).

## Common XSLT Flags

| Flag | Description |
|------|-------------|
| `--output FILE` / `-o FILE` | Write output to FILE |
| `--param NAME VALUE` | Set stylesheet parameter to XPath expression |
| `--stringparam NAME VALUE` | Set stylesheet parameter to string value |
| `--noout` | Run transformation without producing output (rejected with `--output`) |
| `--noent` | Substitute entities, loading the stylesheet's external entities (opt-in; off by default). External loading is confined to the stylesheet's directory |
| `--loaddtd` | Load the stylesheet's external DTD subset (opt-in; off by default). External loading is confined to the stylesheet's directory |
| `--timing` | Print compile/parse/transform timing to stderr |
| `--max-input-bytes N` | Cap bytes read per input (default 100 MiB; `0` = unlimited) |
| `--max-depth N` | Cap element nesting depth (default `256`, `0` = unlimited) |
| `--version` | Display version |

`--output` is refused when it names an input or the stylesheet, or when
combined with `--noout`, so it never truncates a file the command still needs
to read. Local `xsl:include`/`xsl:import` modules resolve relative to the
stylesheet's directory.
