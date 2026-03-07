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
| `c14n_test.go` | c14n | C14N golden file tests |
| `xsd_test.go` | xsd | Schema validation golden tests |
| `relaxng_test.go` | relaxng | RELAX NG golden tests |
| `schematron_test.go` | schematron | Schematron golden tests |

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
| `parseTestDoc(t, path) *Document` | Parse XML with ParseNoEnt, ParseDTDLoad, ParseDTDAttr |
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
1. Parse test XML with ParseNoEnt + ParseDTDLoad + ParseDTDAttr
2. Check for .xpath sidecar → evaluate XPath for node set
3. Check for .ns sidecar → read inclusive namespace prefixes
4. Canonicalize with mode and options
5. Compare output to result file
```

## Skip Maps

Tests are skipped via in-code maps with reasons:
- Parser limitations (duplicate xmlns, single-quoted entity refs, external entity resolution)
- Feature gaps (libxml2 quirks like IDC edge cases)
- Missing expected files in libxml2 test data
