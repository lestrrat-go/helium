# XPath 3.1 — Lexer, Parser, AST

## Tokens (`token.go`)

### Inherited from xpath1

`EOF`, `Number`, `String`, `Name`, `Star`, `VariableRef`, `Slash`, `SlashSlash`, `Pipe`, `Plus`, `Minus`, `Equals`, `NotEquals`, `Less`, `LessEq`, `Greater`, `GreaterEq`, `And`, `Or`, `Mod`, `Div`, `LParen`, `RParen`, `LBracket`, `RBracket`, `At`, `ColonColon`, `Comma`, `Dot`, `DotDot`, `Colon`

### New in XPath 3.1

| Token | Syntax | Disambiguation |
|-------|--------|----------------|
| `Concat` | `\|\|` | Peek after `\|`: if another `\|` → Concat, else Pipe |
| `Bang` | `!` | If followed by `=` → NotEquals, else Bang |
| `Arrow` | `=>` | Peek after `=`: if `>` → Arrow, else Equals |
| `Hash` | `#` | Named function ref: `fn#2` |
| `QMark` | `?` | Lookup / partial application (parser disambiguates) |
| `LBrace` | `{` | Map/array constructor, try-catch body |
| `RBrace` | `}` | |
| `Idiv` | `idiv` | Integer division keyword |
| `Intersect` | `intersect` | Keyword |
| `Except` | `except` | Keyword |
| `Return` | `return` | |
| `For` | `for` | |
| `Let` | `let` | |
| `In` | `in` | |
| `Where` | `where` | |
| `OrderBy` | `order` | (followed by `by`) |
| `By` | `by` | |
| `Ascending` | `ascending` | |
| `Descending` | `descending` | |
| `Stable` | `stable` | |
| `Some` | `some` | |
| `Every` | `every` | |
| `Satisfies` | `satisfies` | |
| `If` | `if` | |
| `Then` | `then` | |
| `Else` | `else` | |
| `Try` | `try` | |
| `Catch` | `catch` | |
| `InstanceOf` | `instance` | (followed by `of`) |
| `Of` | `of` | |
| `CastAs` | `cast` | (followed by `as`) |
| `CastableAs` | `castable` | (followed by `as`) |
| `TreatAs` | `treat` | (followed by `as`) |
| `As` | `as` | |
| `To` | `to` | Range |
| `Eq Ne Lt Le Gt Ge` | value comparison keywords | |
| `Union` | `union` | Keyword form of `\|` |
| `Function` | `function` | Inline function keyword |
| `Map` | `map` | Map constructor |
| `Array` | `array` | Array constructor |
| `Element Attribute DocumentNode NamespaceNode SchemaElement SchemaAttribute Item` | Kind test keywords | |

## Lexer (`lexer.go`)

Same one-pass approach as `xpath1/lexer.go`. All tokens emitted to `[]Token` upfront.

Keyword disambiguation: `and`, `or`, `div`, `mod`, `idiv`, `intersect`, `except`, `to`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `union`, `instance`, `cast`, `castable`, `treat`, `as`, `in`, `return`, `satisfies`, `then`, `else`, `some`, `every`, `for`, `let`, `where`, `if`, `try`, `catch`, `stable`, `ascending`, `descending` are operators/keywords ONLY when preceded by a value-producing token. Otherwise → `Name`.

## Parser (`parser.go`)

Recursive descent. Parse depth limit: 200.

Entry: `Parse(input string) (Expr, error)` → `parseExpression()`.

### Precedence (low → high)

| Level | Construct | Method |
|-------|-----------|--------|
| 0 | `,` sequence | `parseExpression` |
| 1 | for/let/if/quantified/try | `parseExprSingle` |
| 2 | `or` | `parseOrExpr` |
| 3 | `and` | `parseAndExpr` |
| 4 | `= != < <= > >=` general comparison | `parseComparisonExpr` |
| 5 | `eq ne lt le gt ge` value comparison | `parseComparisonExpr` |
| 6 | `\|\|` string concat | `parseConcatExpr` |
| 7 | `to` range | `parseRangeExpr` |
| 8 | `+ -` additive | `parseAdditiveExpr` |
| 9 | `* div idiv mod` multiplicative | `parseMultiplicativeExpr` |
| 10 | `union \|` | `parseUnionExpr` |
| 11 | `intersect except` | `parseIntersectExceptExpr` |
| 12 | `instance of` | `parseInstanceOfExpr` |
| 13 | `treat as` | `parseTreatAsExpr` |
| 14 | `castable as` | `parseCastableAsExpr` |
| 15 | `cast as` | `parseCastAsExpr` |
| 16 | `=>` arrow | `parseArrowExpr` |
| 17 | `!` simple map | `parseSimpleMapExpr` |
| 18 | `/ //` path | `parsePathExpr` |
| 19 | unary `+ -` | `parseUnaryExpr` |
| 20 | `?` lookup | `parseLookupExpr` |
| 21 | `[...]` predicates | `parseStepExpr` |

### Parse-Time Desugarings

- Arrow: `$x => f(a, b)` → `FunctionCall{Name:"f", Args:[$x, a, b]}`
- No `ArrowExpr` AST node.

### Special Parses

