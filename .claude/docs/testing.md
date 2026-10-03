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

- **`*_test.go`** (external `xxx_test` package) — golden file comparison, SAX events, serialization. Preferred for all
  new tests.
- **`*_internal_test.go`** (internal `xxx` package) — tests needing unexported access. In the root package there is one
  per production area: `dtd_internal_test.go`, `node_internal_test.go`, `parser_internal_test.go`,
  `writer_internal_test.go`.

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
| `parser_document_test.go` | root | Element content across input encodings (UTF-8 with/without declaration or BOM, XML 1.1, US-ASCII, ISO-8859-1, windows-1252, UTF-16, UCS-4, EBCDIC 037, Shift_JIS, EUC-JP) and entry points (`Parse`, `ParseReader`, one-byte push, `ParseInNodeContext`, internal and external entity content): tree and error text must match the UTF-8 baseline, and no parse may return `ErrContentCursorForTesting` |
| `parser_decl_test.go` | root | XML declaration, lenient declaration, BOM/UCS-4/UTF-16 encoding detection, encoding declarations |
| `parser_reader_test.go` | root | `ParseReader` streaming, EBCDIC decoding, context cancellation |
| `parser_dtd_test.go` | root | External DTD loading: size/read limits, malformed declarations, PE expansion |
| `parser_dtd_subset_test.go` | root | Conditional sections and external-subset text declarations |
| `parser_attlist_test.go` | root | `<!ATTLIST>` parsing and attribute-value validation by declared type |
| `parser_entity_test.go` | root | General entity substitution, predefined entities, references, entity-value validation, undeclared entities, SAX `GetEntity` error paths |
| `parser_entity_param_test.go` | root | Parameter entities and PE/markup boundary rules |
| `parser_entity_external_test.go` | root | External general and parameter entities, text declarations, resource encodings (BOM, EBCDIC, UTF-16) across the external entity and subset loaders, base-URI resolution |
| `parser_entity_limits_test.go` | root | Entity amplification, depth and size caps |
| `parser_limits_test.go` | root | Depth, name-length, node-content and char-buffer limits |
| `parser_security_test.go` | root | FS confinement, safe defaults, network gating, XXE |
| `parser_whitespace_test.go` | root | Blank stripping, whitespace preservation, over-cap whitespace |
| `parser_charref_test.go` | root | Character references and `CreateCharRef` |
| `parser_xml11_test.go` | root | XML 1.1 characters and prefix undeclaration |
| `parser_xmlchar_test.go` | root | XML character validation and attribute-value parsing |
| `parser_sax_test.go` | root | SAX/dispatch/stop-parser regression coverage |
| `parser_push_test.go` | root | Push parser coverage |
| `writer_test.go` | root | Core serialization, writer options, write errors, allocation bounds (`TestWriteToAllocations`: escaped text, escaped attributes, and prefixed elements on the XML and XHTML paths allocate nothing per repeat), benchmarks |
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
| `copy_test.go` | root | `CopyNode`/`CopyDoc`/`CopyDTDSubsets`/`CopyDTDInfo`/`CopyExtSubset` deep-copy coverage; DTD attribute-declaration order fidelity |
| `dtd_test.go` | root | DTD data-model: internal-subset accessors, element/attr/notation decls, node wrappers |
| `valid_test.go` | root | Document-level DTD validation and external-subset lookup |
| `valid_dtd_decl_test.go` | root | DTD declaration-consistency VCs and declaration-order stability of their diagnostics |
| `valid_attr_test.go` | root | Attribute-type, entity-attribute, required/fixed and per-instance attribute validity; declaration-order stability of attribute diagnostics |
| `valid_content_test.go` | root | Content-model and element-content validation |
| `tree_builder_test.go` | root | SAX-path tree construction (`TreeBuilder`) |
| `c14n_test.go` | c14n | C14N golden file tests |
| `xsd_test.go` | xsd | Schema validation golden tests |
| `xsd_benchmark_test.go` | xsd | Schema compilation and valid-document validation benchmarks |
| `relaxng_test.go` | relaxng | RELAX NG golden tests |
| `interleave_test.go` | relaxng | Interleave §7.4 conflict checks and golden-schema conflict coverage |
| `interleave_internal_test.go` | relaxng | Interleave partition/routing table (internal package) |
| `group_backtrack_differential_test.go` | relaxng | Flag-gated group-backtracking differential harness |
| `interleave_differential_test.go` | relaxng | Flag-gated interleave differential harness (optional `xmllint` oracle) |
| `validate_concurrency_test.go` | relaxng | Every golden instance validated from 8 goroutines sharing one `Grammar` (run with `-race`) |
| `schematron_test.go` | schematron | Schematron golden tests |
| `bytecursor_test.go` | internal/strcursor | ByteCursor read-error and zero-progress handling; `TestCursorPosition` checks line, column, and line text after every advancing method on both ByteCursor and UTF8Cursor |
| `utf8cursor_test.go` | internal/strcursor | UTF-8 cursor boundary/normalization, ASCII QName scanner regression coverage, `ScanCharDataSlice` run/validity checks against a character-at-a-time reference (`FuzzScanCharDataSlice`), `ScanSimpleAttrValue` against a byte-at-a-time reference (`FuzzScanSimpleAttrValue`) and over every code point, and `AdvanceFast`/`AdvanceNoNewline` line/column against `Advance` |

