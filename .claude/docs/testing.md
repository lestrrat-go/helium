# Testing Patterns

## Test Data

All committed test data: `testdata/libxml2-compat/`. Generated from libxml2 source via `testdata/libxml2/generate.sh`.

### Directory Layout

```
testdata/libxml2-compat/
├── *.xml + *.xml.expected           # DOM roundtrip (150+ files)
├── *.xml.sax2.expected              # SAX2 event traces
├── c14n/
│   ├── without-comments/test/ + result/
│   ├── with-comments/test/ + result/
│   ├── exc-without-comments/test/ + result/
│   └── 1-1-without-comments/test/ + result/
├── schemas/test/ + result/          # XSD (.xsd + .xml → .err)
├── relaxng/test/ + result/          # RELAX NG (.rng + .xml → .err)
├── schematron/test/ + result/       # Schematron (.sct + .xml → .err)
├── html/                            # HTML (.html → .sax, .ser, .err)
├── xpath/expr/ + tests/ + docs/     # XPath expression tests
├── xinclude/docs/ + ents/ + result/ # XInclude tests
├── catalogs/                        # Catalog resolution tests
└── valid/dtds/                      # DTD validation tests
```

### Golden File Naming

| Extension | Content |
|-----------|---------|
| `.expected` | Serialized XML output (DOM roundtrip) |
| `.sax2.expected` | SAX2 event stream trace |
| `.sax` | HTML SAX event trace |
| `.ser` | HTML serialization output |
| `.err` | Validation/compilation error output |
| `.xpath` | XPath expression for C14N node-set (sidecar) |
| `.ns` | Inclusive namespace prefixes for exclusive C14N (sidecar) |
| (no extension) | C14N result files |

### Golden File Generation

`testdata/libxml2/generate.sh` copies from libxml2 source and applies:
1. SAX2 buffer artifact fix — truncates displayed attribute values (%.4s → 4 byte limit)
2. SAX character event merging — merges consecutive `SAX.characters()` events
3. Error file patching — corrects parser-specific error messages

## Test File Conventions

### Package Naming

- **`*_test.go`** (external `xxx_test` package) — golden file comparison, SAX events, serialization. Preferred for all new tests.
- **`*_internal_test.go`** (internal `xxx` package) — tests needing unexported access. In the root package there is one per production area: `dtd_internal_test.go`, `node_internal_test.go`, `parser_internal_test.go`, `writer_internal_test.go`.

### Test Shape

Root-package and `xslt3` tests are grouped by the production function or method
under test: one top-level `TestXxx` per production entry point, with `t.Run`
subtests naming the scenario (`TestParseMalformed/"a duplicate attribute"`).
Nesting stops at two subtest levels, and a subtest name never contains `/`,
which would read as a further level. A file is split by production area once it
approaches ~2000 lines. Name a test file after the production file it covers
(`compile_patterns.go` ↔ `compile_patterns_test.go`), never after the scenario or
bug that prompted it.

### Common Test File Names

