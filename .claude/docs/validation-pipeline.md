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
4. **Process includes/imports** — load `xs:include`/`xs:import`/`xs:redefine`, merge declarations. Nested-schema document loads go through `compileConfig.fsys` (`fs.ReadFile(c.fsys, path)`), an injectable `fs.FS` set via `Compiler.FS(...)`; it defaults to `iofs.PermissiveRoot` (`os.Open`) and is propagated to sub-compilers. `xslt3` injects a resolver-backed `fs.FS` (`schemaResolverFS`) so nested includes inside a resolver-loaded schema obey the same default-deny `URIResolver` policy as the top-level load. Schema-location resolution is **URI-aware** and lives in a **single canonical helper**, `xsd.ResolveSchemaURI(ref, base) (string, error)` (`xsd/resolve_uri.go`), shared by both the xsd nested-include path and `xslt3`'s schema loader so the two layers cannot drift. `validateSchemaPath` is a thin wrapper over it. It keys off whether `ref`/`base` carries a URI scheme (`xsd.URIScheme`, the **one** scheme-detector for both packages — multi-char scheme required so Windows drive letters and bare OS paths stay local): an **absolute-URI** `schemaLocation` (e.g. a cross-host `https://cdn.example.com/part.xsd`) is passed through **unchanged** — never `filepath.Join`ed, which would collapse `//` and drop the host; a **relative** location against a **URI base** resolves via `net/url` `ResolveReference` (RFC 3986), keeping authority intact **and re-applying the base's `OmitHost` flag** when the base had no authority (so `mem:/schemas/main.xsd` + `part.xsd` → `mem:/schemas/part.xsd`, never `mem:///schemas/part.xsd`, while canonical `file:///...` bases keep their `///`); a genuine **local** base/location keeps the historical `filepath.Join` + `..`-escape guard (the only branch that can return an error). The import sub-compiler's `baseDir` is `schemaBaseDir(path)` (the full URI for URI bases, `filepath.Dir` for local). Because resolution happens while base and raw `schemaLocation` are still separate, the name reaching the FS is the **canonical** nested URI, so `schemaResolverFS.Open` forwards it verbatim (no string repair of a collapsed name). `xslt3`'s `resolveSchemaURI` delegates the absolute-URI and URI-base cases to `xsd.ResolveSchemaURI` and only handles its own local **file**-base case (xslt3's base is a full file URI/path, not a directory); it seeds the xsd `BaseDir` via `schemaCompileBaseDir(uri)` (full URI when scheme present, `filepath.Dir` otherwise). **targetNamespace match (src-import / src-include):** `loadImport` rejects the located schema when its `targetNamespace` differs from the `namespace` declared on `<xs:import>` — a present `namespace` requires that exact TNS, an absent `namespace` requires the imported schema to have no TNS (so a schema imported as one namespace cannot silently contribute another's declarations). `loadInclude`/`loadRedefine` enforce the analogous include rule (included TNS must equal the including schema's, modulo chameleon includes with no TNS). Both raise a fatal `Schemas parser error` and stop merging that document.
5. **Resolve references** — resolve all QName refs (types, base types, groups, attr groups, union members), build substitution group maps, detect circular substitution
6. **Constraint checks** (when errorCount == 0):
   - `checkFinalOnTypes()` — final attribute enforcement
   - `checkFinalOnSubstGroups()` — substitution group final
   - `checkUPA()` — Unique Particle Attribution (content model determinism)
   - Wildcard overlap detection

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

Enumeration facets are compared in value space, not raw lexical text. A value is
a member if it lexically equals a member OR value-compares equal to one (e.g.
decimal `5.0`≡`5`, boolean `1`≡`true`, float `1.50`≡`1.5`, equal dateTimes in
different timezones). For float/double, NaN equals NaN for enumeration (but
remains incomparable for min/max ordering). QName/NOTATION enumeration resolves
both instance and facet lexical QNames against their respective in-scope
namespaces. Value-space comparison is restricted to an allowlist of numeric,
boolean, date/time, and binary builtins (`enumValueSpaceTypes`); hexBinary and
base64Binary compare by decoded octets (so `"0A"`≡`"0a"`). String-family and
anyURI types stay lexical-only (their value space equals their whitespace-
processed lexical space), so a numeric-looking string enum `"5"` does not accept
`"5.0"`.

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

**Pass 2 — Identity Constraints** (`validateIDConstraints` via second `helium.Walk()`):
- For elements with IDCs (xs:unique, xs:key, xs:keyref):
  1. Evaluate selector XPath → node set
  2. For each selected node, evaluate field XPaths → collect key-sequences
  3. Check unique/key: all key-sequences must be unique
  4. Check keyref: all key-sequences must exist in referenced constraint table
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
    canonicalized via its declared simple type (`resolveFieldBuiltinLocal` →
    `value.CanonicalKey`) before map-key use, so `5`/`+5`/`05` collide for
    xs:integer. Attribute-field type resolution (`attrUseTypeDef`) mirrors the
    content validator's `validationContext.attrUseType`: an inline anonymous
    `<xs:simpleType>` (`au.Type`) is preferred over the named `au.TypeName`
    reference, for both complex-type attribute uses and global attributes, so
    keys on inline-typed attributes canonicalize too. Raw values are retained
    for error display; fields whose type cannot be resolved fall back to
    raw-string comparison.

### Key Data Model

```
Schema { elements, types, groups, attrGroups, globalAttrs, substGroups maps }
ElementDecl { Name QName, Type *TypeDef, MinOccurs/MaxOccurs, Abstract/Nillable, IDCs, Default/Fixed }
TypeDef { ContentType (Empty|Simple|ElementOnly|Mixed), ContentModel *ModelGroup, BaseType, Attributes []*AttrUse, Facets, Variety (Atomic|List|Union) }
ModelGroup { Compositor (Sequence|Choice|All), Particles []*Particle }
IDConstraint { Kind (Unique|Key|KeyRef), Selector/Fields XPath, Refer, Namespaces }
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

### Validate: Document + Schema → Errors

1. Create XPath context with schema's namespaces
2. For each pattern/rule: evaluate `contextExpr` against document root → node set
   - If the context XPath **errors at evaluation**, surface an `XPath error : ...` diagnostic and mark the document invalid (the rule's assertions can't be checked, so it is not silently skipped)
3. For each context node:
   - Bind `<let>` variables (accumulated, later lets see earlier ones)
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