## `examples/`

- `examples/` holds executable Go examples in external package `examples_test`.
- Treat files here as first-class user documentation. Regression coverage is secondary.
- Optimize every example for user clarity, narrow scope, and copy/paste utility.
- Keep each example focused on one concept or one end-to-end workflow. Split broad coverage into multiple files.
- Write comments for users, not maintainers. Explain visible behavior, required context, and why API calls matter.
- Prefer `func Example_*()` + deterministic `// Output:` blocks when behavior is stable.
- Keep shared setup/helpers in `*_helpers_test.go` so example bodies stay easy to read.
- CLI examples call importable entrypoints (e.g. `internal/cli/heliumcmd.Execute`) directly. Do NOT spawn subprocesses
  unless behavior requires it.
- Do NOT use `examples/` for scratch programs, golden fixtures, or temporary experiments.
- An example that needs files on disk writes them under `os.MkdirTemp("", ...)` (the system temp dir), never
  under the package directory, so `go test ./examples/` runs from a read-only checkout.

## Test Helpers

### internal/heliumtest

Shared test utilities in `internal/heliumtest/` (`callerdir.go`, `pollctx.go`):

| Function | Purpose |
|----------|---------|
| `CallerDir(skip)` | Directory of caller's source file (skip=0 for direct caller) |
| `RepoRoot()` | Absolute path to repo root (finds go.mod, cached) |
| `TestDir(path...)` | Join path elements under repo root |
| `NewPollContext(parent, expireAt, err)` | `context.Context` counting its `Err` calls; the `expireAt`-th call and later return `err` and close `Done` (`expireAt <= 0` never expires). `Polls()`, `PollsAfterExpiry()` |

## Timing-Free Assertions

Tests never pass or fail on elapsed time. Express the protected property as work instead:

