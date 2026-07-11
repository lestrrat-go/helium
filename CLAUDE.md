<!-- Agent-consumed file. Keep terse, unambiguous, machine-parseable. -->

# Helium

XML toolkit for Go covering XML parsing, SAX2-style streaming, XPath 3.1,
XInclude, XSD, Relax NG, and Schematron. Started as a libxml2-style port to
Go and grew broader native Go APIs and features along the way.

## XPath 3.1 — XSD Version

The xpath3 package targets **XSD 1.1 only**. This means `+INF` is a valid lexical form for xs:double and xs:float, and xs:dateTimeStamp is a recognized type. QT3 tests with `dependency type="xsd-version" value="1.0"` are skipped.

## XSD — Version Toggle

The xsd package defaults to **XSD 1.0** and treats 1.1 as **opt-in** via `Compiler.Version(xsd.Version11)` (or a `vc:minVersion="1.1"` hint on the root `<xs:schema>` when no explicit version is set). The resolved version is frozen onto the compiled `Schema` so the `Validator` applies the same semantics. 1.0 stays the default so existing behavior and goldens are unchanged.

`resolveVersion` (`compile.go`) resolves in order: a forced `Compiler.Version()` (always wins) → a `vc:minVersion="1.1"`-or-higher hint on the root → a configured `Compiler.DefaultVersion(v)` (the opt-in fallback for a schema silent on version) → `Version10`. `DefaultVersion` never overrides a forced version or a vc hint — it only chooses the fallback. The STANDALONE default stays `Version10`; `DefaultVersion` lets an embedding layer opt its schemas into 1.1 by default while still honoring an explicit version.

`Validator.SkipDatatypeIntegrityChecks(true)` suppresses the document-wide datatype-integrity walks in `validateDocument` (`cfg.skipDatatypeIntegrity`): the xs:ID/xs:IDREF/xs:IDREFS uniqueness+referential-integrity walk (version-INDEPENDENT — runs in both 1.0 and 1.1) and the XSD 1.1-only xs:ENTITY/xs:ENTITIES walk; content-model, type, and xs:key/unique/keyref identity-constraint validation are unaffected. It is for callers that validate an element/subtree as a fragment and enforce document-scope ID/IDREF integrity themselves (xslt3). In 1.0 it suppresses the ID/IDREF walk (the ENTITY walk never runs there).

XSD 1.1 is fully implemented behind the `Version11` opt-in (967/0 on the W3C suite). The complete feature-by-feature implementation state — every 1.1 construct, its file/function, spec clause, version gating, W3C test evidence, and remaining gaps — lives in `.claude/docs/xsd11.md`. **Read that doc before any work in `xsd/`.** Feature areas covered there:

- Type system: xs:assert (complex + simpleContent), xs:assertion facet, conditional type assignment (xs:alternative), simpleContent content-type narrowing, attribute inheritance, 1.1 built-in datatypes, simple-type 1.1 edges.
- Content models: UPA weakening, open content (xs:openContent / xs:defaultOpenContent), xs:all relaxations, wildcard notNamespace/notQName, particle-restriction relaxations, content-model backtracking, Wildcard EDC.
- Identity constraints: field-node classification/canonicalization, @ref, @xpathDefaultNamespace, structural rules, skip-wildcard scoping.
- Document-wide walks: xs:ID/IDREF/IDREFS and xs:ENTITY/ENTITIES integrity.
- Schema composition & representation: xs:override, xsi: attribute references, conditional inclusion (vc:), NCName/QName whitespace collapse, xs:notation, and the many version-INDEPENDENT XML-representation/structural checks.

Do NOT enforce 1.1-only clauses in the 1.0/default path — 1.0 must stay byte-identical to origin.

## XSLT 3.0 — Conformance Scope

The xslt3 package targets **Basic XSLT 3.0** conformance (W3C spec Section 27). The spec defines 8 conformance levels; only "Basic XSLT Processor" is required. The remaining 7 are optional features:

| Feature | Status | Notes |
|---------|--------|-------|
| Basic XSLT Processor | **Target** | Core requirement |
| Schema-Awareness | In progress | `xsl:import-schema`, type annotations |
| Serialization | Implemented | xml/html/text output methods |
| Streaming | Implemented | DOM-materialization; XTSE3430 analysis |
| Higher-Order Functions | Implemented | Via xpath3 |
| XPath 3.1 | Implemented | Via xpath3 |
| Dynamic Evaluation | Implemented | `xsl:evaluate` |
| Backwards-Compatible Processing | Implemented | XSLT 1.0 behavior + XPath 1.0 compatibility mode |

