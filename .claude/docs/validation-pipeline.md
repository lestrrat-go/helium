# Validation Pipeline

Three validation engines: XSD (grammar-based), RELAX NG (pattern-based), Schematron (rule-based). All follow compile→validate pattern.

## XSD

Files: `xsd/xsd.go` (API), `compile*.go` + `read_*.go` + `link_refs.go` + `check_*.go` (compile/read/resolve/constraint pipeline), `validate_context.go` + `validate.go` + `validate_elem.go` (content validation), `validate_idc.go` (IDC), `simplevalue_*.go` + `validate_value_api.go` (simple-value validation), `schema.go` (model)

### Compile: Document → Schema

1. **Parse root** — must be `xs:schema`; extract targetNamespace, form defaults, block/final defaults
2. **Register built-in types** — 46 XSD primitives
3. **First pass: collect** — walk children, populate maps:
   - `schema.elements` (global element decls)
   - `schema.types` (named complex/simple types)
   - `schema.groups` (model groups)
   - `schema.attrGroups` (attribute groups)
   - `schema.globalAttrs` (global attributes)
4. **Process includes/imports** — load `xs:include`/`xs:import`/`xs:redefine`, merge declarations. Nested-schema document loads go through `compileConfig.fsys` (`fs.ReadFile(c.fsys, path)`), an injectable `fs.FS` set via `Compiler.FS(...)`; it defaults to `iofs.PermissiveRoot` (`os.Open`) and is propagated to sub-compilers. `xslt3` injects a resolver-backed `fs.FS` (`schemaResolverFS`) so nested includes inside a resolver-loaded schema obey the same default-deny `URIResolver` policy as the top-level load. Schema-location resolution is **URI-aware** and lives in a **single canonical helper**, `xsd.ResolveSchemaURI(ref, base) (string, error)` (`xsd/resolve_uri.go`), shared by both the xsd nested-include path and `xslt3`'s schema loader so the two layers cannot drift. `validateSchemaPath` is a thin wrapper over it. It keys off whether `ref`/`base` carries a URI scheme (`xsd.URIScheme`, the **one** scheme-detector for both packages — multi-char scheme required so Windows drive letters and bare OS paths stay local): an **absolute-URI** `schemaLocation` (e.g. a cross-host `https://cdn.example.com/part.xsd`) is passed through **unchanged** — never `filepath.Join`ed, which would collapse `//` and drop the host; a **relative** location against a **URI base** resolves via `net/url` `ResolveReference` (RFC 3986), keeping authority intact **and re-applying the base's `OmitHost` flag** when the base had no authority (so `mem:/schemas/main.xsd` + `part.xsd` → `mem:/schemas/part.xsd`, never `mem:///schemas/part.xsd`, while canonical `file:///...` bases keep their `///`); a genuine **local** base/location keeps the historical `filepath.Join` + `..`-escape guard (the only branch that can return an error). The import sub-compiler's `baseDir` is `schemaBaseDir(path)` (the full URI for URI bases, `filepath.Dir` for local). Because resolution happens while base and raw `schemaLocation` are still separate, the name reaching the FS is the **canonical** nested URI, so `schemaResolverFS.Open` forwards it verbatim (no string repair of a collapsed name). `xslt3`'s `resolveSchemaURI` delegates the absolute-URI and URI-base cases to `xsd.ResolveSchemaURI` and only handles its own local **file**-base case (xslt3's base is a full file URI/path, not a directory); it seeds the xsd `BaseDir` via `schemaCompileBaseDir(uri)` (full URI when scheme present, `filepath.Dir` otherwise). **targetNamespace match (src-import / src-include):** `loadImport` rejects the located schema when its `targetNamespace` differs from the `namespace` declared on `<xs:import>` — a present `namespace` requires that exact TNS, an absent `namespace` requires the imported schema to have no TNS (so a schema imported as one namespace cannot silently contribute another's declarations). `loadInclude`/`loadRedefine` enforce the analogous include rule (included TNS must equal the including schema's, modulo chameleon includes with no TNS). Both raise a fatal `Schemas parser error` and stop merging that document. **Fatal-load exception:** an `xs:import` load failure (top-level in `processIncludes` and nested inside `loadImport`) is normally demoted to a non-fatal I/O warning ("Failed to locate a schema ... Skipping the import."); `errImportDepthExceeded` / `errSchemaPathEscape` already bypass that demotion as security limits, and any error whose chain satisfies the exported `xsd.FatalSchemaLoader` interface (`FatalSchemaLoad() bool` → true, found via `errors.As` through the `*fs.PathError` returned by `fs.ReadFile`) does too. `xslt3`'s `schemaResolverFS` wraps an over-cap read (`ErrResourceTooLarge`) in such a marker so the resource cap cannot be silently defeated for an imported schema; the marker `Unwrap`s to the original error so `errors.Is(err, xslt3.ErrResourceTooLarge)` still holds at the xslt3 boundary.
5. **Resolve references** — resolve all QName refs (types, base types, groups, attr groups, union members), build substitution group maps, detect circular substitution. After attribute type refs resolve, `checkAttrUseConstraints()` validates each attribute use's `default`/`fixed` constraint value against the attribute's declared simple type, so a retained-but-invalid constraint (e.g. `default=""` on an `xs:integer` attribute) is reported as a schema parser error rather than injected into the instance at validation time. Presence-based schema checks (`check_elements.go`) use `hasAttr`, and both `hasAttr`/`getAttr` require an **unqualified** attribute (`URI()==""`) so a foreign-namespaced `other:fixed` is not mistaken for the XSD `fixed`. When validation inserts an absent qualified attribute's default/fixed value, it is inserted **namespace-aware** (`SetAttributeNS`, reusing the in-scope prefix) so a later `xs:key` field like `@t:a` matches it. The insertion loop skips `Required` **and** `Prohibited` uses: a prohibited use must never materialize a default/fixed value (the absent attribute is accepted, and a present one is rejected), so it would otherwise mutate a valid document by inserting a forbidden attribute. The compile-time `default`-requires-`use="optional"` check (`checkAttributeUse`, `check_elements.go`) is applied to **ref** attribute declarations as well as named ones, so `<xs:attribute ref="t:a" use="prohibited" default="x"/>` is rejected at compile time (matching xmllint) rather than silently compiling.
   After refs resolve, `checkEnumQNameAndNotation()` (`xsd/check_facets.go`) runs two QName/NOTATION compile-time checks: (a) every `enumeration` literal of a QName/NOTATION-restricted type is resolved against its captured `FacetSet.EnumerationNS` bindings — an unbound prefix makes the literal an invalid QName and is reported as a schema error rather than silently compiling into an unsatisfiable enumeration. This is **variety-aware** (`enumLiteralHasUnboundQName`): an atomic literal is checked directly, a **list** literal item-by-item against the item type, and a **union** literal against whichever member type accepts it under its bindings (a literal that only a QName/NOTATION member could carry, with an unbound prefix, is flagged). (b) A simpleType whose base is directly `xs:NOTATION` with no `enumeration` facet is rejected. `checkNotationOnDeclarations()` extends (b) to **declarations**: an element or attribute whose effective type is the built-in `xs:NOTATION` (or NOTATION-derived) without an effective enumeration facet (`hasEffectiveEnumeration` walks the base chain) is rejected — this catches `type="xs:NOTATION"` placed directly on `<xs:element>`/`<xs:attribute>`, which bypasses the simpleType-level rule. Every attribute use records its source line in `attrUseSources` (merged from import sub-compilers) so the attribute case can report with the right location. Full xs:NOTATION declaration-table semantics (matching enumerated names against declared `<xs:notation>` elements) is deferred. `checkFacetConsistency()` additionally runs `checkFacetValueAgainstBase()` (`xsd/check_facets.go`): each value-bearing range facet (`min`/`maxInclusive`, `min`/`maxExclusive`) is validated as an instance of the restricted base type's value space via `validateValue` with a silenced `validationContext`; a bound that is not a valid instance (e.g. `<xs:minInclusive value="abc"/>` on an `xs:int` base, or a numerically out-of-range bound) is a fatal schema error rather than silently falling through `compareForRangeFacet`'s "can't compare" path and turning the constraint into a no-op at validation time. `checkFacetConsistency()` likewise runs `checkEnumValueAgainstBase()` (`xsd/check_facets.go`): each `enumeration` value is validated against the base type's value space via `validateValue` with a silenced `validationContext` and its captured `FacetSet.EnumerationNS[i]` bindings; an invalid member (e.g. `<xs:enumeration value="+NaN"/>` on an `xs:float`/`xs:double` base — signed NaN is not in their lexical space) is a fatal schema error at compile time rather than an unsatisfiable enumeration that fails only at instance validation. This is **variety-aware** — atomic literals against the base value space, **list** literals item-by-item against the item type, **union** literals against whichever member type accepts them — matching `validateValue`. Suppression is **per literal, narrow**: only a literal that `enumLiteralHasUnboundQName` flags (a QName/NOTATION carrier, at any nesting depth, with an unbound prefix, which `checkEnumQNameAndNotation` already diagnoses) is skipped, to avoid a duplicate diagnostic. It is **not** a blanket skip of QName/NOTATION-carrying types: every other enumeration literal of such a type is still checked against the base value space, so e.g. a QName base restricted with `xs:length value="2"` still rejects an out-of-space `<xs:enumeration value="abc"/>`.
   **Attribute-group reference expansion** (`link_refs.go`): a named attribute group's effective {attribute uses} is the union over the group's own `<xs:attribute>` children (`schema.attrGroups[qn]`) and, transitively, every `<xs:attributeGroup ref>` child (`attrGroupRefChildren[qn]`). `parseNamedAttributeGroup` records nested refs and `expandAttrGroupUses` (cycle-guarded) flattens them into each referencing type so a required/defaulted attribute declared in a nested group is not dropped. Three grammar rules apply: (a) **`use="prohibited"` declared directly inside an `<xs:attributeGroup>` is pointless** — libxml2 warns (`Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.`) and SKIPS it, so it is never propagated as a blocking use and a referencing `xs:anyAttribute` wildcard still admits the attribute (`parseNamedAttributeGroup` / the redefine-override loop in `compile_imports.go` both warn-and-skip). (b) A **circular** reference is a schema error (src-attribute_group.3) outside `<xs:redefine>`. A DIRECT self-reference (`<xs:attributeGroup ref>` resolving to the group being defined) is caught at parse time (`reportCircularAttrGroupRef`) and dropped; an INDIRECT cycle (e.g. `h -> i -> h`, or the 3-node `a -> b -> c -> a`) is caught in `resolveRefs` by `checkCircularAttrGroupRefs`, a deterministic DFS over `attrGroupRefChildren` run BEFORE flattening that reports each back-edge (`reportCircularAttrGroupRefQName`: `Circular reference to the attribute group 'x' defined.`) and CUTS it so the flatten/expand walks terminate without a diagnostic-less truncation and without a duplicate-attribute false positive. The indirect-cycle diagnostic is attributed to the BACK-EDGE `<xs:attributeGroup ref>` element's own line/file (recorded PER edge in `attrGroupRefSources[qn]`, index-aligned with `attrGroupRefChildren[qn]`, populated in `parseNamedAttributeGroup` and the redefine-override loop and merged across imports), NOT the owning group's declaration line — matching the direct-self-reference path and pointing at the right file when the cycle spans included/redefined schemas. The legitimate self-reference inside `<xs:redefine>` is handled by the override path, not `parseNamedAttributeGroup`. (c) The duplicate-attribute-use detection (`flattenAttrGroupRefDuplicates`, ag-props-correct.2) uses `visited` as a **recursion stack** (add on entry, `defer delete` on exit), not a global "seen ever" set — so two SIBLING refs to the same group are each expanded and a name contributed via both (e.g. `g -> h, h` with `h` carrying `x`) surfaces as a duplicate, while true reference cycles are still cut. All these schema diagnostics route through `c.diagSource()` / a per-record `source` so an included/imported schema is cited correctly.
**Particle occurrence validation** (read phase, `read_particles.go`/`read_elements.go`/`check_elements.go`): every particle's `minOccurs`/`maxOccurs` is validated as `xs:nonNegativeInteger` (max additionally allowing `unbounded`) with min<=max and the "prohibited particle" (`min=0 max=0`) carve-out. Lexical-error wording matches xmllint exactly — `The value 'X' is not valid. Expected is 'xs:nonNegativeInteger'.` for minOccurs and `Expected is '(xs:nonNegativeInteger | unbounded)'.` for maxOccurs. `validateOccursAttrs` covers compositor/wildcard/group-ref particles; `checkLocalElement` covers `xs:element`. **xs:all has special occurrence rules** (cos-all-limited): the all compositor's `minOccurs` must be 0 or 1 and its `maxOccurs` must be 1, and each element particle directly inside it must have `minOccurs`/`maxOccurs` of 0 or 1. `parseModelGroup` routes the all compositor to `validateAllOccurs` (dedicated `Expected is '(0 | 1)'.` / `Expected is '1'.` wording, used even for non-integer lexicals) instead of the generic `validateOccursAttrs`, and runs `checkAllElementParticleOccurs` on each direct element child (`Invalid value for {min,max}Occurs (must be 0 or 1).`). The cos-all-limited **group-reference** placement rule (`checkAllGroupRef`, run from `resolveRefs` over `groupRefs`) is also enforced for an `xs:redefine` self-reference: when a redefine rewrites an `all` group around a nested `<xs:group ref="self"/>` (e.g. inside a `sequence`/`choice`), `compile_imports.go` calls `checkAllGroupRef` on that self-reference placeholder **before** deleting it from `groupRefs`, so the nested-all rejection is not bypassed. The placement rule is **also** enforced on the extension-merge path in `link_refs.go`: when an `xs:extension` appends an `all` group (directly or via a group ref that resolves to one) onto a **non-empty** base content model (`modelGroupHasContent(baseMG)`), the merge would build a `sequence` containing an `all` group, so it is rejected with `The 'all' model group needs to be the only child of the model group.` (the `checkAllGroupRef` nested-detection does not fire here because the ref is the derived type's sole top-level particle). **Prohibited particles (`maxOccurs=0`) are not content:** `modelGroupHasContent` returns false for a group whose own `MaxOccurs==0` and skips any child particle with `MaxOccurs==0`, and the extension-`all` rejection is gated on `derivedMG.MaxOccurs != 0` — so a prohibited base/derived all-group particle (e.g. a `<xs:group ref="g" minOccurs="0" maxOccurs="0"/>` that resolves to an `all` group, or a base whose only element is `minOccurs=0 maxOccurs=0`) maps to no particle and is not falsely rejected. **Source attribution:** `validateOccursAttrs`/`validateAllOccurs`/`checkAllElementParticleOccurs` emit through `c.diagSource()`, and `checkAllGroupRef` uses the `source` captured on `groupRefSource` at parse time (it runs deferred, after `c.includeFile` is restored), so an occurrence/all diagnostic for a particle declared in an included/imported schema cites the declaring file, not the top-level label.

**Complex-type child ordering** (read phase, `read_types.go`, XSD 3.4.2): `parseComplexType`, `parseRestriction`, `parseExtension`, and `parseSimpleContentChildren` enforce the fixed child order as an ordered state machine — an OPTIONAL leading model-group particle (`sequence`/`choice`/`all`/`group` ref), at most one, THEN attribute/attributeGroup uses, THEN an OPTIONAL final `anyAttribute`. A model-group particle that appears AFTER any attribute/attributeGroup/anyAttribute is out of order and rejected (`The content model particle '…' must appear before the attribute declaration '…'.`) rather than silently overwriting the content model. A second model group (`more than one content model particle`) and mixing a particle with a `simpleContent`/`complexContent` wrapper are also rejected. The `anyAttribute` wildcard must be the optional FINAL child: an `attribute`/`attributeGroup` use appearing after it is rejected (`The attribute declaration '…' must appear before the attribute wildcard 'anyAttribute'.`), and a second wildcard is rejected (`A complex type definition must not have more than one attribute wildcard …`) via an `anyAttributeSeen` flag tracked in each of the four parse paths, rather than silently overwriting `td.AnyAttribute`. **simpleContent extension prohibited attrs:** `parseSimpleContentChildren` takes the derivation kind; on an EXTENSION a `use="prohibited"` attribute is pointless and is warned+skipped (`Skipping attribute use prohibition, since it is pointless when extending a type.`) exactly like `parseExtension` (complexContent), so it does not propagate as a blocking use and a base attribute wildcard still admits the attribute; on a RESTRICTION the prohibition is kept. **Unresolved type/element ref source attribution:** `reportUnresolvedTypeRef` reports via `c.diagSourceOrRecorded(typeDefSource.source)` and the owner type's actual element kind (`typeDefSource.elemKind`, recorded at parse time — `complexType` vs `simpleType`), not a hard-coded `c.filename`/`"simpleType"`; `elemRefSource` likewise carries the declaring `source` so an unresolved element/ref in an included/imported schema cites the declaring file.

6. **Constraint checks** (when errorCount == 0):
   - `checkFinalOnTypes()` — final attribute enforcement
   - `checkFinalOnSubstGroups()` — substitution group final
   - `checkUPA()` — Unique Particle Attribution (content model determinism)
   - `checkElementConsistent()` (`check_element_consistent.go`) — cos-element-consistent (Element Declarations Consistent): two element declarations with the same expanded name reachable in one effective content model (after group-ref expansion and across nested model groups) must have the same {type definition}. Runs in `compileSchema` AFTER substitution groups are built (NOT inside `resolveRefs`), gated on `errorCount==0` — it consults `schema.substGroups`, so it must run once that map exists. Coverage: (a) complex-type content models (iterated in source line/ordinal order); (b) for each element TERM, the term's substitution-group MEMBERS (`schema.substGroups[term.Name]`) are folded in as implicitly-containable declarations under each member's own name (a head's particle stands in for its members), so a `ref="head"` colliding by name with a different-typed same-named local element is rejected — members' declared types resolve through their head via `resolveDeclaredType`; (c) standalone named `xs:group` definitions (over `schema.groups`, in stable source order via `groupSources` recorded at parse time and merged from import sub-compilers), so a named group no complex type references is still checked. Type identity is by `*TypeDef` pointer (helium shares one pointer per named type and copies the global's type onto a `ref`), with a same-expanded-QName fallback for import-merged duplicates; distinct anonymous inline types are different components and therefore inconsistent. The check is only ever under-strict (a missed violation is safe; it never false-rejects a valid schema). libxml2 does NOT implement this constraint (it is an "URGENT TODO" in `xmlschemas.c`), so the diagnostic uses the existing component-error style rather than mirroring libxml2 wording, and no golden schema trips it.
   - Wildcard overlap detection

7. **Compile result gate:** after linking/checks, `compileSchema` returns `(nil, ErrCompilationFailed)` if `c.errorCount > 0` (fatal diagnostics already delivered to the `ErrorHandler`), otherwise `(schema, nil)`. Sub-compiler `errorCount` is merged into the parent (`compile_imports.go`), so an import/include/redefine fatal also fails the top-level `Compile`. `xslt3` schema-awareness (`compile_schema.go`) maps `ErrCompilationFailed` to `XTSE0220`.

### Validate: Document + Schema → Errors

**Two-pass validation:**

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
validated against the actual type. In `fixedUnionMatches`, when the fixed and
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
length family (`length`/`minLength`/`maxLength`), and `whiteSpace` are NOT in a
union's {applicable facets} set, so `checkFacetApplicability` rejects them at
COMPILE time (`The facet '…' is not allowed.`) — they never reach validation as a
runtime no-op. The union's allowed `pattern`/`enumeration` facets are checked in
the instance active member's value space via `checkFacets` with enumeration
suppressed. The active member for that `checkFacets` call is resolved down to its
LEAF basic member (`fixedUnionActiveMember` descends through nested unions), so a
nested union (`outer=union(inner)`, `inner=union(xs:string)`) resolves to the
leaf type rather than an intermediate union. On an ATOMIC restriction the range facets
(`min`/`maxInclusive`, `min`/`maxExclusive`) apply ONLY to types whose primitive
value space is ORDERED, so `compareForRangeFacet` first gates `builtinLocal` on
the `orderedRangeFacetTypes` allowlist — the numeric leaves (decimal and derived
integers, float, double) AND the date/time/duration family (duration, dateTime,
time, date, and the gregorian g-types). For every NON-ordered leaf — string-family
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
schema error rather than a runtime no-op. `checkFacetSameTypeConsistency` gates EACH
facet-family consistency check to the family's applicable type/variety, so it never
adds a spurious error on top of an applicability rejection: the LENGTH check
(`minLength>maxLength`) runs only on a list variety or a `lengthApplicableTypes`
atomic; the DIGIT check (`fractionDigits>totalDigits`) runs only on a
`decimalFamilyTypes` atomic (so `xs:double`+`totalDigits`/`fractionDigits` reports only
the two "not allowed" errors); the RANGE checks run only on an ordered atomic. It
compares same-type range bounds (`min`/`maxInclusive`, `min`/`maxExclusive`) in the
restricted type's ORDERED VALUE SPACE (`value.CompareFloatFacetBound` for float/double
NaN ordering, else `compareForRangeFacet`), skipping the check on an indeterminate
result, so an inconsistent non-decimal pair like `minInclusive 2021-01-01 >
maxInclusive 2020-01-01` on `xs:date` is rejected. `checkFacetBaseRestriction` compares
each derived range bound against the base bound with the SAME value-space comparator
(gated to ordered atomic; `compareDecimal` only for an unresolved primitive), so a
valid narrowing non-decimal restriction — e.g. base `xs:date` `minInclusive=2021-01-01`
with derived `maxInclusive=2022-01-01` — is no longer false-rejected.

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

**Compile-time IDC checks:** a malformed `xs:selector`/`xs:field` `@xpath` is a fatal schema parser error (`parseIDConstraint` → `reportIDCXPathError`) rather than a silently-dropped `xpath1.Compile` failure that would disable the whole constraint. After all elements are parsed, `checkKeyRefRefers` (in `compileSchema`) resolves every `xs:keyref/@refer` against a schema-wide set of key/unique constraint names (identity-constraint names share one symbol space) and raises a fatal error for an unknown/empty refer. The registry is built by `collectAllIDCs`, which walks EVERY element declaration — not just `schema.elements` (globals) — by recursively descending each global element's/type's/named-group's content model (`idcWalker`, with visited sets on `*ElementDecl`/`*ModelGroup`/`*TypeDef` to bound shared/recursive/circular structures), so a keyref (or the key it refers to) declared on a LOCAL element buried in a content model is checked too. **@refer resolution is schema-wide (the symbol space); keyref VALUE resolution is subtree-scoped** — the two are distinct (see Pass 2 below). A deferred `@refer` error is reported against the constraint's DECLARING file: `IDConstraint.Source` is pinned at parse time in `parseIDConstraint` (`c.includeFile` if inside an include/redefine, else `c.filename` — for an import sub-compiler that is the imported file's display location), and `checkKeyRefRefers` reports with `idc.Source`+`idc.Line` rather than the top-level compiler's filename, so an IMPORTED keyref's dangling-refer error cites the imported schema (where its line number is meaningful), not the importing schema. At validation time, an IDC whose selector/field XPath fails to evaluate is reported as a validity error (`Failed to evaluate identity-constraint '…'`), not swallowed.

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
    xs:integer value space rather than being wrongly accepted as unique.

### Key Data Model

```
Schema { elements, types, groups, attrGroups, globalAttrs, substGroups maps }
ElementDecl { Name QName, Type *TypeDef, MinOccurs/MaxOccurs, Abstract/Nillable, IDCs, Default/Fixed }
TypeDef { ContentType (Empty|Simple|ElementOnly|Mixed), ContentModel *ModelGroup, BaseType, Attributes []*AttrUse, Facets, Variety (Atomic|List|Union) }
ModelGroup { Compositor (Sequence|Choice|All), Particles []*Particle }
IDConstraint { Kind (Unique|Key|KeyRef), Selector/Fields XPath, Refer, Namespaces, Line, Source (declaring file) }
```

## RELAX NG

Files: `relaxng/relaxng.go` (API), `parse.go` (compiler), `validate.go` (engine), `grammar.go` (model)

### Compile: Document → Grammar

1. **Find root** — `<grammar>` or bare pattern (e.g., `<element>`)
2. **Parse grammar content** — process `<start>`, `<define>` elements; handle `combine="choice"/"interleave"`; support `<div>` containers
3. **Parse patterns** (recursive) — element, attribute, group, choice, interleave, optional, zeroOrMore, oneOrMore, ref, parentRef, data, value, list, mixed, text, empty, notAllowed
4. **Resolve references** — copy defines into grammar
5. **Check reference cycles** — detect cycles in `<ref>` bypassing element patterns
6. **Rule checks** — compile-time semantic validation

### Validate: Document + Grammar → Errors

Pattern-matching engine with backtracking:

1. Root element → `validState{seq: [root]}`
2. `validatePattern(grammar.start, state)` dispatches on pattern kind:
   - **Element**: name-class match, consume from seq, validate body (attrs + content)
   - **Attribute**: match against instance attrs
   - **Group**: sequential with backtracking
   - **Choice**: try alternatives, prefer branches making progress
   - **Interleave**: unordered member-by-member matching
   - **ZeroOrMore/OneOrMore/Optional**: repetition with suppressed errors
   - **Ref/ParentRef**: resolve and recurse
   - **Data/Value**: type checking
   - **List**: split text, validate items
3. Element validation: match name, validate attrs, build child list (skip non-content: EntityRef/PI/Comment), validate content, check all attrs+content consumed

### Backtracking Strategy (`backtrackGroupFlexible` / `backtrackGroupNaive`)

When mandatory group child fails:
1. Check if element was consumed (structural vs content error)
2. For each previous flexible child (zeroOrMore/oneOrMore/optional) from nearest to furthest:
   - Try iteration counts from minimum upward to greedy count
   - Re-validate remaining children at each count
   - Keep highest successful count (maximizes consumption — libxml2 semantics)

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
for binary (`hexBinary`/`base64Binary`).

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