- Cancellation/deadline honored mid-run → `heliumtest.PollContext` placing the cancellation or deadline at a
  fixed poll (helium's walks poll through `ctx.Err()`), then assert the context error and a small
  `PollsAfterExpiry()`. A pre-cancelled context covers the entry checks.
- Laziness, bounded backtracking, linear growth → count work, as described in Resource-Measuring Assertions
  below.
- Blocking behavior → synchronize on a signal from the code under test (a context whose `Err` or `Done`
  signals, a reader that signals when a blocking `Read` starts), never on a sleep.
- Code that parks on channels or `sync.Cond` (push parsers, the catalog load dedup) → run the test in a
  `testing/synctest` bubble (`synctest.Test`) and call `synctest.Wait()`. It returns once every goroutine
  in the bubble is blocked, so the parser has consumed what was pushed and is waiting in the push stream's
  `Read`. A goroutine blocked in a syscall or the network poller never counts as blocked, so FIFO and
  socket reads use a `Done` signal instead (`catalog/load_cancel_test.go` `watchContext`).
- Clock-dependent results (`fn:current-dateTime`) → a synctest bubble, whose fake clock moves only by what
  the test sleeps.
- Goroutine leaks → start the work on a goroutine carrying a pprof label (`pprof.SetGoroutineLabels`);
  every goroutine it starts inherits the label. Count labelled goroutines in the goroutine profile until
  none remain (`catalog/load_cancel_test.go` `waitLabeledGoroutinesExit`). Never compare
  `runtime.NumGoroutine`, which counts every parallel test's goroutines.
- `time.After` stays only as a hang guard that fails a test which would otherwise never return.

## Resource-Measuring Assertions

A test that guards a memory or work bound gives the same verdict on every machine and Go toolchain, on every
`GOARCH` and `GOMAXPROCS`, and under `-race`.

- Count the work where the code already counts it, and read the count through `export_test.go`.
  `ParseStateOfParseForTesting` (root package) returns the entity-expansion bytes the parser charged and
  whether the parse allocated the `<!ATTLIST>` default set. xpath3 tests count items read from a counting
  `Sequence`. These counts belong to one call, so the test can be `t.Parallel()`.
- A hook reads state the code keeps anyway, so a normal build pays nothing for it. Never add a
  package-level counter: every parallel test would add to it.
- Show that the check can fail. Add a positive control (an input that must trip it), or require that the
  counted path ran (`require.Positive` on the count), so an error raised earlier cannot pass the test
  without reaching the bound.
- When no count exists, measure allocations with `testing.AllocsPerRun` or the `runtime.MemStats.TotalAlloc`
  delta. Both read process-wide counters, so the measurement is its own top-level test, and neither it nor an
  ancestor calls `t.Parallel()` (`AllocsPerRun` panics in a parallel test). It then runs alone, in the test
  binary's sequential phase. Its doc comment says why it is sequential.
- Compare allocations relatively (N items against 1, a large input against a small one), or bound them by the
  input size with a wide margin: the guarded regression costs a copy of a multi-MiB input, and the bound sits
  far above the fixed cost of the call. `require.Zero` on a path that must not allocate is fine. Never pin an
  absolute count measured on one toolchain.
- Process-wide settings (`syscall.Umask`, package-level variables such as `xpath3.DefaultRegexMatchTimeout`)
  change only in a sequential test, which restores them.

### 32-bit Platforms

386 is the supported 32-bit platform: `ci.yml`'s `test-32bit` job runs the whole suite with `GOARCH=386`,
where `int` is 32 bits. Size, count, and limit arithmetic does not narrow an int64 to int or sum ints where the
result can pass 2^31-1; it parses in int64 and saturates at `math.MaxInt`, compares in int64, or range-checks
into int and reports an error.

- A test guarding such arithmetic uses a value just past 2^31 or 2^32 (`1<<32 + 5` truncates to 5), so the
  64-bit jobs fail too when a value is narrowed. Compare a length against `min(value, math.MaxInt)`.
- Where the code saturates at `math.MaxInt`, an `export_test.go` hook starts the counter next to it
  (`CountOpsForTesting` in xpath1 and xpath3), since no test can run that many operations.
- A limit that genuinely depends on the int size is asserted per platform by branching on `strconv.IntSize`:
  the xpath3 date/time result-year limit, an xs:yearMonthDuration month total past 2^31-1 (FODT0002 where int
  is 32 bits; `requireDurationResult` in `xpath3/arithmetic_datetime_test.go`), and an xslt3 package-version
  component past 2^31-1.

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

## Filesystem Fixtures

- A test that asserts a fixture's permission bits sets them with `os.Chmod` after creating the file
  (`internal/cli/heliumcmd/safety_unix_test.go` `writeFileMode`). `os.WriteFile` and `os.Create` apply the
  process umask, which differs between machines.
- A test that changes the umask (`syscall.Umask`) does not call `t.Parallel()` and restores the old value.

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

`relaxng/interleave_differential_test.go` compares two revisions' interleave verdicts, optionally against the
local `xmllint` as an oracle. `TestInterleaveDifferential` is skipped unless `-relaxng.idiff.out` is given.

| Flag | Meaning |
|------|---------|
| `-relaxng.idiff.out=PATH` | Write the record to PATH (required to run the test) |
| `-relaxng.idiff.cases=N` | Randomized interleave grammars (default 20000) |
| `-relaxng.idiff.seed=N` | Seed (default 1) |
| `-relaxng.idiff.xmllint=DIR` | Also run the local `xmllint` on each case and record its verdict in `DIR/xmllint.txt` |

The recorded procedure and result live in the header comment of `relaxng/interleave_differential_test.go`.

`xpath1/step_order_test.go` `TestStepResultOrder` pins the node-set and order of a matrix of documents (nested
same-name elements, namespaces, comments/PIs, entity references with and without substitution), context nodes
(document, element, attribute, namespace node) and location paths, including unions with a second document bound
to `$nodes`/`$other`, against `xpath1/testdata/step_order.golden`. `XPATH1_UPDATE_STEP_ORDER=1` rewrites the golden
file. It also checks each result against a fresh `DocOrderCache` via `DeduplicateNodes`. Only regenerate the file for
an intended order change.

`xpath3/step_order_test.go` `TestStepResultOrder` is the xpath3 counterpart, against
`xpath3/testdata/step_order.golden` (`XPATH3_UPDATE_STEP_ORDER=1` rewrites it). Besides the xpath1 matrix it covers
positional predicates on forward and reverse axes, path steps whose step expression is not an axis step (`/(a|b)`,
`//b/root()`, `//a/$other`), set operators, and golden-only shapes whose result is not a document-ordered node
sequence (`!`, atomic last steps, `reverse(...)/step`, FLWOR, mixed node/atomic steps raising XPTY0018). Only the
path expressions go through the fresh-`DocOrderCache` check.

`xpath3/eval_path_descendant_test.go` checks the `//` fusion over the same documents and contexts: every `//`
expression must render exactly like its reference, which spells `//` as
`/descendant-or-self::node()/self::node()/` so the path runs one step at a time. The extra self step charges one
op per descendant-or-self node, so `TestDescendantStepFusionLimits` requires the reference's smallest passing
`OpLimit` to be exactly that much higher, and both forms to fail on the same `MaxNodesForTesting` limits; its cases
cover one-pass predicates and nested context nodes (`/descendant::*//b`), and count the nodes of a path in a
predicate (`[count(.//c) > 0]`) because a path that only has to select a node stops at its first one. `TestDescendantStepFusionAnnotated`
annotates every attribute so `[@a = 's']` evaluates the comparison, and the sequential
`TestDescendantNestedContextsAllocate` requires `//a//b` over nested elements to allocate at least 500 times less
than its reference.

`xpath3/vm_path_nodes_test.go` checks the VM node-list consumers the same way. Each template marks an operand
`«X»`: the fast form reads `(X)`, the reference `(if (true()) then X else ())`, which hands X on as a sequence of
node items and charges nothing, so both forms must render alike, need the same smallest `OpLimit` and fail on the
same `MaxNodesForTesting` limits. In the `nodeListExistsExprs` templates the operand is a location path or `E1/path`
that only has to select a node, so the fast form stops at its first node: it must render alike, need no larger
smallest `OpLimit`, and fail on a node-set limit only where the reference fails too. `TestNodeListTypeAnnotations` validates a document against a schema with list and
union types and requires every node item to equal the one `.` gives for its node. `TestNodeListResult` checks the
`Result` accessors of a node-list result against the reference, and the sequential
`TestNodeListConsumersAllocate` requires each fast form to allocate at least 1,000 times less than its reference
over 2,000 nodes.

`xpath3/eval_path_exists_test.go` checks the early stop. `TestPathExists` puts every path of `existsPaths` in places
that only take whether it selects a node (`exists`, `not`, `if`, predicates, and from the document node `empty`,
`boolean`, `and`/`or`, quantifiers, `fn:`/`Q{}` spellings), plus `(exists(X), $other//c | //c)`, whose union order
shows where X registered its document, and requires each to render like its reference `(if (true()) then X else
())` over every document and context node; `TestPathExistsRandom` and `TestPathExistsRandomPaths` repeat it over
generated documents and generated paths. `TestPathExistsLimits` compares `exists(X)` with `count(X)`: no more
operations, and node-set-limit failures only where `count(X)` fails, and exactly the same operations and failures
when X selects nothing. `TestPathExistsStopsEarly` shows that paths over 2,000 elements stay within an `OpLimit`
and a `MaxNodesForTesting` of 100 that their full evaluation exceeds, `TestPathExistsSkippedErrors` that a
predicate error on nodes after the first match no longer fires, and `TestPathExistsCancel` that a cancelled context
stops a probe before it starts and mid-walk (`heliumtest.PollContext`).

`xpath3/evaluator_test.go` checks `Evaluator.EvaluateEBV`. `TestEvaluateEBV` requires it to return the value and
error code of `Evaluate` followed by `Result.EBV()` for `ebvExprs` and `resultEBVExprs` (node-list producers, other
node sequences, every atomic kind, FORG0006 cases, dynamic errors) from every node of a document and from an absent
context node; `TestEvaluateEBVTypeAnnotations` does the same under the type annotations of a schema-validated
document. `TestEvaluateEBVStopsEarly` shows that a root node path over 2,000 elements, with and without type
annotations, stays within an `OpLimit` and a `MaxNodesForTesting` of 100 that `Evaluate` exceeds, and that a
predicate error on nodes after the first match no longer fires.

## Build Tags

- `-tags debug` — used in CI (`go test -v -race -tags debug ./...`)
- No `//go:build` tags in test files

## Fuzzing

- Public-package fuzz coverage lives in package-local `fuzz_test.go` files.
- Direct fuzz targets exist for `.`, `c14n`, `catalog`, `html`, `internal/strcursor` (`FuzzScanCharDataSlice`,
  `FuzzScanSimpleAttrValue`), `relaxng`, `schematron`, `sink`, `stream`, `xinclude`, `xpath1`, `xpath3`, `xpointer`,
  `xsd`, `xmldsig1`, `xmlenc1`, `xslt3`; all are in the `fuzz.yml` matrix. Artifact names replace `/` in the package with `-`
  (`fuzz-corpus-internal-strcursor`).
- `shim` intentionally excluded from repo fuzz matrix.
- `enum` + `sax` intentionally excluded from direct fuzzing → constants/interface-only surface.
- Bound fuzz input sizes early. Return on oversize inputs.
- Prefer in-memory stubs over filesystem/network access.
- Parse/compile/validate/transform fuzz targets MUST tolerate invalid intermediate inputs by returning early instead of
  asserting.
- `xmlenc1`'s `FuzzDecrypt` puts FIXED RSA, EC, and AES key material on one `Decryptor` (all three are
  settable together) and leaves `SessionKey` unset, so the input alone selects which key-protection path runs
  — RSA-OAEP, ECDH-ES, or AES key wrap — and every branch is reachable from the one target. The keys are
  fixed, because a crasher written under `testdata/fuzz` must reproduce across runs.
