# helium

[![CI](https://github.com/lestrrat-go/helium/actions/workflows/ci.yml/badge.svg)](https://github.com/lestrrat-go/helium/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lestrrat-go/helium.svg)](https://pkg.go.dev/github.com/lestrrat-go/helium)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/lestrrat-go/helium)

Helium is a fast XML toolkit for Go covering XML parsing, SAX2-style streaming,
XPath 3.1, XSLT 3.0, XInclude, XSD, Relax NG, and Schematron.

The root `helium` package handles parsing, DOM building, and serialization, but
the module is broader than an XML parser. It also includes
[`xpath3`](xpath3/README.md) for XPath 3.1 querying and
[`xslt3`](xslt3/README.md) for XSLT 3.0 transformations, alongside
[`xpath1`](xpath1/README.md) for XPath 1.0 compatibility,
[`xsd`](xsd/README.md), [`relaxng`](relaxng/README.md), and
[`schematron`](schematron/README.md) for validation,
[`xinclude`](xinclude/README.md) for inclusion processing,
[`c14n`](c14n/README.md) for canonicalization,
[`html`](html/README.md) for HTML parsing, and
[`shim`](shim/README.md) for `encoding/xml`-compatible APIs.

It started as an effort to port libxml2-style capabilities to Go, but grew
broader native Go APIs along the way. The goal is to provide a full Go XML
stack for parsing, querying, transforming, and validating documents, with each
major feature area documented in its own package README.

# SYNOPSIS

<!-- INCLUDE(examples/helium_parse_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
)

func Example_helium_parse() {
  // helium.NewParser().Parse is the simplest way to parse an XML document from a byte slice.
  // It returns a *helium.Document representing the parsed DOM tree.
  doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><child>hello</child></root>`))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // WriteString serializes the entire document back to an XML string,
  // including the XML declaration (<?xml version="1.0"?>).
  s, err := helium.WriteString(doc)
  if err != nil {
    fmt.Printf("failed to serialize: %s\n", err)
    return
  }
  fmt.Println(s)
  // Output:
  // <?xml version="1.0"?>
  // <root><child>hello</child></root>
}
```
source: [examples/helium_parse_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/helium_parse_example_test.go)
<!-- END INCLUDE -->

# Packages

Each public subpackage has its own `README.md` with package-specific details and
an embedded example.

| Package | Description | Notes |
|---------|-------------|-------|
| [`c14n`](c14n/README.md) | W3C Canonical XML support. | C14N 1.0, exclusive C14N 1.0, and C14N 1.1. |
| [`catalog`](catalog/README.md) | OASIS XML Catalog loading and resolution. | Useful with parsers, validators, and external resources. |
| [`enum`](enum/README.md) | Shared typed enums for DTD declarations. | Low-level support package; no standalone example. |
| [`html`](html/README.md) | HTML parser and serializer on top of helium nodes. | Produces helium DOM nodes or SAX-style events. |
| [`relaxng`](relaxng/README.md) | RELAX NG compilation and validation. | Schema compile step plus document validation. |
| [`sax`](sax/README.md) | SAX2 handler interfaces and helpers. | Event-driven parsing surface used by helium and html. |
| [`schematron`](schematron/README.md) | Schematron compilation and validation. | Rule-based XML validation with XPath assertions. |
| [`shim`](shim/README.md) | `encoding/xml`-compatible API backed by helium. | Import-path swap for existing stdlib-style code. |
| [`sink`](sink/README.md) | Generic async event sink. | Also satisfies `helium.ErrorHandler` when `T` is `error`. |
| [`stream`](stream/README.md) | Streaming XML writer. | Writes XML directly without building a DOM. |
| [`xinclude`](xinclude/README.md) | XInclude processing for helium documents. | Supports recursive inclusion and custom resolvers. |
| [`xpath1`](xpath1/README.md) | XPath 1.0 compilation and evaluation. | Includes convenience helpers like `Find` and `Evaluate`. |
| [`xpath3`](xpath3/README.md) | XPath 3.1 compilation and evaluation. | Includes a compiler, evaluator, maps, arrays, and HOFs. |
| [`xpointer`](xpointer/README.md) | XPointer evaluation. | Supports shorthand, `element()`, and XPath-backed schemes. |
| [`xsd`](xsd/README.md) | XML Schema compilation and validation. | XSD 1.0 compiler plus validator APIs. |
| [`xslt3`](xslt3/README.md) | XSLT 3.0 stylesheet compilation and execution. | Targets Basic XSLT 3.0 conformance. |

# `helium` CLI

The command-line interface is exposed as `helium`.
Currently implemented subcommands: `lint`, `xpath`, `xslt`, `xsd validate`, `relaxng validate`, `schematron validate`.
Use `helium lint` in place of the old `heliumlint` command.

| Command | Purpose |
|---------|---------|
| `helium lint` | Parse and lint XML documents |
| `helium xpath` | Evaluate XPath expressions against XML input |
| `helium xslt` | Transform XML with XSLT 3.0 stylesheets |
| `helium relaxng validate` | Validate XML documents against a RELAX NG schema |
| `helium schematron validate` | Validate XML documents against a Schematron schema |
| `helium xsd validate` | Validate XML documents against an XML Schema |

See [`cmd/helium/README.md`](cmd/helium/README.md) for command-specific
documentation.

# Performance

Helium parses XML into a full DOM tree. The benchmark below compares that DOM
build against two lower-level baselines: an `encoding/xml` token loop
(`Decoder.Token`) and libxml2 via cgo.

That is a narrower benchmark than every real `encoding/xml` workload. Many Go
programs use `encoding/xml` to decode directly into structs, and this section is
not meant to dismiss that use case or the package. The point here is simply
that Helium's DOM parse is already quite fast: it is materially faster than the
stdlib token benchmark on all three corpora, it now edges past libxml2 on the
medium corpus, and it is clearly ahead on the largest corpus.

Benchmarks parse real-world XML files of varying sizes (AMD Ryzen 9 7900X3D,
Go 1.26.1, `go test -run '^$' -bench 'Benchmark(HeliumParse|StdlibXMLDecode|Libxml2Parse)$' -benchmem -count=5 -tags libxml2bench ./bench`,
median shown):

| File | Helium | `encoding/xml` | libxml2 (cgo) |
|------|--------|----------------|---------------|
| 109 KB | 139 MB/s | 77 MB/s | 158 MB/s |
| 196 KB | 124 MB/s | 66 MB/s | 109 MB/s |
| 3 MB | 497 MB/s | 120 MB/s | 366 MB/s |

Helium also allocates far fewer objects than `encoding/xml` in this benchmark.
On the 3 MB corpus, the current Helium DOM parse lands around `94 allocs/op`
versus about `155k allocs/op` for `encoding/xml`.

To run the benchmarks yourself:

```text
go test -bench='BenchmarkHeliumParse|BenchmarkStdlibXMLDecode' -benchmem ./bench/
# Include libxml2 (requires cgo and libxml2-dev):
go test -tags cgo,libxml2bench -bench=. -benchmem ./bench/
```

# Current status

* Core functionality is implemented: XML/HTML parsing, DOM building, SAX2, XPath 1.0, XPath 3.1, Basic XSLT 3.0, XInclude, C14N, RELAX NG, Schematron, XSD, XML Catalog, streaming XML writer, and `encoding/xml` compatibility (`shim` package).
* W3C conformance suites: ~22,250 / 22,744 QT3 tests pass for XPath 3.1; ~11,780 / 13,129 W3C tests pass for XSLT 3.0 (skips are XSLT 1.0/2.0 backwards compatibility and other out-of-scope features).
* libxml2-compat golden tests: core XML parsing 100%, XSD 99.6%, RELAX NG 100%, Schematron 100%, C14N 87%, HTML 100%.
* XSLT support is intentionally scoped to Basic XSLT 3.0. Backwards compatibility modes for XSLT 1.0/2.0 are not part of the target feature set.
* A `helium` CLI provides `lint`, `xpath`, `xslt`, `xsd validate`, `relaxng validate`, and `schematron validate` subcommands.
* Some edge cases and parity gaps are still being iterated on; contributions and issue reports are welcome.

# Contributing

## Issues

For bug reports and feature requests, please follow the issue template when possible.
If you can include a minimal reproduction or failing test case, that helps a lot.

## Pull Requests

Please include tests that cover your changes.

If your change touches generated files, update the generator/source first, regenerate,
and commit both the source and generated outputs together.

Please keep pull requests focused and small enough to review quickly.

## Discussions / Usage

For usage questions, design discussion, or "is this approach reasonable?" questions,
please open a GitHub Discussion first.
