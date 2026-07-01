# Validation Pipeline

Three validation engines: XSD (grammar-based), RELAX NG (pattern-based), Schematron (rule-based). All follow compile→validate pattern.

## XSD

Files: `xsd/xsd.go` (API), `compile*.go` + `read_*.go` + `link_refs.go` + `restriction_particle.go` + `check_*.go` (compile/read/resolve/constraint pipeline), `validate.go` + `validate_elem.go` (content validation), `validate_idc.go` (IDC), `simplevalue_core.go` + `simplevalue_facets.go` (simple-value validation), `schema.go` (model)

**XSD version (1.0 default, 1.1 opt-in):** `Compiler.Version(xsd.Version11)` selects 1.1; an explicit call wins, else `resolveVersion` (`compile.go`) reads a `vc:minVersion` hint (ns `lexicon.NamespaceXSDVersioning`) off the root `<xs:schema>` and selects 1.1 iff it is `>= 1.1`; default is `Version10`. The hint is parsed with the SAME rules as the conditional-inclusion pre-pass — ASCII XML whitespace trim only (trims the XSD set space/tab/CR/LF, not `strings.TrimSpace`, so an NBSP-padded value is not silently accepted) and EXACT xs:decimal comparison (`isValidXSDDecimal` + `value.CompareDecimal` against `"1.1"`, not float64) — so a malformed/NBSP value does not auto-select 1.1 (treated as no hint → 1.0) and a high-precision value just below 1.1 is not float-rounded up into 1.1. The resolved `Version` is stored on `compiler.version`, propagated to the import sub-compiler (`compile_imports.go`), and frozen onto `Schema.version`; `validationContext.version` is seeded from it (`newValidationContext`) and from `c.version` for the compile-time facet sub-contexts (`check_facets.go`). The lexical gate lives in `internal/xsd/value.ValidateBuiltin(value, builtinLocal, version)` — `validateFloat` selects `floatRegex10` (no `+INF`) for 1.0; the date validators reject year `0000` for 1.0 (`yearForbiddenInXSD10`). The value package's comparison/canonicalization guards pass `value.Version11` (they run on already-validated values). relaxng pins its `ValidateBuiltin` calls to `value.Version10`. xsd threads the version into `validateBuiltinValue` via `vc.version`; `fixedUnionActiveMember` takes a `version Version` param threaded from every caller (`fixedUnionMatches`/`crossMemberValueEqual(Depth)`/`fixedValueMatches`/`fixedListMatches` carry it, runtime callers pass `vc.version`, and the compile-time restriction-derivation chain in `restriction_particle.go` passes `c.version`), so the throwaway context it builds applies the schema's real version — a 1.1-only lexical form (e.g. `+INF`) inside a union fixed-value or enumeration literal is accepted in 1.1 mode. Two throwaway-context free funcs remain on the `Version10` default — `typeAcceptsValue` (`validate_idc.go`, IDC field active-member) and `TypeDef.Validate` (standalone validation) — a documented Phase-1 gap for 1.1-only lexical forms on those paths. The 1.1 built-in datatypes (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration, xs:anyAtomicType, xs:error) are registered only in 1.1 mode by `registerBuiltinTypes11` (`compile.go`), with `BaseType` linked to their primitives; `Compare`/`CanonicalKey` route dateTimeStamp→dateTime and the durations→duration. **UPA weakening** is implemented: `entriesOverlap` (`check_upa.go`) takes the automaton's `schema.version` and, in 1.1, treats an element-vs-wildcard pair as non-overlapping (the element wins), so such a content model compiles; element-vs-element and wildcard-vs-wildcard are unchanged. Element-over-wildcard precedence is ENFORCED at validation for the CHOICE case (`validate_elem.go` `matchChoice`'s `matchAt`/`matchOnce` and `tryMatchChoice`'s `scanOnce`, both gated on `vc.version == Version11`): a branch that consumes the current child via an element leaf AS ITS FIRST CONSUMING TERM is tried before any branch that would consume it via a wildcard, regardless of declaration order or nesting, so a `skip`/`lax` wildcard declared BEFORE a typed element does not steal the element's child (which would false-accept an invalid value like `<a>not-int</a>` against `xs:int`). The selection is COMMIT-NO-FALLBACK: if ANY branch is element-first for the current child, the choice MUST use an element-first branch and MUST NOT fall back to a wildcard branch — even when the chosen element branch then fails structurally (a later required term is absent, e.g. `choice(sequence(a:int, b:int), any skip)` with only `<a>`) or by content. `matchOnce` first tries each element-first branch via `matchAt`; if none matches fully it commits to the first element-first branch and surfaces its real `matchParticle` failure (recording the genuine content/missing-element error) instead of entering the wildcard pass, and `tryMatchChoice`'s `scanOnce` mirrors this — once an element-first branch exists for the child it returns that branch's success/failure and never falls back — so lookahead and real passes agree. Only when NO branch is element-first for the child may the wildcard pass run. This stays bounded (the commit is to the first element-first branch in declaration order; no backtracking). The classifier `particleConsumesViaElement` delegates to `particleFirstConsumerKind`/`groupFirstConsumerKind` (consumerKind = element/wildcard/none), which is PATH-AWARE: it walks sequence members in order respecting occurrences and emptiable prefixes (reusing `restriction_particle.go`'s `particleEmptiable`), so a LEADING wildcard inside a sequence — e.g. `sequence(any skip, element a)` — is classified wildcard-first and does NOT win element precedence even though a later element leaf in the same group also matches the child; choice/all are element-first iff some member is. This is bounded first-consumer determination, not full backtracking. XSD 1.0 keeps pure declaration-order matching (byte-identical path). The SEQUENCE case is also enforced (`matchSequence`/`tryMatchSequenceOnce`/`tryMatchSequence` via `sequenceElementReservedLimit`, Version11-gated): before matching each particle, the children a LATER element particle in the same sequence is element-first-consumer for are reserved (the particle sees a truncated `children[:limit]`), so a leading minOccurs=0 wildcard (direct or nested as a sub-sequence's first term, e.g. `sequence( any{0,unbounded} lax, state, currency, zip )` from extending a wildcard-only base) cannot greedily consume the required named elements. The reservation fires ONLY when the current particle would consume the child via a wildcard, not an element leaf (element-vs-element stays in declaration order), AND only for a later element REACHABLE from the current position — every intervening particle is emptiable (`particleEmptiable`), so a required element between the wildcard and a trailing optional element keeps its child (no over-rejection); bounded lookahead, no backtracking, no-op for all-element sequences and XSD 1.0. The cos-nonambig position automaton carries a per-position ORIGIN tag (`firstSetEntry.origin`, `positionAutomaton.nextOrigin`): `applyOccurs` RESETS the counter before each occurrence copy it walks, so all occurrence-copies of ONE textual element leaf (any nesting depth, e.g. `(a{1,2}){1,2}`) share an origin while two DISTINCT textual leaves get distinct origins even when same-named and even sharing an `*ElementDecl` via a group ref; in 1.1 `entriesOverlap` treats same-name/same-origin positions as one particle (never a UPA violation) and same-name/DIFFERENT-origin as a violation (`choice(a,a)`, `choice(ref g, ref g)`). XSD 1.0 ignores the origin tag (pure name comparison), byte-identical. **Assertions (xs:assert on complex types)** are implemented (`assert.go`): `parseAssert` reads `@test` from `<xs:assert>` children of complexType/restriction/extension (1.1 only), captures the in-scope namespaces (`collectNSContext`), and pre-compiles via `xpath3.NewCompiler().Compile` (a missing/malformed test is a fatal schema error, mirroring IDC); the result is stored on `TypeDef.Assertions` (the `Assertion` struct mirrors `IDConstraint`). `validateElementContent` extracts the content switch into `validateContentByType` and, after attributes+content validate, calls `checkAssertions`, which walks the type's base chain and evaluates each test with the element as context node via `xpath3.NewEvaluator(...).Namespaces(a.Namespaces).Evaluate(ctx, expr, elem)` then `xpath3.EBV` — false → validity error. StrictPrefixes is intentionally NOT set so xs:/fn: keep default bindings. `parseAssert` also handles `<xs:assert>` inside a simpleContent extension/restriction (`parseSimpleContentChildren`). `$value` is bound (via `Evaluator.Variables`) to the element's TYPED simple value for a simpleContent type (`assertValueSequence`→`buildValueSequence`; empty sequence for complex content). For an EMPTY element `assertValueSequence` substitutes the declaration's fixed/default effective value (from `edecl`, threaded through `checkAssertions`) before typing — mirroring `validateSimpleContent` — so `$value` is the schema-normalized value, not the empty sequence. A QName/NOTATION fixed/default value substituted into an empty element resolves its prefix against the DECLARATION's namespace context (`ElementDecl.FixedNS`/`DefaultNS`, selected by `effectiveValueNS`), not the instance's, for both content validation (1.1 branch) and `$value`; XSD 1.0 keeps instance-context resolution. A materialized QName/NOTATION DEFAULT/FIXED ATTRIBUTE value is prepared by `materializeQNameAttrValue` (1.1 only): it whitespace-COLLAPSES the value before extracting the prefix/local (so `default=" p:x "` still binds `p`), resolves the ACTIVE type — for a UNION attribute type it calls `fixedUnionActiveMember(ctx, collapsed, declNS, …, vc.schema, vc.version)` so a `memberTypes="xs:QName xs:string"` default active in its QName member is materialized too — declares the value's prefix (from `AttrUse.FixedNS`/`DefaultNS`) on the element, minting a fresh non-colliding prefix and rewriting the value (with the collapsed local) if the instance already binds that prefix to a different URI, so an xs:assert/IDC atomizing the attribute resolves it in the schema value space. The whole materialization block is gated on `vc.version == Version11`, so XSD 1.0 inserts the default/fixed value exactly as authored with no namespace-declaration rewrite — byte-identical serialization (no golden exercises this case, so the gate is the only guard). **XDM context isolation** (`isolatedAssertTree`): the test is evaluated against a deep copy (`helium.CopyNode`) of the element rooted in NO document and with comment/PI nodes removed (`stripCommentsAndPIs`), so an absolute path `/`/`//` raises XPDY0050 (root is not a document node) — the assertion cannot navigate outside the element subtree — while the element's in-scope namespaces — INCLUDING an inherited default namespace (prefix "", when not already on the copy), so `namespace-uri-for-prefix('', .)` and unprefixed resolution survive isolation — are re-declared on the copy root and the PSVI type annotations are carried onto DESCENDANT elements and ALL attributes (`mapAssertAnnotations`, fed to `Evaluator.TypeAnnotations`) so a typed attribute (e.g. `@length eq count(...)`) atomizes in its value space and node-scope QName resolution works. The assertion-tree ROOT element is deliberately left UNannotated (an xs:assert is part of determining the element's own validity, so its type is not yet assigned — `data(.)` on the root is untyped, matching Saxon/the conformance tests). The PSVI annotations come from `validationContext.assertAnnotations`, an always-on map (1.1 only) populated by `annotateElement`/`annotateAttrUse`. A single-schema `xpath3.SchemaDeclarations` adapter (`schema_decls.go`, `schemaDecls`, via `vc.assertSchemaDecls()`, carrying `vc.version`) is passed to the evaluator so a node annotated with a NAMED user simple type atomizes through its builtin base (e.g. `data(@x) instance of xs:integer`) and instance-of/schema-element tests see the type hierarchy. The assert/assertion-facet evaluators also set `Evaluator.QNameValueNoDefaultNamespace()` (xpath3) so an UNPREFIXED xs:QName/xs:NOTATION VALUE atomizes to NO namespace — XSD value-space semantics (a QName value, unlike a name, does not pick up the in-scope default namespace; a prefixed value still resolves). The option is off by default, so general XPath/XQuery and XSLT atomization keep resolving an unprefixed QName value against the default namespace. Its `ValidateCast`/`ValidateCastWithNS` run `validateValue` with a `validationContext{schema, version, NilErrorHandler}` (NOT `TypeDef.Validate`, which hardcodes the 1.0 default), so a user-defined `cast`/`castable` inside a 1.1 assertion accepts 1.1-only lexical forms (e.g. year `0000`). BOTH the `castable` AND `cast` evaluation paths (`xpath3/eval_types.go` `evalCastableExpr`/`evalCastExpr`) are namespace-aware for QName/NOTATION-derived USER types: when context-free `CastAtomic` fails and SchemaDeclarations resolve the type's builtin base to `xs:QName`/`xs:NOTATION`, `cast` validates via `ValidateCastWithNS` and returns the namespace-RESOLVED QName/NOTATION value (`castToQName`) carrying the user type annotation — additive, so existing xpath3 cast behavior (no SchemaDeclarations) is unchanged. A self-referential cast (`cast`/`castable as t:T` inside t:T's own assertion) is bounded by a per-validation `(type, value)` cast stack carried in the context (`castGuardKey`, lazily created on the first `validateCast`, self-clearing as the recursion unwinds): a repeat fails closed (treated as not castable / a cast failure) instead of recursing `validateCast → validateValue → checkSimpleTypeAssertions → Evaluate → validateCast …` to a stack overflow. `xpathDefaultNamespace` on `<xs:assert>`/`<xs:assertion>`/root `<xs:schema>` is honored (`resolveXPathDefaultNS`/`assertionNamespaces`): the raw value (element-local OR schema-level) is whitespace-COLLAPSED (schema-for-schemas whiteSpace="collapse") before the empty check and the sentinel switch, so `" ##targetNamespace "` resolves like the sentinel rather than a URI literal; the default element namespace is governed SOLELY by it (the bare `xmlns=""` default is dropped), with `##targetNamespace`/`##defaultNamespace`/`##local` resolved and tracked across chameleon includes (`compile_imports.go` saves/restores `compiler.schemaXPathDefaultNS`) AND carried into the IMPORT sub-compiler (`impC.schemaXPathDefaultNS = getAttr(impRoot, attrXPathDefaultNamespace)` before `parseSchemaChildren`), so an imported schema's assertions resolve their own root default namespace. **xs:assertion simple-type facet** (`assertion_facet.go`): `parseFacets` reads `<xs:assertion>` into `FacetSet.Assertions`; `validateValue` (after lexical+facet validation, factored as `validateValueByVariety`) calls `checkSimpleTypeAssertions`, which binds `$value` to the typed value (`buildValueSequence`/`typedAtomic`/`atomicForType`: a sequence for a list with each item typed by its active member, a NAMED user-defined type's identity PRESERVED (`atomicForType` stamps the user type name as `TypeName`, builtin cast type as `BaseType`, via `builtinAtomicForType`, mirroring `AtomizeItem`/`data()`) so `$value instance of t:T` holds, a union resolved to its active member via SCHEMA-AWARE probing in `buildValueSequence` (so a union whose ACTIVE member is a LIST — e.g. `memberTypes="IntList xs:string"` with value `1 2` — yields the list-item sequence, not a single untypedAtomic) — `typedAtomic`/`buildValueSequence` thread `vc.schema` into `fixedUnionActiveMember` so a member whose own assertion needs `castable as t:T` is selectable, matching the real validation path — a QName/NOTATION lexical resolved against in-scope namespaces into an `xpath3.QNameValue`) and evaluates each assertion along the restriction chain (ANDed) with NO context item — so `.`/`position()`/`last()` raise a dynamic error → unsatisfied; the SchemaDeclarations adapter is passed here too. Because `validateValue` evaluates these facets, the COMPILE-TIME throwaway contexts that reach it are built with `schema: c.schema`: the attribute default/fixed constraint check (`checkAttrUseConstraints`, `link_refs.go`) and the facet value-against-base / enumeration-member / union-member checks (`check_facets.go`). Without it, a schema-aware cast in a facet assertion (`castable as t:T`) would fail closed and reject a VALID schema at compile time. (`typeAcceptsValue` in `validate_idc.go` and standalone `TypeDef.Validate` stay on the Version10/nil-schema default — the documented Phase-1 gap.) XSD 1.1 **attribute inheritance** (`finalizeEffectiveAttrs`, run in `resolveRefs`, gated to 1.1): EVERY derived complex type (extension OR restriction) inherits each base attribute use it does not redeclare. This is mandatory, not optional — `checkRestrictionAttrs` does not report a non-redeclared required base attribute as "missing" in 1.1, so the merge is what actually keeps that inherited requirement enforced (without it an instance omitting the attribute would wrongly validate). The merge is TOPOLOGICAL across BOTH derivation kinds: the base is finalized FIRST (memoized recursion with `merged`/`visiting` sets, following the base pointer regardless of derivation kind), THEN the derivation attribute check runs against td's OWN declarations and the finalized base (`checkRestrictionAttrs` for a restriction, `checkExtensionAttrDuplicates` for an extension — both run in the finalizer in 1.1 so they read a complete base set), THEN td inherits the non-redeclared base uses (a restriction inherits the base wildcard when it declares none; an extension also unions its own wildcard with the base's). Because both the extension and restriction passes DEFER all attribute work to this finalizer in 1.1, an extension-of-restriction or restriction-of-extension at any depth inherits correctly regardless of source order. The passes keep their content-model merge and `checkRestrictionParticles`; XSD 1.0 keeps the original per-pass attribute copy, byte-identical. **simpleContent content-type narrowing** (`TypeDef.ContentSimpleType`, built by `parseSimpleContentRestrictionType`): a simpleContent `<xs:restriction>` narrows the base content type via a nested `<xs:simpleType>` AND/OR its direct facets — `parseSimpleContentRestrictionType` composes BOTH (a restriction of the inline type carrying the sibling facets when both are present), so sibling facets are not dropped when an inline type is also given. Each synthetic restriction it returns is recorded in `typeDefSources` so `checkFacetConsistency` runs facet applicability / value-against-base checks on it — an inapplicable sibling facet (e.g. `xs:minInclusive` on an `xs:string` base) is a compile error rather than silently ignored. The EFFECTIVE content simple type is composed across the WHOLE simpleContent derivation chain by `effectiveContentSimpleType`, which recurses through simpleContent complex types (marked `TypeDef.IsSimpleContent`, set in `parseSimpleContent`) and stops at the underlying simpleType/builtin (which IS its own content type). It terminates via a VISITED-SET cycle guard (tracking `*TypeDef` pointers), NOT a depth cap, so an arbitrarily deep finite acyclic chain is walked fully and a narrowing facet far down the chain (which lives on `ContentSimpleType`, not `Facets`) is never silently skipped; a facet-only narrowing (the synthetic `ContentSimpleType` whose `BaseType` is the owning complex type) is re-based on the effective base content type so ancestor and derived facets compose. `validateSimpleContent` (1.1) validates the text against the composed type whenever `simpleContentNeedsValidation` reports it constrains the value (list/union variety, a non-string builtin, or any facet/assertion along its chain); `assertValueSequence` types `$value` from the same composed type. A NESTED `<xs:simpleType>` content type carries its OWN declared base chain (it is returned by `effectiveContentSimpleType` as-is, NOT re-based onto the base content), so the base complex type's inherited content facets would otherwise be bypassed; `validateSimpleContent` therefore walks the simpleContent base chain and ALSO validates the value against each ancestor `effectiveContentSimpleType(cur.BaseType)` where `cur.ContentSimpleType.BaseType != cur` (nested-type and inline-type-plus-sibling-facets cases — re-based facet-only synthetic excluded), so inline narrowing AND inherited base facets apply through later derivation hops (XSD §3.4.2.2: a nested simpleType RESTRICTS, it does not REPLACE, the base content type — e.g. a `maxLength` on the base still rejects an over-long value the nested type alone would accept). So a simpleContent EXTENSION of a named faceted/asserted simple type enforces that base type's facets/assertions, and a narrowed content type (e.g. an ancestor enumeration) is inherited through a further restriction OR extension. XSD 1.0 keeps the per-type gating, byte-identical. The complexContent-extension-over-simple-base check (cos-ct-extends-1-1) also flags the empty/attribute-only case in 1.1. **Conditional type assignment (xs:alternative)** is implemented (`alternative.go`): `parseTypeAlternatives` reads the `<xs:alternative>` children of an xs:element (1.1 only) in order — each carries an optional `@test` (absent = unconditional default) pre-compiled via xpath3 and a `@type` reference registered on `compiler.altTypeRefs` and resolved by `resolveAltTypeRefs` in `resolveRefs`; results are stored on `ElementDecl.Alternatives` (the `TypeAlternative` struct). At validation `applyTypeAlternatives` runs at every type-selection site (root `validateRootElement`, and `matchAll`/`matchElementParticle`/`matchWildcardParticle`/`validateWildcardChild` plus the xs:anyType/lax-descendant path in `validate_elem.go`) between `effectiveDeclType` and `resolveXsiType`: when no xsi:type is present (`elementHasXsiType`) it returns the type of the first alternative whose `@test` EBV is true (or the testless default), else the declared type — so xsi:type still wins; a failing alternative test is treated as not-matched. At the wildcard/anyType sites CTA selects the governing type FIRST, then the cvc-elt.4.3 derivation-block check and the idc lax-assessment/ID-pass (`assessLaxElement`, `assessed=true`) run against the SELECTED type. An `<xs:alternative>` may carry an INLINE anonymous `<xs:complexType>`/`<xs:simpleType>` (`parseTypeAlternative`; a present-but-empty `@type` is itself a governing-type source, so `@type=""` plus an inline type is a conflict and a bare `@type=""` is an invalid QName), `type="xs:error"` (a selected xs:error invalidates the element even through the xsi:nil nilled path), and `@xpathDefaultNamespace` (`effectiveXPathDefaultNS`: local value resolved against the alternative else inherited from the schema-level default; affects only unprefixed element name tests). The @test runs against a DETACHED single-node CTA context (`inherited_attrs.go` `ctaContextNode`: an orphan element bearing the element's own attributes PLUS inheritable attributes from ancestors (`inheritedAttributes`, decl `inheritable="true"`), in-scope namespaces, position()=last()=1, an `emptyCollectionResolver`, schema base URI, no children/parent — so @test cannot navigate to ancestors/siblings). A non-default `@type`/inline alternative must be VALIDLY SUBSTITUTABLE for the declared type (`checkAltSubstitutability`, compile time): `strictBuiltinAwareDerivedFrom` (`builtin_hierarchy.go`) accepts a genuine `isDerivedFrom` derivation (user types), the built-in simple-type hierarchy via the `builtinSimpleBase` table (the 1.0 built-ins are NOT BaseType-linked), or the anySimpleType simple-content rule; a union declared type also admits any of its base-chain-resolved member types (`resolveVariety`/`resolveUnionMembers`); there is NO permissive simple-vs-simple fallback. The block check is built-in-aware (`isDerivationBlocked`). Per-document xpathDefaultNamespace/base-URI and the `altTypeRefs`/`ctaElems` worklists are saved/restored across include/redefine and seeded onto import sub-compilers. The xs:alternative @test is statically restricted (`alternative.go` `checkCTATestStaticContext`, via the new `xpath3.Expression.StaticReferences(namespaces)`, which resolves every type/function name to a namespace URI — uniform across prefixed, unprefixed, and braced-URI `Q{uri}local` forms — so the check is a pure URI allowlist): a FREE variable reference (cta9002err), a non-built-in type named in cast/castable/instance of/treat as / element()/attribute() kind tests / path-step node tests — verified against the XPath/XDM type hierarchy `xpath3.IsKnownXSDType` so XDM-only names (xs:untyped/xs:untypedAtomic/xs:anyType) are accepted but an unknown xs: name (xs:noSuchType, XPST0008) is rejected (cta9003err, ibm typeAlternatives s3_12ii06) — or an unknown/wrong-arity function call/`#arity` ref/arrow target — verified against the STANDARD registry `xpath3.StandardFunctionAcceptsArity(uri,local,arity)`, which holds the F&O 3.1 fn/math/map/array functions AND the xs: type constructors but EXCLUDES helium extension functions (fn:flatten/array:flat-map), so user-namespace functions, user-type CONSTRUCTOR calls, unknown functions, extension functions, and wrong-arity calls (XPST0017) all reject (arrow arity counts the prepended left operand; an inline-function literal call references no named function and is in-context) — OR a schema-element()/schema-attribute() node test (which references a GLOBAL declaration absent from the CTA static context, §F.2) — is a schema error; a testless (no-`@test`) alternative that is NOT the LAST in document order is rejected (`parseTypeAlternatives`, cta9001err). The inherited schema-level `@xpathDefaultNamespace` is resolved against the HOST alternative element, not the schema root: `effectiveXPathDefaultNS` re-resolves the RAW token (`compiler.schemaXPathDefaultNSToken`, per-document like `xpathDefaultNSSet`) at `elem`, so `##defaultNamespace` uses the alternative's own `xmlns` redeclaration (cta0005) per the {xpath default namespace} mapping (host in-scope namespaces); `##targetNamespace`/`##local`/URI resolve identically either way. `ctaContextNode` carries the INSTANCE document's URI onto the synthetic document so `fn:base-uri(.)` in @test resolves to the instance (cta0021; `fn:static-base-uri()` stays the schema URI). Two element declarations sharing an expanded name in one content model must have EQUIVALENT {type table}s (Element Declarations Consistent extended to type tables, cta9009err/cta9010err), and an element restriction's {type table} must be both-absent or both-present-and-EQUIVALENT to the base's (`restriction_particle.go` `elementRestrictsElement`, Particle Valid (Restriction) clause 4.6, cta0043) — equivalence is the conservative structural comparison (`alternative.go` `typeTablesEquivalent`: same length, identical `@test`s, same {type definition}s via `elementTypesConsistent`). **Open content (xs:openContent + xs:defaultOpenContent)** is implemented (`opencontent.go`): `parseOpenContent` reads `<xs:openContent>` (mode interleave/suffix/none; the `<xs:any>` child becomes the wildcard and must NOT carry minOccurs/maxOccurs — `parseOpenContentWildcard`; mode="none" must not carry a wildcard; at most one annotation) on complexType/restriction/extension into `TypeDef.OpenContent`, setting `TypeDef.openContentExplicit`. The schema-level **`<xs:defaultOpenContent>`** (`readDefaultOpenContent`) is PER-DOCUMENT — saved/reset/restored across xs:include/xs:redefine/xs:override/import like the form defaults (an xs:override/xs:redefine REPLACEMENT child uses the OVERRIDDEN/redefining document's default per §4.2.5, threaded back via `overrideLoadTarget`'s returned `*OpenContent`) — must precede component declarations, and carries mode interleave|suffix (NOT none) plus `appliesToEmpty` (default false). It is captured onto each complex type lacking an explicit openContent as `TypeDef.pendingDefaultOpenContent`. `resolveOpenContent`/`computeEffectiveOpenContent` (in `resolveRefs`, after content models/types are finalized, BASE-FIRST) computes each complex type's EFFECTIVE open content: the explicit one (mode="none" → absent, suppresses the default) else the per-document default (applied unless the content type is empty and appliesToEmpty=false — `contentTypeEmptyForOpenContent`); an EXTENSION inherits/merges the base's (§3.4.2.2 — base interleave mode wins, wildcards union; an extension may not relax a base interleave to suffix — §3.4.6.2) and a RESTRICTION's validity is checked (§3.4.6.4 `checkOpenContentRestriction` — may not add open content to a closed base, subset wildcard, ≥ processContents, mode compatible; all WAIVED when the restriction content model is empty). `validateContentByType` routes the element-only/mixed/empty path through `validateContentModelOpen` when present: **suffix** matches the declared model (`matchContentModelSuffix` — for an `xs:all` it uses the lenient `matchAll11` that stops at the first non-member instead of erroring) then requires every trailing child to match the wildcard (a trailing declared-named child is "not expected"); **interleave** partitions children — those whose expanded name is NOT in `collectModelElementNames(mg)` and which match the wildcard are the open set, the rest must satisfy the declared model — then `refineInterleavePartition` moves a declared-named child the model cannot place (and which matches the wildcard) into the open set (§3.4.4.3.2 existential partition, via a `suppressDepth`-guarded trial match), so open content and declared content may match the same names. Open children are validated via `matchWildcardParticle` (lax/strict/skip). `<xs:complexContent>`/@mixed is honored and must agree with `<xs:complexType>`/@mixed, and an extension and its base must both be mixed or both element-only (§3.4.6.2 cos-ct-extends) — both version-INDEPENDENT (enforced in XSD 1.0 and 1.1). The complexType schema-representation grammar (the `(annotation?, (restriction|extension))` wrapper via `parseDerivationWrapper`, the restriction/extension derivation-body order/cardinality via `parseComplexContentDerivationBody`/`parseSimpleContentChildren`, the direct-complexType annotation-first/at-most-one rule, and the global `<xs:complexType>`/@name `xs:NCName` check) is likewise enforced in both versions; only the 1.1-construct cases (openContent/assert) and the direct-complexType stray-child default stay Version11-gated. KNOWN GAPS (skipped): whitespace in a genuinely-empty (`<xs:sequence/>`) element-only content type is tolerated (open012.n3); schema-component `@id` uniqueness/NCName validity IS enforced (`check_ids.go`, a version-independent XSD rule run in BOTH 1.0 and 1.1: a per-document DOM walk validating each XSD-namespace element's `id` as a valid xs:ID — NCName after whitespace-collapse — and unique within the schema document; run on the entry document AND every nested document loaded by xs:include/xs:redefine/xs:import/xs:override with a fresh per-document `seen` set after conditional-inclusion pruning, so a duplicate/invalid @id in an included/imported/overridden document is caught while the same value may recur across documents; the walk skips xs:appinfo/xs:documentation payload (an xs:annotation's own @id still counts); fixes open038/open039). **xs:all relaxations (XSD 1.1)** are implemented (Saxon `All.testSet` 61/62; all gated on `Version11`, 1.0 byte-identical). (0) The all group's OWN maxOccurs may be 0 or 1 in 1.1 (`validateAllOccurs` accepts `{0,1}`; 1.0 requires exactly 1 — mgO001/mgO018). (1) Element members may carry `minOccurs`/`maxOccurs>1`/`unbounded`: `checkAllElementParticleOccurs` (`check_elements.go`) skips the all-specific 0/1 rule in 1.1 (the generic lexical / min≤max checks still apply). (2) `xs:all` may contain element WILDCARDS, and an `xs:group ref` resolving to an all group may be nested directly inside another `xs:all` with occurrence exactly 1/1 (flattened into the parent); `checkAllGroupRef` (`link_refs.go`, via `groupRefSource.parentCompositor`) allows this in 1.1 and rejects a referenced sequence/choice group, an occurrence ≠ 1/1, or any nested all-group ref in 1.0; an INLINE `<xs:all>` directly inside an `<xs:all>` is rejected too (cos-all-limited — `read_particles.go`, Version11-gated, so it never reaches the flatteners). (3) `matchAll`/`tryMatchAll` (`validate_elem.go`) DISPATCH by version: XSD 1.0 uses the legacy boolean-`seen[]` matcher (`matchAll10`/`tryMatchAll10`), BYTE-IDENTICAL to the pre-1.1 path — a wildcard particle the parser tolerates inside a 1.0 xs:all is never matched and a required one is reported missing (a wildcard particle is not tracked in the 1.0 required-member bookkeeping, guarded by `TestAll10WildcardOnlyRegression`). XSD 1.1 uses a FLAT member list (`flattenAllMembers` — elements, wildcards, and members of a flattened nested 1/1 all) with PER-MEMBER occurrence COUNTING (`matchAll11`/`tryMatchAll11`), matched ORDER-INDEPENDENTLY against each member's min/max; an element member is matched only when the child is ADMISSIBLY substitutable for it (`allMemberForChild`→`elemMatchesDeclOrSubst`, honoring block="substitution"/derivation-block and rejecting abstract heads AND abstract substitution members), and weak-wildcard precedence holds (a declared element with remaining budget wins over a wildcard). (4) Restriction subsumption of a base `xs:all` is occurrence-COUNTING per BASE MEMBER (`all_subsumption.go` `allRestrictsByCounting`/`memberContributions`, routed from `groupRestrictsGroup` for Version11 no-wildcard all:all / sequence:all / choice:all): `memberContributions` computes, per BASE-MEMBER INDEX, the (min,max) the derived side can emit (summing across a sequence/all, merging a choice's branches by per-base-member min/max so alternative branches CORRELATE on the base member rather than summing distinct derived names, scaling by group occurrence); a non-emitting particle (prohibited wildcard / empty group) contributes nothing (checked before rejecting a wildcard term); a derived element maps to a base member by name or substitution group via the shared `admissibleSubstitutionMember` predicate (concrete member, head allows substitution, member's EFFECTIVE type substitutable for the head's — `findBaseAllMember`); the contribution per base member must lie within that member's range and every unmapped base member must be emptiable. Several derived particles (e.g. substitution-group members) may thus collectively restrict ONE base member with maxOccurs>1. The wildcard cases route to `allRestrictsWithWildcards`, which FLATTENS nested 1/1 all-groups on both sides (`flattenAllParticles`) and SUMS concrete derived elements per base element before accounting remaining concrete/wildcard contributions against the base wildcards' cardinality; a nested NON-all derived group (sequence/choice) that survives flattening is NOT decomposed/accounted and is rejected (fail-closed) rather than silently accepted, so it cannot smuggle an out-of-namespace element past a base wildcard. When the syntactic Particle Valid (Restriction) check FAILS in 1.1, a SOUND fallback (`restriction_subsume.go` `particleLanguageSubset`) PROVES `L(derived) ⊆ L(base)` by product simulation of the two content-model automata over the finite alphabet of names the derived model can emit (element leaves expanded to instance-admissible substitution members; a base wildcard matches every admitted name via `wildcardAllowsName`; per-element type/nillable/fixed checked by `elementRestrictsElement`). It is FAIL-CLOSED — a DERIVED wildcard, an xs:all group, or an over-large occurrence unroll (`subsumeNFAStateCap`/`subsumePairStateCap`/`subsumeUnrollCap`) returns "not proven" (keep the rejection) — and runs only as a fallback that ACCEPTS, so it never rejects more (particlesHa161/Hb008/Hb011/Z001/Z028). XSD 1.0 keeps the syntactic verdict, byte-identical. Circular attribute group definitions are also permitted in 1.1 (W3C bug 15795 / attgC010-D015): the direct self-reference (`read_particles.go`) and indirect cycle (`link_refs.go` `checkCircularAttrGroupRefs`) are dropped/cut without a diagnostic (the back-edge is still CUT so the walks terminate); 1.0 rejects both (src-attribute_group.3), byte-identical. (5) An `xs:all` may be EXTENDED by another `xs:all`: the extension content-merge loop (`link_refs.go`) merges the two member sets into a SINGLE all group — both content models must be all groups with the SAME minOccurs; a sequence/choice extending an all, an all extending a sequence/choice, or a minOccurs mismatch is rejected; an open-content `interleave` base may not be extended to `suffix` mode. (6) UPA over all members (`check_upa.go` `walkAllBody`) is fed by MULTI-HEAD substitution groups (`substitutionGroup="a b"` registers the member under every head — `read_elements.go` parses the list into `ElementDecl.SubstitutionGroups`, `buildSubstGroups` registers each), so an element substitutable for two distinct all members is a cos-nonambig violation. UPA competition is by DECLARATION membership: an ABSTRACT substitution member STILL competes (XSD 1.1 §3.8.6.4 / bug 4337, W3C wgData/sg/upa.xsd) even though it can never appear in an instance, so `walkTerm` does NOT skip abstract members. Substitution-group membership is resolved through a TRANSITIVE cycle-guarded closure (`transitiveSubstClosure`) that SEPARATES traversal from inclusion: an edge is traversed when structurally substitutable (`substitutionMemberTypeOK`, abstract ALLOWED) so a concrete descendant behind an abstract intermediate (`h<-abstract m1<-concrete m2`) is reached, while `block="substitution"`/derivation-block prune. The two variants differ only in INCLUSION: `instanceSubstMembers` includes a reached member only when CONCRETE (abstract excluded) for runtime matching (`elemMatchesDeclOrSubst`) and subsumption (`findBaseAllMember`); `substitutableMembersFor` includes abstract members (DECLARATION membership) for UPA (`walkTerm`) and the byte-identical 1.0 matcher (`matchAll10`/`tryMatchAll10`/`resolveSubstDecl`). Both walk the full chain (`h<-m1<-m2`) — a direct `substGroups[head]` lookup misses multi-level members — and the abstract-included-vs-excluded split keeps UPA competition distinct from instance admissibility. (7) A nilled element (`xsi:nil`) with whitespace-only character content is rejected in 1.1 (`validateNilledElement`, cvc-elt.3.2.1; 1.0 still tolerates it). KNOWN GAP: `all308` (an empty mixed `xs:all` base extended by an all — Saxon bug 6202, where Saxon itself treats the schema as a valid extension) is accepted, not rejected. **Wildcard Element Declarations Consistent (EDC)** is implemented (Version11-gated). The STATIC type-table check (`check_element_consistent.go` `checkWildcardElementConsistent`/`collectModelGroupParticles`/`typeTablesConsistent`/`anyWildcardAllows`, run from `checkElementConsistent` over every complex type AND named group) rejects a content model whose LOCAL element declaration particle and a same-named GLOBAL element declaration reachable through a lax/strict (non-skip) wildcard have inconsistent {type table}s (conditional type assignment). Only the type TABLE is compared, NOT the type DEFINITION (a wildcard admits differently-typed elements, so a type-def difference is permitted — e.g. a local union(date,time) element plus a wildcard matching an xs:duration global of the same name is valid); the asymmetric empty-vs-present case is flagged (under-strict, never false-rejecting two distinct-but-present tables). The DYNAMIC check (`validate_elem.go` `validateWildcardElementConsistent` + `edcLocalDecls`) walks the matched type's BASE chain via `vc.edcType` (set/restored by `validateContentByType` in `validate.go`) in addition to the current content model: a restriction that drops a base's local element declaration but admits the name through a wildcard must have its wildcard governing type validly substitutable for the base's local type, else the instance is invalid. **Skip-wildcard IDC selector scoping** is implemented: `annotateSkipChildren` (`validate.go`) records every element inside a `processContents="skip"` subtree in `vc.skipContentNodes`, and `evaluateIDC` (`validate_idc.go`) drops those un-assessed nodes from the selector node-set, so an ancestor `xs:key`/`xs:unique`/`xs:keyref` does not constrain them (the pass-3 ID/IDREF datatype pass already excluded skip content; this fixes the pass-2 selector). 1.1 only — `skipContentNodes` is nil/empty in 1.0, byte-identical.

XSD 1.1 QName references (`@type`, `@base`, `@ref`, `@itemType`, and similar paths through `resolveQName` in `link_refs.go`) reject `lexicon.NamespaceXSDDatatypes` (`http://www.w3.org/2001/XMLSchema-datatypes`), the deprecated XML Schema datatypes namespace. Relax NG still uses that namespace through its own validation pipeline.

### Compile: Document → Schema

1. **Parse root** — must be `xs:schema`; extract targetNamespace, form defaults, block/final defaults
2. **Register built-in types** — 46 XSD primitives
2a. **Conditional inclusion pre-pass** (`conditional_inclusion.go`, `applyConditionalInclusion`) — run AFTER built-in registration (it consults the registry for type availability) and BEFORE the first collect pass, in BOTH 1.0 and 1.1 mode, on the top-level root AND every included/imported/redefined document (`loadInclude`/`loadRedefine`/`loadImport`). **Caller-document safety:** the pre-pass `helium.UnlinkNode`s excluded elements, which would mutate the caller's parsed `*helium.Document` and make `Compile` non-idempotent (compiling the same doc under 1.0 then 1.1 would lose the 1.1 branch). So `compileSchema` first runs a cheap `documentHasVCDirective(root)` scan and, ONLY when a vc attribute is actually present, compiles against a `helium.CopyDoc(doc)` clone (URL preserved via `SetURL`, so include/import/redefine `schemaLocation` resolution is unchanged) and prunes the CLONE; a vc-free schema keeps the fast no-copy path (no perf regression). Nested include/import/redefine documents are parsed fresh on every `Compile`, so the pre-pass prunes those in place. It walks the schema element tree and `helium.UnlinkNode`s (collected first, unlinked after — never mid-iteration) any element — with its whole subtree — excluded by its version-control (`vc:`, ns `lexicon.NamespaceXSDVersioning`) attributes for the active `c.version`. Prune rules: `vc:minVersion`/`vc:maxVersion` (xs:decimal) keep iff `minVersion <= processorVersion < maxVersion`, compared EXACTLY via `value.CompareDecimal` (math/big.Rat) against the processor version as the exact string "1.0"/"1.1" — NOT float64, so a high-precision bound (`1.1000…001`, kept) is not mis-rounded and a many-digit valid bound does not float-overflow into a spurious "malformed" error; `vc:typeAvailable`/`vc:facetAvailable` keep iff EVERY listed QName is available; `vc:typeUnavailable`/`vc:facetUnavailable` keep iff at least one listed QName is UNavailable (empty `*Unavailable` = unconditional exclude; empty `*Available` = no-op — both fall out of the "all available" formulation). "Available" is a fixed PROCESSOR-CAPABILITY check, NOT "is it declared somewhere": an XSD-namespace built-in TYPE name is matched against the IMMUTABLE per-version set (`builtinTypeAvailable` over `builtinTypeSet10` + the `builtinType11Bases` 1.1-only names — both the single source `registerBuiltinTypes`/`registerBuiltinTypes11` register from), NOT `c.schema.types` (which can already hold user/included declarations — a 1.0 schema literally declaring `{XSD}error` must not make `vc:typeAvailable="xs:error"` true); an XSD-namespace FACET name is matched against `xsdFacetNames`/`xsdFacetNames11` (`assertion`/`explicitTimezone` are 1.1-only). xs:error and the other 1.1 types are available only in 1.1; a non-XSD QName is unavailable. A vc-excluded ROOT `<xs:schema>` drops all its children (empty schema); `applyConditionalInclusion` returns that root-excluded flag so each caller SHORT-CIRCUITS to an empty contribution BEFORE interpreting/validating the root's other (non-preserved) attributes — at top level `compileSchema` skips blockDefault/finalDefault validation (an excluded root must not fail on, e.g., a bogus `blockDefault` it would never use; `registerBuiltinTypes` is moved ahead of that validation so the pre-pass has the type registry), but if the pre-pass already reported a malformed-vc error (`errorCount > 0`) it returns `ErrCompilationFailed` rather than swallowing it behind the empty-schema return. For a NESTED document the pre-pass runs right after the root is confirmed `<xs:schema>` and BEFORE the targetNamespace-compatibility / src-import check (`loadInclude`/`loadRedefine`/`loadImport`), so a vc-excluded nested root with an incompatible TNS is an empty contribution, not a TNS error. A vc-excluded `loadInclude` root returns `nil` (nothing to merge), but a vc-excluded `loadRedefine` root does NOT short-circuit: it contributes an EMPTY Phase-A set and still runs `processRedefineOverrides` against it, so an `<xs:redefine>` override targeting a (now-absent) component is REJECTED per XSD redefine semantics rather than silently accepted. For `loadImport` the sub-compiler's diagnostics are flushed to the parent on EVERY exit path after the sub-collector is installed via an idempotent `propagateImpErrors` (a `propagated` guard) that is `defer`red right after install — so a 1.1 malformed-vc value recorded during the pre-pass is propagated (and fails the compile) even when a later early return fires, e.g. the `parseSchemaChildren`-error or fatal nested-load path; the explicit calls that remain only fix ordering (forward while the parent is still error-free, before a TNS error is reported, and skip the declaration merge on failure). Malformed `vc:` values — a non-`xs:decimal`-lexical version (`isValidXSDDecimal`: sign/digits/one-dot form only, magnitude-agnostic; the value is trimmed of ASCII XML whitespace only — `" \t\r\n"`, NOT `strings.TrimSpace`, so a NBSP-padded version stays malformed) or an invalid/unbound-prefix QName in a list (`xmlchar.IsValidQName`+`lookupNS`) — are fatal schema errors ONLY under 1.1; under 1.0 they are tolerated and the condition skipped (so a schema with a bad `vc:minVersion` is valid under 1.0, invalid under 1.1, matching W3C VC vc902/vc903). Only the six exact attribute local names are consulted; a misspelt versioning-namespace attribute (e.g. `vc:minversion`) is an inert foreign attribute. Empirically calibrated against the W3C saxonData/VC + ibmData conditionalInclusion suites (note: the spec's "Unavailable" quantifier is the surprising one — vc011 single-known prunes, vc013 mixed keeps, vc014 xs:error single-known prunes).
3. **First pass: collect** — walk children, populate maps:
   - `schema.elements` (global element decls)
   - `schema.types` (named complex/simple types)
   - `schema.groups` (model groups)
   - `schema.attrGroups` (attribute groups)
   - `schema.globalAttrs` (global attributes)
