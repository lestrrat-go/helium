# XSLT 3.0 Schema-Awareness Design Document

## 1. Executive Summary

This document describes the design of schema-aware processing in the `xslt3` package. The helium XSLT 3.0 processor previously operated as a Basic XSLT 3.0 processor; W3C test cases tagged with the `schema_aware` feature were skipped. The `feat-xslt3-schema-aware` branch adds the infrastructure necessary to pass these tests.

**Key objectives:**

1. Compile `xsl:import-schema` (file-backed and inline) into the `Stylesheet` struct as `[]*xsd.Schema`.
2. Apply XSD type annotations to result-tree nodes via `xsl:element type=`, `xsl:attribute type=`, and `validation=` attributes on constructors.
3. Thread annotations into `xpath3` evaluation through `WithTypeAnnotations`, enabling typed node-test matching and typed atomization.
4. Implement `type-available()` and update `system-property("is-schema-aware")` to return `"yes"`.
5. Update `xslt3gen` to stop skipping `schema_aware` tests so they run against the implementation.

**Success criterion:** All W3C XSLT 3.0 tests tagged `schema_aware` that are not blocked by an unrelated missing feature (e.g., `streaming`) should pass without a skip reason.

---

## 2. Requirements Analysis

### 2.1 Functional Requirements

1. **FR-1** — `xsl:import-schema` with `schema-location` compiles the referenced XSD file into `Stylesheet.schemas`.
2. **FR-2** — `xsl:import-schema` with an inline `xs:schema` child element compiles the embedded schema.
3. **FR-3** — `xsl:import-schema` with neither attribute nor child schema (namespace-only declaration) is accepted without error.
4. **FR-4** — `xsl:element` and `xsl:attribute` with `type=` compile the QName to a canonical `xs:*` type string stored in `ElementInst.TypeName` / `AttributeInst.TypeName`.
5. **FR-5** — At execution time, `execElement` and `execAttribute` call `ec.annotateNode` / `ec.annotateAttr` to record the annotation in `execContext.typeAnnotations`.
6. **FR-6** — `xsl:copy-of validation="preserve"` propagates type annotations from the source node to the copy in the result tree.
7. **FR-7** — `execContext.newXPathContext` / `baseXPathContext` pass `typeAnnotations` to `xpath3` via `WithTypeAnnotations` when the map is non-empty.
8. **FR-8** — `xpath3` node-test matching for `element(name, type)` and `attribute(name, type)` checks the type annotation from `ec.typeAnnotations` against `isSubtypeOf`.
9. **FR-9** — Typed atomization (`atomizeItemTyped`) reads the type annotation and casts the string value to the annotated atomic type.
10. **FR-10** — `type-available(name)` returns `true` for all built-in XSD types and for named types in imported schemas.
11. **FR-11** — `system-property("xsl:is-schema-aware")` returns `"yes"`.
12. **FR-12** — `xslt3gen` removes the `schema_aware` skip rule so that generated test files include schema-aware test cases.

### 2.2 Non-Functional Requirements

- **NFR-1** — The `typeAnnotations` map is shared by reference from `execContext` into `xpath3` evaluation contexts; it must not be cloned on every XPath call.
- **NFR-2** — Schema compilation happens at stylesheet compile time; it must not repeat on each `Transform` call.
- **NFR-3** — The `xpath3` package must never import `xsd`. Type annotations flow in as a `map[helium.Node]string`; the `xs:*` prefix form is the canonical representation.
- **NFR-4** — `NodeItem.TypeAnnotation` field is optional and only set when a node is materialized from a schema-validated context; nodes in a Basic-mode transformation remain `xs:untypedAtomic`.

### 2.3 Constraints and Assumptions

- `xpath3` MUST NOT import `xsd`. This is enforced by the dependency rules in `dependencies.md`.
- Schema-aware processing is driven entirely by `xsl:import-schema` plus explicit `type=` / `validation=` attributes. The processor does not implicitly validate source documents unless `validation=` is specified.
- Only file-backed and inline `xs:schema` child schemas are supported in the initial phase; `namespace-only` `xsl:import-schema` is silently accepted.
- The `xsd.Schema.LookupType(local, ns)` API is used for user-defined type resolution in `type-available()`.

### 2.4 Out of Scope