- The `xslt3` targets count each input's heap allocations over parse+compile (and compile+transform) inline
  and fail via `t.Errorf` when the count exceeds `maxInputAllocs()` (`fuzz_test.go` `flagIfHeavy`, read from
  the `/gc/heap/allocs:objects` runtime metric), so the fuzzing engine persists the exact bytes as a crasher.
  Go's own worker already turns a genuine hang into a crasher via a 10s deadlock detector (`internal/fuzz`
  `worker.go`), so this targets the heavy-but-finite input that 10s net misses — the input that drags run
  throughput toward the fuzztime deadline and surfaces only as an unactionable `context deadline exceeded`
  with no reproducer. The count is the same for an input on every run and machine, so a flagged input
  replays as flagged. The bound defaults to 100 million objects (a million-body `xsl:for-each` transform
  allocates about 56 million) and is overridable via `HELIUM_FUZZ_MAX_ALLOCS` (a decimal count). Checking
  inline (not in a child goroutine) keeps panics on `testing`'s normal minimizable-crasher path.

## Fuzz CI

- Each matrix job restores newest Go fuzz corpus for same `go.mod`/package/ref, then falls back to same `go.mod`/package
  across refs.
- Each matrix job saves its corpus under an immutable run/attempt key after success or failure.
- Pull requests run NO fuzzing — `ci.yml` is normal test/build/lint/vuln verification only, so PR turnaround stays fast
  and deterministic (live fuzzing is nondeterministic and cannot gate a PR without flaking).