4. **Process includes/imports** — load `xs:include`/`xs:import`/`xs:redefine`, merge declarations. Nested-schema document loads go through `compiler.readNestedSchema(path)` (FS `Open` preferred, falling back to `fs.ReadFile` for a `ReadFileFS`-only FS whose `Open` errors, both under a `maxNestedSchemaSize` 10 MiB byte cap via `internal/iolimit`, so an endless source cannot exhaust memory), reading from `compileConfig.fsys`, an injectable `fs.FS` set via `Compiler.FS(...)`; it **defaults to `iofs.DenyAll`** (opens nothing — secure by default, mirroring `helium.NewParser`) so an untrusted schema cannot disclose local files or exhaust resources via a hostile `schemaLocation` — opt into host access via `Compiler.FS(helium.PermissiveFS())` or a confined FS — and is propagated to sub-compilers. An over-cap read returns `errSchemaTooLarge`, classified fatal by `IsFatalSchemaLoad`. `xslt3` injects a resolver-backed `fs.FS` (`schemaResolverFS`) so nested includes inside a resolver-loaded schema obey the same default-deny `URIResolver` policy as the top-level load. Schema-location resolution is **URI-aware** and lives in a **single canonical helper**, `xsd.ResolveSchemaURI(ref, base) (string, error)` (`xsd/resolve_uri.go`), shared by both the xsd nested-include path and `xslt3`'s schema loader so the two layers cannot drift. `validateSchemaPath` is a thin wrapper over it. It keys off whether `ref`/`base` carries a URI scheme (`xsd.URIScheme`, the **one** scheme-detector for both packages — multi-char scheme required so Windows drive letters and bare OS paths stay local): an **absolute-URI** `schemaLocation` (e.g. a cross-host `https://cdn.example.com/part.xsd`) is passed through **unchanged** — never `filepath.Join`ed, which would collapse `//` and drop the host; a **relative** location against a **URI base** resolves via `net/url` `ResolveReference` (RFC 3986), keeping authority intact **and re-applying the base's `OmitHost` flag** when the base had no authority (so `mem:/schemas/main.xsd` + `part.xsd` → `mem:/schemas/part.xsd`, never `mem:///schemas/part.xsd`, while canonical `file:///...` bases keep their `///`); a genuine **local** base/location keeps the historical `filepath.Join` + `..`-escape guard (the only branch that can return an error). The import sub-compiler's `baseDir` is `schemaBaseDir(path)` (the full URI for URI bases, `filepath.Dir` for local). Because resolution happens while base and raw `schemaLocation` are still separate, the name reaching the FS is the **canonical** nested URI, so `schemaResolverFS.Open` forwards it verbatim (no string repair of a collapsed name). `xslt3`'s `resolveSchemaURI` delegates the absolute-URI and URI-base cases to `xsd.ResolveSchemaURI` and only handles its own local **file**-base case (xslt3's base is a full file URI/path, not a directory); it seeds the xsd `BaseDir` via `schemaCompileBaseDir(uri)` (full URI when scheme present, `filepath.Dir` otherwise). **targetNamespace match (src-import / src-include):** `loadImport` rejects the located schema when its `targetNamespace` differs from the `namespace` declared on `<xs:import>` — a present `namespace` requires that exact TNS, an absent `namespace` requires the imported schema to have no TNS (so a schema imported as one namespace cannot silently contribute another's declarations). `loadInclude`/`loadRedefine` enforce the analogous include rule (included TNS must equal the including schema's, modulo chameleon includes with no TNS). Both raise a fatal `Schemas parser error` and stop merging that document. **Fatal-load exception:** an `xs:import` load failure (top-level in `processIncludes` and nested inside `loadImport`) is normally demoted to a non-fatal I/O warning ("Failed to locate a schema ... Skipping the import."). An `xs:include`/`xs:redefine` load failure is ALSO demoted to the same warning pair (`reportSchemaLoadWarning`, "Skipping the include."/"Skipping the redefine.") — a `schemaLocation` is only a hint, so a genuinely-missing target lets compilation continue (libxml2 parity; fixes the W3C anyURI_a001 case whose valid schema carries three unresolvable include/import/redefine hints) — but ONLY when `nestedLoadFailureFatal` allows it: the error must be a plain `fs.ErrNotExist` (a parse/structural failure such as a node-content-size breach stays fatal), it must NOT be `IsFatalSchemaLoad`, AND the configured FS must NOT be the default `iofs.DenyAll` (a deny-all refusal is surfaced as fatal so an untrusted schema's missing include is not silently swallowed — `TestDefaultFSDeniesUntrustedInclude`). `errImportDepthExceeded` / `errIncludeDepthExceeded` / `errSchemaPathEscape` / `errSchemaTooLarge` (the nested-schema byte-cap breach) already bypass that demotion as security limits (the include-depth sentinel is classified fatal by `IsFatalSchemaLoad` so an over-deep `xs:include`/`xs:redefine` chain nested inside an imported schema is not swallowed by `loadImport`'s nested-processing fallback), and any error whose chain satisfies the exported `xsd.FatalSchemaLoader` interface (`FatalSchemaLoad() bool` → true, found via `errors.As` through the `*fs.PathError` returned by the FS `Open`/`fs.ReadFile`) does too. `xslt3`'s `schemaResolverFS` wraps an over-cap read (`ErrResourceTooLarge`) in such a marker so the resource cap cannot be silently defeated for an imported schema; the marker `Unwrap`s to the original error so `errors.Is(err, xslt3.ErrResourceTooLarge)` still holds at the xslt3 boundary. **Transitive includes:** after parsing an included/redefined schema's declarations, `loadInclude`/`loadRedefine` recurse via `processNestedIncludes` (switching `baseDir`/`filename` to the included schema) so that schema's OWN `xs:include`/`xs:import`/`xs:redefine` resolve — a chain `main → inc1 → inc2(defines T)` where `main` uses `T` compiles. Recursion is bounded by `includeVisited` (a per-compiler loaded-set keyed on resolved fs path, so each document loads at most once and circular includes terminate) plus a `maxIncludeDepth` cap (`errIncludeDepthExceeded`). `includeVisited` only records documents pulled in via `loadInclude`/`loadRedefine`, so `CompileFile` seeds it with the TOP-LEVEL schema's own resolved key (`compileConfig.rootKey`, computed via `ResolveSchemaURI(filepath.Base(path), baseDir)` in the same canonical form a nested include would use): a cycle that points back at the root (`main → inc → main`) then treats the root as already-loaded instead of re-parsing it and emitting spurious duplicate-component errors. **Repeated `xs:redefine` of the same document:** XSD permits multiple `xs:redefine` targeting one document (redefining disjoint components, or a no-op repeat), so a redefine whose target is already in `includeVisited` is NOT rejected for the path repeating. When a document is first loaded (via `loadInclude` or `loadRedefine`), the set of redefinable component names it contributes (the `(afterKeys − beforeKeys)` delta, split by kind via `computeRedefinableKeys`) is cached per resolved path in `loadedRedefinable`; a later `xs:redefine` of that already-loaded document skips Phase A and processes its override children (`processRedefineOverrides`) against the cached Phase-A set. A shared `consumed` set per document tracks which components have already been redefined across every `xs:redefine` of it, so a component redefined twice (`g` in two redefines) is reported as a duplicate (`… does already exist.`) while disjoint redefinitions and an empty/no-op repeat compile clean. **`xs:override` (XSD 1.1, `override.go`, gated to `Version11`):** processed in `processIncludes` alongside include/import/redefine. Unlike redefine (restriction/extension of the SAME component), override is WHOLESALE REPLACEMENT of any top-level component (element/attribute/simpleType/complexType/group/attributeGroup/notation) in the referenced document matched by an override child of the same (expanded-name, symbol space — simpleType and complexType share ONE type symbol space). `loadOverride` collects the override children, then `overrideLoadTarget` loads the target (like include: TNS/chameleon/form-default handling), parses its SURVIVING components via `overrideParseTargetChildren` (a matched component is SUPPRESSED and recorded), and recurses into the target's own include/override via `overrideProcessNested` — the transform is TRANSITIVE, a nested `xs:include` is treated as an `xs:override` carrying the cascaded set and a nested `xs:override` merges its children (OUTER override wins on collision). The active override set is BRANCH-LOCAL: a nested `xs:override` derives a FRESH map (inherited ∪ its own children, outer-precedence, `activeFromEntries`) so its children cannot leak into a later SIBLING include/override target's suppression set; `overrideLoadTarget` RETURNS the matched subset and each `xs:override` REGISTERS its own matched children IMMEDIATELY (`registerMatchedChildren`/`registerOverrideChild`), while the DECLARING document's context (form/block/final defaults, xpathDefaultNamespace, base URI, includeFile) is still active — never deferred to an outer context. An override child that matched nothing is DROPPED, so a reference to it is a dangling-ref error (conformance-verified against over026 — registering unmatched children makes over026 the only failing case; NOT the literal §4.2.5 "add all children" reading). Termination uses a SEPARATE `overrideVisited` set (distinct from plain-include `includeVisited`) keyed by `path + overrideActiveFingerprint(active)` (sorted symbol/name/replacement-element identity) plus the document's `rootKey`: a re-reach under the SAME active set or a back-edge to the root terminates without re-loading (permissible circular override), but the SAME document reached with a DIFFERENT active set is a DISTINCT transformed document and is loaded again — keying by path alone would over-terminate and silently drop a sibling override's replacements, while loading distinct lets a genuine collision surface via the duplicate-component check. A disallowed cycle surfaces as a dangling reference. A document pulled in by BOTH a plain `xs:include`/`xs:redefine` AND an `xs:override` is a FATAL conflict (`reportOverrideIncludeConflict`, enforced in `overrideLoadTarget` and symmetrically in `loadInclude`/`loadRedefine`) — not a silent no-op, since the override yields a distinct constituent whose components collide with the included originals. Per-document override-target sets in `processIncludes`/`overrideProcessNested` reject overriding the same document twice; duplicate override children of one `xs:override` are rejected. `xs:import` inside a target is handled by the SHARED `processImport` helper (NOT transformed) so it emits the same already-imported / load-failure warnings as a top-level import; override children may reference components imported only by the overriding schema. Override+redefine interplay is unexercised by the suite and processes redefine without cascade — a documented gap.
5. **Resolve references** — resolve all QName refs (types, base types, groups, attr groups, union members), detect circular substitution. **Circular model-group references** are detected and CUT by `checkCircularGroupRefs` (`link_refs.go`) BEFORE the group-ref resolution loop shares group content slices: a named `xs:group` that references itself (directly or transitively) would otherwise make the resolved content-model tree cyclic and stack-overflow the downstream Glushkov (UPA)/element-consistency/open-content walks. A DFS over the named-group reference graph (built from each group's UNRESOLVED definition) reports each back-edge as a `Circular reference to the model group definition '...' defined.` schema error and leaves the back-edge placeholder an empty model group so resolution stays acyclic. This is forbidden in BOTH XSD 1.0 and 1.1 (a model group must not contain itself), so it is reported in both versions (unlike circular ATTRIBUTE groups, which 1.1 permits). `check_upa.go`'s `positionAutomaton` also carries a defensive recursion-stack guard (`walking`, a `map[*ModelGroup]struct{}`) that terminates the Glushkov walk on any self-referential model group that ever slips through — never fired by a valid schema, since an acyclic model never has a group as its own ancestor. The substitution-group membership map itself is built by `buildSubstGroups` just BEFORE this step (global element `@substitutionGroup` affiliations are fixed at read/include time; XSD 1.1 list-valued attributes register the member under every listed head), so the UPA (`checkUPA`) check run WITHIN `resolveRefs` can expand a content model's substitution-group head leaves to their transitive members. `buildSubstGroups` only populates and sorts the direct-affiliation map and emits no diagnostics, so the circular/final/element-consistency checks keep their original error ordering. For an untyped member, `inheritedTypeFromFirstSubstitutionHead` follows only the FIRST QName in the actual `@substitutionGroup` value at each step to find the inherited declared type; the full list remains for membership and affiliation checks. After refs resolve, `checkSubstGroupAffiliations()` rejects any member whose effective declared type is not validly derived from each affiliated head's effective declared type. After attribute type refs resolve, `checkAttrUseConstraints()` validates each attribute use's `default`/`fixed` constraint value against the attribute's declared simple type, so a retained-but-invalid constraint (e.g. `default=""` on an `xs:integer` attribute) is reported as a schema parser error rather than injected into the instance at validation time. **au-props-correct.3** is enforced during attr-ref resolution (`checkAttrRefFixedConflict`): when an `<xs:attribute ref>` use carries its OWN value constraint AND the referenced global declaration has a `fixed`, the use's constraint must ALSO be `fixed` and value-equal (compared under the global's type via `fixedConstraintRestricts`) — a local `default`, or a `fixed` with a different value, is a schema error. This is enforced for EVERY referencing use, not only inside a restriction (the derivation-ok-restriction check covers only the derived-vs-base relationship), so a plain complexType with `<xs:attribute ref="t:a" default="2"/>` against a `fixed` t:a is rejected. **XSD 1.1 only**, `checkElementDeclConstraints()` (`link_refs.go`) does the same for ELEMENT declarations: an EXPLICIT `default`/`fixed` is validated against the element's EFFECTIVE declared type (`effectiveDeclType`, so a no-type substitution-group member is checked against its inherited head type; element-only/mixed complex content is skipped) through the SHARED `validateSimpleContentValue` — the same path the instance value uses, covering the effective content simple type (`effectiveContentSimpleType`) PLUS inherited simpleContent base-content facets (`validateNestedSimpleContentBases`) — on the version-/schema-aware `validateValue`, so an invalid list/union/builtin default, or one violating an inherited base-content facet, is a fatal schema error (W3C idIDREF s3_3_4si07/si08). Sources are recorded (`elemDeclConstraintSources`) only under Version11 and merged from import sub-compilers (a `ref=""` element inheriting the global's value is covered by the global's own entry), so XSD 1.0 stays byte-identical. Presence-based schema checks (`check_elements.go`) use `hasAttr`, and both `hasAttr`/`getAttr` require an **unqualified** attribute (`URI()==""`) so a foreign-namespaced `other:fixed` is not mistaken for the XSD `fixed`. When validation inserts an absent qualified attribute's default/fixed value, it is inserted **namespace-aware** (`SetAttributeNS`, reusing the in-scope prefix) so a later `xs:key` field like `@t:a` matches it. The insertion loop skips `Required` **and** `Prohibited` uses: a prohibited use must never materialize a default/fixed value (the absent attribute is accepted, and a present one is rejected), so it would otherwise mutate a valid document by inserting a forbidden attribute. The compile-time `default`-requires-`use="optional"` check (`checkAttributeUse`, `check_elements.go`) is applied to **ref** attribute declarations as well as named ones, so `<xs:attribute ref="t:a" use="prohibited" default="x"/>` is rejected at compile time (matching xmllint) rather than silently compiling. **Unbound QName prefix (src-resolve):** the shared QName-resolution helper `resolveQName` (`link_refs.go`), used for every QName-valued schema attribute (`@type`, `@ref`, `@base`, `@itemType`, `@substitutionGroup`, union `@memberTypes`), reports a fatal schema error via `reportUnboundQNamePrefix` when a **prefixed** ref's prefix is not bound in scope (`lookupNS` returns ""), instead of silently mapping it to the empty namespace. An **absent** prefix still maps to the in-scope default namespace (else the target namespace), and the predeclared `xml` prefix is never flagged (`lookupNS` returns the XML namespace for it).
   After refs resolve, `checkEnumQNameAndNotation()` (`xsd/check_facets.go`) runs two QName/NOTATION compile-time checks: (a) every `enumeration` literal of a QName/NOTATION-restricted type is resolved against its captured `FacetSet.EnumerationNS` bindings — an unbound prefix makes the literal an invalid QName and is reported as a schema error rather than silently compiling into an unsatisfiable enumeration. This is **variety-aware** (`enumLiteralHasUnboundQName`): an atomic literal is checked directly, a **list** literal item-by-item against the item type, and a **union** literal against whichever member type accepts it under its bindings (a literal that only a QName/NOTATION member could carry, with an unbound prefix, is flagged). (b) A simpleType whose base is directly `xs:NOTATION` with no `enumeration` facet is rejected. `checkNotationOnDeclarations()` extends (b) to **declarations**: an element or attribute whose effective type is the built-in `xs:NOTATION` (or NOTATION-derived) without an effective enumeration facet (`hasEffectiveEnumeration` walks the base chain) is rejected — this catches `type="xs:NOTATION"` placed directly on `<xs:element>`/`<xs:attribute>`, which bypasses the simpleType-level rule. Every attribute use records its source line in `attrUseSources` (merged from import sub-compilers) so the attribute case can report with the right location. (c) **XSD 1.1 only**: an `xs:NOTATION` atomic restriction's `enumeration` values must each name a notation declared in the schema. `parseSchemaChildren` collects every top-level `<xs:notation name="...">` into `compiler.notations` (keyed `QName{name, targetNamespace}`, merged from import sub-compilers); `checkEnumQNameAndNotation` resolves each enumeration value of a NOTATION-carrier type via `resolveLexicalQName`+`EnumerationNS` and reports a schema error when the resolved QName is not in `notations`. Gated on Version11 so XSD 1.0 stays byte-identical (the historical "declaration-table matching deferred" behavior). Full list/union-nested NOTATION declaration-table matching beyond the direct atomic case remains deferred. `checkAnyAtomicTypeUsage()` (`xsd/check_facets.go`, Version11) additionally rejects a user-defined simple type that names `xs:anyAtomicType` as its restriction base, list item type, or union member type (W3C bug 11103); it is the abstract base of all atomic types and must not appear in a user derivation (`xs:anyAtomicType` IS valid as an element/attribute/xsi:type type). `checkAnySimpleTypeUsage()` (`xsd/check_facets.go`, Version11) does the analogous check for the simple ur-type `xs:anySimpleType` (note in XML Schema Part 2 §2.4.1 / W3C bug 14559): it must not be RESTRICTED — as a simpleType restriction base, list item type, or union member type, nor as a simpleContent complexType RESTRICTION whose effective content simple type (`effectiveContentSimpleType`) is left as `xs:anySimpleType` (an empty or non-narrowing restriction of a base whose content type is the ur-type derives a content simple type that restricts the ur-type). It stays valid as an element/attribute/xsi:type type and as the base of a simpleContent EXTENSION (e.g. a substitution-group head). `checkFacetConsistency()` additionally runs `checkFacetValueAgainstBase()` (`xsd/check_facets.go`): each value-bearing range facet (`min`/`maxInclusive`, `min`/`maxExclusive`) is validated as an instance of the restricted base type's value space via `validateValue` with a silenced `validationContext`; a bound that is not a valid instance (e.g. `<xs:minInclusive value="abc"/>` on an `xs:int` base, or a numerically out-of-range bound) is a fatal schema error rather than silently falling through `compareForRangeFacet`'s "can't compare" path and turning the constraint into a no-op at validation time. The length-family and digit facet {value}s are lexically validated at PARSE time (`parseFacets`/`validateNumericFacetValue`, `xsd/read_types.go`): `length`/`minLength`/`maxLength`/`fractionDigits` must be a valid `xs:nonNegativeInteger` and `totalDigits` a valid `xs:positiveInteger` (whitespace-collapsed first, via `validateBuiltinValue`), so `<xs:maxLength value="1e2"/>`, a negative/non-numeric/empty value, or `totalDigits="0"` is a fatal schema error instead of being silently collapsed to `0` by `parseOccurs` (which would drop the constraint). This is an XSD 1.0 rule enforced in BOTH the default (1.0) and Version11 mode (goldens are byte-identical — no libxml2-compat golden uses an out-of-space length/digit facet value). `checkFacetConsistency()` likewise runs `checkEnumValueAgainstBase()` (`xsd/check_facets.go`): each `enumeration` value is validated against the base type's value space via `validateValue` with a silenced `validationContext` and its captured `FacetSet.EnumerationNS[i]` bindings; an invalid member (e.g. `<xs:enumeration value="+NaN"/>` on an `xs:float`/`xs:double` base — signed NaN is not in their lexical space) is a fatal schema error at compile time rather than an unsatisfiable enumeration that fails only at instance validation. This is **variety-aware** — atomic literals against the base value space, **list** literals item-by-item against the item type, **union** literals against whichever member type accepts them — matching `validateValue`. Suppression is **per literal, narrow**: only a literal that `enumLiteralHasUnboundQName` flags (a QName/NOTATION carrier, at any nesting depth, with an unbound prefix, which `checkEnumQNameAndNotation` already diagnoses) is skipped, to avoid a duplicate diagnostic. It is **not** a blanket skip of QName/NOTATION-carrying types: every other enumeration literal of such a type is still checked against the base value space, so e.g. a QName base restricted with `xs:length value="2"` still rejects an out-of-space `<xs:enumeration value="abc"/>`.
   **Attribute-group reference expansion** (`link_refs.go`): a named attribute group's effective {attribute uses} is the union over the group's own `<xs:attribute>` children (`schema.attrGroups[qn]`) and, transitively, every `<xs:attributeGroup ref>` child (`attrGroupRefChildren[qn]`). `parseNamedAttributeGroup` records nested refs and `expandAttrGroupUses` (cycle-guarded) flattens them into each referencing type so a required/defaulted attribute declared in a nested group is not dropped. Three grammar rules apply: (a) **`use="prohibited"` declared directly inside an `<xs:attributeGroup>` is pointless** — libxml2 warns (`Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.`) and SKIPS it, so it is never propagated as a blocking use and a referencing `xs:anyAttribute` wildcard still admits the attribute (`parseNamedAttributeGroup` / the redefine-override loop in `compile_imports.go` both warn-and-skip). (b) A **circular** reference is a schema error (src-attribute_group.3) outside `<xs:redefine>`. A DIRECT self-reference (`<xs:attributeGroup ref>` resolving to the group being defined) is caught at parse time (`reportCircularAttrGroupRef`) and dropped; an INDIRECT cycle (e.g. `h -> i -> h`, or the 3-node `a -> b -> c -> a`) is caught in `resolveRefs` by `checkCircularAttrGroupRefs`, a deterministic DFS over `attrGroupRefChildren` run BEFORE flattening that reports each back-edge (`reportCircularAttrGroupRefQName`: `Circular reference to the attribute group 'x' defined.`) and CUTS it so the flatten/expand walks terminate without a diagnostic-less truncation and without a duplicate-attribute false positive. The indirect-cycle diagnostic is attributed to the BACK-EDGE `<xs:attributeGroup ref>` element's own line/file (recorded PER edge in `attrGroupRefSources[qn]`, index-aligned with `attrGroupRefChildren[qn]`, populated in `parseNamedAttributeGroup` and the redefine-override loop and merged across imports), NOT the owning group's declaration line — matching the direct-self-reference path and pointing at the right file when the cycle spans included/redefined schemas. The legitimate self-reference inside `<xs:redefine>` is handled by the override path, not `parseNamedAttributeGroup`. (c) The duplicate-attribute-use detection (`flattenAttrGroupRefDuplicates`, ag-props-correct.2) uses `visited` as a **recursion stack** (add on entry, `defer delete` on exit), not a global "seen ever" set — so two SIBLING refs to the same group are each expanded and a name contributed via both (e.g. `g -> h, h` with `h` carrying `x`) surfaces as a duplicate, while true reference cycles are still cut. All these schema diagnostics route through `c.diagSource()` / a per-record `source` so an included/imported schema is cited correctly.
**Schema default attributes** (read/resolve phase, XSD 1.1 only): `xs:schema/@defaultAttributes` is read per schema document (including includes/redefines/imports with save/restore around nested documents) and resolved as a QName. Lexically invalid QNames, unbound prefixes, and deprecated XSD-datatypes namespace refs are reported but not recorded for the later unresolved-attribute-group pass, avoiding duplicate follow-on diagnostics. `xs:complexType/@defaultAttributesApply` defaults true in 1.1; when true and a schema default is present, the complex type records an implicit attribute-group reference to that default group, so the existing attribute-group expansion path handles required/default/fixed uses, duplicate detection, wildcards, and source diagnostics. Attribute uses contributed by the implicit default group are tracked separately (`defaultAttrUses`): extension duplicate checking suppresses only the case where the derived type re-applies the same schema default use component already inherited from its base; explicit attribute/attributeGroup redeclarations and same-name uses from different schema default groups still report duplicate attribute uses. **`xs:override` governance** (§4.2.5): an override replacement complex type is copied into the TRANSFORMED TARGET document, so the TARGET (overridden) document's `@defaultAttributes` — not the overriding document's — governs it. `overrideLoadTarget` resolves the target root's `@defaultAttributes` (via the side-effect-free `resolveSchemaDefaultAttributes`, shared with the normal read path's `readSchemaDefaultAttributes`), applies it as the active schema default for the duration of the target's parse (so surviving target complex types are governed correctly too — save/reset/restore alongside form/block/final defaults), and RETURNS that `schemaDefaultAttrsState`; the owner level (`loadOverride` / `overrideProcessNested`) reapplies it around `registerMatchedChildren` so the replacement records the implicit ref (or none) of the target document. Without this, an override target lacking its own `@defaultAttributes` would inherit the overriding schema's default group (W3C ibmMeta/defaultAttributesApply s3_4_2_4ii08). A replacement matched in a DEEPER nested include of the target is registered under the DIRECT target's state — a documented edge not exercised by the conformance suite. XSD 1.0 schemas ignore both attributes.