Schemas imported via `xsl:import-schema` (and a source-document schema) default to **XSD 1.1** (`compile_schema.go`/`source_schema.go` build the `xsd.Compiler` with `.DefaultVersion(xsd.Version11)`), so a schema silent on version compiles with 1.1 semantics (CTA/xs:alternative, unions, etc.) while an explicit `Compiler.Version()` or a schema `vc:minVersion` hint still wins. xslt3 validates a constructed element/subtree through `schemaRegistry.ValidateDoc` with `xsd.Validator.SkipDatatypeIntegrityChecks(true)`: content/type/CTA validation runs at 1.1 but the XSD 1.1 document-wide xs:ID/IDREF/ENTITY integrity walks are suppressed, because element-level validation (`xsl:validation="strict"` on an LRE) must not enforce whole-document ID uniqueness — xslt3 applies document-scope ID/IDREF integrity itself via `validateDocIDConstraints` at the true document/result-document scope (XTTE1555), matching the W3C validation-16xx semantics.

## XSLT — Backwards-Compatible Processing

Backwards-compatible processing (XSLT 3.0 §3.10) is enabled per element when its **effective version < 2.0** — the nearest-in-scope `[xsl:]version` (on the element, an ancestor, an included/imported module root, a global variable/param, or a literal result element's `xsl:version`; a `_version` shadow attribute takes precedence over the literal). Effective version in `[2.0, 3.0)` is identical to 3.0, so there is no separate "XSLT 2.0 compatibility" behavior. The one exception is the **`xsl:output` boolean serialization-parameter value space** (`compile_formats.go` `compileOutput`/`serializationYesNoOnly`): Serialization 3.0 widened the boolean parameters (indent, omit-xml-declaration, byte-order-mark, escape-uri-attributes, include-content-type, undeclare-prefixes, allow-duplicate-names, build-tree — and standalone's boolean synonyms) to the full xs:boolean lexical space `{yes, no, true, false, 1, 0}`, but an effective XSLT version below 3.0 (the module's in-scope `c.effectiveVersion`, NOT the xsl:output element's own `@version`, which is the serialization output-version parameter) restricts them to `{yes, no}` (standalone: `{yes, no, omit}`) — any other lexical form, including `true`/`false`/`1`/`0`, is an SEPM0016 static error (W3C output-0197/0198/0199/0280/0281/0282/0283, all `version="2.0"`). Uppercase forms (`TRUE`/`YES`) are invalid in every version (xs:boolean is lowercase-only). An absent/unparseable version defaults to 3.0 (permissive).

The core is **XPath 1.0 compatibility mode**, a runtime flag on the xpath3 evaluator (`Evaluator.XPath10Compat()`, default off, so xsd/relaxng/schematron and ordinary xpath3/xslt3 are unchanged). xslt3 records every expression compiled under an effective version < 2.0 by pointer identity (`Stylesheet.compatExprs`, set in `compiler.compileXPath`) and evaluates it in compat mode (`execContext.evalXPath`/`withCompat`). XPath 1.0 mode (`internal xpath10_compat.go`): a single-item function parameter given >1 items keeps only the first; an `xs:string(?)` parameter coerces via `fn:string` and an `xs:double(?)`/`xs:numeric` parameter via `fn:number` (invalid/empty → NaN); arithmetic operands convert to `xs:double` (÷0 → ±INF); general comparisons apply the 1.0 boolean/numeric/string rules, and the relational operators (`<`,`<=`,`>`,`>=`) always convert both operands to number. Functions/operators that bypass signature coercion (`format-number` value+picture, `subsequence` position/length, `string-join` separator, `fn:number`-family node args, the `to` range operator) consult the flag directly. Runtime-compiled expressions not in `compatExprs` are covered by a context flag instead: match-pattern predicates (`execContext.patternCompat`, set from `pattern.compat` at the match entry, honored in `evalXPath` and the predicate evaluators) and `xsl:evaluate`'s dynamic expression (compat when its static `xpath` attribute is compat-marked). Backwards-compatible processing is NOT applied to the compile-time static context — `use-when`, `static="yes"` variables/params, and shadow attributes are evaluated version-independently.

