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

## Security

External resource access is a security boundary, and `xslt3` is **default-deny**:
with no resolver configured, a compiled stylesheet can neither read the
filesystem nor reach the network. The engine itself never calls `os.Open`,
`os.ReadFile`, `os.Create`, or `http.DefaultClient` — every external retrieval is
routed through a resolver the caller must install explicitly. This is the same
posture as the parser, `xinclude`, and `xsd` compiler (see the repository-wide
[`# Security`](../README.md#security) section and
[`SECURITY.md`](../SECURITY.md), the repository security policy;
[`CONFORMANCE.md`](CONFORMANCE.md) records external-resource loading as
default-deny).

### What is gated

Every operation that would reach outside the in-memory stylesheet and source
document is gated:

| Operation | Without a resolver |
|-----------|--------------------|
| `xsl:import` / `xsl:include` (module loads) | compile fails — `XTSE0165` ("no URIResolver configured") |
| `xsl:use-package` | compile fails — no `PackageResolver` configured |
| `xsl:import-schema`, `xsi:schemaLocation` source schemas | schema load refused |
| `doc()` / `fn:doc()` / `document()` | dynamic error — "no URIResolver configured" (or "no HTTPClient or URIResolver" for `http(s)`) |
| `xsl:source-document`, `xsl:merge` `for-each-source` | dynamic error — `FODC0002` |
| `unparsed-text()` / `unparsed-text-lines()` | retrieval error — `FOUT1170` |
| Availability probes — `doc-available()`, `unparsed-text-available()`, `stream-available()` | return `false` (no error, no host access) |

`xsl:result-document` never writes to the filesystem either: secondary result
documents are delivered in-memory to a `ResultDocumentHandler` (or collected),
so output is confined to the caller's process. Retrieval functions raise their
spec-mandated errors; the *availability* probes report `false` rather than
stat-ing the host.

### Granting access

Access is granted by installing a resolver. There are two resolver interfaces —
one for compile time, one for the transform:

- Compile time: `Compiler.URIResolver(r)` where `r` satisfies
  `xslt3.URIResolver` (`Resolve(uri string) (io.ReadCloser, error)`), plus
  `Compiler.PackageResolver(r)` (`ResolvePackage(name, version string) (io.ReadCloser, string, error)`)
  for `xsl:use-package`.
- Transform time: `Invocation.URIResolver(r)` where `r` satisfies
  `xpath3.URIResolver` (`ResolveURI(uri string) (io.ReadCloser, error)`),
  `Invocation.CollectionResolver(r)` for `fn:collection`, and
  `Invocation.HTTPClient(client)` to opt in to network retrieval for
  `http(s)` URIs. Without an `HTTPClient` (and without a resolver that reaches
  the network) network access stays refused.

Back your resolver with a **confined** `fs.FS` rooted at a trusted directory and
map incoming URIs to paths within it, rejecting anything that escapes the root.
Grant only what the transform legitimately needs; do not hand untrusted
stylesheets or untrusted source documents an OS-rooted or network-capable
resolver.

```go
// A confined resolver backed by an fs.FS rooted at a trusted directory.
type confinedResolver struct{ fsys fs.FS }

// ResolveURI satisfies xpath3.URIResolver (used at transform time).
func (r confinedResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	// Map uri -> a path inside fsys however your deployment requires,
	// rejecting any path that would escape the trusted root.
	return r.fsys.Open(pathWithinRoot(uri)) // your own validating mapper
}

// Resolve satisfies xslt3.URIResolver (used at compile time, e.g. xsl:import).
func (r confinedResolver) Resolve(uri string) (io.ReadCloser, error) {
	return r.fsys.Open(pathWithinRoot(uri))
}

res := confinedResolver{fsys: trustedFS}

ss, err := xslt3.NewCompiler().
	URIResolver(res). // allow xsl:import / xsl:include from trustedFS
	Compile(ctx, stylesheetDoc)
// ...
out, err := ss.Transform(source).
	URIResolver(res). // allow doc() / xsl:source-document / unparsed-text()
	// HTTPClient(client). // opt in to network only if genuinely required
	Serialize(ctx)
```

Even after a resolver grants access, the bytes it returns are parsed with the
hardened default parser: external DTD/entity loading (XXE) stays **blocked**
unless you explicitly opt in via `Compiler.AllowExternalEntities(true)` /
`Invocation.AllowExternalEntities(true)`, and the injected `Compiler.Parser` /
`Invocation.Parser` governs the parse policy of every loaded document.

### Cancellation

Both compilation and transformation honor `context.Context` cancellation and
deadlines. `Compile`, and the transform delivery methods (`Serialize`, `Do`,
`WriteTo`), take a `ctx`, and the compile and execution loops poll `ctx.Err()`
at their iteration points (e.g. `execute_control.go`, `execute_apply.go`,
`execute_streaming.go`, `execute_misc.go`; `compile.go`, `compile_imports.go`,
`compile_templates.go`), so a cancelled or expired context aborts the work
promptly rather than running to completion. Always pass a `ctx` with a deadline
when processing untrusted input.

### Caller responsibilities

The engine bounds a few things itself: each external resource read is capped at
`MaxResourceBytes` (10 MiB default; `Invocation.MaxResourceBytes` overrides it),
which also bounds `xsl:analyze-string` match enumeration. But the engine cannot
know your overall resource budget, so the caller must still:

- Enforce a maximum **raw input size** before parsing the stylesheet and source
  document — the byte cap governs resolver-fetched resources, not the primary
  source you hand in.
- Bound **output size**: a stylesheet can fan out large numbers of
  `xsl:result-document`s and materialize large result trees in memory. Cap the
  work (or reject the stylesheet) when transforming untrusted input.
- Pass a `context.Context` with a deadline (see Cancellation above).

## Conformance

`xslt3` targets **Basic XSLT 3.0** conformance (W3C XSLT 3.0 spec §27).
"Basic XSLT Processor" is the only required conformance level; the spec's
other seven levels are optional features. See
[`CONFORMANCE.md`](CONFORMANCE.md) for the explicit, auditable conformance
declaration (level table, skip taxonomy, and the isolated narrow-quirk list).

Against the W3C XSLT 3.0 test suite (run from the sibling `helium-w3c-tests`
module; see "Running the conformance tests" below):

| Outcome | Count |
|---------|-------|
| Pass    | 12,346 |
| Skip    | 781    |
| Fail    | 0      |
| Total   | 13,127 |

This is the **reproducible local baseline** (`go test ./xslt3/`). The on-demand
`.github/workflows/conformance.yml` workflow runs the same suite with
`HELIUM_SLOW_TESTS=1` (the `slow` toggle), which additionally executes the
**+481** performance-gated cases the default run skips. They all pass, so the
slow run is a strict superset with **0 failures**:

| Outcome | Count (slow, `HELIUM_SLOW_TESTS=1`) |
|---------|-------|
| Pass    | 12,827 |
| Skip    | 300    |
| Fail    | 0      |
| Total   | 13,127 |

The slow figures are on-demand CI evidence (committed as
`results-xslt30-slow.xml`); the default baseline is what a local run reproduces.

There are **no failing tests**. **Every skip is expected, not a deficiency.**
Each carries a precise, individually-recorded reason and falls into one of the
legitimate categories below — **none is a missing mandatory Basic XSLT 3.0
facility.** The two sources of truth for the per-case reasons are:

- the harness's `w3cImplicitSkips` map in
  [`helium-w3c-tests`](https://github.com/lestrrat-go/helium-w3c-tests)
  `xslt3/w3c_helpers_test.go` — one entry per skipped case, each with a precise
  reason string; and
- the committed `xslt3/summary-xslt30.md` "Skipped by reason" table (regenerated
  from those same reason strings), beside this package.

A reader can audit any individual skip against either.

The `spec="XSLT20"`/`spec="XSLT10"` version-specific bucket (~1,120 cases) is
now **in scope and un-gated** — the generator runs it against our 3.0 processor.
About 1,015 pass as-is; the ~80 that remain skipped are **XSLT 2.0-vs-3.0
divergences** where our 3.0 output is correct (see below), not gaps.

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

Every count below is bucketed from the regenerated `summary-xslt30.md`
"Skipped by reason" table for the **default** run; the category counts sum to
the 781 total. None represents a missing mandatory Basic XSLT 3.0 facility.

| Category | Count | Why it is a legitimate skip |
|----------|------:|-----------------------------|
| Performance-gated (slow streaming / large-corpus regex; run with `HELIUM_SLOW_TESTS=1` or the on-demand workflow) | 605 | CI runtime only; these pass — not capability gaps |
| **XSLT 2.0-vs-3.0 divergences** (our 3.0 output is correct) | 80 | The 2.0-only test asserts a behavior/error XSLT 3.0 deliberately changed; it *cannot* pass on a conformant 3.0 processor |
| Spec-version divergences (XSD 1.0 test vs our XSD 1.1 target) | 4 | We target XSD 1.1; the case asserts an XSD-1.0-only regex/type error |
| Test requires a feature to be *absent* that we support | 70 | The test only applies to a processor *without* the feature (schema-awareness, disable-output-escaping, dynamic evaluation, XSD 1.1, out-of-range year components, …); we exceed its requirement |
| External / non-interoperable resources | 9 | XQuery `load-xquery-module`, network access, Saxon-format `?select=` URIs, missing upstream fixtures — out of scope or W3C-noted non-interoperable |
| XML-parser-layer limits | 2 | External/parameter entity resolution — the parser layer, not the XSLT engine |
| XPath 1.0 *grammar* differences | 3 | `div`/`mod` as a name after an operator, unprefixed `function` name test, empty function arguments — compat mode changes semantics, not the grammar |
| Narrow defects / fixture or Unicode-version dependence | 8 | Individually-tracked quirks (`base-uri()` fixture dependence, a Unicode-version `\w` classification, `format-number` shadowing, byte-exact xhtml serialization, the 1.0-only default output method) |

In the on-demand slow run (`HELIUM_SLOW_TESTS=1`) the performance-gated bucket
drops from **605 to 124** — the **+481** `HELIUM_SLOW_TESTS=1`-gated cases now
run and all pass — so the slow skip total falls to **300** (781 − 481). The
remaining 124 are the handful still too slow even for the slow CI bucket
(large-corpus regex, large-iteration `xsl:evaluate`); every other category above
is unchanged between the two runs.

#### The XSLT 2.0-vs-3.0 divergence category

These are **not gaps** — they are cases where a `spec="XSLT20"` test asserts a
behavior or error code that XSLT 3.0 deliberately removed or changed, so a
conformant 3.0 processor produces a different (correct) result and the 2.0
assertion cannot hold. They are correctly skipped, not deficiencies. Examples,
all drawn verbatim from the harness reason strings:

- **3.0-only regex constructs** (non-capturing groups, reluctant quantifiers)
  the 2.0 test expects to reject with `FORX0002`; 3.0 accepts them.
- **Removed static/dynamic error codes** the 2.0 test still expects:
  `XTSE0340` (relaxed match/error pattern syntax), `XTSE0010`
  (`xsl:sequence` with a contained sequence constructor), `XPST0017`
  (functions/arities added in 3.0 / XPath 3.1), `XTTE0520` and `XTTE1120`
  (`apply-templates`/`for-each-group` `select` type errors — a non-node
  population is now handled by the built-in atomic template rule / never matches
  a pattern), and `XTDE0047`/`XTDE0060` (initial-template + initial-mode /
  required-param conflicts removed by W3C bug 28418).
- **`current-group()` / `current-grouping-key()` out of context**, an empty
  sequence in 2.0 but a dynamic error (`XTDE1061`/`XTDE1071`) in 3.0 — our
  processor correctly errors (the paired 3.0 variant passes).
- **Conflicting `xsl:strip-space`/`xsl:preserve-space`**, a recoverable error in
  1.0/2.0 but a static error `XTSE0270` in 3.0 — which we correctly raise.
- **XPath 3.1 fractional-second truncation** in `format-date`/`format-time`
  where the 2.0 case asserts rounding; the 3.0+ variant passes.
- **3.0 function-library availability** (`element-available('xsl:key')`,
  `generate-id` / document access in `use-when`, run-time `xs:QName(string)`
  casts, `copy-of`/`snapshot`/`parse-json` and the F+O 3.0 library) that the
  2.0 test asserts is absent.

Each such case is paired in `w3cImplicitSkips` with the passing 3.0 variant, so
the divergence is auditable.

Backwards-compatible processing (XSLT 1.0 behavior + XPath 1.0 compatibility
mode, enabled per element when the effective `[xsl:]version` is below 2.0) **is
implemented and in scope**; see the Backwards-Compatible Processing section in
the repository `CLAUDE.md`. Only XSLT 1.0/2.0 *syntax* support (the grammar, not
the semantics) remains out of scope.

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

The performance-gated cases (the 605 counted above) are not run in the default
CI pass. Helium ships an on-demand GitHub Actions workflow
(`.github/workflows/conformance.yml`, `workflow_dispatch` with a `slow` toggle,
plus a nightly cron) whose `slow` toggle sets `HELIUM_SLOW_TESTS=1`; a slow run
passes **481 additional** performance-gated tests at **0 failures** (12,827 pass
/ 300 skip / 0 fail, total 13,127), confirming they are CI runtime gates, not
capability gaps. That workflow is how the slow figures above are produced.

Helium keeps only the `xslt3` unit tests plus committed, point-in-time evidence
beside this package — a stamped `summary-xslt30.md` and JUnit
`results-xslt30.xml` for the default run, plus `results-xslt30-slow.xml` for the
on-demand slow run. Regenerate the default evidence from the sibling module:

```sh
# in ../helium-w3c-tests, after fetch + generate
go run ./cmd/w3ctest -no-system-out \
  -out ../helium/xslt3/results-xslt30.xml \
  -summary ../helium/xslt3/summary-xslt30.md \
  -helium-commit "$(git -C ../helium rev-parse --short HEAD)" \
  xslt30
```