| File | Package | Purpose |
|------|---------|---------|
| `libxml2_compat_test.go` | root, html, catalog | Golden file comparison suite |
| `parser_test.go` | root | Core parse entry points: `Parse`/`ParseFile`, names/QNames, namespaces, malformed input, options, recovery |
| `parser_decl_test.go` | root | XML declaration, lenient declaration, BOM/UCS-4/UTF-16 encoding detection, encoding declarations |
| `parser_reader_test.go` | root | `ParseReader` streaming, EBCDIC decoding, context cancellation |
| `parser_dtd_test.go` | root | External DTD loading: size/read limits, malformed declarations, PE expansion |
| `parser_dtd_subset_test.go` | root | Conditional sections and external-subset text declarations |
| `parser_attlist_test.go` | root | `<!ATTLIST>` parsing and attribute-value validation by declared type |
| `parser_entity_test.go` | root | General entity substitution, predefined entities, references, entity-value validation, undeclared entities, SAX `GetEntity` error paths |
| `parser_entity_param_test.go` | root | Parameter entities and PE/markup boundary rules |
| `parser_entity_external_test.go` | root | External general and parameter entities, text declarations, base-URI resolution |
| `parser_entity_limits_test.go` | root | Entity amplification, depth and size caps |
| `parser_limits_test.go` | root | Depth, name-length, node-content and char-buffer limits |
| `parser_security_test.go` | root | FS confinement, safe defaults, network gating, XXE |
| `parser_whitespace_test.go` | root | Blank stripping, whitespace preservation, over-cap whitespace |
| `parser_charref_test.go` | root | Character references and `CreateCharRef` |
| `parser_xml11_test.go` | root | XML 1.1 characters and prefix undeclaration |
| `parser_xmlchar_test.go` | root | XML character validation and attribute-value parsing |
| `parser_sax_test.go` | root | SAX/dispatch/stop-parser regression coverage |
| `parser_push_test.go` | root | Push parser coverage |
| `writer_test.go` | root | Core serialization, writer options, write errors, benchmarks |
| `writer_escape_test.go` | root | Invalid-character rejection, character maps, normalization, injection rejection |
| `writer_namespace_test.go` | root | Namespace emission and subtree reconciliation |
| `writer_dtd_test.go` | root | DTD serialization (subset/escaping/formatting/self-close, entity and literal emission) |
| `writer_xhtml_test.go` | root | XHTML serialization output |
| `writer_encoding_test.go` | root | Output encoding, US-ASCII escaping, the no-override path |
| `writer_declaration_test.go` | root | XML declaration emission |
| `attr_test.go` | root | Attribute creation, lookup, namespaces, list repair |
| `element_test.go` | root | Element creation, content, AddChild/AddSibling/Replace |
| `node_test.go` | root | Generic node linkage shape, consistency, defensive copies, cycle and owned-boundary guards |
| `node_leaf_test.go` | root | Text/Comment/CDATA/PI/Entity/Notation node methods and guards |
| `node_namespace_test.go` | root | `DeclareNamespace` collapse and namespace lookup |
| `tree_test.go` | root | Base URIs, tree mutation, `Walk`, node accessors |
| `copy_test.go` | root | `CopyNode`/`CopyDoc`/`CopyDTDInfo`/`CopyExtSubset` deep-copy coverage; DTD attribute-declaration order fidelity |
| `dtd_test.go` | root | DTD data-model: internal-subset accessors, element/attr/notation decls, node wrappers |
| `valid_test.go` | root | Document-level DTD validation and external-subset lookup |
| `valid_dtd_decl_test.go` | root | DTD declaration-consistency VCs and declaration-order stability of their diagnostics |
| `valid_attr_test.go` | root | Attribute-type, entity-attribute, required/fixed and per-instance attribute validity; declaration-order stability of attribute diagnostics |
| `valid_content_test.go` | root | Content-model and element-content validation |
| `tree_builder_test.go` | root | SAX-path tree construction (`TreeBuilder`) |
| `c14n_test.go` | c14n | C14N golden file tests |
| `xsd_test.go` | xsd | Schema validation golden tests |
| `relaxng_test.go` | relaxng | RELAX NG golden tests |
| `group_backtrack_differential_test.go` | relaxng | Flag-gated group-backtracking differential harness |
| `schematron_test.go` | schematron | Schematron golden tests |
| `utf8cursor_test.go` | internal/strcursor | UTF-8 cursor boundary/normalization and ASCII QName scanner regression coverage |

## `examples/`

- `examples/` holds executable Go examples in external package `examples_test`.
- Treat files here as first-class user documentation. Regression coverage is secondary.
- Optimize every example for user clarity, narrow scope, and copy/paste utility.
- Keep each example focused on one concept or one end-to-end workflow. Split broad coverage into multiple files.
- Write comments for users, not maintainers. Explain visible behavior, required context, and why API calls matter.
- Prefer `func Example_*()` + deterministic `// Output:` blocks when behavior is stable.
- Keep shared setup/helpers in `*_helpers_test.go` so example bodies stay easy to read.
- CLI examples call importable entrypoints (e.g. `internal/cli/heliumcmd.Execute`) directly. Do NOT spawn subprocesses unless behavior requires it.
- Do NOT use `examples/` for scratch programs, golden fixtures, or temporary experiments.

