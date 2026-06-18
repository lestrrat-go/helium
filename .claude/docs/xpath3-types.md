# XPath 3.1 — Type System

## Item Hierarchy

```
Item (interface)
├── NodeItem        → wraps helium.Node
├── AtomicValue     → XSD atomic (string, int64, float64, bool, time.Time, etc.)
├── FunctionItem    → callable (inline, named ref, partial application)
├── MapItem         → immutable key-value (keys: AtomicValue)
└── ArrayItem       → immutable indexed sequence members
```

## Core Types

```go
type Item interface { itemType() ItemType }

type ItemType interface {
    Matches(Item) bool
    String() string
}

type Sequence []Item
```

## NodeItem

```go
type NodeItem struct {
    Node           helium.Node
    TypeAnnotation string
    AtomizedType   string
}
```

- `TypeAnnotation` → schema-aware node type annotation (`xs:*` or `Q{ns}local`)
- `AtomizedType` → built-in base type used when atomizing schema-derived node types

## AtomicValue

```go
type AtomicValue struct {
    TypeName string // "xs:string", "xs:integer", etc.
    Value    any    // Go native backing value
    BaseType string // built-in base type for user-defined/derived types (drives normalization)
}
```

### Atomic Type Constants and Go Backing Types

| Constant | XSD Type | Go type |
|----------|----------|---------|
| `TypeString` | xs:string | `string` |
| `TypeInteger` | xs:integer | `*big.Int` |
| `TypeDecimal` | xs:decimal | `*big.Rat` |
| `TypeDouble` | xs:double | `float64` |
| `TypeFloat` | xs:float | `float64` |
| `TypeBoolean` | xs:boolean | `bool` |
| `TypeDate` | xs:date | `time.Time` |
| `TypeDateTime` | xs:dateTime | `time.Time` |
| `TypeTime` | xs:time | `time.Time` |
| `TypeDuration` | xs:duration | `Duration` struct |
| `TypeDayTimeDuration` | xs:dayTimeDuration | `Duration` struct |
| `TypeYearMonthDuration` | xs:yearMonthDuration | `Duration` struct |
| `TypeAnyURI` | xs:anyURI | `string` |
| `TypeQName` | xs:QName | `QNameValue{Prefix,Local,URI string}` |
| `TypeBase64Binary` | xs:base64Binary | `[]byte` |
| `TypeHexBinary` | xs:hexBinary | `[]byte` |
| `TypeUntypedAtomic` | xs:untypedAtomic | `string` |
| `TypeAnyAtomicType` | xs:anyAtomicType | (abstract) |

## FunctionItem

```go
type FunctionItem struct {
    Arity  int
    Name   string  // empty for anonymous
    invoke func(ctx context.Context, args []Sequence) (Sequence, error)
}
```

Produced by: inline functions, named refs (`fn#2`), partial application.

## MapItem

Immutable. `Put` returns new map (copy-on-write).
Construction + outward-facing accessors clone `Sequence` values. Caller mutation MUST NOT change stored contents.

```go
type MapItem struct {
    entries []mapEntry        // ordered for stable iteration
    index   map[mapKey]int    // O(1) lookup
}
type mapKey struct { typeName string; value any }

func NewMap(entries []MapEntry) MapItem
func (m MapItem) Get(key AtomicValue) (Sequence, bool)
func (m MapItem) Put(key AtomicValue, value Sequence) MapItem
func (m MapItem) Keys() []AtomicValue
func (m MapItem) Contains(key AtomicValue) bool
func (m MapItem) Size() int
func (m MapItem) ForEach(fn func(AtomicValue, Sequence) error) error
func MergeMaps(maps []MapItem, policy MergePolicy) (MapItem, error)
```

Merge policies: `UseFirst`, `UseLast`, `Combine`, `Reject`.

