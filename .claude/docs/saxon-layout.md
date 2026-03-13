# Saxon-HE Source Layout (XPath 3.0/3.1 Reference)

Source location: `testdata/saxon/source/12/` (latest release line).
Java packages under `net.sf.saxon`.

## Compilation Pipeline

```
Source string → Tokenizer → XPathParser → AST (Expression tree)
  → Static Analysis (type check, scope) → Optimizer (minimal in HE)
  → Elaborator (evaluator selection) → Evaluator (runtime)
```

## Key Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `expr.parser` | 21 | Tokenizer, XPathParser, Token definitions, TypeChecker |
| `expr` | 400+ | Expression AST nodes (base `Expression.java` 72KB) |
| `expr.flwor` | 46 | FLWOR clauses: For, Let, Where, OrderBy, GroupBy, Window, Count |
| `expr.elab` | 31 | Elaboration: creates specialized evaluators (pull/push/item/boolean) |
| `expr.compat` | 4 | XPath 1.0 backward compatibility |
| `functions` | 194 | Built-in function implementations |
| `functions.registry` | 15 | Function sets: XPath20, XPath30, XPath31, XSLT30, Array |
| `functions.hof` | ~3 | Higher-order functions (partial apply, function refs) |
| `pattern` | 38 | XSLT pattern matching (NodeTest, QNameTest, UnionPattern, etc.) |
| `type` | 51 | Type hierarchy: ItemType, AtomicType, ComplexType, TypeHierarchy |
| `value` | 49 | Atomic values: IntegerValue, StringValue, DateTimeValue, etc. |
| `om` | 71 | Object model: Item, NodeInfo, SequenceIterator, AxisInfo, NamePool |
| `tree.tiny` | 27 | Default compact tree model (array-based) |
| `tree.linked` | 24 | DOM-like linked tree model |
| `tree.iter` | 31 | Axis iterators: Descendant, Following, Preceding, etc. |
| `event` | 58 | SAX-like push processing (Receiver, Outputter, builders) |
| `xpath` | 8 | JAXP XPath API (XPathParser, XPathEvaluator, XPathExpressionImpl) |
| `query` | 20 | XQuery support (not relevant for XPath-only work) |
| `serialize` | 46 | Output serialization (not relevant for XPath-only work) |

## Tokenizer

File: `expr/parser/Tokenizer.java` (~1300 lines)

- Converts XPath string → token stream
- State machine: `BARE_NAME_STATE`, `SEQUENCE_TYPE_STATE`, `OPERATOR_STATE`
- `languageLevel` field controls XPath 2.0 / 3.0 / 3.1 syntax

### XPath 3.0+ Tokens (in `Token.java`)

| Token | Syntax | Constant |
|-------|--------|----------|
| Concat | `\|\|` | `CONCAT` |
| Simple map | `!` | `BANG` |
| Arrow | `=>` | `ARROW` |
| Lookup | `?` | `QMARK` |
| String concat | `\|\|` | `CONCAT` |
| Named function ref | `#` | `NAMED_FUNCTION_REF` |

## XPathParser

File: `expr/parser/XPathParser.java` (~5500 lines, recursive descent)

### Entry Points

| Method | Parses |
|--------|--------|
| `parseExpression()` | Comma-separated expression sequence |
| `parseExprSingle()` | Single expr: for/let/if/quantified/switch |
| `parseOrExpression()` | Binary operators with precedence climbing |
| `parseBasicStep()` | Location steps: axis + nodetest + predicates |
| `parseFunctionCall()` | Function invocations |
| `parseInlineFunction()` | Inline function expressions (3.0+) |

### Operator Precedence (low → high)

1. `,` (comma / sequence)
2. `for` / `let` / `some` / `every` / `if`
3. `or`
4. `and`
5. `eq ne lt le gt ge` / `= != < <= > >=`
6. `||` (string concat, 3.0+)
7. `to` (range)
8. `+ -` (additive)
9. `* div idiv mod` (multiplicative)
10. `union \|`
11. `intersect except`
12. `instance of`
13. `treat as`
14. `castable as`
15. `cast as`
16. `=>` (arrow, 3.0+)
17. `!` (simple map, 3.0+)
18. `/` `//` (path)
19. Unary `+ -`
20. `?` (lookup, 3.1)
21. Predicates `[...]`, arguments `(...)`

## Expression AST Nodes

### Literals & Variables

| Class | XPath |
|-------|-------|
| `Literal` | Numbers, strings, booleans |
| `VariableReference` | `$var` |
| `RootExpression` | `/` (document root) |

### Path & Axis

| Class | XPath |
|-------|-------|
| `SlashExpression` | `a/b/c` |
| `AxisExpression` | `child::*`, `@attr`, `ancestor::node()` |
| `SimpleStepExpression` | Single-step optimization |

### Operators

| Class | XPath |
|-------|-------|
| `ArithmeticExpression` | `+ - * div mod idiv` |
| `ValueComparison` | `eq ne lt le gt ge` |
| `GeneralComparison` | `= != < <= > >=` |
| `BooleanExpression` | `and or` |
| `VennExpression` | `union intersect except` |
| `ConcatExpression` | `\|\|` (3.0+) |
| `LookupExpression` | `?key` (3.1) |

### Complex Expressions

| Class | XPath |
|-------|-------|
| `ForExpression` | `for $x in ... return ...` |
| `LetExpression` | `let $x := ... return ...` |
| `QuantifiedExpression` | `some/every $x in ... satisfies ...` |
| `Choose` (IfExpression) | `if (...) then ... else ...` |
| `TryCatch` | `try { ... } catch * { ... }` |
| `FunctionCall` | `fn:name(...)` |
| `DynamicFunctionCall` | `$f(...)` (3.0+) |

