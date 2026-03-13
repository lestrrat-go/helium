# XPath 3.1 Design — Index

New `xpath3` package. Shared infra in `internal/xpath`. Saxon-HE as reference (`saxon-layout.md`).

## Sub-Documents

Read ONLY the file relevant to your current task.

| Trigger | Doc |
|---------|-----|
| Architecture, packages, file layout, import rules | `xpath3-architecture.md` |
| Public API, Context, Result, errors | `xpath3-api.md` |
| Item/Sequence/AtomicValue/Map/Array type system | `xpath3-types.md` |
| Lexer tokens, parser grammar, AST nodes | `xpath3-parser.md` |
| Evaluator, comparison, casting, eval rules | `xpath3-eval.md` |
| Built-in function system, all function categories | `xpath3-functions.md` |
| Phased task breakdown with dependencies | `xpath3-tasks.md` |

## Constraints

- `xpath3` MUST NOT import `xpath1`. Both import `internal/xpath`.
- `xpath3` MUST NOT import `xsd`. XSD atomic types inline in `xpath3/types.go`.
- Avoid gratuitous external deps. Standard-adjacent packages (e.g., `golang.org/x/text`) are fine when justified.
- `xpath1` public API unchanged after refactor.
- Eager sequences (`[]Item`). No lazy iterators in v1.
- No static type checking in v1. Type errors at runtime.
- Helium DOM (`helium.Node`) is sole node model.

## Out of Scope

- Static type checking / type inference
- Lazy/streaming sequences
- FLWOR `group by`, `count`, `window`
- `fn:doc()`, `fn:collection()`, `fn:transform()`, `fn:serialize()`
- XQuery
- Saxon-style optimization / elaboration
- XML Schema import for user-defined types
- Unicode collation beyond codepoint order