XSLT-level 1.0 behaviors (keyed off the compat-marked expression or an instruction `Compat` flag): `xsl:value-of` and AVTs discard all but the first item; `xsl:number/@value` uses the first atom and outputs `"NaN"` for an empty or non-integer value (no XTDE0980); `xsl:sort` uses the first sort-key item; `xsl:call-template` silently ignores a surplus `with-param` (no XTSE0680); `xsl:key`/`key()` compare values as `xs:string`; `system-property('xsl:supports-backwards-compatibility')` is `"yes"`. Known gaps (skipped in `expectations/xslt30.json` with specific reasons): the 1.0-only default output method (xhtml→xml for an implicit 1.0 result tree); `base-uri()` fixture dependence; and XPath **1.0 grammar** differences (`div`/`mod` as a name after an operator, unprefixed `function` as a name test, empty function arguments) — compat mode changes semantics, not the grammar, so these stay out of scope.

The `spec="XSLT20"`/`spec="XSLT10"` version-specific test bucket (~1120 cases) is **in scope** and un-gated: `specSupported` in the helium-w3c-tests generator treats `XSLT10`/`XSLT20` like their `+` forms, so a conformant 3.0 processor runs them. ~1015 pass as-is, plus 7 more from version-gating the `xsl:output` boolean serialization parameters (below). The ~80 remaining are documented per-case in the `w3cImplicitSkips` map (helium-w3c-tests `xslt3/w3c_helpers_test.go`), overwhelmingly as **legitimate 2.0-vs-3.0 divergences** where our 3.0 output is correct and the case asserts a 2.0-only error a 3.0 processor no longer raises: 3.0-only regex constructs the 2.0 test expects to reject with FORX0002; match/error pattern-syntax relaxations that dropped XTSE0340; `xsl:sequence` with a contained sequence constructor, no longer XTSE0010; functions/arities added in 3.0/XPath 3.1, no longer XPST0017; `apply-templates`/`for-each` `select` required-type errors XTTE0520/XTTE1120, both removed in 3.0 (a non-node population is handled by the atomic built-in template rule / never matches a pattern); initial-entry conflict errors XTDE0047/XTDE0060, removed in 3.0 (W3C bug 28418); `current-group()`/`current-grouping-key()` outside a grouping context, which 3.0 makes a dynamic error XTDE1061/XTDE1071 rather than the 2.0 empty sequence; and a conflicting `xsl:strip-space`/`xsl:preserve-space` at equal precedence/priority, a RECOVERABLE error in 1.0/2.0 but a STATIC error `XTSE0270` in 3.0 (helium correctly raises `XTSE0270` via `compile_formats.go` `checkSpaceConflicts`; the 3.0 counterpart `strip-space-019a` passes); and `format-date`/`format-time` fractional-second handling, where XPath 3.1 truncates but the 2.0 case asserts rounding (the 3.0+ variant passes). **The genuine-gaps list is now essentially empty**: every residual skip in the XSLT20/XSLT10 bucket is a legitimate 2.0-vs-3.0 divergence (a correct-skip where our 3.0 output is right), not a missing mandatory Basic 3.0 facility. `xsl:output` boolean serialization parameters are version-gated (`compile_formats.go` `serializationYesNoOnly`): under an effective XSLT version < 3.0 they accept only `yes`/`no` (raising SEPM0016 otherwise, per Serialization 1.0), while 3.0 keeps the full `xs:boolean` value space. Do NOT implement XSLT 1.0/2.0 **syntax** support — compat mode changes semantics, not the grammar.

## Generated Files

- NEVER modify generated files by hand. Regenerate through the owning generator (e.g. `go generate`).

## Pre-Read Rules

Read the linked doc BEFORE working in that area. No exceptions.

