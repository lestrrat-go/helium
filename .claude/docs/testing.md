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
- **`*_internal_test.go`** (internal `xxx` package) — tests needing unexported access

### Common Test File Names

| File | Package | Purpose |
|------|---------|---------|
| `libxml2_compat_test.go` | root, html, catalog | Golden file comparison suite |
| `parser_test.go` | root | Core parser coverage: XML decl, names/QNames, depth limits, general parse regressions |
| `parser_sax_test.go` | root | SAX/dispatch/stop-parser regression coverage |
| `parser_entity_test.go` | root | Entity expansion, XXE, amplification, and boundary regressions |
| `parser_push_test.go` | root | Push parser coverage |
| `writer_test.go` | root | XML writer/escaping coverage and related benchmarks |
| `c14n_test.go` | c14n | C14N golden file tests |
| `xsd_test.go` | xsd | Schema validation golden tests |
| `relaxng_test.go` | relaxng | RELAX NG golden tests |
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

## Build Tags

- `-tags debug` — used in CI (`go test -v -race -tags debug ./...`)
- No `//go:build` tags in test files

## Fuzzing

- Public-package fuzz coverage lives in package-local `fuzz_test.go` files.
- Direct fuzz targets exist for `.`, `c14n`, `catalog`, `html`, `relaxng`, `schematron`, `sink`, `stream`, `xinclude`, `xpath1`, `xpath3`, `xpointer`, `xsd`, `xslt3`.
- `shim` intentionally excluded from repo fuzz matrix.
- `enum` + `sax` intentionally excluded from direct fuzzing → constants/interface-only surface.
- Bound fuzz input sizes early. Return on oversize inputs.
- Prefer in-memory stubs over filesystem/network access.
- Parse/compile/validate/transform fuzz targets MUST tolerate invalid intermediate inputs by returning early instead of asserting.

## Fuzz CI

- `ci.yml` runs smoke fuzzing on push/PR.
- Smoke fuzz duration → `5s` per discovered `Fuzz...` target.
- `fuzz.yml` runs deep fuzzing on weekly schedule + manual dispatch.
- Deep fuzz duration default → `5m` per discovered `Fuzz...` target.
- Workflows discover fuzz targets via `go test ./<pkg>/ -list '^Fuzz' -run '^$'`.

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

### 4. QT3 Tests (XPath 3.1)

W3C QT3 test suite for XPath 3.1. Generated by `tools/qt3gen/`.

```
testdata/qt3ts/
├── source/           # QT3 test suite (fetched by fetch.sh, gitignored)
│   └── catalog.xml   # Test catalog
├── testdata/         # Context documents (copied by qt3gen)
└── fetch.sh          # Clones QT3 test suite
```

#### Generated test files

`qt3gen` groups tests by set-name prefix into per-category files:

| File | Category |
|------|----------|
| `qt3_fn_gen_test.go` | `fn-*` built-in functions |
| `qt3_op_gen_test.go` | `op-*` operators |
| `qt3_prod_gen_test.go` | `prod-*` language productions |
| `qt3_app_gen_test.go` | `app-*` application tests |
| `qt3_math_gen_test.go` | `math-*` math functions |
| `qt3_map_gen_test.go` | `map-*` map functions |
| `qt3_array_gen_test.go` | `array-*` array functions |
| `qt3_xs_gen_test.go` | `xs-*` type constructor tests |
| `qt3_misc_gen_test.go` | `misc-*` miscellaneous |
| `qt3_method_gen_test.go` | `method-*` serialization methods |

Regenerate: `go run ./tools/qt3gen/`
Tests with `feature="xpath-1.0-compatibility"` are omitted rather than generated.

Old single file `qt3_generated_test.go` is auto-removed on regeneration.

#### QT3 test helpers (`qt3_helpers_test.go`)

| Type/Function | Purpose |
|---------------|---------|
| `qt3Test` | Test case struct: Name, XPath, DocPath, Namespaces, DefaultLanguage, Skip, ExpectError, Assertions |
| `qt3Assertion` | Assertion interface for result checking |
| `qt3RunTests(t, []qt3Test)` | Table-driven runner: parse doc, build `xpath3.Context` options (timezone, language, namespaces, explicit base URI, default `/fots/fn/` base for `unparsed-text*()` fixtures, HTTP), compile XPath, evaluate, check assertions |
| `qt3AssertEq(string)` | Assert result equals literal |
| `qt3AssertStringValue(string)` | Assert string value of result |
| `qt3AssertTrue()` / `qt3AssertFalse()` | Assert boolean result |
| `qt3AssertEmpty()` | Assert empty sequence |
| `qt3AssertCount(n)` | Assert sequence length |
| `qt3AssertType(string)` | Assert result type |
| `qt3AssertDeepEq(string)` | Assert deep equality |
| `qt3AnyOf(checks...)` | Any-of assertion (first passing check wins) |

### 5. W3C XSLT 3.0 Tests

W3C XSLT 3.0 test suite for XSLT 3.0. Generated by `tools/xslt3gen/`.

```
testdata/xslt30/
├── source/           # W3C XSLT 3.0 test suite (fetched, gitignored)
│   └── catalog.xml   # Test catalog
├── testdata/         # Context documents (copied by xslt3gen)
└── fetch.sh          # Clones XSLT 3.0 test suite
```

#### Generated test files

`xslt3gen` groups tests by category:

| File | Category |
|------|----------|
| `w3c_attr_gen_test.go` | Attribute tests |
| `w3c_decl_gen_test.go` | Declaration tests |
| `w3c_expr_gen_test.go` | Expression tests |
| `w3c_fn_gen_test.go` | Function tests |
| `w3c_insn_gen_test.go` | Instruction tests |
| `w3c_misc_gen_test.go` | Miscellaneous tests |
| `w3c_strm_gen_test.go` | Streaming tests |
| `w3c_type_gen_test.go` | Type tests |

Regenerate: `go run ./tools/xslt3gen/`

#### W3C XSLT test helpers (`w3c_helpers_test.go`)

Test runner for W3C XSLT 3.0 conformance: compiles stylesheet, transforms input, compares output against expected results.

## Skip Maps

Tests are skipped via in-code maps with reasons:
- Parser limitations (duplicate xmlns, single-quoted entity refs, external entity resolution)
- Feature gaps (libxml2 quirks like IDC edge cases)
- Missing expected files in libxml2 test data