**Particle occurrence validation** (read phase, `read_particles.go`/`read_elements.go`/`check_elements.go`): every particle `minOccurs`/`maxOccurs` is validated as `xs:nonNegativeInteger` (max also allows `unbounded`) with `min<=max` and the prohibited-particle (`min=0 max=0`) carve-out. Lexical-error wording matches xmllint for `minOccurs` and `maxOccurs`. `validateOccursAttrs` covers compositor/wildcard/group-ref particles; `checkLocalElement` covers `xs:element`.

**xs:all occurrence rules are versioned.** XSD 1.0 enforces cos-all-limited: the all compositor `minOccurs` is 0 or 1, its `maxOccurs` is 1, and each direct element particle has `minOccurs`/`maxOccurs` in {0,1}. XSD 1.1 keeps the compositor rule but relaxes member particles: `checkAllElementParticleOccurs` skips the direct-element 0/1 rule, `xs:all` may contain wildcard members, and `matchAll11` validates flattened all members with per-member occurrence counting. Inline nested `<xs:all>` remains rejected. An `xs:group ref` resolving to an all group may be nested directly in an XSD 1.1 all only with 1/1 occurrence, then it is flattened; XSD 1.0 rejects that nested all ref. Extension placement still rejects appending an all group to a non-empty non-all base content model. Prohibited particles (`maxOccurs=0`) are not content for these placement checks. Occurrence diagnostics use declaring-file source attribution via `c.diagSource()`/recorded `groupRefSource`.

