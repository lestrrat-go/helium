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
    Node             helium.Node
    TypeAnnotation   string
    AtomizedType     string
    ListItemType     string
    ListItemAtomized string
    UnionMemberTypes []string
    ActiveUnionMember *NodeItemUnionMember
    ListItemLeaves   []*NodeItemUnionMember
    QNameNoDefaultNS bool
}

type NodeItemUnionMember struct {
    TypeName     string // member type annotation name
    Atomized     string // built-in base of an atomic member
    ListItem     string // non-empty if the member is itself a list
    ListItemAtom string // built-in base of the list item
    ListItemLeaves []*NodeItemUnionMember // per-token leaves when ListItem is a union
}
```

- `TypeAnnotation` → schema-aware node type annotation (`xs:*` or `Q{ns}local`)
- `AtomizedType` → built-in base type used when atomizing schema-derived node types
- `ListItemType` → item type name for a list-typed node; `ListItemAtomized` → that item type's built-in base (e.g. `xs:QName` for a QName-derived list item), so list-token atomization (`atomizeListToken`) can resolve QName/NOTATION items namespace-aware; `UnionMemberTypes` → member type names for a union-typed node
- `ActiveUnionMember` → the value-dependent ACTIVE LEAF member of a union node (the first direct member the value FULLY validates against — lexical/value cast AND facets/list-length via `SchemaDeclarations.ValidateCastWithNS`; when that member is itself a union, resolution descends recursively to the nested leaf, matching the `$value` selection), or nil. A list leaf expands to per-item atoms; an atomic leaf yields a single atom typed as that member. The `NodeItemUnionMember` it points at is immutable (shared across clones)
- `ListItemLeaves` → populated only when a list node's ITEM type is a UNION: the per-token ACTIVE union leaf (resolved value-dependently in `nodeItemFor` via `resolveActiveUnionLeafForValue`, one entry per token in document order). `atomizeListTokenAt(i, …)` atomizes token `i` through `ListItemLeaves[i]` when present, so `xs:list itemType="<union>"` atomizes each token by its own active member (agreeing with `$value`) instead of forcing every token through one static base; nil/short slice falls back to `atomizeListToken` on the declared item type
- `QNameNoDefaultNS` → set from `Evaluator.QNameValueNoDefaultNamespace()`; when true an UNPREFIXED QName/NOTATION value atomizes to no namespace (XSD value-space semantics) instead of the node's default namespace

`resolveQNameFromNode` predeclares the `xml` prefix (→ the XML namespace) without requiring a binding on any node, matching `xsd.resolveLexicalQName`, so an xs:QName value such as `xml:lang`/`xml:space` atomizes correctly. List-typed node atomization splits on XSD whitespace ONLY (`xsdListFields`: space/tab/CR/LF — NOT NBSP/other Unicode whitespace, matching XSD list tokenization and the validation/`$value` paths) and atomizes each token via `atomizeListToken` (used by both `atomizeStream` and the value-comparison atom iterator). When the item type's built-in base is `xs:QName`/`xs:NOTATION` it resolves each token against the node's in-scope namespaces (preserving the user/list-item type name and its built-in base); a NON-QName USER item type (a `Q{...}` name `CastFromString` can't resolve) is cast through its `ListItemAtomized` built-in base and typed as the user type with that base, so a list whose item type derives from xs:int yields numeric atoms usable by `sum()`. A UNION-typed node is atomized through its precomputed ACTIVE LEAF member (`ActiveUnionMember`, resolved by `resolveActiveUnionLeaf` in `nodeItemFor` with FULL schema-aware validation — cast validity AND facets/list-length/assertions via `SchemaDeclarations.ValidateCastWithNS`, the ctx threaded through `nodeItemFor`): `atomizeUnionItems` (used by both the stream and comparison paths) expands a LIST leaf to per-item atoms (via `atomizeListTokenAt`, so when the leaf's list item type is ITSELF a UNION the per-token active leaves carried on `NodeItemUnionMember.ListItemLeaves` — precomputed in `resolveActiveUnionLeafRec` — type each token by its own active member, matching `$value`) and yields an ATOMIC leaf as a single atom typed as that member. Resolution descends recursively through NESTED unions to the leaf (mirroring `fixedUnionActiveMember`), so `data()` and `$value` agree for `Outer=union(Inner,…)` / `Inner=union(IntList,…)`. Because selection is full validation, a member that casts but fails its own facets (e.g. an xs:list with `xs:length=2` against `"1 2 3"`) is correctly skipped — matching the `$value` path. All of this only activates when `ActiveUnionMember`/`ListItemType` are populated (schema-aware atomization), so non-schema-aware xpath3/xslt3 behavior is unchanged.

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
Construction + outward-facing accessors clone `Sequence` values via `cloneSequence`. Caller mutation MUST NOT change stored contents.

KEYS are cloned too: every map ingress (`newSingleEntryMap`, `NewMap`, `Put`, `MergeMaps`, `MapBuilder.Add`) clones the `AtomicValue` key via `cloneMapKey` (an O(1) single-atomic copy reusing `deepCloneAtomicValue`), and outward-facing `Keys`/`ForEach` return cloned keys. This stops a pointer-backed key (e.g. `xs:integer` backed by `*big.Int`) from being mutated after insertion — single-entry maps recompute the stored key in `Get`, so a mutated key would otherwise silently change the map's key. Public `ForEach` clones both key and value; trusted internal read-only/bounded/lazy callers use the private no-clone `forEach0`/`keys0`/`get0`/`entries0` accessors (caller must not mutate, and clones once at the call-site egress).

`cloneSequence` (`types.go`) is a **deep** clone of each item, not a shallow slice copy: pointer/slice-backed `AtomicValue` payloads (`*big.Int`, `*big.Rat`, `*FloatValue`, `[]byte`, and a `Duration`'s `*big.Rat` fields) are duplicated so mutating the caller's original payload cannot reach the stored value. It does NOT recurse into nested `MapItem`/`ArrayItem` items: those are immutable (all mutators are copy-on-write) and every value they hold was already deep-cloned at its own ingress, so they are shared by value — this keeps incremental construction of a depth-N nested structure O(N) rather than O(N²) and preserves the OpLimit/maxNodes resource bounds (charged once per inserted value, not per nesting level). Immutable atomic payloads (int64, string, bool, float64, time.Time, QNameValue, by-value Duration without rationals) are returned as the original boxed item to avoid an interface re-allocation per item.

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

In the `fn:data` / typed-value accessor path (`atomizeForFnData`, `functions_node.go`), an element node's complex content kind (XDM 3.1 §5.15) selects a typed-value ACTION: **element-only** content has no typed value → error `FOTY0012`; **empty** content has typed value `()` → the item is SKIPPED (contributes no atoms, no error); **mixed** content still atomizes to `xs:untypedAtomic` and simple/simpleContent to its typed value. This fires only when the active `SchemaDeclarations` also implements the optional `ContentTypeKindProvider` (reached by type assertion; the xsd adapter `schemaDecls` implements it) and the annotation resolves. The content-kind check (`checkContentKindItem`, returning `(skip, err)`) is INTERLEAVED with atomization — threaded into `atomizeStreamCont` as an optional per-item pre-check — so it walks items in the SAME encounter order and with the SAME array recursion as atomization: the FIRST offending item wins (a map/function atomized earlier still raises `FOTY0013` before a later element-only element is reached), and element-only / empty nodes nested inside arrays are handled. Non-schema-aware nodes and every other `AtomizeItem` caller (string value, comparisons, casts) are unaffected — the check is scoped to `fn:data` (no provider ⇒ `atomizeForFnData` is exactly `AtomizeSequence`).

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