## Test Helpers

### internal/heliumtest

Shared test utilities in `internal/heliumtest/callerdir.go`:

| Function | Purpose |
|----------|---------|
| `CallerDir(skip)` | Directory of caller's source file (skip=0 for direct caller) |
| `RepoRoot()` | Absolute path to repo root (finds go.mod, cached) |
| `TestDir(path...)` | Join path elements under repo root |

### SAX Event Normalization

| Function | Package | Purpose |
|----------|---------|---------|
| `mergeCharactersEvents(s string) string` | root | Merge consecutive `SAX.characters()` events |
| `mergeHTMLCharEvents(s string) string` | html | Merge HTML `characters()` + `cdata()` events |
| `normalizeCharDisplays(s string) string` | html | Replace truncated display strings in merged events |
| `newLibxml2EventEmitter(io.Writer) sax.SAX2Handler` | root | SAX2 handler matching libxml2 output format |
| `newHTMLSAXEventEmitter(*bytes.Buffer) html.SAXHandler` | html | HTML SAX handler matching libxml2 format |

### C14N Helpers

| Function | Purpose |
|----------|---------|
| `parseTestDoc(t, path) *Document` | Parse XML with SubstituteEntities, LoadExternalDTD, DefaultDTDAttributes |
| `readExpected(t, path) []byte` | Read expected result file |
| `parseXPathFile(t, path) (string, map[string]string)` | Parse .xpath sidecar → expression + namespace bindings |
| `parseNSFile(t, path) []string` | Parse .ns sidecar → inclusive namespace prefixes |
| `evaluateNodeSet(t, doc, expr, nss) []Node` | Evaluate XPath → node set |

### Validation Helpers (shared pattern across xsd, relaxng, schematron)

| Function | Purpose |
|----------|---------|
| `discoverTests(t) []testCase` | Walk result/ dir for `{base}_{N}.err` → (schema, instance, result) triples |
| `partitionCompileErrors([]error) (warnings, errors string)` | Split errors by ErrorLevelFatal |
| `shouldSkip(name) string` | Check skip maps (prefix + exact match) → skip reason |

## Environment Variable Filtering

Run specific test subsets via env vars:

| Variable | Test Suite |
|----------|-----------|
| `HELIUM_LIBXML2_TEST_FILES` | Root XML compatibility tests |
| `HELIUM_LIBXML2_SAX2_TEST_FILES` | SAX2 event tests |
| `HELIUM_HTML_TEST_FILES` | HTML parser tests |
| `HELIUM_XMLSCHEMA_TEST_FILES` | XSD validation tests |
| `HELIUM_RELAXNG_TEST_FILES` | RELAX NG tests |
| `HELIUM_SCHEMATRON_TEST_FILES` | Schematron tests |

## Differential Harnesses

`relaxng/group_backtrack_differential_test.go` compares two revisions on
accept/reject decisions AND exact error text. It prints one line per case
(identity, verdict, quoted error text) for the golden RELAX NG schema
cross-product and for seeded random group grammars, so the same file run on two
checkouts produces two outputs that can be diffed. It uses only the exported
API, so it can be copied into an older checkout unchanged.

| Flag | Meaning |
|------|---------|
| `-relaxng.differential.out=PATH` | Write the record to PATH. Absent: discard the output and, unless `cases=0`, run a small subset |
| `-relaxng.differential.cases=N` | Randomized grammars (default 20000; `0` runs the golden cross-product only, with or without `out`) |
| `-relaxng.differential.seed=N` | Seed for the randomized grammars (default 1) |

