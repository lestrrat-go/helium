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
| [`xmldsig1`](xmldsig1/README.md) | W3C XML Digital Signatures 1.1 over helium documents. | **Experimental**; API may change. |
| [`xmlenc1`](xmlenc1/README.md) | W3C XML Encryption 1.1 over helium documents. | **Experimental**; API may change. |
| [`xpath1`](xpath1/README.md) | XPath 1.0 compilation and evaluation. | Includes convenience helpers like `Find` and `Evaluate`. |
| [`xpath3`](xpath3/README.md) | XPath 3.1 compilation and evaluation. | Includes a compiler, evaluator, maps, arrays, and HOFs. |
| [`xpointer`](xpointer/README.md) | XPointer evaluation. | Supports shorthand, `element()`, and XPath-backed schemes. |
| [`xsd`](xsd/README.md) | XML Schema compilation and validation. | XSD 1.0 (default) and opt-in XSD 1.1 compiler plus validator APIs. |
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

# Security

`NewParser()` is **secure by default** — it is safe to point at untrusted XML
with no extra configuration. By default:

- External entity and DTD loading is **blocked** (`BlockXXE(true)`), so XML
  External Entity (XXE) attacks are rejected.
- No filesystem is exposed: the parser's `FS` is a **deny-all** filesystem, so
  even a document that reaches a loader cannot open host paths.
- Network access is **forbidden** (`AllowNetwork(false)`). The core parser has
  no network loader, so this is belt-and-suspenders.
- Element nesting depth is **capped at 256** (`MaxDepth(256)`; `0` = unbounded).
- Entity substitution and external DTD loading are off
  (`SubstituteEntities(false)`, `LoadExternalDTD(false)`); the entity-expansion
  amplification, name-length, and content-model-depth guards are at their
  defaults (`MaxEntityAmplification`, `MaxNameLength`, `MaxContentModelDepth`);
  and any external DTD subset — once explicitly enabled — is capped at 10 MiB.

The builder is clone-on-write, so one configured parser is safe to reuse across
goroutines.

To deliberately load external resources from a **trusted** source, opt back in
explicitly:

```go
doc, err := helium.NewParser().
    BlockXXE(false).            // allow external entities and DTDs
    LoadExternalDTD(true).      // read the external DTD subset
    SubstituteEntities(true).   // expand entities
    FS(helium.PermissiveFS()).  // open any os.Open path (or pass a confined fs.FS)
    Parse(ctx, xmlBytes)
```

`helium.PermissiveFS()` returns an `fs.FS` that opens any path via `os.Open`,
restoring the historical unsandboxed behavior; prefer a confined `fs.FS` rooted
at a trusted directory when the document's external references are known.
Passing `FS(nil)` restores the deny-all default.

The parser cannot know your resource budget, so even with the safe defaults the
caller should also:

- Enforce a maximum raw document size before calling `Parse`.
- Pass a `context.Context` with a deadline to `Parse` / `ParseReader`.
- Leave the entity-amplification, name-length, and content-model-depth limits
  at their defaults — passing a negative value to `MaxEntityAmplification`,
  `MaxNameLength`, or `MaxContentModelDepth` removes that guard.
- Be cautious enabling XInclude, catalogs, DTD validation, or
  default-DTD-attribute processing for untrusted input; when you do, keep every
  external resource allowlisted and size-bounded. The `xinclude` processor is
  also secure by default — with no resolver configured it denies all filesystem
  access; grant access with `Resolver(xinclude.NewFSResolver(fsys))` backed by a
  confined `fs.FS` (`os.Root.FS`), or restore historical OS-path access with
  `xinclude.NewFSResolver(helium.PermissiveFS())`.
  `xinclude.Processor.MaxIncludeDepth` bounds the nesting depth of included
  documents, and `MaxIncludeSize` caps the bytes read per included resource.

The `xsd` schema compiler is likewise **secure by default**: `xsd.NewCompiler()`
denies all nested-schema filesystem access, so an untrusted schema cannot
disclose local files or exhaust resources through a hostile
`xs:include`/`xs:import`/`xs:redefine` `schemaLocation`. Each nested schema is
read through a fixed byte cap regardless of the `fs.FS` in use. Opt into host
access with `Compiler.FS(helium.PermissiveFS())` or a confined `fs.FS`;
`Compiler.FS(nil)` restores the deny-all default.

**Caveat:** a permissive or directory-rooted `FS` is not yet a complete sandbox.
External-resource paths are joined against the document base URI and may be
absolute or use OS-specific separators, so `os.DirFS`-style roots (which enforce
`fs.ValidPath`) reject them. Until path normalization lands, rely on the deny-all
default for confinement rather than a chroot-style `fs.FS`.