- `fuzz.yml` runs fuzzing OFF the PR path, always non-gating:
  - on every `push` to `main` (in practice, each PR merge) → short `60s` per target, for a prompt signal attributed to
    the pushed commit.
  - on the weekly `schedule` → deep `5m` per target.
  - on manual `workflow_dispatch` → its `fuzz-time` input (default `5m`).
- Fuzz targets are discovered per package via `go test ./<pkg>/ -list '^Fuzz' -run '^$'`; each target writes a separate
  log and retains its raw `go test` status.
- Every nonzero raw status or log-capture failure uploads target log + metadata (`package`, target, Go version, fuzz
  budget, raw/log statuses, classification) as a diagnostic artifact; log-capture failures fail the job.
- Only Go coordinator failures matching the complete deadline-only signature are warnings
  ([golang/go#75804](https://github.com/golang/go/issues/75804)); any extra diagnostic, panic, worker hang, source
  failure, or crashing input fails the job.
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

XSD benchmarks use valid `extension0_0` and `nvdcve_0` schema/instance pairs from the same fixture tree, plus
`assert_cta_1000`, an inline schema whose `<item>` type is chosen by `xs:alternative` and checked by `xs:assert`,
validated against an in-memory `<order>` of 1000 items, and `assert_paths_1000`, an inline schema whose
`xs:assert` tests only check whether node paths select a node (a bare path, `exists`/`empty`/`not`, `count(...) > 0`),
validated against the same `<order>`. Every case runs once per version as `<case>/1.0` and `<case>/1.1`;
`assert_cta_1000` and `assert_paths_1000` run only at 1.1. Compilation times `Compiler.Compile` with a parsed schema
document; validation times `Validator.Validate` with a compiled schema and parsed instance document.

Parse benchmarks (`bench/parse_bench_test.go`) share a corpus of `relaxng/test/spec_0.xml` (`118KB`, declared
iso-8859-1), `schemas/test/nvdcve_0.xml` (`287KB`) and `relaxng/test/comps_0.xml` (`608KB`), loaded by `loadCorpus`.
`BenchmarkHeliumParse` times `Parse` on the byte slice. `BenchmarkHeliumParseReader` times `ParseReader` in two
sub-cases per size: `BytesReader` reads from a `bytes.Reader`, and `64BReads` wraps it in `cappedReader`, which returns
at most 64 bytes per `Read`, so most tags and text runs straddle a read boundary. A cap of a few KB times the same as
`BytesReader`, because the parser's input buffer is 8KB. `BenchmarkHeliumParseSmall` times documents of 1KB or less
through `Parse`, `ParseReader` and a reused `Parser`. `BenchmarkStdlibXMLDecode` tokenizes the same corpus with
`encoding/xml` for comparison.

`BenchmarkWrite` (`writer_test.go`) serializes the parsed `nvdcve_0.xml`, `relaxng/test/comps_0.xml`, and
`relaxng/test/ISO19005-1-XMP_Packet.rng` (`xmprng`: every element `rng:`-prefixed, about thirty namespace
declarations on the root) with `helium.Write` into `io.Discard`. `BenchmarkIdentityTransform` (`xslt3/identity_bench_test.go`) runs an identity
transform over `comps_0.xml` in two stylesheet forms (`template`: a `match="@*|node()"` copy rule; `mode`:
`xsl:mode on-no-match="shallow-copy"`), each timed as `writer` (`TransformToWriter`, transform plus serialization)
and `tree` (`Transform` only). `BenchmarkConditionalNodeTests` (same file, same document) times `Transform` for
stylesheets whose `xsl:if`/`xsl:when` tests only check whether a node path selects a node: `path` (bare paths per
`package`), `exists` (the same through `exists`/`empty`/`count(...) > 0`) and `document` (whole-document paths per
`group`). `BenchmarkValidateNodePathTests` (`schematron/validate_bench_test.go`) validates the 500-record catalog
against an `xslt3`-binding schema of such `assert`/`report` tests.

RELAX NG benchmarks (`relaxng/relaxng_benchmark_test.go`) use `tutor10_8` (`small`) and `libvirt` from the same
tree. `BenchmarkValidate/large` validates `libvirt_0.xml` with its single `<disk>` repeated 300 times, built in
memory; `BenchmarkCompile` skips it because the schema is the same as `libvirt`.

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

Helium keeps only the **unit regression** `xsd/union_cycle_overflow_test.go` (cyclic simpleType must error, not
stack-overflow), guarding the in-tree fix (`baseChain` in `simplevalue_core.go`, `checkCircularSimpleTypes` in
`check_facets.go`).

## Skip Maps

Tests are skipped via in-code maps with reasons:
- Parser limitations (duplicate xmlns, single-quoted entity refs, external entity resolution)
- Feature gaps (libxml2 quirks like IDC edge cases)
- Missing expected files in libxml2 test data