| Trigger | Doc |
|---------|-----|
| Package purpose, API, files | `.claude/docs/packages.md` |
| Cross-package imports | `.claude/docs/dependencies.md` |
| Working with `context.Context`, package `Context` payloads, carrier/accessor patterns | `.claude/docs/context.md` |
| Feature status, test counts, known gaps, ParseOption support | `.claude/docs/libxml2-parity.md` |
| Writing/running tests, golden files, test data, helpers | `.claude/docs/testing.md` |
| Writing/editing `examples/`, example scope, example comments | `.claude/docs/testing.md` |
| Error types, format strings, ErrorHandler, ValidateError | `.claude/docs/error-formatting.md` |
| Parse pipeline, encoding, entities, SAX→DOM, push parser | `.claude/docs/parser-internals.md` |
| DOM node hierarchy, struct fields, namespace/attr storage | `.claude/docs/node-types.md` |
| XSD/RELAX NG/Schematron compile→validate flow | `.claude/docs/validation-pipeline.md` |
| XSD 1.1 feature state (`Version11`), any work in `xsd/` — index → area sub-docs | `.claude/docs/xsd11.md` |
| XSD 1.1 types: assert, alternative/CTA, simpleContent, attr inheritance | `.claude/docs/xsd11-types.md` |
| XSD 1.1 content models: UPA, open content, xs:all, wildcards, restriction, backtracking, EDC | `.claude/docs/xsd11-content-models.md` |
| XSD 1.1 identity constraints (xs:key/unique/keyref) | `.claude/docs/xsd11-identity-constraints.md` |
| XSD 1.1 document-wide ID/IDREF/ENTITY walks | `.claude/docs/xsd11-doc-walks.md` |
| XSD 1.1 representation/structural/composition checks, xs:override, vc:, notation | `.claude/docs/xsd11-representation.md` |
| helium CLI commands, flags, pipeline, exit codes | `.claude/docs/helium-command.md` |
| Cutting a release, editing release/conformance workflows, bumping the harness pin | `.claude/docs/releasing.md` |
| XPath 3.1 design overview, constraints, sub-doc index | `.claude/docs/xpath3-design.md` |
| XPath 3.1 architecture, file layout, internal/xpath | `.claude/docs/xpath3-architecture.md` |
| XPath 3.1 public API, Context, Result, errors | `.claude/docs/xpath3-api.md` |
| XPath 3.1 Item/Sequence/Map/Array type system | `.claude/docs/xpath3-types.md` |
| XPath 3.1 lexer, parser, AST nodes | `.claude/docs/xpath3-parser.md` |
| XPath 3.1 evaluator, comparison, casting | `.claude/docs/xpath3-eval.md` |
| XPath 3.1 function system, built-in categories | `.claude/docs/xpath3-functions.md` |
| Saxon-HE source layout (reference) | `.claude/docs/saxon-layout.md` |

## Cache Maintenance

These docs cache repository state. Still read source before modifying code.

1. When your changes affect a doc below, update it in the same commit.
2. If you notice any doc is wrong or stale — even on an unrelated task — fix it immediately.
3. **Write current state, not history.** These are caches of what the code *is*, not a changelog of how it got there. State what the code does now; never append "was X, now Y", "no longer / previously / used to", "the regression where…", "fixes #NNN", "round-N", or resolved "KNOWN RESIDUAL / deferred" notes — that framing rots into staleness the moment the state changes. When you fix a gap, delete its gap note (don't rewrite it to "now fixed"); when you change behavior, describe the new behavior in the present tense. Keep design rationale (spec citations, version-gating / byte-identical constraints, genuine *current* gaps) — that is current state; the journey belongs in git history.

| Doc | Update trigger |
|-----|----------------|
| `packages.md` | Public API, package, or key file changes |
| `dependencies.md` | Inter-package import changes |
| `context.md` | `context.Context` conventions, package `Context` payload pattern, `NewContext`/`GetContext` guidance changes |
| `libxml2-parity.md` | Test count, parser limitation, feature, or ParseOption changes |
| `testing.md` | Test data layout, helper, env var, or test pattern changes |
| `error-formatting.md` | Error format, error type, or ErrorHandler changes |
| `parser-internals.md` | Parse pipeline, encoding, entity, SAX, or parserCtx changes |
| `node-types.md` | Node type, struct field, or tree operation changes |
| `validation-pipeline.md` | Compile/validate phase, data model, or backtracking changes |
| `xsd11.md` (+ `xsd11-*.md` sub-docs) | XSD 1.1 feature, gating, gap, or version-resolution changes — update the sub-doc for the affected area; keep the index table in sync |
| `helium-command.md` | CLI command, flag, pipeline, or exit code changes |
| `xpath3-design.md` | Design constraints, sub-doc structure changes |
| `xpath3-architecture.md` | Package layout, file additions/removals, import graph changes |
| `xpath3-api.md` | Public API, Context, Result, error type changes |
| `xpath3-types.md` | Item/Sequence/AtomicValue/Map/Array type changes |
| `xpath3-parser.md` | Lexer, parser, AST node, token type changes |
| `xpath3-eval.md` | Evaluator, comparison, casting logic changes |
| `xpath3-functions.md` | Function registry, built-in function additions/changes |
| `saxon-layout.md` | Reference layout updates |
