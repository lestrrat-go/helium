# XPath 3.1 — Function System

## Function Interface

```go
type Function interface {
    MinArity() int
    MaxArity() int  // -1 = variadic
    Call(ctx context.Context, args []Sequence) (Sequence, error)
}

type FunctionContext interface {
    Node()       helium.Node
    Position()   int
    Size()       int
    Namespace(prefix string) (string, bool)
    Variable(name string)   (Sequence, bool)
}
```

## Registry

Package-level `builtinFunctions3 map[QualifiedName]Function` populated in `init()`.

## Namespace URIs

| Prefix | URI |
|--------|-----|
| `fn:` (default) | `http://www.w3.org/2005/xpath-functions` |
| `math:` | `http://www.w3.org/2005/xpath-functions/math` |
| `map:` | `http://www.w3.org/2005/xpath-functions/map` |
| `array:` | `http://www.w3.org/2005/xpath-functions/array` |
| `err:` | `http://www.w3.org/2005/xqt-errors` |
| `xs:` | `http://www.w3.org/2001/XMLSchema` |

`fn:` is default → `string()` and `fn:string()` resolve to same function.

## Functions by File

### `functions_node.go`
`node-name`, `nilled`, `string`, `data`, `base-uri`, `document-uri`, `root`, `path`, `has-children`, `innermost`, `outermost`, `id`, `idref`, `lang`, `local-name`, `name`, `namespace-uri`, `number`, `generate-id`

### `functions_string.go`
`codepoints-to-string`, `string-to-codepoints`, `compare`, `codepoint-equal`, `concat`, `string-join`, `substring`, `string-length`, `normalize-space`, `normalize-unicode`, `upper-case`, `lower-case`, `translate`, `contains`, `starts-with`, `ends-with`, `substring-before`, `substring-after`, `matches`, `replace`, `tokenize`, `analyze-string` (partial; result DOM built with helium)

Regex: use Go `regexp` package. Map XPath flags (`i`,`m`,`s`,`x`) to Go equivalents.

### `functions_numeric.go`
`abs`, `ceiling`, `floor`, `round`, `round-half-to-even`, `format-integer` (stub v1), `format-number` (stub v1)

### `functions_boolean.go`
`boolean`, `not`, `true`, `false`

### `functions_aggregate.go`
`count`, `avg`, `max`, `min`, `sum`, `distinct-values`

### `functions_sequence.go`
`empty`, `exists`, `head`, `tail`, `insert-before`, `remove`, `reverse`, `subsequence`, `unordered`, `zero-or-one`, `one-or-more`, `exactly-one`, `deep-equal`, `index-of`

### `functions_datetime.go`
Constructors: `dateTime`
Accessors: `year-from-dateTime`, `month-from-dateTime`, `day-from-dateTime`, `hours-from-dateTime`, `minutes-from-dateTime`, `seconds-from-dateTime`, `timezone-from-dateTime`, (same for date/time variants), `years-from-duration`, `months-from-duration`, `days-from-duration`, `hours-from-duration`, `minutes-from-duration`, `seconds-from-duration`
Formatting: `format-date`, `format-dateTime`, `format-time`
Misc: `adjust-dateTime-to-timezone`, `adjust-date-to-timezone`, `adjust-time-to-timezone`

### `functions_uri.go`
`resolve-uri`, `encode-for-uri`, `iri-to-uri`, `escape-html-uri`, `base-uri`, `document-uri`

### `functions_qname.go`
`QName`, `resolve-QName`, `prefix-from-QName`, `local-name-from-QName`, `namespace-uri-from-QName`, `namespace-uri-for-prefix`, `in-scope-prefixes`

### `functions_hof.go`
`for-each`, `filter`, `fold-left`, `fold-right`, `apply`, `function-lookup`, `function-arity`, `function-name`

### `functions_map.go`
`map:merge`, `map:size`, `map:keys`, `map:contains`, `map:get`, `map:put`, `map:entry`, `map:remove`, `map:for-each`, `map:find`

### `functions_array.go`
`array:size`, `array:get`, `array:put`, `array:append`, `array:subarray`, `array:remove`, `array:insert-before`, `array:head`, `array:tail`, `array:reverse`, `array:join`, `array:flat-map`, `array:filter`, `array:fold-left`, `array:fold-right`, `array:for-each`, `array:for-each-pair`, `array:sort`

### `functions_math.go`
`math:pi`, `math:exp`, `math:exp10`, `math:log`, `math:log10`, `math:pow`, `math:sqrt`, `math:sin`, `math:cos`, `math:tan`, `math:asin`, `math:acos`, `math:atan`, `math:atan2`

All delegate to Go `math` package.

### `functions_error.go`
`error` (raises `FOER0000` with optional code/description/value), `trace` (logs and returns input)

### `functions_misc.go`
`static-base-uri`, `default-collation`, `available-environment-variables`, `environment-variable`, `current-dateTime`, `current-date`, `current-time`, `implicit-timezone`, `generate-id`

## Stubs (v1)

Return "not yet implemented" error:
- `fn:format-number`, `fn:format-integer`
- `fn:serialize` (needs helium serializer integration)

## FunctionItem Mechanics

Inline functions, named refs, partial applications → all produce `FunctionItem`.

`FunctionItem.invoke` is a closure: `func(ctx context.Context, args []Sequence) (Sequence, error)`.

Evaluator calls uniformly regardless of origin.
