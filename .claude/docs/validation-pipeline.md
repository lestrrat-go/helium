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
4. **Process includes/imports** — load `xs:include`/`xs:import`/`xs:redefine`, merge declarations
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

QName/NOTATION simple-value validation compares enumeration facets in value
space, not raw prefix text. Instance lexical QNames are resolved against the
instance node's in-scope namespaces; facet lexical QNames are resolved against
the schema facet's in-scope namespaces.

**Pass 2 — Identity Constraints** (`validateIDConstraints` via second `helium.Walk()`):
- For elements with IDCs (xs:unique, xs:key, xs:keyref):
  1. Evaluate selector XPath → node set
  2. For each selected node, evaluate field XPaths → collect key-sequences
  3. Check unique/key: all key-sequences must be unique
  4. Check keyref: all key-sequences must exist in referenced constraint table
  - XPath uses namespace context from schema, not instance

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

### Backtracking Strategy (`backtrackGroupFlexible`)

When mandatory group child fails:
1. Check if element was consumed (structural vs content error)
2. For each previous flexible child (zeroOrMore/oneOrMore/optional) from nearest to furthest:
   - Try iteration counts from minimum upward to greedy count
   - Re-validate remaining children at each count
   - Keep highest successful count (maximizes consumption — libxml2 semantics)

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
3. For each context node:
   - Bind `<let>` variables (accumulated, later lets see earlier ones)
   - Create rule-specific XPath context with variables
4. For each test:
   - Evaluate XPath, convert to boolean
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