- Automatic validation of source documents against imported schemas.
- `xs:schema` inline schema extraction from inside `xsl:import-schema` is a gap (see Section 9).
- Deriving type annotations from a separate schema-validated source document (`document()` + `validate()`).
- The `xsl:validate` instruction (not part of XSLT 3.0 Basic or Schema-Aware spec).
- Static type checking (no type inference at compile time).

---

## 3. Technical Design

### 3.1 Architecture Overview

```
Stylesheet Compilation
  xsl:import-schema ──► xsd.CompileFile / xsd.Compile ──► Stylesheet.schemas []*xsd.Schema
  xsl:element type= ──► resolveXSDTypeName ──► ElementInst.TypeName string
  xsl:attribute type= ──► resolveXSDTypeName ──► AttributeInst.TypeName string
  default-validation ──► Stylesheet.defaultValidation string

XSLT Transformation  (execContext)
  execElement ──► annotateNode(elem, TypeName) ──► typeAnnotations map[helium.Node]string
  execAttribute ──► annotateAttr(elem, TypeName, ...) ──► typeAnnotations map[helium.Node]string
  execCopyOf(validation=preserve) ──► copyTypeAnnotations(srcNode) ──► typeAnnotations (transfer)
  newXPathContext / baseXPathContext ──► WithTypeAnnotations(ctx, typeAnnotations)

XPath 3.1 Evaluation  (evalContext)
  evalConfig.typeAnnotations ──► nodeTypeAnnotation(n, ec)
  matchNodeTest(ElementTest / AttributeTest) ──► isSubtypeOf(ann, target)
  atomizeItemTyped(NodeItem) ──► CastFromString(s, ann)
  matchItemType(ElementTest / AttributeTest) ──► ni.TypeAnnotation or typeAnnotations lookup

Functions
  fnTypeAvailable(name) ──► IsKnownXSDType(resolved) || schema.LookupType(local, ns)
  fnSystemProperty("is-schema-aware") ──► "yes"
```

### 3.2 Detailed Component Specifications

#### 3.2.1 `Stylesheet` struct (`xslt3/stylesheet.go`)

**Already implemented.**

```go
type Stylesheet struct {
    // ... existing fields ...
    schemas           []*xsd.Schema   // imported schemas (xsl:import-schema)
    defaultValidation string          // "strict", "lax", "preserve", "strip"
}
```

`schemas` accumulates compiled `*xsd.Schema` values in `compileImportSchema`. `defaultValidation` is set from the `default-validation` attribute on the root `xsl:stylesheet` element.

#### 3.2.2 `compileImportSchema` (`xslt3/compile.go`)

**Partially implemented. Gap: inline schema not supported.**

Current implementation handles `schema-location=` (file-backed schemas). The URI is resolved relative to the stylesheet base URI by joining it with `filepath.Dir(c.baseURI)`.

```go
func (c *compiler) compileImportSchema(elem *helium.Element) error {
    schemaLoc := getAttr(elem, "schema-location")
    if schemaLoc == "" {
        // Gap: inline xs:schema child is not parsed
        return nil
    }
    // resolve + xsd.CompileFile → c.stylesheet.schemas
}
```

**Gap: inline schema support.** When `schema-location=""` and the element has an `xs:schema` child, that child should be compiled. See Section 9.

#### 3.2.3 `ElementInst` and `AttributeInst` (`xslt3/instruction.go`)

**Already implemented.**

```go
type ElementInst struct {
    // ...
    TypeName string // XSD type annotation, e.g., "xs:integer"
}

type AttributeInst struct {
    // ...
    TypeName string // XSD type annotation, e.g., "xs:ID"
}
```

`TypeName` is populated during compilation by `resolveXSDTypeName(getAttr(elem, "type"), c.nsBindings)`.

#### 3.2.4 `resolveXSDTypeName` (`xslt3/compile.go`)

**Already implemented.** Normalizes QName references:

- `xs:ID` → `xs:ID`
- `xsd:integer` → `xs:integer`
- `Q{http://www.w3.org/2001/XMLSchema}ID` → `xs:ID`
- Any other prefix resolved via `nsBindings`; if prefix maps to XSD NS, normalizes to `xs:`.

#### 3.2.5 `CopyInst` and `CopyOfInst` (`xslt3/instruction.go`)

**Already implemented.**

```go
type CopyInst struct {
    Validation string // "strict", "lax", "preserve", "strip"
    // ...
}
type CopyOfInst struct {
    Validation string
    Select     *xpath3.Expression
}
```