Map key normalization (`normalizeMapKey`, value-space keys so equal values collide):
- string/int64/float64/bool → Go map equality
- string-like fold to `xs:string`: `xs:untypedAtomic`, `xs:anyURI`, and any
  `xs:string`-derived type, including schema-derived atomics whose `BaseType` is
  string-like (checked via `TypeName` AND `BaseType`)
- numerics fold to one `familyNumeric` bucket via exact rational arithmetic
  (int64 fast path; `xs:integer`/`xs:decimal`-derived use big.Int/big.Rat; float/double via big.Rat)
- durations fold to `xs:duration` and canonicalize to signed rational months/seconds
  (`durationToRat`): `"<months>|<seconds>"`. Covers `xs:duration`,
  `xs:yearMonthDuration`, `xs:dayTimeDuration`, and schema-derived durations
  (folded via `BaseType` and, since the Go value is unambiguously a `Duration`,
  unconditionally in the `case Duration` branch). `-PT0S` == `PT0S`.
- `time.Time` → UTC-normalized RFC3339Nano string (no-tz keys prefixed `notz:`, kept distinct from tz keys)
- `[]byte` → hex-encoded string
- `QNameValue` → `URI + "\x00" + Local` (prefix-independent)

`NewMap` is a low-level constructor: it does NOT enforce XPath duplicate-key
semantics (`XQDY0137`) — that is enforced by the map constructor expression
(`eval_funcall.go`) before `NewMap`. Given value-equal duplicate keys, `NewMap`
retains both `entries` rows (`Size()` over-counts) while the lookup `index`
collapses them last-wins.

## ArrayItem

Immutable. 1-based indexing.
Construction + outward-facing accessors clone member sequences. Caller mutation MUST NOT change stored contents.

```go
type ArrayItem struct { members []Sequence }

func NewArray(members []Sequence) ArrayItem
func (a ArrayItem) Get(index int) (Sequence, error)    // 1-based; error if OOB
func (a ArrayItem) Size() int
func (a ArrayItem) Put(index int, value Sequence) (ArrayItem, error)
func (a ArrayItem) Append(value Sequence) ArrayItem
func (a ArrayItem) Members() []Sequence
func (a ArrayItem) SubArray(start, end int) (ArrayItem, error)  // 1-based inclusive
func (a ArrayItem) Flatten() Sequence
```

## Sequence Helpers (`sequence.go`)

```go
func SingleNode(n helium.Node) Sequence
func SingleAtomic(v AtomicValue) Sequence
func SingleBoolean(b bool) Sequence
func SingleInteger(n int64) Sequence
func SingleDouble(f float64) Sequence
func SingleString(s string) Sequence
func EmptySequence() Sequence
func NodesFrom(seq Sequence) ([]helium.Node, bool)
func AtomizeSequence(seq Sequence) ([]AtomicValue, error)
func EBV(seq Sequence) (bool, error)
```

## Effective Boolean Value (EBV)

Per XPath 3.1 Section 2.4.3:
- Empty sequence → false
- Single boolean → that boolean
- Single string/anyURI/untypedAtomic → false iff empty string
- Single numeric → false iff 0 or NaN
- Sequence starting with node → true
- Otherwise → dynamic error `FORG0006`

## Atomization

Per XPath 3.1 Section 2.6.2:
- Node → typed cast via `TypeAnnotation` when available
- Schema-derived node type → fallback to `AtomizedType` built-in base for atomization
- Unannotated node → `xs:untypedAtomic` with `StringValue(node)`
- Atomic → identity
- Function/map/array → error `FOTY0013`

## SequenceType (used in `instance of`, `cast as`, etc.)

```go
type SequenceType struct {
    ItemType   NodeTest     // reuses NodeTest interface
    Occurrence Occurrence
}
type Occurrence int
const (
    OccurrenceExactlyOne Occurrence = iota  // (default)
    OccurrenceZeroOrOne                     // ?
    OccurrenceZeroOrMore                    // *
    OccurrenceOneOrMore                     // +
)
```
