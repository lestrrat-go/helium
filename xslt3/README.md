# xslt3

The `xslt3` package compiles XSLT 3.0 stylesheets and applies them to helium
documents.

Import path: `github.com/lestrrat-go/helium/xslt3`

<!-- INCLUDE(examples/xslt3_transform_string_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform_string() {
  const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <greeting>Hello, <xsl:value-of select="person/@name"/>!</greeting>
  </xsl:template>
</xsl:stylesheet>`

  const sourceSrc = `<person name="World"/>`

  ctx := context.Background()
  p := helium.NewParser()

  stylesheetDoc, err := p.Parse(ctx, []byte(stylesheetSrc))
  if err != nil {
    fmt.Printf("parse stylesheet error: %s\n", err)
    return
  }

  stylesheet, err := xslt3.CompileStylesheet(ctx, stylesheetDoc)
  if err != nil {
    fmt.Printf("compile error: %s\n", err)
    return
  }

  sourceDoc, err := p.Parse(ctx, []byte(sourceSrc))
  if err != nil {
    fmt.Printf("parse error: %s\n", err)
    return
  }

  // TransformString is a convenience that compiles+transforms+serializes
  // in one call, returning the result as a string.
  result, err := xslt3.TransformString(ctx, sourceDoc, stylesheet)
  if err != nil {
    fmt.Printf("transform error: %s\n", err)
    return
  }

  fmt.Println(result)
  // Output:
  // <?xml version="1.0" encoding="UTF-8"?><greeting>Hello, World!</greeting>
}
```
source: [examples/xslt3_transform_string_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xslt3_transform_string_example_test.go)
<!-- END INCLUDE -->

## Conformance

`xslt3` targets **Basic XSLT 3.0** conformance (W3C XSLT 3.0 spec §27).
"Basic XSLT Processor" is the only required conformance level; the spec's
other seven levels are optional features.

Against the W3C XSLT 3.0 test suite (run from the sibling `helium-w3c-tests`
module; see "Running the conformance tests" below):

| Outcome | Count |
|---------|-------|
| Pass    | 11,156 |
| Skip    | 1,971  |
| Fail    | 0      |
| Total   | 13,127 |

There are **no failing tests**. Every skip carries an explicit reason and
falls into one of the categories below — none is a missing mandatory 3.0
instruction.

### What is implemented

All mandatory Basic XSLT 3.0 facilities work and are exercised by the suite:
template matching and modes, `xsl:apply-templates` / `xsl:call-template` /
`xsl:next-match` / `xsl:apply-imports`, variables and parameters,
`xsl:function`, `xsl:for-each-group`, sorting, `xsl:number`, keys,
attribute-sets, `xsl:result-document` (multiple result documents),
accumulators, `xsl:merge`, `xsl:iterate` / `xsl:fork`, `xsl:try` / `xsl:catch`,
maps and arrays, packages / `xsl:use-package`, and the XPath 3.1 data model.

Unsupported optional features are reported with concrete error codes
(e.g. `XTSE0090`, `XTSE0220`, `FOXT0003`) rather than silently
misinterpreted, and external resource access is default-deny.

### What is skipped, and why

| Category | ~Count | Reason |
|----------|-------:|--------|
| XSLT 1.0/2.0-only tests and backwards-compatibility mode | ~1,265 | Optional 1.0/2.0 compatibility level — **intentionally not implemented** |
| Performance-gated (run with `HELIUM_SLOW_TESTS=1`) | ~605 | CI runtime only; not capability gaps |
| Schema-awareness | ~55 | Optional level, in progress |
| Tests requiring a feature to be *absent* (we support it) | ~35 | We exceed the test's requirement |
| XML-parser-level limits (XML 1.1 control chars / ns-undeclaration, certain external entities) | ~33 | Parser layer, not the XSLT engine |
| External / non-interoperable (XQuery `load-xquery-module`, network, Saxon-specific URIs) | ~12 | Out of scope or noted non-interoperable by the W3C catalog |
| Genuine edge defects | ~25 | Narrow, individually-tracked quirks (e.g. type-annotation propagation, `snapshot()/root()` namespace nodes) |

Backwards-compatibility modes for XSLT 1.0/2.0 are **not** part of the target
feature set and will not be implemented.

### Running the conformance tests

The W3C XSLT 3.0 conformance suite (generator, harness, generated case tables,
and fixtures) lives in the sibling
[`helium-w3c-tests`](https://github.com/lestrrat-go/helium-w3c-tests) module,
which depends on this module via a `replace` directive. Run it from there:

```sh
# in ../helium-w3c-tests
go run ./cmd/w3cgen fetch xslt30      # clone upstream + copy fixtures
go run ./cmd/w3cgen generate xslt30   # regenerate the case tables
go run ./cmd/w3ctest xslt30           # run the suite, emit JUnit XML

# include the performance-gated tests skipped by default
HELIUM_SLOW_TESTS=1 go test ./xslt3/ -run TestXSLT30W3C
```

Helium keeps only the `xslt3` unit tests plus committed, point-in-time evidence
beside this package — a stamped `summary-xslt30.md` and JUnit
`results-xslt30.xml`. Regenerate them from the sibling module:

```sh
# in ../helium-w3c-tests, after fetch + generate
go run ./cmd/w3ctest -no-system-out \
  -out ../helium/xslt3/results-xslt30.xml \
  -summary ../helium/xslt3/summary-xslt30.md \
  -helium-commit "$(git -C ../helium rev-parse --short HEAD)" \
  xslt30
```