`Validation` is read from the `validation=` attribute. When empty, the `defaultValidation` on the stylesheet is consulted at runtime.

#### 3.2.6 `execContext.typeAnnotations` (`xslt3/execute.go`)

**Already implemented.**

```go
type execContext struct {
    // ...
    typeAnnotations map[helium.Node]string // node → xs:... type annotation
}
```

The map is lazily allocated on first write by `annotateNode`.

**`annotateNode(node helium.Node, typeName string)`** — Sets `typeAnnotations[node] = typeName`. No-op when `typeName` is empty.

**`annotateAttr(elem *helium.Element, typeName, localName, nsURI, value string)`** — Locates the attribute node on the element and calls `annotateNode`. Also registers `xs:ID` with `resultDoc.RegisterID` for `id()` lookup.

**`copyTypeAnnotations(srcNode helium.Node)`** — Currently a stub (copies `srcNode` annotation to itself; doesn't propagate to the copied result node). This is a known gap described in Section 9.

#### 3.2.7 XPath Context Wiring (`xslt3/execute.go`)

**Already implemented.** Both `newXPathContext` and `baseXPathContext` contain:

```go
if len(ec.typeAnnotations) > 0 {
    ctx = xpath3.WithTypeAnnotations(ctx, ec.typeAnnotations)
}
```

The map is passed by reference (not cloned) per NFR-1.

#### 3.2.8 `evalConfig.typeAnnotations` (`xpath3/context.go`)

**Already implemented.**

```go
type evalConfig struct {
    // ...
    typeAnnotations map[helium.Node]string
}
```

Populated via `WithTypeAnnotations(ctx, annotations)`.

#### 3.2.9 `nodeTypeAnnotation` (`xpath3/eval_path.go`)

**Already implemented.**

```go
func nodeTypeAnnotation(n helium.Node, ec *evalContext) string {
    if ec == nil || ec.typeAnnotations == nil {
        return ""
    }
    return ec.typeAnnotations[n]
}
```

Used by `matchNodeTest` for `ElementTest` and `AttributeTest` when `test.TypeName != ""`.

#### 3.2.10 `NodeItem.TypeAnnotation` field (`xpath3/types.go`)

**Already implemented.**

```go
type NodeItem struct {
    Node           helium.Node
    TypeAnnotation string // optional xs:... type annotation (schema-aware)
}
```

`AtomizeItem` prefers `TypeAnnotation` over the map lookup when the field is non-empty. `atomizeItemTyped` (used internally in `eval_path.go`) checks both the field and `ec.typeAnnotations`.

**Gap:** `NodeItem` produced during normal XPath location path evaluation is created with `NodeItem{Node: n}` — the `TypeAnnotation` field is never populated from the map during path evaluation. The `nodeTypeAnnotation` lookup from the map is used for node test matching, but `NodeItem` values produced in sequences still carry an empty `TypeAnnotation`. This is generally acceptable because `atomizeItemTyped` already checks both, but it means that a `NodeItem` passed out of the XPath evaluator as a `Result.Sequence()` item won't carry an annotation for downstream use. This is a minor gap for external consumers.

#### 3.2.11 `IsKnownXSDType` (`xpath3/types.go`)

**Already implemented.**

```go
func IsKnownXSDType(name string) bool {
    if _, ok := xsdTypeParent[name]; ok { return true }
    return name == TypeAnyAtomicType || name == TypeNumeric
}
```

Used by `fnTypeAvailable` to check built-in XSD types without importing `xsd`.

#### 3.2.12 `fnTypeAvailable` (`xslt3/functions.go`)

**Already implemented.**

The function:
1. Atomizes the argument.
2. Resolves the QName to canonical `xs:...` form using `resolveQName` + XSD namespace normalization.
3. Calls `xpath3.IsKnownXSDType(resolved)` for built-in types.
4. Falls through to `schema.LookupType(local, ns)` for each imported schema.

#### 3.2.13 `fnSystemProperty` (`xslt3/functions.go`)

**Already implemented.**

```go
case "is-schema-aware":
    return xpath3.SingleString("yes"), nil
```

Returns `"yes"` unconditionally. This is correct: any stylesheet importing this package has schema-aware capability.

#### 3.2.14 `xslt3gen` feature filter (`tools/xslt3gen/main.go`)

**Gap: `schema_aware` is not in the `featureSupported` allow-list — it falls through to `return true`, which means it is already NOT being skipped by the generator.**

Wait — reading the current code:

```go
func featureSupported(feature string) bool {
    switch feature {
    case "streaming",
        "higher_order_functions",
        "backwards_compatibility",
        "dynamic_evaluation",
        "Saxon-PE", "Saxon-EE":
        return false
    }
    return true  // everything else, including schema_aware, is supported
}
```

`schema_aware` is already treated as supported by the generator. However, the generated test files contain `Skip: "unsupported feature: schema_aware"` lines. This can only happen if the `schema_aware` skip reason was introduced somewhere else, or if the generator was invoked with the old filter at the time the files were generated. The generated files in the worktree still contain the skip annotations because those files pre-date the generator change.

**Action required:** The generated test files (`w3c_*_gen_test.go`) need to be regenerated after the `featureSupported` function correctly allows `schema_aware`. Since the function already returns `true` for `schema_aware`, the fix is simply to re-run `xslt3gen`. The skip strings currently in the generated files are stale.

### 3.3 Data Design

No new persistent data structures are required. The runtime annotation map is:

```
map[helium.Node]string
```

Where the key is a pointer-comparable `helium.Node` interface and the value is a canonical XSD type string in `xs:T` form.

The map lives on `execContext` for the duration of a single `Transform` call and is discarded afterward.

### 3.4 API Specifications

#### New public API

```go
// xpath3/context.go
func WithTypeAnnotations(ctx context.Context, annotations map[helium.Node]string) context.Context

// xpath3/types.go
func IsKnownXSDType(name string) bool
```

Both are already implemented.

#### No changes to `xslt3` public API

The `xslt3.Transform`, `xslt3.CompileStylesheet` etc. signatures are unchanged. Schema awareness is determined purely by the stylesheet content (`xsl:import-schema`, `type=`, `validation=` attributes).

### 3.5 State Management

- `Stylesheet.schemas` — populated at compile time, immutable for the lifetime of the `Stylesheet`.
- `execContext.typeAnnotations` — created lazily during transformation, lives for one `Transform` call.
- `evalConfig.typeAnnotations` — a pointer alias into `execContext.typeAnnotations` passed to each XPath call; not separately owned.

---

## 4. Implementation Guidelines

### 4.1 Coding Standards Compliance

- Follow the `getAttr` / `compileXPath` / `resolveQName` patterns already in `compile.go`.
- New helper functions in `compile.go` follow the `(c *compiler) compileXxx` receiver pattern.
- Runtime helpers on `execContext` use `(ec *execContext) xxxYyy` receivers.
- Type name constants use the `TypeXxx = "xs:xxx"` format from `xpath3/types.go`.
- No new exported symbols in `xpath3` unless strictly necessary.
- Inline schema compilation re-uses `xsd.Compile(ctx, doc)` (not `xsd.CompileFile`).

### 4.2 Error Handling

- `compileImportSchema` returns a wrapped `fmt.Errorf("xsl:import-schema: ...")` on schema compilation failure; this surfaces as a compile-time error, not a runtime error.
- `fnTypeAvailable` never returns an error; it returns `false` for unresolvable names.
- `annotateNode` / `annotateAttr` are silent no-ops when `typeName == ""`.
- `copyTypeAnnotations` should not return an error; failed transfers are silently ignored.

### 4.3 Testing Strategy

- The primary test vehicle is the W3C XSLT 3.0 test suite. After regenerating `w3c_*_gen_test.go` with `go run ./tools/xslt3gen`, the `schema_aware` tests will run instead of being skipped.
- Unit tests for `fnTypeAvailable` can be added directly in `xslt3_test.go` following the existing test harness pattern.
- The inline-schema gap (FR-2) blocks a subset of `import-schema-xxx` tests; those tests may remain skipped with a more specific reason (`"inline schema not yet supported"`) until the gap is filled.

---

## 5. Detailed Task Breakdown

### Task 1: Regenerate W3C test files [Simple]

**File:** `xslt3/w3c_*_gen_test.go` (all 8 generated files)

**Action:** Run `go run ./tools/xslt3gen` from the worktree root. This regenerates all 8 test files. The `featureSupported` function already returns `true` for `schema_aware`, so the regenerated files will have `Skip: ""` for schema-aware tests that have no other blocking dependency.

**Verification:** `grep -c 'schema_aware' xslt3/w3c_decl_gen_test.go` should return `0` after regeneration.

**Dependencies:** None (xslt3gen is already correct).

**Complexity:** Simple

---

### Task 2: Fix `copyTypeAnnotations` to actually propagate annotations [Moderate]

**File:** `xslt3/execute.go`

**Problem:** The current implementation of `copyTypeAnnotations` reads the source node's annotation but does nothing with it. In `execCopyOf`, after `copyNodeToOutput(v.Node)`, the newly-added node in the result document is a different pointer from `v.Node`. The annotation for the source is present in `ec.typeAnnotations` keyed by the source pointer, but the new result node pointer has no entry.

**Fix:** After `copyNodeToOutput`, retrieve the last-added node from the current output frame and copy the annotation:

```go
func (ec *execContext) execCopyOf(ctx context.Context, inst *CopyOfInst) error {
    // ... existing eval ...
    preserve := inst.Validation == "preserve" ||
        (inst.Validation == "" && ec.stylesheet.defaultValidation == "preserve")

    for _, item := range result.Sequence() {
        switch v := item.(type) {
        case xpath3.NodeItem:
            if err := ec.copyNodeToOutput(v.Node); err != nil {
                return err
            }
            if preserve {
                ec.transferAnnotations(v.Node)  // NEW
            }
        // ...
        }
    }
}

// transferAnnotations copies the type annotation for srcNode (and its
// descendants) to the most recently appended result-tree node.
func (ec *execContext) transferAnnotations(srcNode helium.Node) {
    if ec.typeAnnotations == nil {
        return
    }
    ann, ok := ec.typeAnnotations[srcNode]
    if !ok {
        return
    }
    out := ec.currentOutput()
    // The last child of out.current corresponds to the copied node.
    last := out.current.LastChild()
    if last != nil && ann != "" {
        ec.annotateNode(last, ann)
    }
}
```

**Note:** Deep copy annotation transfer (for subtrees) requires recursive traversal. For the initial pass, shallow transfer (only the root of the copied node) satisfies most test cases. The recursive form can be added when test failures demand it.

**Complexity:** Moderate

---

### Task 3: Implement inline schema support in `compileImportSchema` [Moderate]

**File:** `xslt3/compile.go`

**Problem:** When `schema-location` is absent, the element may contain an inline `xs:schema` child. The current code returns early with `nil` without checking for it.

**Fix:**

```go
func (c *compiler) compileImportSchema(elem *helium.Element) error {
    schemaLoc := getAttr(elem, "schema-location")
    if schemaLoc != "" {
        // Existing file-based path
        uri := resolveSchemaURI(schemaLoc, c.baseURI)
        ctx := context.Background()
        schema, err := xsd.CompileFile(ctx, uri)
        if err != nil {
            return fmt.Errorf("xsl:import-schema: cannot compile %q: %w", uri, err)
        }
        c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
        return nil
    }

    // Look for inline xs:schema child
    for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
        childElem, ok := child.(*helium.Element)
        if !ok {
            continue
        }
        if childElem.LocalName() == "schema" &&
            childElem.URI() == "http://www.w3.org/2001/XMLSchema" {
            // Build a minimal document wrapping the inline schema
            inlineDoc := helium.NewDocument()
            copied := helium.CopyNode(childElem, inlineDoc)
            inlineDoc.AddChild(copied)
            ctx := context.Background()
            schema, err := xsd.Compile(ctx, inlineDoc)
            if err != nil {
                return fmt.Errorf("xsl:import-schema: cannot compile inline schema: %w", err)
            }
            c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
            return nil
        }
    }
    // namespace-only declaration — no schema to compile, accepted silently
    return nil
}
```

**Dependencies:** Task 1 (regenerated tests reveal which tests need this).

**Complexity:** Moderate

---

### Task 4: Populate `NodeItem.TypeAnnotation` during path evaluation [Simple]

**File:** `xpath3/eval_path.go`

**Problem:** `evalLocationPath` and the step evaluation functions build result sequences using `NodeItem{Node: n}` without checking the type annotation map. The annotation map is consulted by `matchNodeTest` but the resulting `NodeItem` in the sequence carries no annotation. This does not affect matching but affects typed atomization in downstream stages that use `AtomizeItem` (which checks `ni.TypeAnnotation`) rather than `atomizeItemTyped` (which also checks the map).

**Fix:** Populate `TypeAnnotation` when building the result sequence:

```go
func evalLocationPath(ec *evalContext, lp *LocationPath) (Sequence, error) {
    // ... existing traversal ...
    result := make(Sequence, len(nodes))
    for i, n := range nodes {
        ni := NodeItem{Node: n}
        if ec.typeAnnotations != nil {
            ni.TypeAnnotation = ec.typeAnnotations[n]
        }
        result[i] = ni
    }
    return result, nil
}
```

Apply the same pattern in `evalStepWithPredicates` and `evalStepNoPredicates` when materializing the final result sequence (not intermediate node lists, which stay as `[]helium.Node`).

**Complexity:** Simple

---

### Task 5: Validate `type=` against imported schemas at compile time [Simple]

**File:** `xslt3/compile_instructions.go`

**Current behavior:** `compileElement` and `compileAttribute` accept any `type=` value, including unknown types, and store the raw normalized string. No validation against `c.stylesheet.schemas` occurs.

**Desired behavior:** If the type name does not correspond to a built-in XSD type and is not present in any imported schema, emit a static error `XTTE1520` (for element) or `XTTE1510` (for attribute).

```go
func (c *compiler) validateTypeAttr(typeName string) error {
    if typeName == "" {
        return nil
    }
    if xpath3.IsKnownXSDType(typeName) {
        return nil
    }
    // Check imported schemas
    local := typeName
    ns := "http://www.w3.org/2001/XMLSchema"
    if strings.HasPrefix(typeName, "xs:") {
        local = typeName[3:]
    }
    for _, schema := range c.stylesheet.schemas {
        if _, ok := schema.LookupType(local, ns); ok {
            return nil
        }
    }
    return staticError(errCodeXTTE1520, "unknown type %q in type= attribute", typeName)
}
```

Call this after `inst.TypeName = resolveXSDTypeName(...)` in `compileElement` and `compileAttribute`.

**Note:** This is a quality-of-implementation improvement. Many test cases use built-in `xs:*` types that are already in `IsKnownXSDType`, so this will not change pass/fail status for those. It primarily matters for user-defined type references.

**Complexity:** Simple

---

### Task 6: Handle `validation=` on `xsl:copy` at runtime [Simple]

**File:** `xslt3/execute_instructions.go`

**Current state:** `execCopy` exists but does not check `inst.Validation` or `ec.stylesheet.defaultValidation` for annotation preservation. It should mirror the pattern already used in `execCopyOf`.

**Fix:** After `copyNodeToOutput` inside `execCopy`, call `ec.transferAnnotations(ec.contextNode)` when `preserve` is in effect.

**Complexity:** Simple

---

### Task 7: Update `packages.md` cache doc [Simple]

**File:** `.claude/docs/packages.md`

The `xslt3/` entry already documents schema awareness accurately. Verify it reflects the current state after the above tasks.

**Complexity:** Simple

---

## 6. Security Considerations

- Inline `xs:schema` may reference external DTDs or entities. `xsd.Compile` uses the helium parser which enforces `ParseNoXXE` by default; this is sufficient.
- The `schema-location` URI is resolved relative to the stylesheet base URI and must be a filesystem path or absolute URI. No URL fetching occurs at compile time beyond what `xsd.CompileFile` already does.
- Type annotations are stored as `string` values keyed by `helium.Node` pointers. There is no risk of annotation injection from untrusted input because annotations are set only via XSLT stylesheet instructions, not from source document data.

---

## 7. Performance Considerations

- **Schema compilation is O(schema size) once at compile time.** The compiled `*xsd.Schema` objects are retained on the `Stylesheet` and reused across all `Transform` calls.
- **`typeAnnotations` map** grows at most proportionally to the number of result-tree nodes that carry type annotations. Most transformations produce far fewer annotated nodes than total nodes, so the map stays small.
- **`WithTypeAnnotations` passes the map by reference** (not cloned). This means the `evalConfig` in `xpath3` shares the live map with `execContext`. Since `xpath3` evaluation is synchronous within a single `Transform` call and the map is only written to before evaluation begins (or, during `execElement` / `execAttribute`, which happen between XPath evaluations), there is no concurrent mutation risk.
- **`nodeTypeAnnotation` is a single map lookup** — O(1).

---

## 8. Rollout Plan

1. **Phase 1 — Implement:** Implement all tasks from Section 5 and regenerate test files with `go run ./tools/xslt3gen`.
2. **Phase 2 — Iterate on test failures:** This is a loop, not a single pass:
   1. Run the schema-aware tests (e.g., `go test ./xslt3/ -count=1 -run <test-pattern> -timeout 120s`).
   2. Identify the **first** failing test.
   3. Diagnose the root cause.
   4. Either **fix** the issue in code, or **mark won't-fix** by adding a specific skip reason to the test generator (not a blanket skip — the reason must explain *why* this specific test cannot pass).
   5. **Commit** the fix or skip.
   6. Go back to step 2.1 — run the tests again.
   7. Repeat until there are no more failing schema-aware tests.
3. **Phase 3 — Merge:** Merge `feat-xslt3-schema-aware` back into `feat-xslt3`. All changes land on the parent feature branch.
4. **Phase 4 — Cleanup:** Remove the `feat-xslt3-schema-aware` worktree and delete the branch after merge.

**No feature flags required.** Schema awareness is always available when any `xsl:import-schema` is present. The `is-schema-aware` property returning `"yes"` is correct.

### Completion Criteria

- All schema-aware W3C tests either pass or have an explicit, justified skip reason (not a blanket "unsupported feature" skip).
- No regressions in non-schema-aware tests.
- Changes merged to `feat-xslt3` branch.
- Worktree `.worktrees/feat-xslt3-schema-aware` and branch `feat-xslt3-schema-aware` removed.

---

## 9. Open Questions and Risks

### Gap 1: Inline schema compilation (FR-2)

`compileImportSchema` currently skips when `schema-location=""`. The inline schema path (`xs:schema` child) is not compiled. This blocks test cases like `import-schema-001` through `import-schema-003` that use inline schemas. **Risk: Medium.** Task 3 addresses this.

The implementation requires building a synthetic `helium.Document` containing a clone of the `xs:schema` element, then calling `xsd.Compile`. The `helium.CopyNode` and `helium.NewDocument` APIs are available for this. However, `helium.NewDocument` is not currently part of the public API — verify if it needs to be added or if the xsd package can be called differently.

### Gap 2: `copyTypeAnnotations` stub (FR-6)

The current `copyTypeAnnotations` implementation is a no-op. Task 2 provides a shallow fix. Deep subtree annotation transfer (for recursive copy-of of element trees with type annotations on descendants) requires traversing the copied subtree in parallel with the source subtree. This is complex to implement correctly and may not be required by the test suite's initial schema-aware cases.

### Gap 3: `NodeItem.TypeAnnotation` field not populated in sequences

Addressed by Task 4. Low risk; `atomizeItemTyped` already handles the map-based fallback.

### Gap 4: Generated test files contain stale skip annotations

The `w3c_*_gen_test.go` files in the current worktree were generated with an older version of the generator (or the generator was re-run pointing at the `feat-xslt3` worktree's testdata, not the `feat-xslt3-schema-aware` worktree). The fix is to re-run `xslt3gen`. **Risk: None** once regenerated.

### Gap 5: `featureSupported("schema_aware")` — generator already allows it

The current `featureSupported` implementation in `tools/xslt3gen/main.go` returns `true` for `schema_aware` (it is not in the explicitly-unsupported list). This means the generator never emits `Skip: "unsupported feature: schema_aware"` for current runs. The stale skip strings in the existing generated files must be from an older generator run. Regeneration will clear them.

### Open Question: `helium.NewDocument()` availability

Task 3 requires creating a new `helium.Document` to wrap an inline `xs:schema`. Check whether `helium.NewDocument()` is exported. If not, use `helium.Parse(ctx, []byte("<placeholder/>"))` and replace the root, or serialize the inline schema element and re-parse it with `helium.Parse`.

### Open Question: `xsd.LookupType` signature

`fnTypeAvailable` currently calls `schema.LookupType(local, ns)` where `ns = "http://www.w3.org/2001/XMLSchema"`. Verify that the `xsd.Schema.LookupType` method signature matches and that it returns user-defined complex and simple types (not just built-in types).

### Risk: Test failures from incorrect annotation propagation

If `copyTypeAnnotations` incorrectly transfers an annotation (wrong node pointer), it could corrupt node-test matching in subsequent XPath expressions. The current stub (no-op) is safe in that it cannot produce false positives. Task 2's fix must be carefully tested before being activated.
