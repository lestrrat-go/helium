<!-- Agent-consumed file. Keep terse, unambiguous, machine-parseable. -->

# Helium

Go libxml2 implementation.

## XPath 3.1 — XSD Version

The xpath3 package targets **XSD 1.1 only**. This means `+INF` is a valid lexical form for xs:double and xs:float, and xs:dateTimeStamp is a recognized type. QT3 tests with `dependency type="xsd-version" value="1.0"` are skipped.

## Pre-Read Rules

Read the linked doc BEFORE working in that area. No exceptions.

| Trigger | Doc |
|---------|-----|
| Package purpose, API, files | `.claude/docs/packages.md` |
| Cross-package imports | `.claude/docs/dependencies.md` |
| Feature status, test counts, known gaps, ParseOption support | `.claude/docs/libxml2-parity.md` |
| Writing/running tests, golden files, test data, helpers | `.claude/docs/testing.md` |
| Error types, format strings, ErrorHandler, ValidateError | `.claude/docs/error-formatting.md` |
| Parse pipeline, encoding, entities, SAX→DOM, push parser | `.claude/docs/parser-internals.md` |
| DOM node hierarchy, struct fields, namespace/attr storage | `.claude/docs/node-types.md` |
| XSD/RELAX NG/Schematron compile→validate flow | `.claude/docs/validation-pipeline.md` |
| heliumlint CLI flags, pipeline, exit codes | `.claude/docs/heliumlint.md` |
| XPath 3.1 design overview, constraints, sub-doc index | `.claude/docs/xpath3-design.md` |
| XPath 3.1 architecture, file layout, internal/xpath | `.claude/docs/xpath3-architecture.md` |
| XPath 3.1 public API, Context, Result, errors | `.claude/docs/xpath3-api.md` |
| XPath 3.1 Item/Sequence/Map/Array type system | `.claude/docs/xpath3-types.md` |
| XPath 3.1 lexer, parser, AST nodes | `.claude/docs/xpath3-parser.md` |
| XPath 3.1 evaluator, comparison, casting | `.claude/docs/xpath3-eval.md` |
| XPath 3.1 function system, built-in categories | `.claude/docs/xpath3-functions.md` |
| XPath 3.1 phased tasks, dependencies, risks | `.claude/docs/xpath3-tasks.md` |
| Saxon-HE source layout (reference) | `.claude/docs/saxon-layout.md` |

## Cache Maintenance

These docs cache repository state. Still read source before modifying code.

1. When your changes affect a doc below, update it in the same commit.
2. If you notice any doc is wrong or stale — even on an unrelated task — fix it immediately.

| Doc | Update trigger |
|-----|----------------|
| `packages.md` | Public API, package, or key file changes |
| `dependencies.md` | Inter-package import changes |
| `libxml2-parity.md` | Test count, parser limitation, feature, or ParseOption changes |
| `testing.md` | Test data layout, helper, env var, or test pattern changes |
| `error-formatting.md` | Error format, error type, or ErrorHandler changes |
| `parser-internals.md` | Parse pipeline, encoding, entity, SAX, or parserCtx changes |
| `node-types.md` | Node type, struct field, or tree operation changes |
| `validation-pipeline.md` | Compile/validate phase, data model, or backtracking changes |
| `heliumlint.md` | CLI flag, pipeline, or exit code changes |
| `xpath3-design.md` | Design constraints, sub-doc structure changes |
| `xpath3-architecture.md` | Package layout, file additions/removals, import graph changes |
| `xpath3-api.md` | Public API, Context, Result, error type changes |
| `xpath3-types.md` | Item/Sequence/AtomicValue/Map/Array type changes |
| `xpath3-parser.md` | Lexer, parser, AST node, token type changes |
| `xpath3-eval.md` | Evaluator, comparison, casting logic changes |
| `xpath3-functions.md` | Function registry, built-in function additions/changes |
| `xpath3-tasks.md` | Task completion, dependency, or risk changes |
| `saxon-layout.md` | Reference layout updates |