- **Named function ref**: `fn:upper-case#1` → `NamedFunctionRef{Prefix:"fn", Name:"upper-case", Arity:1}`. Detected when `Hash` follows name in `parsePrimaryExpr`.
- **Inline function**: `function($x as xs:integer) as xs:boolean { $x > 0 }` → `InlineFunctionExpr`. Body delimited by `{ }`.
- **Partial application**: `substring(?, 2)` → `FunctionCall{Args:[PlaceholderExpr{}, NumberExpr{2}]}`. Evaluator detects placeholders → returns `FunctionItem`.
- **Map constructor**: `map { "key": "value" }` → `MapConstructorExpr`.
- **Array (square)**: `[1, 2, 3]` → `ArrayConstructorExpr{SquareBracket:true}`. Each expr = one member.
- **Array (curly)**: `array { 1, 2, 3 }` → `ArrayConstructorExpr{SquareBracket:false}`. Single sequence.
- **FLWOR**: `for`/`let` clauses collected until `return`. `for` and `let` can interleave.
- **Sequence types**: `parseSequenceType` → `parseItemType` → `parseKindTest`. Kind tests: `node()`, `element(name, type)`, `attribute(name, type)`, `document-node(element(...))`, `text()`, `comment()`, `processing-instruction(target)`, `namespace-node()`, `schema-element(name)`, `schema-attribute(name)`, `function(*)`, `map(*)`, `array(*)`, `item()`.

## AST Nodes (`expr.go`)

All implement `Expr` interface (`exprNode()` marker method).

### Literals & Variables

| Node | Fields |
|------|--------|
| `LiteralExpr` | `Value any` (string or float64) |
| `VariableExpr` | `Prefix, Name string` |
| `RootExpr` | (none — bare `/`) |

### Location Paths

```go
type LocationPath struct { Absolute bool; Steps []Step }
type Step struct {
    Axis       ixpath.AxisType
    NodeTest   NodeTest
    Predicates []Expr
}
```

### Node Tests

| Node | Fields | Matches |
|------|--------|---------|
| `NameTest` | `Prefix, Local string` | QName or `*` wildcard |
| `TypeTest` | `Kind NodeKind` | `node()`, `text()`, `comment()`, `processing-instruction()` |
| `PITest` | `Target string` | PI with optional target |
| `ElementTest` | `Name, TypeName string; Nillable bool` | `element(name, type?)` |
| `AttributeTest` | `Name, TypeName string` | `attribute(name, type?)` |
| `DocumentTest` | `Inner NodeTest` | `document-node(element(...))` |
| `SchemaElementTest` | `Name string` | `schema-element(name)` |
| `SchemaAttributeTest` | `Name string` | `schema-attribute(name)` |
| `NamespaceNodeTest` | (none) | `namespace-node()` |
| `FunctionTest` | (none) | `function(*)` |
| `MapTest` | (none) | `map(*)` |
| `ArrayTest` | (none) | `array(*)` |
| `AnyItemTest` | (none) | `item()` |

### Operators

| Node | Fields | XPath |
|------|--------|-------|
| `BinaryExpr` | `Op TokenType; Left, Right Expr` | `+ - * div idiv mod = != < > eq ne lt le gt ge and or` |
| `UnaryExpr` | `Operand Expr` | `-expr` |
| `ConcatExpr` | `Left, Right Expr` | `\|\|` |
| `SimpleMapExpr` | `Left, Right Expr` | `!` |
| `RangeExpr` | `Start, End Expr` | `to` |
| `UnionExpr` | `Left, Right Expr` | `union \|` |

### Filter & Path

| Node | Fields |
|------|--------|
| `FilterExpr` | `Expr Expr; Predicates []Expr` |
| `PathExpr` | `Steps []Expr` |

### Lookup

| Node | Fields |
|------|--------|
| `LookupExpr` | `Expr, Key Expr; All bool` |
| `UnaryLookupExpr` | `Key Expr; All bool` |

### FLWOR

```go
type FLWORExpr struct { Clauses []FLWORClause; Return Expr }
// FLWORClause implementations:
type ForClause    struct { Var string; Expr Expr }
type LetClause    struct { Var string; Expr Expr }
type WhereClause  struct { Predicate Expr }
type OrderByClause struct { Specs []OrderSpec; Stable bool }
type OrderSpec    struct { Expr Expr; Descending, EmptyGreatest bool; Collation string }
```

### Control Flow

| Node | Fields |
|------|--------|
| `QuantifiedExpr` | `Some bool; Var string; Domain, Satisfies Expr` |
| `IfExpr` | `Cond, Then, Else Expr` |
| `TryCatchExpr` | `Try Expr; Catches []CatchClause` |
| `CatchClause` | `Codes []string; Expr Expr` |

### Type Expressions

| Node | Fields |
|------|--------|
| `InstanceOfExpr` | `Expr Expr; Type SequenceType` |
| `CastExpr` | `Expr Expr; Type AtomicTypeName; AllowEmpty bool` |
| `CastableExpr` | `Expr Expr; Type AtomicTypeName; AllowEmpty bool` |
| `TreatAsExpr` | `Expr Expr; Type SequenceType` |

### Functions

| Node | Fields |
|------|--------|
| `FunctionCall` | `Prefix, Name string; Args []Expr` |
| `DynamicFunctionCall` | `Func Expr; Args []Expr` |
| `NamedFunctionRef` | `Prefix, Name string; Arity int` |
| `InlineFunctionExpr` | `Params []FunctionParam; ReturnType SequenceType; Body Expr` |
| `PlaceholderExpr` | (none — `?` in partial application) |
| `FunctionParam` | `Name string; TypeHint SequenceType` |

### Constructors

```go
type MapConstructorExpr struct { Pairs []MapConstructorPair }
type MapConstructorPair struct { Key, Value Expr }
type ArrayConstructorExpr struct { Items []Expr; SquareBracket bool }
type SequenceExpr struct { Items []Expr }  // comma-separated: (a, b, c)
```
