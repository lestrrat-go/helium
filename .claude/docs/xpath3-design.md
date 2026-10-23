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

## Constraints

- `xpath3` MUST NOT import `xpath1`. Both import `internal/xpath`.
- `xpath3` MUST NOT import `xsd`. XSD atomic types inline in `xpath3/types.go`.
- Avoid gratuitous external deps. Standard-adjacent packages (e.g., `golang.org/x/text`) are fine when justified.
- `xpath1` public API unchanged after refactor.
- `Sequence` supports eager `ItemSlice` values and lazy ranges/iterators; consumers preserve bounds and cancellation.
- No static type checking in v1. Type errors at runtime.
- Helium DOM (`helium.Node`) is sole node model.

## Out of Scope

- Static type checking / type inference
- FLWOR `group by`, `count`, `window`
- XSLT execution inside `xpath3` itself (`fn:transform()` is supplied by `xslt3`)
- XQuery
- Saxon-style optimization / elaboration
- Unicode collation beyond codepoint order
