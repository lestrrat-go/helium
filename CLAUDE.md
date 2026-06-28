<!-- Agent-consumed file. Keep terse, unambiguous, machine-parseable. -->

# Helium

XML toolkit for Go covering XML parsing, SAX2-style streaming, XPath 3.1,
XInclude, XSD, Relax NG, and Schematron. Started as a libxml2-style port to
Go and grew broader native Go APIs and features along the way.

## XPath 3.1 — XSD Version

The xpath3 package targets **XSD 1.1 only**. This means `+INF` is a valid lexical form for xs:double and xs:float, and xs:dateTimeStamp is a recognized type. QT3 tests with `dependency type="xsd-version" value="1.0"` are skipped.

## XSD — Version Toggle

The xsd package defaults to **XSD 1.0** and treats 1.1 as **opt-in** via `Compiler.Version(xsd.Version11)` (or a `vc:minVersion="1.1"` hint on the root `<xs:schema>` when no explicit version is set). The resolved version is frozen onto the compiled `Schema` so the `Validator` applies the same semantics. 1.0 stays the default so existing behavior and goldens are unchanged.

Implemented in 1.1 mode so far: the 1.1-only lexical forms (`+INF` for xs:double/xs:float; year `0000` on the date types — both gated in `internal/xsd/value` via a `value.Version` argument; relaxng is pinned to `value.Version10`); the 1.1 built-in datatypes (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration, xs:anyAtomicType, xs:error); **UPA weakening** (`check_upa.go` `entriesOverlap`: in 1.1 an element competing with a wildcard is not a cos-nonambig violation — the element wins. Element-over-wildcard precedence is now ENFORCED at validation for the CHOICE case (`validate_elem.go` `matchChoice`/`tryMatchChoice`, gated on `vc.version == Version11`): a branch that consumes the current child via an element leaf AS ITS FIRST CONSUMING TERM takes precedence over one that would consume it via a wildcard, regardless of declaration order or nesting. The classifier (`particleConsumesViaElement` → `particleFirstConsumerKind`/`groupFirstConsumerKind`) is PATH-AWARE: it respects compositor order, occurrences, and emptiable prefixes (reusing `particleEmptiable`), so a LEADING wildcard inside a sequence (e.g. `sequence(any skip, element a)`) is correctly classified as wildcard-first and does NOT win element precedence even though a later element leaf in the same group also matches — only bounded first-consumer determination, no backtracking. A `skip` wildcard declared before a typed element (directly or as a leading nested term) thus no longer steals the element's child and false-accepts an invalid value. Selection is COMMIT-NO-FALLBACK: once any branch is element-first for the current child the choice MUST use an element-first branch and never falls back to a wildcard branch, even if the chosen element branch then fails structurally (a later required term missing, e.g. `choice(sequence(a:int, b:int), any skip)` with only `<a>`) or by content. The SEQUENCE case is a remaining limitation — sequence matching is position-based/greedy, so a minOccurs=0 wildcard preceding an element can still consume the element's child); **xs:assert on complex types** (`assert.go`: parsed from complexType/restriction/extension, pre-compiled via xpath3, evaluated against the element after content validation — EBV false → invalid; inherited down the base chain); and **conditional type assignment** (`alternative.go`: `<xs:alternative>` on an element declaration selects the governing type via the first @test that holds against the element, else a testless default, else the declared type; xsi:type still takes precedence; applied at the root and all three per-child match sites); and **open content** (`opencontent.go`: `<xs:openContent>` interleave/suffix — interleave removes children whose names aren't declared and which match the wildcard, then matches the declared model on the rest; suffix matches the declared model then requires every trailing child to match the wildcard; declared-named children always go through the model per weak-wildcard precedence); and **wildcard `notNamespace`/`notQName`** (`xs:any`/`xs:anyAttribute`): `readWildcard` (`read_elements.go`) parses `@notNamespace` (anyURI/`##targetNamespace`/`##local`, mutually exclusive with `@namespace`) into `Wildcard.NotNamespace` and `@notQName` (QName/`##defined`/`##definedSibling`) into `Wildcard.NotQName`/`NotQNameDefined`/`NotQNameDefinedSibling` (`schema.go`). Matching honors them everywhere: `wildcardMatches` excludes `NotNamespace`; `wildcardAllowsExpandedName`/`wildcardExcludesName` (`validate_elem.go`) add the name-level `notQName`/`##defined` exclusion (consulting `schema.elements`/`schema.globalAttrs`), used by element content (`matchWildcardParticle`, `matchAll`), attribute (`validateWildcardAttr` call site, `validate.go`), and open content. `##definedSibling` is resolved post-`resolveRefs` by `resolveDefinedSiblings` (`opencontent.go`) into `SiblingNames`. The UPA overlap test (`check_upa.go` `firstSetEntry.wc`, `wildcardsOverlap`→`constraintsIntersect`) and the restriction-subset check (`wildcardConstraintSubset11`, `wildcard_algebra.go`) are 1.1-aware — the latter rejects a derived wildcard that DROPS a `##defined` the base carries, or (for `##definedSibling`) whose resolved `SiblingNames` set is NARROWER than the base's (it iterates `super.SiblingNames` and requires `sub` to exclude each, not just compare the marker bit). `resolveDefinedSiblings` runs INSIDE `resolveRefs` (after group-ref expansion, before the restriction-derivation checks) so those checks see resolved `SiblingNames`, and it visits ALL parsed complex types — named `schema.types` AND inline ANONYMOUS types (iterating `c.typeDefSources` keys, deduped by `*TypeDef` pointer) — so a local element's inline `##definedSibling` wildcard is resolved too. The restriction-derivation functions thread `*Schema` (and an `isAttr` flag into `wildcardConstraintSubset`/`wildcardConstraintSubset11`) so EVERY derivation site uses the full `wildcardAllowsExpandedName` (notQName/`##defined`-aware), not a namespace-only test: `wildcardAllowsName` (element-restricts-wildcard); `checkRestrictionAttrs` (`link_refs.go`, a derived CONCRETE attribute vs the base `anyAttribute`, isAttr=true — so a base wildcard excluding it by `notQName`/`##defined` rejects the restriction); and the per-name SUBSET checks in `wildcardConstraintSubset11` (cos-ns-subset — each name `super` disallows must be DISALLOWED by `sub` under the full test, so a derived wildcard's `##defined` can discharge a base wildcard's explicit `notQName="g"` when `g` is globally declared, while the `super.NotQNameDefined && !sub.NotQNameDefined` whole-marker check still requires `sub` to carry `##defined` when `super` excludes every defined name). `##definedSibling` is permitted ONLY on element wildcards (`parseNotQName` rejects it on `xs:anyAttribute`, passed an `isAttr` flag from `readWildcard`). A normalized wildcard algebra (`wildcard_algebra.go`: `wcConstraint` + union/intersection) drives attribute-wildcard UNION (extension, `wildcardUnion`) and INTERSECTION (the type's "complete wildcard" across `xs:attributeGroup` refs — group wildcards captured in `attrGroupWildcards`, intersected in `link_refs.go`). `attrGroupWildcards` is initialized AND merged into the parent in import sub-compilers (`compile_imports.go`; an imported group's `xs:anyAttribute` otherwise panics on a nil map), and the `xs:redefine` override path parses the group's `xs:anyAttribute`, clears the stale base wildcard, intersects a self-reference's original wildcard, and stores the replacement. A group's `xs:anyAttribute` must be the optional FINAL, unique child (enforced in BOTH `parseNamedAttributeGroup` and the redefine override, Version11). `notQName` QName tokens are validated with the shared `internal/xmlchar.IsValidQName` (accepts non-ASCII XML NameChars like the middle dot). `constraintToWildcard` and its union/intersection callers carry `SiblingNames` through materialization (intersection unions the sibling exclusions; union retains the sibling names neither operand admits) so a materialized base-`all` wildcard union does not drop `##definedSibling` names and silently accept a re-admitting restriction. The intersection carries the STRONGER processContents of its operands (`strongerProcessContents`, strict>lax>skip) so it is ORDER-INDEPENDENT — a strict group is not weakened to skip by a sibling skip group. The UNION `{disallowed names}` follows cos-aw-union (§3.10.6.3): a candidate QName is excluded iff NEITHER operand admits it by NAMESPACE-and-explicit-`notQName` (`wildcardAdmitsNameIgnoringDefined`); `##defined`/`##definedSibling` are folded ONLY as whole markers (kept iff BOTH operands carry them) and do NOT make an individual QName disallowed — so a global-attr name one operand excludes via `##defined` but admits by namespace, while the other excludes it explicitly, is still ADMITTED by the union (W3C wild083 `surprise`). `xs:all` may contain wildcards — matched by `matchAll` (gated on `Version11`, so the 1.0 path is byte-identical to before the feature); restriction handled by `allRestrictsWithWildcards` (`restriction_particle.go`), which enforces per-base-wildcard MIN and MAX cardinality (the MAX bound applied only to base wildcards that EXCLUSIVELY own their namespace, so disjoint base wildcards are not collapsed into one aggregate capacity). CONCRETE derived elements admitted by a base wildcard (not mapped to a base element) participate in that cardinality accounting (combined totals and per-base min/max) keyed on their exact name, so extra concrete elements cannot overload a base wildcard's `maxOccurs` and a required concrete element satisfies a base wildcard's `minOccurs`. A derived wildcard's processContents must be at least as strong as EVERY base wildcard it INTERSECTS (not merely the weakest in the union), so a `skip` derived wildcard cannot restrict a `strict` same-namespace base wildcard just because a disjoint base wildcard is `skip`. A schema declaring an attribute in the XSI namespace is rejected (no-xsi). All gated on `Version11`; 1.0 drops group wildcards and ignores notNamespace/notQName, unchanged. **Not yet implemented** (planned, gated behind the same toggle): the **xs:assertion simple-type facet** (needs `$value` bound to the typed atomic), **xs:defaultOpenContent** (schema-level default), `xs:all` element member `maxOccurs>1`, the **dynamic and static Element Declarations Consistent (EDC)** checks for wildcards (a strict/lax wildcard selecting a governing type that differs from a same-named sibling/global element declaration), identity-constraint scoping that excludes `skip`-wildcard-matched (un-assessed) subtrees, `xs:override`, the general user-declarable `explicitTimezone` facet, and `vc:typeAvailable`/`vc:facetAvailable` conditional pruning. Do NOT assume these work in 1.1 mode yet. (Limitations: xs:assert's test runs against the element in the full document so it can navigate to ancestors; xs:alternative does not yet support an inline anonymous type — use a named @type.)

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