Without flags an ordinary `go test ./relaxng` run walks every eighth golden
schema plus 200 random grammars in about half a second, and
`TestGroupBacktrackDifferentialDeterministic` checks that two runs of that
subset agree byte for byte. The reduced subset is what the absent `out` flag
selects for the default case count; `cases=0` turns the randomized half off and
walks the full golden cross-product either way. The recorded procedure and
result for the current run live in the header comment of
`relaxng/group_backtrack_test.go`.

## Build Tags

- `-tags debug` — used in CI (`go test -v -race -tags debug ./...`)
- No `//go:build` tags in test files

## Fuzzing

- Public-package fuzz coverage lives in package-local `fuzz_test.go` files.
- Direct fuzz targets exist for `.`, `c14n`, `catalog`, `html`, `relaxng`, `schematron`, `sink`, `stream`, `xinclude`, `xpath1`, `xpath3`, `xpointer`, `xsd`, `xmldsig1`, `xmlenc1`, `xslt3`.
- `shim` intentionally excluded from repo fuzz matrix.
- `enum` + `sax` intentionally excluded from direct fuzzing → constants/interface-only surface.
- Bound fuzz input sizes early. Return on oversize inputs.
- Prefer in-memory stubs over filesystem/network access.
- Parse/compile/validate/transform fuzz targets MUST tolerate invalid intermediate inputs by returning early instead of asserting.
- `xmlenc1`'s `FuzzDecrypt` puts FIXED RSA, EC, and AES key material on one `Decryptor` (all three are
  settable together) and leaves `SessionKey` unset, so the input alone selects which key-protection path runs
  — RSA-OAEP, ECDH-ES, or AES key wrap — and every branch is reachable from the one target. The keys are
  fixed, because a crasher written under `testdata/fuzz` must reproduce across runs.
- The `xslt3` targets time each input's parse+compile (and transform) inline and fail via `t.Errorf` when it
  crosses `slowInputThreshold()` (`fuzz_test.go` `flagIfSlow`), so the fuzzing engine persists the exact bytes
  as a crasher. Go's own worker already turns a genuine hang into a crasher via a 10s deadlock detector
  (`internal/fuzz` `worker.go`), so this targets the slow-but-finite input (a few seconds) that 10s net misses
  — the input that drags run throughput toward the fuzztime deadline and surfaces only as an unactionable
  `context deadline exceeded` with no reproducer. The threshold MUST stay below Go's 10s worker deadline to
  fire first; it defaults to 5s (ample headroom over any legitimate compile, so no false trips under CI
  scheduler jitter) and is overridable via `HELIUM_FUZZ_SLOW_INPUT` (a Go duration). Timing inline (not in a
  child goroutine) keeps panics on `testing`'s normal minimizable-crasher path.

## Fuzz CI

- Each matrix job restores newest Go fuzz corpus for same `go.mod`/package/ref, then falls back to same `go.mod`/package across refs.
- Each matrix job saves its corpus under an immutable run/attempt key after success or failure.
- Pull requests run NO fuzzing — `ci.yml` is normal test/build/lint/vuln verification only, so PR turnaround stays fast and deterministic (live fuzzing is nondeterministic and cannot gate a PR without flaking).
- `fuzz.yml` runs fuzzing OFF the PR path, always non-gating:
  - on every `push` to `main` (in practice, each PR merge) → short `60s` per target, for a prompt signal attributed to the pushed commit.
  - on the weekly `schedule` → deep `5m` per target.
  - on manual `workflow_dispatch` → its `fuzz-time` input (default `5m`).