**Named model group definition content model** (`read_particles.go` `parseNamedGroup`, version-INDEPENDENT — enforced in both XSD 1.0 and 1.1 per §3.7.2): a `<xs:group name="...">` must have content `(annotation?, (all | choice | sequence))` — exactly one model group child, at most one annotation which must PRECEDE the model group, and no other element children. A missing model group, a second model group, a second/misplaced annotation, or any stray child (e.g. a direct `<xs:element>`/`<xs:complexType>`/`<xs:attribute>`/`<xs:group ref>`) is a schema error. The single model group is still recorded so references resolve; the redundant missing-model-group diagnostic is suppressed when a stray/misplaced child already produced a grammar error on the group.

**XSD 1.1 local element targetNamespace:** `parseLocalElement` uses explicit local `@targetNamespace` when `Version11`; otherwise it derives namespace from `@form`/`elementFormDefault`. Presence matters: `targetNamespace=""` means absent namespace. `checkLocalElement` rejects `@targetNamespace` on `@ref`, requires `@name` when `@targetNamespace` is present, rejects `@targetNamespace` with `@form`, and requires local `@targetNamespace` values that differ from the effective ancestor schema target namespace, or appear when the effective ancestor schema has no target namespace, to be inside a non-`xs:anyType` complex-content restriction before the nearest `xs:complexType`; chameleon includes/redefines inherit the including schema's target namespace for this check.

**Complex-type child ordering** (read phase, `read_types.go`, XSD 3.4.2): `parseComplexType`, `parseRestriction`, `parseExtension`, and `parseSimpleContentChildren` enforce the fixed child order as an ordered state machine — an OPTIONAL leading model-group particle (`sequence`/`choice`/`all`/`group` ref), at most one, THEN attribute/attributeGroup uses, THEN an OPTIONAL final `anyAttribute`. A model-group particle that appears AFTER any attribute/attributeGroup/anyAttribute is out of order and rejected (`The content model particle '…' must appear before the attribute declaration '…'.`) rather than silently overwriting the content model. A second model group (`more than one content model particle`) and mixing a particle with a `simpleContent`/`complexContent` wrapper are also rejected. The `anyAttribute` wildcard must be the optional FINAL child: an `attribute`/`attributeGroup` use appearing after it is rejected (`The attribute declaration '…' must appear before the attribute wildcard 'anyAttribute'.`), and a second wildcard is rejected (`A complex type definition must not have more than one attribute wildcard …`) via an `anyAttributeSeen` flag tracked in each of the four parse paths, rather than silently overwriting `td.AnyAttribute`. **simpleContent extension prohibited attrs:** `parseSimpleContentChildren` takes the derivation kind; on an EXTENSION a `use="prohibited"` attribute is pointless and is warned+skipped (`Skipping attribute use prohibition, since it is pointless when extending a type.`) exactly like `parseExtension` (complexContent), so it does not propagate as a blocking use and a base attribute wildcard still admits the attribute; on a RESTRICTION the prohibition is kept. **Unresolved type/element ref source attribution:** `reportUnresolvedTypeRef` reports via `c.diagSourceOrRecorded(typeDefSource.source)` and the owner type's actual element kind (`typeDefSource.elemKind`, recorded at parse time — `complexType` vs `simpleType`), not a hard-coded `c.filename`/`"simpleType"`; `elemRefSource` likewise carries the declaring `source` so an unresolved element/ref in an included/imported schema cites the declaring file.

