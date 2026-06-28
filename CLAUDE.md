<!-- Agent-consumed file. Keep terse, unambiguous, machine-parseable. -->

# Helium

XML toolkit for Go covering XML parsing, SAX2-style streaming, XPath 3.1,
XInclude, XSD, Relax NG, and Schematron. Started as a libxml2-style port to
Go and grew broader native Go APIs and features along the way.

## XPath 3.1 — XSD Version

The xpath3 package targets **XSD 1.1 only**. This means `+INF` is a valid lexical form for xs:double and xs:float, and xs:dateTimeStamp is a recognized type. QT3 tests with `dependency type="xsd-version" value="1.0"` are skipped.

## XSD — Version Toggle

The xsd package defaults to **XSD 1.0** and treats 1.1 as **opt-in** via `Compiler.Version(xsd.Version11)` (or a `vc:minVersion="1.1"` hint on the root `<xs:schema>` when no explicit version is set). The resolved version is frozen onto the compiled `Schema` so the `Validator` applies the same semantics. 1.0 stays the default so existing behavior and goldens are unchanged.

Implemented in 1.1 mode so far: the 1.1-only lexical forms (`+INF` for xs:double/xs:float; year `0000` on the date types — both gated in `internal/xsd/value` via a `value.Version` argument; relaxng is pinned to `value.Version10`); the 1.1 built-in datatypes (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration, xs:anyAtomicType, xs:error); **UPA weakening** (`check_upa.go` `entriesOverlap`: in 1.1 an element competing with a wildcard is not a cos-nonambig violation — the element wins. Element-over-wildcard precedence is now ENFORCED at validation for the CHOICE case (`validate_elem.go` `matchChoice`/`tryMatchChoice`, gated on `vc.version == Version11`): a branch that consumes the current child via an element leaf AS ITS FIRST CONSUMING TERM takes precedence over one that would consume it via a wildcard, regardless of declaration order or nesting. The classifier (`particleConsumesViaElement` → `particleFirstConsumerKind`/`groupFirstConsumerKind`) is PATH-AWARE: it respects compositor order, occurrences, and emptiable prefixes (reusing `particleEmptiable`), so a LEADING wildcard inside a sequence (e.g. `sequence(any skip, element a)`) is correctly classified as wildcard-first and does NOT win element precedence even though a later element leaf in the same group also matches — only bounded first-consumer determination, no backtracking. A `skip` wildcard declared before a typed element (directly or as a leading nested term) thus no longer steals the element's child and false-accepts an invalid value. Selection is COMMIT-NO-FALLBACK: once any branch is element-first for the current child the choice MUST use an element-first branch and never falls back to a wildcard branch, even if the chosen element branch then fails structurally (a later required term missing, e.g. `choice(sequence(a:int, b:int), any skip)` with only `<a>`) or by content. The SEQUENCE case is a remaining limitation — sequence matching is position-based/greedy, so a minOccurs=0 wildcard preceding an element can still consume the element's child); **xs:assert on complex types** (`assert.go`: parsed from complexType/restriction/extension, pre-compiled via xpath3, evaluated against the element after content validation — EBV false → invalid; inherited down the base chain); and **conditional type assignment** (`alternative.go`: `<xs:alternative>` on an element declaration selects the governing type via the first @test that holds against the element, else a testless default, else the declared type; xsi:type still takes precedence; applied at the root and all three per-child match sites); and **open content** (`opencontent.go`: `<xs:openContent>` interleave/suffix — interleave removes children whose names aren't declared and which match the wildcard, then matches the declared model on the rest; suffix matches the declared model then requires every trailing child to match the wildcard; declared-named children always go through the model per weak-wildcard precedence); **document-wide xs:ID/xs:IDREF/xs:IDREFS validation** (`validate_id.go`: a third validation walk, gated to 1.1, enforcing ID uniqueness and IDREF referential integrity — an ID identifies the element bearing it, so an attribute ID is owned by its element and an element-content ID by its PARENT, and the same value may recur as long as it identifies one element, matching the 1.1 multiple-ID relaxation; list/union values are decomposed to their atomic ID/IDREF leaves via the active-member resolver; element-content default/fixed values are applied EXCEPT on a CONFIRMED nilled element — one DECLARED nillable (`idcHostDecl(elem).Nillable`) carrying `xsi:nil="true"` (a nilled element has no value, so substituting its default/fixed would fabricate a duplicate ID or dangling IDREF). The check is by DECLARATION, not raw `xsi:nil`: a lax no-declaration element with `xsi:nil` is NOT validly nilled, so its real (assessed) content is still collected. Attribute IDs always apply on a nilled element; element/attribute typing uses ONLY provenance recorded at genuine pass-1 ASSESSMENT sites — the `assessedElemType`/`actualElemDecl` and `actualAttrType` maps — never `actualElemType` (which is ALSO written, with `assessed=false`, for `processContents="skip"` subtrees and lax elements with no resolvable governing type, purely for pass-2 IDC canonicalization) and never a global fallback. A `processContents="lax"` element with no declaration but a RESOLVABLE `xsi:type` IS laxly assessed by the shared `assessLaxElement` helper (validated against that type, `assessed=true`) — used for BOTH a directly wildcard-matched lax element (`matchWildcardParticle`) and an xs:anyType/lax descendant (`annotateAnyTypeChildren`) — so its `xsi:type="xs:ID"`/`xs:IDREF` content participates in the ID/IDREF pass. An undeclared element has no nillable declaration, so `xsi:nil="true"` does NOT exempt it from validation: its content is still checked against the governing type (a nilled lax element with invalid/forbidden content is rejected). `skip` content is NEVER assessed, so even one carrying `xsi:type="xs:ID"` is not mis-typed; **identity-constraint @xpathDefaultNamespace** (resolved on xs:selector/xs:field or inherited from the root xs:schema — inheritance happens ONLY when the attribute is ABSENT, detected by PRESENCE (`hasAttr`), so an explicit `xpathDefaultNamespace=""` means "no default element namespace" and does NOT inherit the schema-level default; supporting `##targetNamespace`/`##defaultNamespace`/`##local`/URI, applied via the opt-in `xpath1.Evaluator.DefaultElementNamespace`, which affects only unprefixed ELEMENT name tests, never attributes; the schema-level default is PER-document — like elementFormDefault/blockDefault/finalDefault it is saved/set/restored across `xs:include`/`xs:redefine` and set on the import sub-compiler in `compile_imports.go`, so an included/imported schema's IDCs use ITS root's value, not the including/importing schema's); and **identity-constraint @ref** (`resolveConstraintRefs` in `compile.go`: a key/unique/keyref using `@ref` reuses a same-kind referenced constraint's selector/fields — and a keyref's refer — adopting its QName identity. The `@ref` form is detected by PRESENCE (`hasAttr`, so a literal `ref=""` is recognized, not silently dropped). A dangling ref, an empty `@ref`, an unbound `@ref` prefix, a kind mismatch, or a `@ref` that also carries name/selector/field/refer is a fatal schema error. The `name`/`refer` companions are detected by PRESENCE (`hasAttr`, so an empty-but-present `name=""`/`refer=""` is still rejected), and `refer` is rejected for EVERY kind, not just keyref). **Not yet implemented** (planned, gated behind the same toggle): the **xs:assertion simple-type facet** (needs `$value` bound to the typed atomic), **xs:defaultOpenContent** (schema-level default), `xs:all` member `maxOccurs>1` and wildcards-in-all, wildcard `notNamespace`/`notQName`, `xs:override`, the general user-declarable `explicitTimezone` facet, `vc:typeAvailable`/`vc:facetAvailable` conditional pruning, **xs:ENTITY/xs:ENTITIES instance validation** (needs the DTD unparsed-entity table), **element/attribute default-value validity against the declared type at compile time**, and **ID/IDREF members inside a union at instance level** (only atomic and list-of-ID/IDREF are decomposed). Do NOT assume these work in 1.1 mode yet. (Limitations: xs:assert's test runs against the element in the full document so it can navigate to ancestors; xs:alternative does not yet support an inline anonymous type — use a named @type.)

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
| **Compatibility (1.0/2.0)** | **Not planned** | Optional per spec; 122 tests skipped legitimately |

Do NOT implement XSLT 1.0/2.0 backwards compatibility mode. Tests skipped with reason `"unsupported feature: backwards_compatibility"` or `"unsupported spec: XSLT20"` are intentionally out of scope.

## Generated Files

- NEVER modify generated files by hand. Regenerate through owning generator, e.g. `xslt3gen`.

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
| helium CLI commands, flags, pipeline, exit codes | `.claude/docs/helium-command.md` |
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
| `helium-command.md` | CLI command, flag, pipeline, or exit code changes |
| `xpath3-design.md` | Design constraints, sub-doc structure changes |
| `xpath3-architecture.md` | Package layout, file additions/removals, import graph changes |
| `xpath3-api.md` | Public API, Context, Result, error type changes |
| `xpath3-types.md` | Item/Sequence/AtomicValue/Map/Array type changes |
| `xpath3-parser.md` | Lexer, parser, AST node, token type changes |
| `xpath3-eval.md` | Evaluator, comparison, casting logic changes |
| `xpath3-functions.md` | Function registry, built-in function additions/changes |
| `saxon-layout.md` | Reference layout updates |