The `xmldsig1` (signatures) and `xmlenc1` (encryption) packages are
**experimental** and should not be relied on inside a security or compliance
boundary yet.

# `encoding/xml` compatibility

The [`shim`](shim/README.md) package is an import-path-compatible replacement
for `encoding/xml` backed by helium's parser (`Marshal`, `Unmarshal`,
`Encoder`, `Decoder`, and the usual struct tags). It is a migration aid, not a
byte-for-byte behavioral clone. Known differences:

- `Decoder.Strict = false` is not supported; `Decoder.AutoClose` is a no-op and
  `HTMLAutoClose` is omitted.
- Undeclared namespace prefixes are rejected rather than passed through.
- Namespace declarations are emitted before regular attributes.
- `Decoder.InputOffset` is approximate rather than exact.
- Empty elements captured via `,innerxml` may re-serialize as self-closed tags.

Migrate behind your own tests rather than assuming a transparent swap.

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

* **Implemented:** XML/HTML parsing, DOM building, SAX2, XPath 1.0, XPath 3.1, Basic XSLT 3.0, XInclude, C14N, RELAX NG, Schematron, XSD, XML Catalog, streaming XML writer, and `encoding/xml` compatibility (`shim` package).
* **Experimental:** W3C XML Digital Signatures 1.1 (`xmldsig1`) and XML Encryption 1.1 (`xmlenc1`) — these APIs may change and may move to a separate repository.
* **CLI:** the `helium` command provides `lint`, `xpath`, `xslt`, `xsd validate`, `relaxng validate`, and `schematron validate` subcommands.

Some edge cases and parity gaps are still being iterated on; contributions and issue reports are welcome.

## W3C conformance suites

Each row links to the package's README (for scope and details) and its committed, point-in-time evidence (a stamped summary; a matching JUnit `results-*.xml` sits beside it).

| Suite | Result | Package | Evidence |
|-------|--------|---------|----------|
| XPath 3.1 (QT3) | 21,987 / 22,473 | [`xpath3`](xpath3/README.md) | [summary](xpath3/summary-qt3.md) |
| XSLT 3.0 | 12,343 / 13,127 · **0 failures** | [`xslt3`](xslt3/README.md) | [summary](xslt3/summary-xslt30.md) |
| XSD 1.0 — schema validity (default) | 14,082 / 14,399 (97.8%) | [`xsd`](xsd/README.md) | [summary](xsd/summary-xsd10.md) |
| XSD 1.0 — instance validity (default) | 24,613 / 25,004 | [`xsd`](xsd/README.md) | [summary](xsd/summary-xsd10.md) |
| XSD 1.1 (opt-in) | 1,049 cases · **0 failures** | [`xsd`](xsd/README.md) | [summary](xsd/summary-xsd11.md) |

XSLT 3.0 has **zero failing tests**; every one of the 784 default skips is accounted for — performance-gated slow tests, legitimate XSLT 2.0-vs-3.0 divergences (where our 3.0 output is correct), optional features, and other out-of-scope cases — none a missing mandatory Basic 3.0 facility. Running with `HELIUM_SLOW_TESTS=1` (the on-demand [`conformance.yml`](.github/workflows/conformance.yml) workflow) passes **12,824**, executing +481 performance-gated tests with 0 failures. The XSD 1.1 cases are drawn from the IBM, Saxon, Oracle, and W3C-WG collections.

## libxml2 compatibility (golden tests)

| Core XML | XSD | RELAX NG | Schematron | C14N | HTML |
|:--------:|:---:|:--------:|:----------:|:----:|:----:|
| 100% | 99.6% | 100% | 100% | 100% | 100% |

## Scope

XSLT support is intentionally scoped to Basic XSLT 3.0. Backwards-compatible processing (XSLT 1.0 semantics + XPath 1.0 compatibility mode, enabled per element when the effective version is below 2.0) is implemented and in scope; only XSLT 1.0/2.0 *syntax* support is out of scope.

# For coding agents

If you are an AI coding agent (Claude Code, Codex, Gemini, etc.) working in this
repository, start with [`AGENTS.md`](AGENTS.md) (also available as `CLAUDE.md`).
It points to the cached navigation and architecture docs under
[`.claude/docs/`](.claude/docs/) and lists the pre-read rules, scope boundaries,
and generated-file policy you must follow before making changes.

Runnable usage examples live in the [`examples/`](examples/) directory as
`*_example_test.go` files — read those first to see how the public APIs are meant
to be used.

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