6. **Constraint checks** (when errorCount == 0):
   - `checkFinalOnTypes()` — final attribute enforcement. In **Version11**, `finalDefault="extension"` also reaches a simple type's `{final}` (the simpleType final-default mask in `read_types.go` includes `FinalExtension`; spec bug 2074), so a simpleContent extension of an extension-final simple type is rejected. XSD 1.0 masks `extension` out of a simple type's `{final}`, byte-identical.
   - `checkFinalOnSubstGroups()` — substitution group final
   - `checkUPA()` (`check_upa.go`) — Unique Particle Attribution / cos-nonambig (content model determinism). Builds a **position automaton** (Glushkov construction) over the content-model particle tree: each element/wildcard leaf particle becomes a numbered position, and the walk yields nullable/firstpos/lastpos/followpos. Occurrence (`minOccurs`/`maxOccurs`) is folded in by `applyOccurs` **once per particle** — a particle wrapping a model group is NOT re-counted, since the parser stores the same range on both the particle and the group (`walkParticle` defers a group term to `walkModelGroup`, which applies the group's own range). `applyOccurs` distinguishes two cases: a **non-nullable body** expands its required copies (`minOccurs`) into a strict chain of distinct positions and attaches at most ONE optional remainder copy (looping only if it can repeat) — this keeps counted models like `a{2}, a` deterministic, and the single optional copy avoids falsely flagging interchangeable repeats like `<any maxOccurs="5"/>`; a **nullable body** collapses to a single loop copy (expanding it is unsound because a skipped occurrence would make later copies compete with earlier ones, e.g. `(a?, b?){1,3}`). The Glushkov sequence concatenation keys lastpos on the CURRENT particle's nullability (`last(rs) = last(s) ∪ last(r)` only when `s` is nullable), so an optional prefix before a required element does not linger in lastpos (`(a?, b), b` stays deterministic). Substitution-group members of an element term each get their own position (any member can match where the head is expected) — so a choice/sequence that lets a member match both via a `ref="head"` leaf and an explicit `ref="member"` leaf is correctly flagged. This head→member expansion depends on `schema.substGroups` being populated by `buildSubstGroups` BEFORE `resolveRefs` (and thus before this check, which runs within `resolveRefs`). A model is ambiguous if, from any single state — the start state firstpos(root), or followpos(p) for any position p — two distinct positions reachable on the same element name (or via overlapping wildcards, by `entriesOverlap`/`wildcardsOverlap`) compete. This catches non-adjacent ambiguity such as `a?, b?, a` (skipping the optional first `a` re-introduces it as a competitor of the final `a`). `xs:all` is walked as an order-independent group by `walkAllBody` (NOT as a sequence): every member's firstpos competes from the start state, and after any member is consumed every OTHER member is still reachable (mutual-reachability followpos — each member's lastpos may be followed by every other member's firstpos). This faithfully models all's order-independence and is what catches a duplicate same-name member (two members with the same element name overlap in the union of firstpos). The wildcard-vs-wildcard overlap test (`wildcardsOverlap`) is namespace-constraint-aware: two wildcards overlap (and so are a UPA conflict) ONLY if their namespace sets INTERSECT. A finite-set vs finite-set pair overlaps iff they share a member; two negation constraints (`##any`/`##other`/`##not-absent`) always intersect; a negation vs a finite set reuses `wildcardMatchesNS` per member (so `##other`, which here = not(targetNS) and not(absent), is DISJOINT from `##targetNamespace` and from `##local`). A disjoint pair such as `##other?, ##targetNamespace` is therefore accepted, not falsely rejected. The diagnostic message is `The content model is not determinist.`
   - `checkElementConsistent()` (`check_element_consistent.go`) — cos-element-consistent (Element Declarations Consistent): two element declarations with the same expanded name reachable in one effective content model (after group-ref expansion and across nested model groups) must have the same {type definition}. Runs in `compileSchema` AFTER substitution groups are built (NOT inside `resolveRefs`), gated on `errorCount==0` — it consults `schema.substGroups`, so it must run once that map exists. Coverage: (a) complex-type content models (iterated in source line/ordinal order); (b) for each element TERM, the term's substitution-group MEMBERS (`schema.substGroups[term.Name]`) are folded in as implicitly-containable declarations under each member's own name (a head's particle stands in for its members), so a `ref="head"` colliding by name with a different-typed same-named local element is rejected — untyped members' declared types resolve through the first substitution-group head via `resolveDeclaredType`; (c) standalone named `xs:group` definitions (over `schema.groups`, in stable source order via `groupSources` recorded at parse time and merged from import sub-compilers), so a named group no complex type references is still checked. Type identity is by `*TypeDef` pointer (helium shares one pointer per named type and copies the global's type onto a `ref`), with a same-expanded-QName fallback for import-merged duplicates; distinct anonymous inline types are different components and therefore inconsistent. The check is only ever under-strict (a missed violation is safe; it never false-rejects a valid schema). libxml2 does NOT implement this constraint (it is an "URGENT TODO" in `xmlschemas.c`), so the diagnostic uses the existing component-error style rather than mirroring libxml2 wording, and no golden schema trips it.
   - Wildcard overlap detection
   - **Restriction derivation checks** (`link_refs.go`, run per `DerivationRestriction` complex type in source-line order): `checkRestrictionAttrs` enforces derivation-ok-restriction for ATTRIBUTE uses (optional cannot restrict required, every derived attribute must have a matching base use/wildcard, every required base attribute must be matched, and the attribute wildcard NSSubset/processContents rules). For a redeclared same-QName attribute it additionally enforces derivation-ok-restriction.2.1.2/2.1.3: the derived attribute's type must be the same as, or derived by restriction from, the base attribute's type (`simpleTypeValidlyRestricts` — the `*TypeDef` pointer chain via `isDerivedFrom`, with a builtin-hierarchy fallback `builtinDerivesFrom`/`builtinRestrictionParent` for the builtin-to-builtin narrowing case like xs:integer→xs:int that the pointer chain cannot see because builtins carry no `BaseType` links). An ABSENT attribute type is `xs:anySimpleType` (XSD §3.2.2.1, via `attrUseEffectiveTypeDef`), so an UNTYPED derived attribute restricting a narrower base (e.g. xs:int) is the simple ur-type WIDENING the base and is REJECTED. The builtin fallback applies ONLY when the BASE is an ACTUAL XSD builtin (`base.Name.NS == NamespaceXSD`): walking the DERIVED side to its builtin ancestor is sound, but treating a user simple type that RESTRICTS a builtin (e.g. xs:int with `maxInclusive="10"`) as that builtin would WIDEN the base and wrongly accept a derived type that drops the user-added facets — so a user-restricted base must be derived from through the pointer chain. Two exceptions are handled BEFORE the builtin-base early return, because `builtinBaseLocal` is empty for a union and for a directly-declared list, so the early return would otherwise accept ANY derived type unconditionally (a false-accept). (a) A user-defined UNION base (`resolveVariety(base)==Union`): per cos-st-derived-ok (§3.14.6, the XSD 1.0 Type Derivation OK Simple rule) a type validly derived from (at least) ONE of the union's (transitively-walked) member types is a valid derivation. So a base `@a` of `IntOrString` (memberTypes="xs:int xs:string") restricted to derived `@a` xs:int is ACCEPTED, while xs:date (derived from NEITHER member) is REJECTED. This holds REGARDLESS of facets the base union carries: XSD 1.0 has NO "facets empty" condition on a union base, so a FACETED union base (e.g. `IntOrString` further restricted by an `enumeration`) — and likewise a no-facet restriction ALIAS over a faceted union — STILL admits a derived member type and is ACCEPTED. (The "facets empty" condition is XSD 1.1-only, §3.16.6.3 Type Derivation OK Simple; this package targets XSD 1.0 / libxml2 parity, so it is intentionally NOT enforced.) Intervening MEMBER unions are walked through `simpleTypeValidlyRestricts`' recursion. (b) A LIST base: a type that did NOT pass the pointer chain (`isDerivedFrom`, which a real `<xs:restriction base="theList">` satisfies via the `BaseType` pointer) is NOT a valid restriction of the list and is REJECTED. This covers (i) a user-defined list base (`resolveVariety(base)==List`) — both an unrelated list (e.g. base `IntList`=list(xs:int) redeclared as `StringList`=list(xs:string)) AND xs:anySimpleType (the simple ur-type is a SUPERTYPE; restricting a list "down to" it would WIDEN to accept non-list values, so it is NOT a valid restriction) are rejected — and (ii) the builtin LIST types (xs:IDREFS/xs:ENTITIES/xs:NMTOKENS), which are registered as bare atomic-variety names with no list marker so `resolveVariety` reports Atomic and clause (i) does not catch them: `base.Name` ∈ {IDREFS,ENTITIES,NMTOKENS} (in the XSD namespace) is rejected whenever the pointer chain failed, so an xs:IDREFS base "restricted" by an unrelated user list (xs:list itemType="xs:string") — which reaches the `db==""` shortcut and would otherwise be accepted — or by an atomic xs:string is REJECTED. Both branches run BEFORE the builtin db/bb fallback (`builtinBaseLocal` is empty for a directly-declared list, so the early db/bb return would otherwise accept ANY derived type unconditionally). `builtinRestrictionParent` covers EVERY atomic builtin primitive (string/decimal families plus boolean, float, double, the date/time/g* family, duration, hexBinary, base64Binary, anyURI, QName, NOTATION — all rooted at anySimpleType), so a cross-family pair (e.g. base xs:int restricted by derived xs:boolean) is decided and REJECTED rather than treated as "unknown" and accepted. `builtinDerivesFrom` (via `builtinListItem`) still DECIDES a builtin list on the DERIVED side (a list type is the same only as itself and validly derives only from xs:anySimpleType per cos-st-derived-ok.2.2.3, otherwise UNRELATED to every atomic type and to the other two list types); the base-side builtin-list case is short-circuited to REJECT in `simpleTypeValidlyRestricts` (above) to close the `db==""` user-list false-accept. (c) A CONSTRUCTED derived list or union (`resolveVariety(derived)` is List/Union) that reaches the end having FAILED both the pointer chain (`isDerivedFrom`) and the union-member shortcut can only be validly derived from `xs:anySimpleType` (the simple ur-type) or through a real base-type chain, so it is ACCEPTED only when the base IS the actual `xs:anySimpleType` and otherwise REJECTED — closing the `db==""` "unknown ⇒ valid" fallback that would wrongly accept an ATOMIC base (e.g. xs:string) redeclared as a user `xs:union` or `xs:list`. A base `fixed` value constraint forces the derived attribute to carry a value-space-equal `fixed` value (`fixedConstraintRestricts`): each lexical is compared under ITS OWN simple type (a same-type fast path via `fixedValueMatches`, else the cross-type `crossMemberValueEqual`), so base `fixed="1"` accepts derived `fixed="01"` for xs:int, base xs:decimal `fixed="1.0"` accepts a narrowing derived xs:int `fixed="1"`, but a dropped or `fixed="2"` constraint is rejected. It is CONSERVATIVE — it accepts whenever derivation cannot be decided (unresolved types, or a builtin pair the tables do not cover), so it never false-rejects a legitimate restriction; base `@a` xs:int restricted to derived `@a` xs:string is rejected. Every `checkRestrictionAttrs` diagnostic is attributed via `source := c.diagSourceOrRecorded(src.source)` (paired with the type's recorded `src.line`) so an invalid attribute restriction in an included/imported schema cites the DECLARING file, not the top-level `c.filename`; the processContents 4.3 case, which deliberately reports at the BASE type's line/component, likewise uses `c.diagSourceOrRecorded(baseSrc.source)` so its filename follows that base line. `checkRestrictionParticles` (`restriction_particle.go`) complements it with a CONSERVATIVE subset of derivation-ok-restriction for the CONTENT MODEL (§3.9.6 Particle Valid (Restriction)): the derived type's effective content model must be a valid restriction of the base's, via `particleValidRestriction` — element→element NameAndTypeOK (same expanded name, occurrence subset, type `isDerivedFrom` base type — **in Version11, when the base element's type is a UNION the derived type may instead be validly derived from one of the union's (transitive) members via `isXsiTypeDerivedFromDeclared`, union member substitutability**, no nillable-widening, and a base `fixed` value forces the derived element to carry a VALUE-SPACE-equal fixed value — compared via `fixedValueMatches` in the element's simple-type value space, NOT raw lexical string equality, so base `fixed="1"` accepts derived `fixed="01"` for xs:integer while `fixed="2"` is rejected), occurrence-range subset (`occurrenceValidRestriction`: rMin>=bMin, rMax<=bMax with -1 = unbounded), order-preserving Recurse for sequence→sequence and choice→choice (`recurseOrdered`: each base particle either restricts the next derived particle or must be emptiable; every derived particle must be consumed), all→all distinct-mapping Recurse (`recurseAll`), wildcard NSSubset (any→any) and element→wildcard NSCompat (`wildcardAllowsName`), **Recurse-As-If-Group** (derived ELEMENT vs base MODEL GROUP, `elementRestrictsGroup`: the element is mapped as a singleton group through the base group's children — for sequence/all it must restrict exactly one base child with every OTHER base child emptiable, for choice it must restrict SOME alternative; occurrence subset still required), **NSRecurseCheckCardinality** (derived MODEL GROUP vs base WILDCARD, `groupRestrictsWildcard`/`groupLeavesWithinWildcard`: the group particle's range must be within the base wildcard particle's range, and every element/wildcard LEAF inside the derived group — with its effective occurrence folded through the enclosing group ranges via `mulOccurs` — must be admitted by the base wildcard's namespace constraint and within its cardinality; a nested wildcard leaf must also be a namespace subset with at-least-as-strong processContents), and emptiable handling (`particleEmptiable`). **Prohibited particles (`maxOccurs=0`) emit nothing:** `particleEmitsNothing` (max element-emission == 0 via `particleElementRange`) treats a prohibited leaf/group as contributing (0,0) — such a derived particle is dropped before the order-preserving mapping (`nonEmittingFiltered` in `recurseOrdered`) and skipped in `recurseAll`/`recurseChoiceUnordered`/the map-and-sum loop/`groupLeavesRestrictElement`/`groupLeavesWithinWildcard` (it needs no base counterpart and admits no content), and a prohibited BASE particle never requires a derived counterpart — so a derived restriction that omits or prohibits a maxOccurs=0 leaf is not false-rejected. It catches the CLEAR violations — reordered, added, or renamed particles, widened occurrence, an element that drops a required base group child or matches no base alternative, and a group leaf outside the base wildcard's namespace/cardinality — and emits `The content model is not a valid restriction of the content model of the base complex type definition '{ns}name'.`. Map-and-sum (derived SEQUENCE restricting a base CHOICE, in `groupRestrictsGroup`) compares the derived sequence's TOTAL element-emission range (`particleElementRange`, folding each group's own occurrence range times its children's emission and recursing through nesting) against the base choice particle's range, then requires every emitting derived member to restrict SOME base branch; group-vs-element (`groupRestrictsElement`/`groupLeavesRestrictElement`) requires the derived group be a pointless wrapper emitting exactly one element that restricts the base element. The mixed-compositor group:group pairs with NO §3.9.6 derivation rule — derived choice:sequence, choice:all, all:sequence, all:choice — are REJECTED in `groupRestrictsGroup`'s default branch, but only after `reduceSingletonGroup` first folds away any "pointless" single-emitting-child wrapper on either side and re-dispatches (so e.g. choice(a) restricting sequence(a) reduces to element→element and stays valid); derived sequence:all is **RecurseUnordered** (`recurseAll`, after a group-occurrence subset check): each derived sequence particle must map to a DISTINCT base all particle it validly restricts (order is irrelevant in the base all) and every unmatched base particle must be emptiable, so a derived sequence that adds/renames a particle or drops a required base all member is rejected. By design it NEVER false-rejects: any sub-case the recursion cannot decide with confidence (unresolved types, restriction off xs:anyType / empty / simple base) returns "valid". No golden schema trips it. The restriction diagnostic is attributed via `c.diagSourceOrRecorded(src.source)` so an invalid restriction in an included/imported schema cites the declaring file. **Deferred sub-cases:** substitution-group containment is accepted unconditionally rather than fully verified.

7. **Compile result gate:** after linking/checks, `compileSchema` returns `(nil, ErrCompilationFailed)` if `c.errorCount > 0` (fatal diagnostics already delivered to the `ErrorHandler`), otherwise `(schema, nil)`. Sub-compiler `errorCount` is merged into the parent (`compile_imports.go`), so an import/include/redefine fatal also fails the top-level `Compile`. `xslt3` schema-awareness (`compile_schema.go`) maps `ErrCompilationFailed` to `XTSE0220`.

### Validate: Document + Schema → Errors

**Three-pass validation** (pass 3 runs only in XSD 1.1 mode):

**Pass 1 — Content Model** (`validateDocument` via `helium.Walk()`):
- For each element:
  1. Match against global element declaration
  2. Resolve `xsi:type` against block flags
  3. Check abstract type constraint
  4. Handle `xsi:nil`
  5. Validate attributes against type's AttrUses
  6. Validate content by ContentType:
     - Empty: no child elements
     - Simple: no child elements, validate text vs type facets
     - Element-only/Mixed: match children against ModelGroup (`matchSequence()`/`matchChoice()`)

Fixed value constraints (element content and attribute values) are compared in
the declared simple type's value space via `fixedValueMatches`. Both the fixed
and instance values are first whitespace-normalized using the type's *effective*
whiteSpace facet (`resolveWhiteSpace` walks the derivation chain, so a facet
derived on a restriction — e.g. `xs:string` restricted with
`whiteSpace="collapse"` — is honoured, not just the builtin default). Then the
comparison dispatches on variety (`resolveVariety`): list types split into items
and recurse each item through the variety-aware comparator on the actual item
type, so an `xs:integer` list fixed `1 2` accepts `01 +2` **and** a list whose
item type is a union dispatches each item to the union's member value spaces;
union types accept when any member's value space matches; atomic types compare
via `value.Compare` for the value-comparable builtins in `enumValueSpaceTypes`
(numeric, boolean, date/time, and binary — so `xs:hexBinary` fixed `0A` accepts
`0a` and integer fixed `1` accepts `+1`/`01`), falling back to normalized-lexical
equality for string-family/anyURI (so a numeric-looking `xs:string` fixed `5`
does not accept `5.0`). `xs:QName`/`xs:NOTATION` fixed values are resolved in
namespace context: each lexical QName is resolved against its own in-scope
namespaces — the instance side via `collectNSContext(elem)`, the schema fixed
side via the `FixedNS` map captured on the `ElementDecl`/`AttrUse` at read time
(`collectNSContext` over the declaring schema element) — and the resolved
`{namespace URI, local name}` pairs are compared, so two different prefixes bound
to the same URI are equal while a same-prefix different-URI binding is not. An
unresolved prefix on either side is a *rejection*, not a lexical fallback (a
QName/NOTATION whose prefix cannot be resolved is itself invalid, so the fixed
comparison must not pass on raw lexical match). `fixedValueMatches`
takes the instance and fixed namespace contexts as parameters. When `td` is nil it
falls back to raw string equality. The element fixed-value comparison uses the
element *declaration's* type (`edecl.Type`), not an `xsi:type` actual type, so a
declared `xs:string` (whiteSpace="preserve") fixed `abc ` keeps its trailing space
even when the instance's `xsi:type` collapses whitespace — element content is still
validated against the actual type. In XSD **1.1** `fixedValueMatches` itself narrows
its comparison type via `effectiveContentSimpleType` (gated on `version == Version11`,
right after the nil-`td` guard), so for a simpleContent complex type the fixed value
is compared in its NARROWED content simple type (`ContentSimpleType`) — e.g. a content
type restricted to `xs:QName` accepts a different-prefix/same-URI value by QName
value-space equality instead of a lexical comparison against the outer complex type's
own base chain. This is CENTRALIZED in `fixedValueMatches` so every caller is
consistent: the runtime non-empty element fixed check, the attribute fixed checks,
and the COMPILE-TIME content-model restriction check (`restriction_particle.go`
NameAndTypeOK, where a base/derived element `fixed` must be value-space equal).
`effectiveContentSimpleType` returns a non-simpleContent type unchanged, so the
simple (attribute / list-item / union-member) callers are unaffected; XSD 1.0 keeps
the historical declared-type comparison, byte-identical. In `fixedUnionMatches`, when the fixed and
instance values resolve to *different* active members, the cross-member
comparison (`crossMemberValueEqual`) is **recursive over variety**: when both
active members are **lists** (e.g. `memberTypes="intList decimalList"`) each value
is split and compared item-by-item in the item types' shared value space — so the
literal `1.0 2.0` (active in `decimalList`) accepts the instance `1 2` (active in
`intList`); a list-vs-atomic variety mismatch has no shared value space and stays
unequal. When both active members are **atomic** they are value-equal iff
their members reduce to the same *primitive* value-space family
(`primitiveValueSpaceFamily`, XSD 1.1 §2.3 — restrictions create no new values):
all integer types → `decimal`; all xs:string-derived types
(string/normalizedString/token/language/Name/NCName/NMTOKEN/IDREF/ENTITY/…) and
anyURI → `string`; each remaining comparable primitive (boolean, float, double,
date/time family, hexBinary, base64Binary) is its own family; QName/NOTATION have
no shared family (namespace-context dependent). Each operand is first
whitespace-normalized with *its* active member's effective whiteSpace facet; the
`decimal`/comparable families then compare via `value.Compare` (so union fixed
`1.0` accepts both `1` and ` 1 `), while the `string` family compares the
normalized lexical forms (so fixed `a b` active in one xs:string restriction
accepts instance ` a   b ` active in another xs:string restriction with
whiteSpace="collapse", both denoting `a b`). This includes string-derived members
— it is **not** gated on the `enumValueSpaceTypes` allowlist. `unionActiveMember` returns the active *basic*
(atomic) member, descending through nested unions to the basic member that
actually accepts the value, so an outer union `memberTypes="inner xs:decimal"`
(with `inner` a union `xs:integer xs:boolean`) compares instance `1` (active
basic member xs:integer) against fixed `1.0` (xs:decimal) in the shared decimal
value space rather than rejecting. Global attributes matched through an `xs:anyAttribute`
wildcard (`validateWildcardAttr`, processContents strict/lax) also enforce the
global attribute's `Fixed`/`FixedNS` via `fixedValueMatches`.

Enumeration facets are compared in value space, not raw lexical text. Each
enumeration *literal* is first whitespace-normalized with the constrained type's
effective whiteSpace facet (`checkFacets` takes the `whiteSpace` mode resolved by
`resolveWhiteSpace(td)` in `validateFacets`), mirroring the normalization the
instance value already underwent — so an `xs:token` enumeration `"a  b"` (two
spaces) collapses to `a b` and matches the instance `a b`. A value is a member if
it lexically equals a normalized member OR value-compares equal to one (e.g.
decimal `5.0`≡`5`, boolean `1`≡`true`, float `1.50`≡`1.5`, equal dateTimes in
different timezones). For float/double, NaN equals NaN for enumeration (but
remains incomparable for min/max ordering). QName/NOTATION enumeration resolves
both instance and facet lexical QNames against their respective in-scope
namespaces; the facet literal is whitespace-normalized before its prefix is
resolved (both at validation time and in the compile-time `checkEnumQNameAndNotation`
prefix-binding check), so a literal like `" p:a "` is not falsely rejected as an
invalid QName. Value-space comparison is restricted to an allowlist of numeric,
boolean, date/time, and binary builtins (`enumValueSpaceTypes`); hexBinary and
base64Binary compare by decoded octets (so `"0A"`≡`"0a"`). String-family and
anyURI types stay lexical-only (their value space equals their whitespace-
processed lexical space), so a numeric-looking string enum `"5"` does not accept
`"5.0"`. **List** enumeration (`checkListEnumeration`) splits both the instance and
each enumeration member into items and compares item-by-item in the item type's
value space via `fixedListMatches` (so `xs:list itemType="xs:int"` enum `"1 2"`
accepts `"01 +2"`; QName item types resolve instance items against the instance's
namespaces and each member's items against its captured `FacetSet.EnumerationNS`).
**Union** enumeration (`checkUnionEnumeration`) resolves the active member
INDEPENDENTLY for the instance value and for each enumeration literal, then
compares with the same ordered-union value-family logic fixed-value comparison
uses (`fixedUnionMatches`), recursing through list/nested-union member value
spaces — so a literal active in a string member is not value-equal to an instance
active in a numeric member (`memberTypes="zeroString xs:int"` enum `"0"` rejects
`"+0"`). A **union** restriction may carry ONLY `pattern` and `enumeration`
facets: per XSD §4.1.5 the range facets (`min`/`maxInclusive`,
`min`/`maxExclusive`), the digit facets (`totalDigits`/`fractionDigits`), the
length family (`length`/`minLength`/`maxLength`), `whiteSpace`, and
`explicitTimezone` are NOT in a union's {applicable facets} set, so
`checkFacetApplicability` rejects them at COMPILE time (`The facet '…' is not
allowed.`) — they never reach validation as a runtime no-op. The union's allowed
`pattern`/`enumeration` facets are checked in
the instance active member's value space via `checkFacets` with enumeration
suppressed. The active member for that `checkFacets` call is resolved down to its
LEAF basic member (`fixedUnionActiveMember` descends through nested unions), so a
nested union (`outer=union(inner)`, `inner=union(xs:string)`) resolves to the
leaf type rather than an intermediate union. On an ATOMIC restriction the range facets
(`min`/`maxInclusive`, `min`/`maxExclusive`) apply ONLY to types whose primitive
value space is ORDERED, so `compareForRangeFacet` first gates `builtinLocal` on
`value.Orderable` (the shared `orderedRangeFacetTypes` allowlist in
`internal/xsd/value`, also used by relaxng) — the numeric leaves (decimal and derived
integers, float, double) AND the date/time/duration family (duration,
dayTimeDuration, yearMonthDuration, dateTime, dateTimeStamp, time, date, and the
gregorian g-types). For every NON-ordered leaf — string-family
and anyURI, boolean, the binary types (hexBinary/base64Binary), QName/NOTATION,
and any non-atomic list/union carrier (empty/unknown local) — it returns
`ok=false`, leaving the range facet INAPPLICABLE rather than forcing a comparison.
The gate matters even though `value.Compare` returns a deterministic order for
boolean and the binary types (that order exists only so enumeration can use
`cmp==0`; it is NOT the XSD value-space order and must never fire a range facet).
For the ordered leaves the actual ordering is deferred to `value.Compare`. So a
numeric-looking string/boolean like `5` under `minInclusive` on a string, boolean,
binary, or list/union leaf is no longer wrongly rejected, and there is no
empty-`builtinLocal` decimal fallback. `checkAtomicFacetApplicability` also rejects
the length family (`length`/`minLength`/`maxLength`) on any atomic primitive
OUTSIDE `lengthApplicableTypes` (string-derived, the binary types, anyURI, QName,
NOTATION); on a numeric/decimal, boolean, float/double or date/time/duration atomic
the length facets are inapplicable and are reported at COMPILE time (`The facet '…'
is not allowed on types derived from the type xs:…`), so e.g. `xs:int`+`length` is a
schema error rather than a runtime no-op. `explicitTimezone` is accepted only on the
date/time family (dateTime/dateTimeStamp/date/time and the gregorian g-types); it is
stored on `FacetSet.ExplicitTimezone` with `FacetSet.ExplicitTimezoneFixed`,
checked at validation time for required or prohibited timezone presence, and
participates in compile-time restriction checks so `xs:dateTimeStamp` can only
retain `required` and inherited user `fixed="true"` values cannot be changed.
`parseFacets` rejects duplicate singleton facets in a single restriction step
(range, digit, length, whiteSpace, and explicitTimezone facets); only
`enumeration`, `pattern`, and the XSD 1.1 `assertion` facet are repeatable there.
Built-in temporal whiteSpace is treated as fixed `collapse`.
`checkFacetSameTypeConsistency` gates EACH
facet-family consistency check to the family's applicable type/variety, so it never
adds a spurious error on top of an applicability rejection: the LENGTH check
(`minLength>maxLength`) runs only on a list variety or a `lengthApplicableTypes`
atomic; the DIGIT check (`fractionDigits>totalDigits`) runs only on a
`value.IsDecimalFamily` atomic (so `xs:double`+`totalDigits`/`fractionDigits` reports only
the two "not allowed" errors); the RANGE checks run only on an ordered atomic. It
compares same-type range bounds (`min`/`maxInclusive`, `min`/`maxExclusive`) in the
restricted type's ORDERED VALUE SPACE (`value.CompareFloatFacetBound` for float/double
NaN ordering, else `compareForRangeFacet`), skipping the check on an indeterminate
result, so an inconsistent non-decimal pair like `minInclusive 2021-01-01 >
maxInclusive 2020-01-01` on `xs:date` is rejected. `checkFacetBaseRestriction` compares
each derived range bound against the EFFECTIVE inherited lower/upper bounds with
the SAME value-space comparator (gated to ordered atomic; `compareDecimal` only
for an unresolved primitive), so a valid narrowing non-decimal restriction — e.g.
base `xs:date` `minInclusive=2021-01-01` with derived
`maxInclusive=2022-01-01` — is no longer false-rejected, while an empty derived
range such as `minExclusive` equal to the effective inherited `maxInclusive` is
rejected. Range facet
`fixed="true"` is tracked per bound (`FacetSet.*Fixed`) and prevents a derived type
from changing that bound except by value-space equality. The facet value-against-base
check also allows a derived exclusive bound equal to the base exclusive bound (for
example the same effective `maxExclusive` on `xs:dateTimeStamp`), because that is
a valid restriction even though the exclusive boundary value is not itself an
instance of the base type; the shortcut is only used when that exclusive bound is
still the effective inherited bound for that side after considering any tighter
intermediate inclusive/exclusive bounds.

Pattern facets are stored per restriction step as `FacetSet.Patterns []string`,
compiled once into `FacetSet.compiledPatterns` (`[]*xsdregex.Regexp`) at schema
compile time via `xsdregex.Compile`. Patterns in the same step are ORed (value
valid if it matches any); patterns from different derivation steps are ANDed,
enforced by `validateFacets` walking the base-type chain and validating each
step's `FacetSet` independently. `xsdregex.Compile` translates XSD regex to Go's
RE2 (the same translator `xpath3` uses for `fn:matches`), so XSD-only constructs
(`\i`, `\c`, `\p{Is...}` blocks) are enforced rather than silently skipped;
patterns using XML Schema character-class subtraction (`[a-z-[aeiou]]`) or
quantifier bounds beyond RE2's limit are compiled with the regexp2 backtracking
engine, which RE2 cannot. A pattern that is not a valid XSD regular expression is
reported as a schema parser error (`The value '…' is not a valid regular
expression.`); its `compiledPatterns` entry stays nil and is skipped at validation.

The XSD xs:pattern grammar (XML Schema Part 2 Appendix F) is stricter than the
shared XPath/XQuery regex flavor, so `xsdregex.CompileVersion` (the XSD compiler
entry point, with the 1.1 toggle) and its `xsdregex.Compile` wrapper (used by
relaxng) — NOT the `Translate`/`Validate` used by `xpath3` —
run an extra `rejectNonXSDConstructs` pass that rejects, in BOTH 1.0 and 1.1
mode, three construct classes valid in XPath but forbidden in XSD: reluctant
(non-greedy) quantifiers (`a*?`, `b{1,3}?`), `(?...)` group extensions
(non-capturing `(?:…)` and inline flags), and unbalanced parentheses
(`)(`, `(abc`, `abc)`). The check is scoped to the Compile path because the
XPath flavor `xpath3` shares (`fn:matches`/`tokenize`/`replace`) legitimately
permits reluctant quantifiers and `(?:…)`. Stray-hyphen character-class ranges
(`[^a-d-b-c]`) are a known remaining false-accept, deferred.

**Compile-time IDC checks:** a malformed `xs:selector`/`xs:field` `@xpath` is a fatal schema parser error (`parseIDConstraint` → `reportIDCXPathError`) rather than a silently-dropped `xpath1.Compile` failure that would disable the whole constraint. A structurally malformed IDC declaration is likewise fatal (`parseIDConstraint` → `reportIDCStructureError`, src-identity-constraint): a missing required `@name` (`The attribute 'name' is required but missing.`, drops the constraint), a missing `<selector>` child (`A child element is missing.`, short-circuits the field check like libxml2), or a `<selector>` present with zero `<field>` children (`The content is not valid. Expected is (annotation?, (selector, field+)).`). The `(annotation?, (selector, field+))` content model is enforced as an ordered scan (ORDER + CARDINALITY): an optional leading `<annotation>` (at most once, first), then EXACTLY ONE `<selector>`, then ONE-OR-MORE `<field>`, nothing else — a `<field>` before the `<selector>`, a second `<selector>`, a misplaced `<annotation>`, or any unexpected child — XSD-namespaced OR foreign-namespaced — all raise `The content is not valid. Expected is (annotation?, (selector, field+)).` (the content model has no element wildcard, so a foreign direct child is rejected, matching libxml2; extension content belongs inside `xs:annotation`/`xs:appinfo`). Both `<selector>` and `<field>` REQUIRE an unqualified non-empty `@xpath` (`idcXPathAttr`, `hasAttr`-checked): an absent or empty `@xpath` (which would leave the selector/field `""` and silently disable the constraint) is fatal with `Element '{…}selector'`/`'{…}field': The attribute 'xpath' is required but missing.` (distinct from the missing-child diagnostics), and an empty field is then skipped by the XPath-compile loop so no redundant "not a valid field expression" follows. `reportIDCStructureError`/`reportIDCXPathError`/`idcXPathAttr` all attribute via `c.diagSource()` (not `c.filename`), so a malformed IDC in an INCLUDED/REDEFINED schema is cited under the declaring file, matching its line. After all elements are parsed, `checkKeyRefRefers` (in `compileSchema`) resolves every `xs:keyref/@refer` against a schema-wide set of key/unique constraint names (identity-constraint names share one symbol space) and raises a fatal error for an unknown/empty refer. The registry is built by `collectAllIDCs`, which walks EVERY element declaration — not just `schema.elements` (globals) — by recursively descending each global element's/type's/named-group's content model (`idcWalker`, with visited sets on `*ElementDecl`/`*ModelGroup`/`*TypeDef` to bound shared/recursive/circular structures), so a keyref (or the key it refers to) declared on a LOCAL element buried in a content model is checked too. **@refer resolution is schema-wide (the symbol space); keyref VALUE resolution is subtree-scoped** — the two are distinct (see Pass 2 below). A deferred `@refer` error is reported against the constraint's DECLARING file: `IDConstraint.Source` is pinned at parse time in `parseIDConstraint` (`c.includeFile` if inside an include/redefine, else `c.filename` — for an import sub-compiler that is the imported file's display location), and `checkKeyRefRefers` reports with `idc.Source`+`idc.Line` rather than the top-level compiler's filename, so an IMPORTED keyref's dangling-refer error cites the imported schema (where its line number is meaningful), not the importing schema. At validation time, an IDC whose selector/field XPath fails to evaluate is reported as a validity error (`Failed to evaluate identity-constraint '…'`), not swallowed.

**Pass 2 — Identity Constraints** (`validateIDConstraints` via second `helium.Walk()`):
- **Host declaration resolution** (`idcHostDecl`): the declaration whose IDCs apply
  to an element instance is the non-ref declaration recorded during pass-1 if one is
  present — used even when it carries ZERO IDCs, because a local element that merely
  shadows a same-named global must NOT inherit the global's IDCs. It falls back to the
  GLOBAL lookup (`lookupElemDecl`) only when no declaration was recorded OR the recorded
  declaration is a ref (`IsRef`). Pass-1 records the matched
  `*ElementDecl` for every element instance in `validationContext.actualElemDecl`
  (`recordElemDecl`, called at the content-model match sites alongside
  `annotateElement` and at the validation root), so an `xs:key`/`xs:unique`/`xs:keyref`
  declared on a LOCAL element buried in a content model is EVALUATED rather than
  silently skipped — `lookupElemDecl` finds only globals. The ref fallback exists
  because an `<xs:element ref="g">` matches a ref declaration that does NOT copy the
  global's IDCs (IDCs are a property of the referenced global declaration), so for a
  `ref` the global lookup is the one carrying the constraints.
- For elements with IDCs (xs:unique, xs:key, xs:keyref):
  1. Evaluate selector XPath → node set
  2. For each selected node, evaluate field XPaths → collect key-sequences
  3. Check unique/key: all key-sequences must be unique
  4. Check keyref: all key-sequences must exist in referenced constraint table.
     **Keyref tables are SUBTREE-SCOPED** (XSD identity-constraint scope, matching
     xmllint): the key/unique table a keyref resolves against is the one in scope
     for the keyref's host OCCURRENCE — the constraints declared directly on the
     host (`validateIDConstraints` builds a per-occurrence `keyTables
     map[QName]*idcTable` and resolves the occurrence's keyrefs against it after
     every key/unique on the occurrence is evaluated, so a keyref declared before
     its key still resolves) PLUS key/unique tables that PROPAGATE UP from the
     host's DESCENDANT subtree (`collectSubtreeKeyTable` walks the host's children
     recursively, gathering — via `idcHostDecl` per descendant — every key/unique of
     the referenced QName and merging their key-sequences; descendant evaluation is
     done under `suppressDepth` so cvc field/key-missing diagnostics are reported
     only once, by that descendant's own pass-2 walk). So a key on a CHILD element
     satisfies a keyref on an ancestor host (bug322411). A keyref whose referenced
     key/unique is declared OUTSIDE the host's subtree — on a SIBLING, or on a
     different occurrence of a repeating host — resolves against an EMPTY key space →
     every key-sequence is a "no match" failure. This is deliberate and matches
     xmllint: two sibling occurrences of a repeating host never leak key spaces into
     each other (a doc-wide merged table would falsely accept a cross-scope
     reference), and a key on a sibling element is out of the keyref's scope. No
     false accepts.
  - Field presence (cvc-identity-constraint.4.2.1): an `xs:key` requires every
    field to evaluate to a node for each selected node; an absent field is a
    validity error (`Not all fields of key identity-constraint '…' evaluate to a
    node.`). `xs:unique` and `xs:keyref` tolerate absent fields — the node drops
    out of the qualified node-set.
  - Field cardinality (cvc-identity-constraint.3): for each selected node every
    field must evaluate to an empty node-set or a node-set with exactly one
    member. A field selecting more than one node is a validity error for all IDC
    kinds (`The XPath '…' of a field of <kind> identity-constraint '…' evaluates
    to a node-set with more than one member.`) rather than silently using the
    first node.
  - XPath uses namespace context from schema, not instance
  - Key comparison is value-space aware (XSD 3.11.4): each field value is
    canonicalized via its resolved simple type (`resolveFieldType` →
    `canonicalFieldKey`/`canonicalValueKey`) before map-key use, so
    `5`/`+5`/`05` collide for xs:integer. Field-type resolution first consults
    the actual `*TypeDef` recorded for each element during pass-1 content
    validation (`validationContext.actualElemType`, populated by
    `annotateElement`), so an IDC field whose type is contributed by an
    `xsi:type` actual type is canonicalized in the correct value space; it falls
    back to descending the declared content model only when the actual type is
    unknown. Attribute-field type resolution (`attrUseTypeDef`) mirrors the
    content validator's `validationContext.attrUseType`: an inline anonymous
    `<xs:simpleType>` (`au.Type`) is preferred over the named `au.TypeName`
    reference, for both complex-type attribute uses and global attributes.
    Canonicalization is full-type aware via per-variety dispatch: QName/NOTATION
    fields resolve the lexical prefix against the field node's in-scope
    namespaces to a `{uri,local}` Clark-name key (so `p:a`/`q:a` bound to the
    same URI collide, different URIs stay distinct), list fields canonicalize
    each item in the item type's value space (so `5 6`/`+5 06` collide for
    itemType="xs:integer"), and union fields resolve the **active member** the
    same way `validateUnionValue` does — the first **direct** member type
    (declaration order) the value **fully validates against** (lexical space AND
    that member's facets AND, for a nested-union member, the union wrapper's own
    facets and member resolution, via `typeAcceptsValue` → `validateValue`, not
    lexical space alone). Members are **not** pre-flattened to leaves: each direct
    member (`resolveUnionMembers`) is validated as-is, so a nested-union member
    whose wrapper restriction rejects the value by facet is correctly skipped
    (flattening to the bare leaf would drop that wrapper facet and falsely accept
    the value). Once the active member is chosen, the value is canonicalized in
    THAT member's space by **recursing** through `canonicalValueKey`
    (`unionActiveMember` → `canonicalValueKey`), so a **list** member canonicalizes
    item-by-item and a nested-**union** member resolves its own active member;
    an atomic member reaches `canonicalAtomicKey`, where value-comparable members
    use `value.CanonicalKey` and lexical-only members (xs:string family, anyURI)
    use the whitespace-processed lexical value. So memberTypes="xs:string
    xs:integer" keeps `5` and `+5` distinct (active member xs:string), while
    "xs:integer xs:string" collapses them; memberTypes="intList xs:string" (intList
    = xs:list itemType="xs:integer") collapses `5 6` and `+5 06`; and a member
    whose facets reject the value (e.g. an xs:integer restriction with
    maxInclusive="0" fed `5`) is skipped so the value falls through to the next
    member, exactly as the validator does. `typeAcceptsValue` (and thus
    active-member selection) threads `fieldNodeNSContext(fieldNode)` as the value's
    namespace context, so a union member with a QName/NOTATION-valued facet (e.g.
    an enumeration of prefixed names) resolves its prefixes against the same
    bindings as the instance value. Variety dispatch in `canonicalValueKey` and the
    list/union member resolution use the same base-chain helpers the validator
    uses (`resolveVariety`, `resolveItemType`, `resolveUnionMembers`), so a
    restriction over an inline list/union (which keeps `Variety==Atomic` on the
    derived type) is still canonicalized in the correct variety. `canonicalAtomicKey`
    first whitespace-processes the value per the resolved type's effective
    whiteSpace facet (`resolveWhiteSpace`), so a restriction of xs:string with
    whiteSpace="collapse" makes `a b` and `a  b` collide. Raw values are retained
    for error display;
    fields whose type cannot be resolved fall back to raw-string comparison.
  - Field type resolution (`resolveElemType`) consults the `actualElemType` map
    populated in pass 1, so xsi:type ACTUAL types reach IDC canonicalization. Pass
    1 annotates not only model-group children but also descendants of an
    xs:anyType / mixed element with no content model: `validateElementContent`
    routes that case to `annotateAnyTypeChildren`, which lax-validates each child
    (look up global decl, resolve xsi:type, `annotateElement`, recurse) so a
    descendant under an anyType ancestor still has its actual type recorded before
    pass-2 IDC evaluation — otherwise a nested `<item xsi:type="itemType" n="5"/>`
    / `n="+5"` pair would be compared lexically and wrongly accepted as unique.
    The same recursion runs for the **lax** wildcard path: when
    `matchWildcardParticle` (`xs:any processContents="lax"`) matches an element
    that has no global declaration, that element is not schema-assessed but its
    subtree is still walked via `annotateAnyTypeChildren`, so a nested global IDC
    host deeper under an unknown wildcard wrapper has its descendants' actual
    types recorded before pass-2 IDC — otherwise the same lexical-vs-value-space
    `5`/`+5` collision would be missed. The **skip** wildcard path
    (`processContents="skip"`) is not schema-assessed at all, so it must NOT run
    content-model validation or raise errors; instead `matchWildcardParticle`
    walks each matched subtree with `annotateSkipChildren`, an annotation-only
    traversal that records (via `annotateElement`) the ACTUAL type for every
    descendant carrying a resolvable `xsi:type` — including LOCAL descendants with
    no global declaration — using a non-reporting `resolveXsiTypeQuiet`, then
    recurses. This is what lets a nested `<item xsi:type="itemType" n="5"/>` /
    `n="+5"` pair under an `xs:any processContents="skip"` wrapper collide in
    xs:integer value space rather than being wrongly accepted as unique. These
    skip-only actual type records are NOT written to the XSD 1.1 `assertAnnotations`
    map, so an ancestor `xs:assert` still atomizes skipped content as unassessed
    (`xs:untypedAtomic`) even when the skipped node carries a resolvable `xsi:type`.

    XSD 1.1 assertion PSVI annotations register INLINE ANONYMOUS list item and
    union member metadata across the annotated type's full base chain, not just
    the immediate `TypeDef`. This preserves anonymous faceted members inherited
    through a restriction wrapper, so `data()` active-member selection in
    `xs:assert` sees the same value space as validation.

**XSD 1.1 identity-constraint extras** (compile time): `@xpathDefaultNamespace`
on `xs:selector`/`xs:field` (or inherited from the root `xs:schema`) becomes a
default ELEMENT namespace URI stored per selector/field
(`IDConstraint.SelectorDefaultNS`/`FieldDefaultNS`). The token→URI resolution
(`resolveXPathDefaultNSToken`, `read_elements.go`: `##targetNamespace` → the
target namespace, `##defaultNamespace` → the element's in-scope default namespace,
`##local`/empty → none, else literal URI) is namespace-context-sensitive for
`##defaultNamespace`. A LOCALLY-PRESENT selector/field value is resolved against
THAT element; an inherited schema-level value is resolved ONCE against the SCHEMA
ROOT at root-read time and stored as the resolved URI (`compiler.schemaXPathDefaultNS`),
so an inherited `##defaultNamespace` uses the ROOT's default namespace, NOT a
selector/field that redeclares `xmlns`. `evaluateIDC` applies the URI via the
opt-in `xpath1.Evaluator.DefaultElementNamespace`, which matches unprefixed
ELEMENT name tests against it (attributes are never affected — they have no
default namespace). `resolveXPathDefaultNS` decides inheritance by PRESENCE
(`hasAttr(elem, attrXPathDefaultNS)`), not value — xs:anyURI admits the empty
string and `getAttr` cannot tell `xpathDefaultNamespace=""` from an absent
attribute — so an EXPLICIT empty value means "no default element namespace" and
does NOT inherit; only an ABSENT attribute inherits the resolved schema-level URI.
The schema-level default is a PER-document setting: `compile.go` resolves it for
the top-level root and `compile_imports.go` saves/sets/restores it (resolved
against each loaded root) across `xs:include`/`xs:redefine` (alongside
elementFormDefault/blockDefault/finalDefault and `includeFile`) and sets it on the
import sub-compiler from the imported root — so an included/imported schema's IDCs
inherit ITS root's value, not the including/importing schema's (otherwise an
included `xpath="emp"` selector would silently resolve to no-namespace and miss
duplicates). `@ref` on an identity
constraint (`resolveConstraintRefs`, `compile.go`, run after `checkDuplicateIDCs`
and before `checkKeyRefRefers`) makes the constraint reuse a referenced
constraint's selector/fields — and a keyref's `refer` — and adopt its QName
identity (so a ref'd keyref resolves against a ref'd key on the same host); the
reference must resolve to an existing constraint of the SAME kind, else a fatal
schema error. A `@ref` constraint has no name of its own and is skipped in the
duplicate-name and key-name registries. The ref form is detected by PRESENCE
(`hasAttr(elem, attrRef)`, since `getAttr` cannot tell an absent attribute from an
empty one) so a literal `ref=""` is recognized as the (invalid) ref form and
reported as a fatal error rather than silently dropped. `resolveIDCNameQName`
first validates `@ref` as a lexical `xs:QName` (`validateQName`) before splitting
the prefix, so a malformed value such as `:u` (empty prefix) is a fatal error
(`reportInvalidQNameValue`) instead of being silently resolved as an unprefixed
reference — `strings.Cut(":u", ":")` would otherwise yield an empty prefix that
bypasses the unbound-prefix check. `@refer` (`resolveIDCReferQName`) is validated
the same way, and its malformed/unbound-prefix diagnostics are attributed to the
constraint's DECLARING file (`idc.Source`, falling back to `c.diagSource()`), so a
bad `@refer` in an included/redefined schema cites the included file paired with
its own line number, not the including schema. A prefixed `@ref` whose prefix is not bound in scope is a fatal error
(`resolveIDCNameQName` → `reportUnboundQNamePrefix`, the same path every other
QName-valued schema attribute uses), not a silent map to no-namespace; the
`constraintRefUnbound` flag suppresses the follow-up "unknown constraint"
diagnostic. The `@ref` form is mutually
exclusive with the full form: a `@ref` constraint that ALSO carries
`name`/`xs:selector`/`xs:field`/`refer` is rejected (`reportIDCRefConflict`). The
`name` and `refer` companions are detected by PRESENCE (`hasAttr`), not value, so
an empty-but-present `name=""`/`refer=""` is still rejected (consistent with the
ref-form detection); and `refer` is rejected for EVERY kind (key/unique/keyref),
not only on `xs:keyref`.

**Pass 3 — ID/IDREF/IDREFS** (`validateIDIDREF`, `validate_id.go`, XSD 1.1 only):
a third `helium.Walk()` enforcing cvc-id document-wide. Every `xs:ID` value must
be unique, **except** that the same value may identify a single element more than
once. An ID's owning element is the element BEARING it — an attribute ID on its
owning element, an element-content ID on its **parent** (`idOwner`) — so two ID
attributes of one element, or two `<id>` children of one parent, sharing a value
is valid, while the same value reaching two different owners is a duplicate. An
element-content ID on the DOCUMENT ROOT has no parent element, so it denotes NO
element: `idOwner` returns nil and `recordID` skips it (the value never enters
the table), so any `xs:IDREF` to it dangles — W3C idIDREF s3_3_4ii26/ii27, "ID on
root does not denote any element". Each
`xs:IDREF`/`xs:IDREFS` token must resolve to some collected ID. Values are
decomposed against their type variety (`collectIDFromValue`, mirroring
`canonicalValueKey`): a list splits into items, a union resolves to its active
member (`unionActiveMember`), reaching the atomic ID/IDREF leaves; the built-in
`xs:IDREFS` (a flat atomic placeholder) is split by name. Element-content
collection is SKIPPED entirely when the element has CHILD ELEMENTS
(`hasChildElement`): simple content forbids child elements, so pass 1 already
rejected such an element structurally and there is no valid simple value here —
`elemTextContent` ignores the children and a default/fixed would otherwise be
substituted for non-empty content, fabricating an ID/IDREF on top of the real
structural error. Otherwise, genuinely-empty element content falls back to the
declaration's default/fixed value — EXCEPT on a CONFIRMED nilled element: one
DECLARED nillable (`idcHostDecl(elem).Nillable`) carrying `xsi:nil="true"`
(checked quietly via `isXsiNilTrue`). A nilled element has no element value, so
substituting its default/fixed would fabricate a duplicate ID or a dangling IDREF
and false-reject a valid document; its element-content collection is skipped
(attribute IDs still apply). The check is by DECLARATION, NOT raw
`xsi:nil`: a `processContents="lax"` element with no declaration but a resolvable
`xsi:type` is not validly nilled (xsi:nil requires a nillable declaration), and
`assessLaxElement` validated its real content, so its ID/IDREF content is still
collected — raw `xsi:nil` would wrongly drop it and false-accept a duplicate. Element
typing for this pass uses `assessedElemType` as the SOLE source; attribute typing
uses `actualAttrType` (populated in `annotateAttrUse` and `validateWildcardAttr`
for explicit uses and strict/lax wildcard-admitted global attributes) — with NO
global-declaration fallback. Crucially it reads NEITHER `actualElemType` (ALSO
written, `assessed=false`, by `annotateSkipChildren` and the lax-no-resolvable-type
branch purely for pass-2 IDC canonicalization) NOR `actualElemDecl` (written by
`recordElemDecl` at the particle-MATCH scan BEFORE content validation — so a
matched-but-UNASSESSED child, e.g. an unsatisfied `minOccurs`, would otherwise be
misclassified as ID/IDREF and produce a spurious duplicate/dangling on top of the
real structural error). Only `assessedElemType` reflects genuine assessment, so
both skip content and matched-but-failed children are excluded. The host
declaration (`idcHostDecl`, for default/fixed/nillable metadata) is consulted only
AFTER `elementTypeForID` returns a non-nil assessed type. `annotateElement` takes an `assessed` bool — true at the root,
content-model matches, xs:anyType/lax children WITH a global declaration, AND a
`processContents="lax"` element with no declaration but a RESOLVABLE `xsi:type`
(which per XSD lax IS assessed: validated against that type and counted for pass 3,
so its `xsi:type="xs:ID"`/`xs:IDREF` content participates); false at
`skip`/lax-no-resolvable-type sites (writes only `actualElemType`). The lax
no-declaration assessment lives in ONE shared helper, `assessLaxElement`, called
from BOTH `matchWildcardParticle` (a directly wildcard-matched lax element) and
`annotateAnyTypeChildren` (an xs:anyType/lax descendant), so a directly-matched
lax element is validated and assessed just like a descendant — previously
`matchWildcardParticle` only recursed into the subtree and never assessed the
matched element itself, so it both false-accepted invalid `xsi:type` content and
missed its ID. `assessLaxElement` NEVER lets `xsi:nil="true"` bypass validation: an
undeclared element has no nillable declaration, so its content is always validated
against the governing type (a nilled lax element with invalid or type-forbidden
content is rejected; empty content a type permits stays valid). The STRICT
wildcard FAILURE path (a strict wildcard matching an element with no global
declaration) is, like `skip`, NOT assessed: `validateWildcardChild` reports the
strict error then walks the subtree with `annotateSkipChildren` (canonicalization-
only, never `assessedElemType`), NOT `annotateAnyTypeChildren` — using the lax
traversal there would laxly ASSESS globally-declared / xsi:typed descendants of a
strict-FAILED subtree and fabricate a spurious duplicate-ID/dangling on top of the
real strict error. So a `skip` wildcard element AND a strict-failed subtree — even
ones carrying `xsi:type="xs:ID"` or globally-declared `xs:ID` descendants — are
NEVER assessed and not treated as xs:ID/xs:IDREF, while a lax xsi:type'd element
IS. This avoids both false-rejecting duplicate skipped/strict-failed IDs and
false-accepting duplicate (or invalid) lax-assessed xsi:type content. The pass never runs in 1.0
mode, so the libxml2-compat goldens stay byte-identical. ID/IDREF members inside a
union ARE covered: `collectIDFromValue`'s union branch resolves the active member
(`unionActiveMember`) and recurses to the atomic ID/IDREF leaf (and `idFamilyType`
recurses into union members), so a duplicate union xs:ID across owners and a
dangling union xs:IDREF both fail. `xs:ENTITY`/`xs:ENTITIES` value-space validity is enforced by the separate `validateEntities` pass (`validate_entity.go`), not this one. NOTE: this skip-exclusion is for the
ID/IDREF DATATYPE pass only; pass-2 IDC selectors (`xs:key`/`xs:unique`) still
match skip-content nodes by XPath — helium deliberately includes skip-matched
nodes in an ancestor IDC (see `TestIDCFieldSkipWildcardSelectedSelf`), so the
saxon `Wild.testSet` `wild101–104` "IDC-with-skip" cases (which require the
opposite) remain a separate, unresolved pass-2 design question.

### Current XSD 1.1 Assertion Details

`materializeQNameAttrValue` handles default/fixed QName and NOTATION carriers
value-dependently: it selects active union members with `fixedUnionActiveMember`,
walks XSD-list item tokens recursively, and rewrites only tokens whose schema
prefix collides with the instance's in-scope binding. `assertEffectiveValues`
records the effective content type for empty descendant defaults so
`isolatedAssertTree` applies the same union/list materialization before ancestor
assertions atomize `data(c)`.

Schema-aware XPath casts to user-defined types still return the built-in cast
atomic value stamped with the user type. String and untypedAtomic sources validate
facets against the original source lexical string via `ValidateCastWithNS`,
preserving lexical facets such as a pattern on an integer restriction where
`"05"` and canonical `"5"` must not be treated as the same lexical input; already
typed sources validate the builtin cast result's lexical form.
For schema-aware union casts, member types are tried recursively; after a member
accepts, target-union facets/assertions are checked against the original lexical
for string/untypedAtomic sources and against the accepted member-cast result for
already typed sources.

### Key Data Model

```
Schema { elements, types, groups, attrGroups, globalAttrs, substGroups maps }
ElementDecl { Name QName, Type *TypeDef, MinOccurs/MaxOccurs, Abstract/Nillable, IDCs, Default/Fixed }
TypeDef { ContentType (Empty|Simple|ElementOnly|Mixed), ContentModel *ModelGroup, BaseType, Attributes []*AttrUse, Facets, Variety (Atomic|List|Union) }
ModelGroup { Compositor (Sequence|Choice|All), Particles []*Particle }
IDConstraint { Kind (Unique|Key|KeyRef), Selector/Fields XPath, Refer, Namespaces, Selector/FieldDefaultNS (1.1 xpathDefaultNamespace), IsConstraintRef/ConstraintRef(QName) (1.1 @ref), Line, Source (declaring file) }
```

## RELAX NG

Files: `relaxng/relaxng.go` (API), `parse.go` (compiler), `validate.go` (engine), `grammar.go` (model)

### Compile: Document → Grammar

1. **Find root** — `<grammar>` or bare pattern (e.g., `<element>`)
2. **Parse grammar content** — process `<start>`, `<define>` elements; handle `combine="choice"/"interleave"`; support `<div>` containers
3. **Parse patterns** (recursive) — element, attribute, group, choice, interleave, optional, zeroOrMore, oneOrMore, ref, parentRef, data, value, list, mixed, text, empty, notAllowed
4. **Resolve references (scoped)** — each `<grammar>` (including nested ones) gets its own lexical `grammarScope` with a `defines` table and a `parent` link. Every `<ref>`/`<parentRef>` node is recorded with the scope it was parsed in (`compiler.pendingRefs`); after the whole tree is parsed, `resolveScopedRefs` fixes each node's `pattern.resolved` pointer: `<ref>` resolves the name in its OWN grammar scope, `<parentRef>` in that scope's PARENT scope. A name not found in the target scope, or a `<parentRef>` with no parent grammar scope, is a FATAL compile error (`reportUnresolvedRef`, bumps `errorCount`) per RELAX NG §4.18 — not a silently-unresolved node. Scoped resolution keeps same-named defines across nested grammars distinct (D-RNG-001). The flat `Grammar.defines` map is populated by `resolveRefs` but is not the resolution authority. **`<include>` override interaction:** a start/define that the `<include>` body OVERRIDES is REMOVED from the included grammar per RELAX NG include semantics, so `parseInclude` collects the override names FIRST and `parseGrammarContentSkipping` skips parsing those top-level start/defines (through transparent `<div>`) when reading the included grammar — refs that live only inside a removed component are never recorded in `pendingRefs` and so never trigger a spurious fatal unresolved-ref. The override names are also `delete`d from the scope after parsing (to clear any entry leaked by a nested `<include>`) before the overrides are applied.
5. **Check reference cycles** — `checkRefCycles` walks each define body across every scope, following `pattern.resolved` (cycle set keyed by define-pattern POINTER, not name); element patterns break the chain
6. **Rule checks** — compile-time semantic validation (`checkPattern` also follows `pattern.resolved`; visited set keyed by `{define pattern, ruleFlags}` so a define reached under a new ancestor context — e.g. once normally and once under `<list>` — is re-checked rather than suppressed)
7. **Content-type checks (§7.2)** — `checkContentTypes` (runs AFTER scoped-ref resolution, so it follows `<ref>`/`<parentRef>` into their resolved define body) checks every `<element>` body reachable in the LIVE grammar. The live element bodies are gathered by `collectLiveElements` walking ONLY from the resolved start pattern, descending children/attrs/resolved refs under a pointer-keyed cycle guard — reaching every `<element>` reachable through the live start/define graph (including those inside referenced defines and nested grammars), NOT a parse-time append-only list of every parsed `<element>` and NOT every entry of the append-only `c.scopes`. Seeding from the start alone is what keeps a removed/overridden define out of the check: a nested-grammar scope created while parsing a define that an `<include>` override later deletes lingers in `c.scopes` forever but is unreachable from the live start, so its dead content (e.g. a content-type-bad define inside a nested `<grammar>` under the overridden define) is not flagged. The define that REPLACED an overridden one is reachable from start and so IS checked. Reporting cites the original element node via the `elementNodes` (`*pattern`→`*helium.Element`) map. A `<ref>`/`<parentRef>` contributes the content-type of its TARGET define body (NOT the unconditional `complex()` the §7.2 table lists for the fully-simplified form), matching libxml2: a ref to a group-of-attributes is `empty` and groups with `<value>` (libvirt.rng), while a ref to `<data>` is `simple` and is rejected when mixed with sibling element content. It derives a RELAX NG §7.2 content-type (`empty`<`complex`<`simple`) for each body via `contentTypeOf`/`contentTypeOfSeq`: `empty`/`notAllowed`/`attribute`→empty, `text`/`element`→complex, `data`/`value`/`list`→simple (it STOPS at `element`/`attribute`/`list` boundaries — each opens a fresh content context). `group`/`interleave`/repetition bodies fold children as an implicit group requiring `groupableContentTypes` at each step (`empty` groupable with anything; `complex`+`complex` ok; `simple` groupable only with `empty`), so mixing `simple` content with `complex` (e.g. `group(data, element)`, `group(data, text)`), grouping two `simple` values (`group(data, value)`), or repeating `simple` content (`oneOrMore(data)`) is a content-type error. `choice` combines branches with max and NO groupable constraint (branches never coexist), so `choice(data, element)` COMPILES. A ref cycle (degenerate non-element recursion, already reported by step 5) contributes nothing; unresolved refs contribute nothing.

### Validate: Document + Grammar → Errors

Pattern-matching engine with backtracking:

1. Root element → `validState{seq: [root]}`
2. `validatePattern(grammar.start, state)` dispatches on pattern kind:
   - **Element**: name-class match, consume from seq, validate body (attrs + content)
   - **Attribute**: match against instance attrs
   - **Group**: sequential with backtracking
   - **Choice**: try alternatives, prefer branches making progress
   - **Interleave**: unordered member-by-member matching; a repeatable member-group (zeroOrMore/oneOrMore of group) restarts its members each iteration so a sibling branch can consume elements between group members across iterations
   - **ZeroOrMore/OneOrMore/Optional**: repetition with suppressed errors
   - **Ref/ParentRef**: follow the compile-time-resolved `pattern.resolved` scoped pointer and recurse (no by-name lookup)
   - **Data/Value**: type checking
   - **List**: split text, validate items
3. Element validation: match name, validate attrs, build child list (skip non-content: EntityRef/PI/Comment), validate content, check all attrs+content consumed

### Backtracking Strategy (`backtrackGroupFlexible` / `backtrackGroupNaive`)

When mandatory group child fails:
1. Check if element was consumed (structural vs content error)
2. For each previous flexible child (zeroOrMore/oneOrMore/optional) from nearest to furthest:
   - Try iteration counts from minimum upward to greedy count
   - Re-validate remaining children via the group routine (`validateGroupChildren` /
     `validateGroupSeq`) so flexible members in the retry range can themselves
     backtrack — this recovers groups with 2+ flexible members that must each
     yield (e.g. `group(zeroOrMore(x), zeroOrMore(x), x)`)
   - Keep highest successful count (maximizes consumption — libxml2 semantics)

The recursive retry would be exponential (`O(M^N)` for `N` flexible members over
`M` elements) without memoization, since the cascading reductions re-explore
overlapping subproblems. `validateGroupChildren`/`validateGroupSeq` therefore
cache each call in `validator.groupMemo`, keyed by the inputs that fully determine
the result (child-range start pattern + length, owning element, first remaining
node + sequence length, packed `attrUsed`, `suppressDepth>0`, content-vs-naive
discriminator). A hit reproduces the original call's effect exactly — resulting
position, attribute usage, appended errors, return value — so memoization is sound
(no valid document rejected) while collapsing the fan-out to polynomial. Regression
guard: `TestMultiFlexibleGroupBacktrackingNotExponential`.

Two parallel implementations share this strategy. `validateGroupContent` +
`backtrackGroupFlexible` runs inside element bodies (threads attrs/attrUsed and
emits content-failure diagnostics). `validateGroup` + `backtrackGroupNaive` runs
the bare-`<group>` start path (and any group reached via `validatePattern`),
operating only on the node sequence with no attribute/element-content context.

### Token-Level Backtracking (`<list>` / attribute values)

`matchAttrContent` (attribute values) and `matchListContent` (`<list>` content)
match whitespace-separated tokens. `matchAttrTokensCounts` returns every
possible token-consumption count for a pattern in greedy-preferred (descending)
order; `groupCounts` composes children sequentially across those options
(memoized by child index + token offset to avoid exponential blow-up) and
`repeatCounts` enumerates repetition counts. A group succeeds when some
combination consumes exactly all tokens, so a greedy `oneOrMore`/`zeroOrMore`
can yield tokens back to a later mandatory member, and a zero-token `choice`
branch (e.g. `empty`) does not shadow a consuming branch. `matchAttrTokens`
is a thin greedy-max wrapper over `matchAttrTokensCounts`. `validateList` (the
naive `validatePattern` path) delegates to `matchListContent`, so every `<list>`
path shares these semantics.

### XSD Datatype Library (`<data>` / `<value>`)

When `datatypeLibrary` is the XSD datatypes namespace, `validateXSDType`
(`<data>`) and `matchXSDValue` (`<value>`) route through the shared XSD value
validator (`internal/xsd/value`): `ValidateBuiltin` enforces lexical/value
spaces (date/time/duration ranges, integer subtype bounds, binary alphabets) so
RELAX NG and XSD stay consistent. `xsdDatatypeNames` is the recognized-name
allowlist; any name outside it is an unknown datatype and is rejected (no silent
accept). `xs:string` keeps the local `<param>`-facet path (whiteSpace=preserve).
For `<value>`, `matchXSDValue` first requires **both** the instance text and the
`<value>` literal to be lexically valid via `ValidateBuiltin` for **every**
recognized XSD datatype — this gate runs before the lexical-equality fast-path
and the value-space branch, so an identical-but-invalid lexical is rejected for
both comparable types (e.g. `type="integer"` with both forms `5.0`) and
constrained non-comparable string-family types (e.g. `type="NCName"` with both
forms `1foo`). `ValidateBuiltin` imposes no constraint on `xs:string`/`xs:anyURI`,
so those stay effectively lexical-only. After the gate, value-space-comparable
types in `xsdValueSpaceTypes` (numeric, boolean, date/time, binary; mirrors
xsd's `enumValueSpaceTypes`) match by `value.Compare` value-space equality (e.g.
integer `5`≡`+5`≡`05`, NaN≡NaN for float/double); all other recognized types
(string-family, anyURI) match by whitespace-processed lexical equality.

**Empty / absent `datatypeLibrary`.** The empty built-in library provides only
`string` and `token`. For libxml2/golden compat, `matchData`/`matchValue` fall
back to the XSD value path for a recognized bare XSD name (e.g. `<data
type="integer"/>`) **only when `datatypeLibrary` is genuinely absent** —
`dataType.libraryDeclared == false`. An explicit `datatypeLibrary=""` (including
one that resets an inherited XSD library) selects the built-in library and
rejects bare XSD names. `getDatatypeLibrary` returns `(value, declared)`,
testing attribute presence (`getAttrOpt`) up the ancestor walk so an explicit
`""` stops the walk instead of leaking the inherited library. Unknown
names/libraries fail rather than matching by raw equality.

**Length facets.** `validateWithParams(value, typeName, params)` enforces exact
`length` as well as `minLength`/`maxLength`, computing length by datatype via
`facetLength`: rune count for string-family types, XML-whitespace token COUNT
for XSD list builtins (`NMTOKENS`/`IDREFS`/`ENTITIES`), and decoded OCTET count
for binary (`hexBinary`/`base64Binary`). The length facets are APPLICABLE only to
the string-derived family, the binary types, anyURI, QName and NOTATION
(`value.LengthApplicable` in `internal/xsd/value`, shared with relaxng; xsd's
`check_facets.go` keeps an equivalent table);
a length facet on a numeric, boolean or date/time datatype is rejected at COMPILE
time by `checkDataFacets`. On every applicable type the length facets are
CONSTRAINING — including `xs:QName` and `xs:NOTATION` (XSD 1.0 / libxml2 parity):
a value whose rune count violates the bound is REJECTED, exactly as the shared xsd
validator's `facetLength` rejects it (so RELAX NG and xsd cannot diverge — a
multi-rune QName with `maxLength="1"` fails in both). Each
length bound is COMPILE-validated as an
`xs:nonNegativeInteger` (XSD-collapse normalized, NOT Go `TrimSpace`) so a
negative, fractional, non-digit, or NBSP-padded bound is a fatal schema error
(including on QName/NOTATION)
rather than a facet that silently accepts every value (`minLength="-1"`) or whose
bound is leniently trimmed. At validation `validateWithParams` parses the bound
with `math/big` (`parseNonNegFacetBound`) and compares against `facetLength` via
`big.Int.Cmp`, so a huge-but-valid `maxLength` cannot overflow `int` into a
reject-all.

**Pattern facet.** The `pattern` facet is an XSD/XPath regular expression matched
through the shared XSD-regex engine (`internal/xsdregex`, the same translator
`xsd` and `xpath3` use), NOT Go's `regexp` — so XSD-only constructs (`\i`/`\c`
name-class shorthands, `\p{...}` blocks, character-class subtraction) are honoured
rather than false-rejected. `checkDataFacets` compiles the pattern once at compile
time (caching the `*xsdregex.Regexp` on the `param`, guarded by `patternChecked`
against shared-`<ref>` re-visits); an invalid pattern is a fatal schema error, and
`validateWithParams` reuses the cached compilation (whole-value anchored).

**Ordering / digit facets.** `validateWithParams` also enforces the range facets
(`min`/`maxInclusive`, `min`/`maxExclusive`) via `facetOrderingOK` (shared
`value.Compare`) and the digit facets (`totalDigits`/`fractionDigits`) via
`value.CountTotalDigits`/`CountFractionDigits` on the `value.IsDecimalFamily`
types. `facetOrderingOK` returns SATISFIED when `value.Compare` reports `ok=false`
for two valid ORDERED operands that are genuinely indeterminate (e.g. mixed-timezone
`xs:dateTime`), matching XSD semantics — but a NaN operand is the exception: an
`xs:float`/`xs:double` NaN instance value OR NaN bound is excluded by the bounding
facets (`value.IsFloatNaN`), so the facet FAILS rather than slipping through. The
digit- and length-facet bounds are parsed with `math/big` (`parseNonNegFacetBound`,
normalizing via the XSD collapse whiteSpace facet — NOT Go's
`strconv.Atoi`+`strings.TrimSpace`, which trims NBSP and overflows large bounds
into a reject-all). Facet APPLICABILITY
and bound LEXICAL VALIDITY are enforced at COMPILE time by `checkDataFacets`
(`parse_check.go`, run from the `patternData` case of
`checkPattern`): an ordering facet on a non-ordered datatype (`!value.Orderable`)
or with an invalid bound, a digit facet on a non-decimal datatype, a length facet
on a non-string-derived datatype (`!value.LengthApplicable`), an uncompilable
`pattern`, a digit-facet
bound that is not a valid `xs:positiveInteger` (`totalDigits`) /
`xs:nonNegativeInteger` (`fractionDigits`), and a length-facet bound that is not a
valid `xs:nonNegativeInteger` — including an NBSP-padded or
out-of-range bound — are fatal
schema errors — which makes the whole grammar unmatchable (`compileSchema`
replaces `start` with `notAllowed`). `effectiveXSDDatatype` resolves the `<data>`
datatype the same way `matchData` does (explicit XSD library, or a bare recognized
name only when `datatypeLibrary` is absent). Any other `<param>` name
(`enumeration`, `whiteSpace`, unknown) fails closed at validation.

**Tokenization.** All `<list>`, attribute `<group>`/repetition, and
`<value type="token">` token splitting uses `xmlFields` (XML whitespace #x20,
#x9, #xA, #xD only) — never `strings.Fields` — so NBSP stays part of a token.

### Error Suppression

- `suppressDepth` counter incremented during choice branch exploration
- Errors only emitted on definitive failures (top-level or after element consumed)

### Key Data Model

```
Grammar { start *pattern, defines map[string]*pattern }
pattern { kind, name, ns, value, dataType, children, attrs, nameClass, params }
nameClass { kind (ncName|ncAnyName|ncNsName|ncChoice), name, ns, left/right, except }
```

## Schematron

Files: `schematron/schematron.go` (API), `parse.go` (compiler), `validate.go` (engine), `schema.go` (model)

### Compile: Document → Schema

Three-phase parsing:
1. **Phase 1: Title** — optional `<title>`
2. **Phase 2: Namespace declarations** — all `<ns prefix="x" uri="...">` → `schema.namespaces` map
3. **Phase 3: Patterns** — `<pattern>` → `<rule context="xpath">` → `<let>`, `<assert test="xpath">`, `<report test="xpath">`

Message content parsed into `[]messagePart`: text literals, `<name path="..."/>` (element name), `<value-of select="..."/>` (XPath value).

**Namespace gating:** structural elements are only recognized when in the detected Schematron namespace (`isSchematronElement`/`elementInNamespace`). Foreign-namespaced elements are handled differently depending on position:
- **Required structural position → fatal/rejected.** Where a specific Schematron element is expected (e.g. a `<rule>` under `<pattern>`, checked via `isSchematronElement(elem, schNS, "rule")` in `compilePattern`), a foreign element like `<x:rule>` does NOT satisfy the requirement and is rejected with a fatal `Expecting a rule element instead of ...` diagnostic. The same applies at the top level (`Expecting a pattern element instead of ...`).
- **Free-content children → ignored.** Foreign-namespaced children inside rules, asserts, and reports are skipped as free content. `compileRuleChild` returns early when `!elementInNamespace(...)`, so e.g. `<x:assert>` inside a `<rule>` is not executed; likewise foreign `<name>`/`<value-of>` inside message content (`parseMessageElement`) are ignored, not interpolated.

Structural attributes (`context`, `test`, `select`, `name`, `id`, `prefix`, `uri`, `value`, `path`) are read unqualified-only via `getStructuralAttr` (`NSPredicate{..., NamespaceURI: ""}`); a prefixed `x:test` is not read as Schematron.

**Fatal compile errors:** `compileSchema` wraps the configured handler in a `fatalTrackingHandler`. If any `ErrorLevelFatal` diagnostic is emitted (no pattern, pattern with no rule, rule with no test, etc.), `Compile`/`CompileFile` return `ErrCompileFailed` with a **nil** `*Schema` — even when no error handler is configured, so a broken schema can never validate as success.

### Validate: Document + Schema → Errors

`Validate` returns `ErrNoSchema` (typed) when the Validator has no compiled schema (`NewValidator(nil)` or zero-value), guarding against a nil-deref panic.

1. Create XPath context with schema's namespaces
2. For each pattern/rule: evaluate `contextExpr` against document root → node set
   - If the context XPath **errors at evaluation**, surface an `XPath error : ...` diagnostic and mark the document invalid (the rule's assertions can't be checked, so it is not silently skipped)
   - **First-match-only (ISO Schematron):** within a pattern, each node is processed by only the FIRST rule whose context matches it. A per-pattern `map[helium.Node]bool` (reset each pattern) skips nodes already claimed by an earlier rule. Scope is per pattern, so a later pattern still fires for the same node.
3. For each context node:
   - Bind `<let>` variables in **document order** (accumulated, so a later let sees earlier ones, e.g. `<let name="b" value="$a"/>` after `a`). A let whose expression **errors at evaluation** surfaces an `XPath error : ...` diagnostic rather than being silently dropped.
   - Create rule-specific XPath context with variables
4. For each test:
   - Evaluate XPath, convert to boolean
   - If the test XPath **errors at evaluation**, surface an `XPath error : ...` diagnostic and treat the test as `false` (mirrors libxml2 `xmlSchematronRunTest` returning 0): an **assert** then fires/fails, a **report** stays silent. A broken test is never treated as satisfied.
   - **Assert**: error if false
   - **Report**: error if true
5. Format message (interpolate text/name/value-of parts)
6. Report as ValidationError or append to string builder

### Key Data Model

```
Schema { patterns []*pattern, namespaces map[string]string }
pattern { name, rules []*rule }
rule { context string, contextExpr *xpath.Expression, tests []*test, lets []*letBinding }
test { typ (Assert|Report), expr, compiled *xpath.Expression, message []messagePart }
```

## Comparison

| Aspect | XSD | RELAX NG | Schematron |
|--------|-----|----------|-----------|
| Paradigm | Grammar (content models) | Pattern (recursive descent) | Rule (XPath queries) |
| Determinism | Compile-time UPA | Runtime backtracking | N/A |
| Namespace | Form qualification | Name classes | Schema prefix map |
| Constraints | xs:unique/key/keyref | None | Assert/report |
| Include | xs:include/import/redefine | include/externalRef | None |
| Interleave | xs:all (limited) | Full interleave | XPath predicates |