### FLWOR (3.0+)

| Class | Clause |
|-------|--------|
| `FLWORExpression` | Container |
| `ForClause` | `for $x in ...` |
| `LetClause` | `let $x := ...` |
| `WhereClause` | `where ...` |
| `OrderByClause` | `order by ...` |
| `GroupByClause` | `group by ...` |
| `WindowClause` | `for tumbling/sliding window ...` |
| `CountClause` | `count $c` |

## Type System

### Type Hierarchy

```
ItemType
├── AtomicType → BuiltInAtomicType (62 built-in types)
├── NodeKindTest (element, attribute, text, comment, PI, document, namespace)
├── FunctionItemType (3.0+)
├── MapType (3.1)
├── ArrayItemType (3.1)
└── AnyItemType (matches all)
```

### Built-in Atomic Types (in `BuiltInAtomicType.java`)

Numeric: `integer`, `decimal`, `double`, `float`, `long`, `int`, `short`, `byte`
String: `string`, `normalizedString`, `token`, `Name`, `NCName`, `QName`
Calendar: `dateTime`, `date`, `time`, `gYear`, `gMonth`, `gDay`, `gYearMonth`, `gMonthDay`
Duration: `duration`, `dayTimeDuration`, `yearMonthDuration`
Other: `boolean`, `anyURI`, `base64Binary`, `hexBinary`, `NOTATION`

### Sequence Cardinality

| Indicator | Meaning | Constant |
|-----------|---------|----------|
| (none) | exactly one | `EXACTLY_ONE` |
| `?` | zero or one | `ZERO_OR_ONE` |
| `*` | zero or more | `ZERO_OR_MORE` |
| `+` | one or more | `ONE_OR_MORE` |

## Function Library

### Registry Structure

`BuiltInFunctionSet` → versioned subsets:

| Class | Functions added |
|-------|----------------|
| `XPath20FunctionSet` | Core 2.0 (string, math, node, aggregate, date/time) |
| `XPath30FunctionSet` | 3.0 additions (higher-order, format, environment) |
| `XPath31FunctionSet` | 3.1 additions (map, array, JSON, random) |

### Key Function Files

| File | Functions |
|------|-----------|
| `Concat.java` | `concat()` |
| `Contains.java` | `contains()`, `starts-with()`, `ends-with()` |
| `Count.java` | `count()` |
| `Sum.java` | `sum()` |
| `Round.java` | `round()` |
| `Tokenize_1.java` | `tokenize()` |
| `Replace.java` | `replace()` |
| `CurrentDate.java` | `current-date()`, `current-dateTime()` |
| `Name_1.java` | `name()` |
| `LocalName.java` | `local-name()` |
| `ApplyFn.java` | `fn:apply()` (3.0+) |

## Object Model (om/)

### Core Abstractions

| Interface | Role |
|-----------|------|
| `Item` | Base: atomic value, node, or function |
| `NodeInfo` | Tree node (element, attr, text, comment, PI, doc, namespace) |
| `SequenceIterator` | Lazy iteration over sequences |
| `GroundedValue` | Materialized in-memory sequence |
| `TreeInfo` | Tree metadata (root, base URI, etc.) |

### Axes (in `AxisInfo.java`)

13 XPath axes: `child`, `descendant`, `parent`, `ancestor`, `following-sibling`, `preceding-sibling`, `following`, `preceding`, `attribute`, `namespace`, `self`, `descendant-or-self`, `ancestor-or-self`

### Name Handling

| Class | Use |
|-------|-----|
| `StructuredQName` | Full QName (prefix + URI + local) |
| `FingerprintedQName` | Fast comparison via integer fingerprint |
| `NamePool` | Shared name registry |
| `NameChecker` | XML name validation |

## XPath 3.0 Features (vs. 2.0)

| Feature | Key files |
|---------|-----------|
| Simple map (`!`) | `SimpleMapExpression.java` |
| String concat (`\|\|`) | Token `CONCAT`, `ConcatExpression` |
| Inline functions | `XPathParser.parseInlineFunction()` |
| Named function refs (`fn#1`) | `NamedFunctionRef.java` |
| Partial application | `functions.hof.PartialApply` |
| Higher-order functions | `DynamicFunctionCall`, `FunctionItemType` |
| Let expressions | `LetExpression.java` |
| Arrow operator (`=>`) | Parsed in XPathParser, desugared to function call |
| FLWOR extensions | `flwor/GroupByClause`, `WindowClause`, `CountClause` |
| Try-catch | `TryCatch.java` |

## XPath 3.1 Features (vs. 3.0)

| Feature | Key files |
|---------|-----------|
| Maps | `MapType`, `map:*` functions |
| Arrays | `ArrayItemType`, `array:*` functions |
| Lookup (`?key`) | `LookupExpression`, `LookupAllExpression` |
| Unary lookup | `UnaryLookup` |
| Arrow functions | Extended arrow syntax |

## Static Analysis

| Class | Role |
|-------|------|
| `StaticContext` | Namespace bindings, variable types, function declarations |
| `ExpressionVisitor` | Walks AST, applies analyses |
| `TypeChecker` | Validates expression types against declarations |
| `RoleDiagnostic` | Describes operand context for error messages |

## Runtime

| Class | Role |
|-------|------|
| `XPathContext` | Runtime state interface |
| `XPathContextMajor` | New focus (context item change) |
| `XPathContextMinor` | Variable scope change |
| `Evaluator` | Base evaluator interface |
| `PullEvaluator` | Iterate values |
| `PushEvaluator` | Send events |