- Fuzz targets are discovered per package via `go test ./<pkg>/ -list '^Fuzz' -run '^$'`; each target writes a separate log and retains its raw `go test` status.
- Every nonzero raw status or log-capture failure uploads target log + metadata (`package`, target, Go version, fuzz budget, raw/log statuses, classification) as a diagnostic artifact; log-capture failures fail the job.
- Only Go coordinator failures matching the complete deadline-only signature are warnings ([golang/go#75804](https://github.com/golang/go/issues/75804)); any extra diagnostic, panic, worker hang, source failure, or crashing input fails the job.
- Go-written crashing corpus files are uploaded separately for real failures; fuzz artifacts are not committed.

## Common Test Patterns

### 1. Golden File Comparison (DOM/SAX)

```
1. Iterate testdata dir for input files (skip .expected, .err, .sax2.*)
2. Check skip map and env var filter
3. Parse input → serialize output
4. Compare against .expected golden file
5. On mismatch, save actual to .err for debugging
```

### 2. Schema Validation (XSD/RELAX NG/Schematron)

```
1. discoverTests() walks result/ for {base}_{N}.err files
2. Extract schema path (test/{base}.xsd) + instance path (test/{base}_{N}.xml)
3. Compile schema with ErrorCollector (ErrorLevelNone to capture all)
4. Validate instance against schema
5. Partition compile + validation errors by severity
6. Compare concatenated output against .err golden file
```

### 3. C14N Tests

```
1. Parse test XML with SubstituteEntities + LoadExternalDTD + DefaultDTDAttributes
2. Check for .xpath sidecar → evaluate XPath for node set
3. Check for .ns sidecar → read inclusive namespace prefixes
4. Canonicalize with mode and options
5. Compare output to result file
```

### 4. QT3 Tests (XPath 3.1) — moved out of this module

The W3C QT3 (XPath/XQuery 3.1) conformance suite lives in the **sibling
`github.com/lestrrat-go/helium-w3c-tests` module**, not here. That module owns the generator
(`internal/suites/qt3`), the harness and generated per-category case tables (`xpath3/qt3_*_gen_test.go`, run
via one `TestQT3W3C`), the on-demand-fetched context/resource fixtures plus the committed curated overlay
(`fixtures/qt3ts`), and the skip/expectation metadata (`expectations/qt3.json`); it `replace`s `helium =>
../helium` and uses a local `go.work`. Run it from there: `go run ./cmd/w3cgen fetch qt3 && go run
./cmd/w3cgen generate qt3 && go run ./cmd/w3ctest qt3`. Helium keeps only the xpath3 **unit** tests.

### 5. W3C XSLT 3.0 Tests — moved out of this module

The W3C XSLT 3.0 conformance suite lives in the **sibling `github.com/lestrrat-go/helium-w3c-tests` module**,
not here. That module owns the generator (`internal/suites/xslt30`), the harness and generated per-category
case tables (`xslt3/xslt30_*_gen_test.go`, run via one `TestXSLT30W3C`), the on-demand-fetched fixtures plus
the committed curated overlay (`fixtures/xslt30`), and the skip/expectation metadata
(`expectations/xslt30.json`); it `replace`s `helium => ../helium` and uses a local `go.work`. Run it from
there: `go run ./cmd/w3cgen fetch xslt30 && go run ./cmd/w3cgen generate xslt30 && go run ./cmd/w3ctest
xslt30`. Helium keeps only the xslt3 **unit** tests.

### 6. W3C XML Schema Test Suite (XSTS) — moved out of this module

The heavyweight W3C XML Schema (XSD 1.1) conformance suite lives in the **sibling
`github.com/lestrrat-go/helium-w3c-tests` module**, not here. That module owns the generated tests, the
on-demand-fetched fixtures, and the skip/expectation metadata (`expectations/xsd11.json`); it `replace`s
`helium => ../helium` and uses a local `go.work` to test against an in-progress branch. Run it from there: `go
run ./cmd/w3cgen fetch xsd11 && go run ./cmd/w3cgen generate xsd11 && go test ./...`.

Helium keeps only the **unit regression** `xsd/union_cycle_overflow_test.go` (cyclic simpleType must error, not stack-overflow), guarding the in-tree fix (`baseChain` in `simplevalue_core.go`, `checkCircularSimpleTypes` in `check_facets.go`).

## Skip Maps

Tests are skipped via in-code maps with reasons:
- Parser limitations (duplicate xmlns, single-quoted entity refs, external entity resolution)
- Feature gaps (libxml2 quirks like IDC edge cases)
- Missing expected files in libxml2 test data
