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

## `helium xpath`

```text
helium xpath [--engine 1|3] EXPR [XMLfiles ...]
```

Evaluates an XPath expression against XML input. Engine `3` is the default;
engine `1` selects the XPath 1.0 implementation.

## `helium xslt`

```text
helium xslt [options] STYLESHEET [XMLfiles ...]
```

Applies an XSLT 3.0 stylesheet to one or more XML documents.

## `helium relaxng validate`

```text
helium relaxng validate [--timing] SCHEMA [XMLfiles ...]
```

Compiles a RELAX NG schema once, then validates each input XML document
against it.

## `helium schematron validate`

```text
helium schematron validate [--timing] SCHEMA [XMLfiles ...]
```

Compiles a Schematron schema once, then validates each input XML document
against it.

## `helium xsd validate`

```text
helium xsd validate [--timing] SCHEMA [XMLfiles ...]
```

Compiles an XML Schema once, then validates each input XML document against
it.

## Common XSLT Flags

| Flag | Description |
|------|-------------|
| `--output FILE` / `-o FILE` | Write output to FILE |
| `--param NAME VALUE` | Set stylesheet parameter to XPath expression |
| `--stringparam NAME VALUE` | Set stylesheet parameter to string value |
| `--noout` | Run transformation without producing output |
| `--timing` | Print compile/parse/transform timing to stderr |
| `--version` | Display version |
