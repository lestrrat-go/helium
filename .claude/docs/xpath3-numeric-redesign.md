# XPath 3.1 — Numeric Type Redesign

## Problem

Current numeric representation:
- `xs:integer` → `int64` — bounded, overflows at ±9.2×10¹⁸
- `xs:decimal` → `string` — parsed to `float64` for arithmetic, losing precision and type identity
- `xs:double` → `float64` — correct
- `xs:float` → `float64` — correct (truncated to float32 on output)

XSD spec requires:
- `xs:integer` — arbitrary precision, unbounded
- `xs:decimal` — arbitrary precision, exact decimal (no binary float artifacts)
- `xs:double` — IEEE 754 64-bit
- `xs:float` — IEEE 754 32-bit

Current bugs: arithmetic returns wrong types (decimal*integer→double instead of decimal), integer overflow on large values, decimal loses precision through float64 round-trip.

## Design

### Backing Types

| XSD Type | Go Backing | Notes |
|----------|-----------|-------|
| `xs:integer` (and all derived) | `*big.Int` | Arbitrary precision, no overflow |
| `xs:decimal` | `*big.Rat` | Exact rational arithmetic, represents all decimals exactly |
| `xs:double` | `float64` | Unchanged |
| `xs:float` | `float64` | Unchanged (truncated to float32 precision on serialization) |

### Why `*big.Rat` for decimal (not `*big.Float`)

- `*big.Float` is binary floating point — still can't represent `0.1` exactly
- `*big.Rat` represents any finite decimal as exact `p/q` rational
- `Rat.SetString("3.14")` → exact `157/50`
- `Rat.FloatString(n)` → decimal string with `n` digits of precision
- All four arithmetic ops (`Add`, `Sub`, `Mul`, `Quo`) are exact
- Comparison via `Cmp` is exact
- Only downside: output formatting requires determining precision (see Serialization below)

### AtomicValue Changes

```go
// AtomicValue.Value type table:
//   xs:integer (derived)  → *big.Int
//   xs:decimal             → *big.Rat
//   xs:double              → float64   (unchanged)
//   xs:float               → float64   (unchanged)
//   xs:boolean             → bool      (unchanged)
//   xs:string (derived)    → string    (unchanged)
//   xs:dateTime/date/time  → time.Time (unchanged)
//   xs:duration            → Duration  (unchanged)
//   xs:QName               → QNameValue(unchanged)
//   xs:hexBinary           → []byte    (unchanged)
//   xs:base64Binary        → []byte    (unchanged)
```

### Accessor Method Changes

```go
// Remove: IntegerVal() int64, DoubleVal() float64
// Add:
func (a AtomicValue) BigInt() *big.Int       // panics if not integer-derived
func (a AtomicValue) BigRat() *big.Rat       // panics if not decimal
func (a AtomicValue) Float64Val() float64    // panics if not double/float

// ToFloat64 stays but uses big.Int/Rat conversion:
func (a AtomicValue) ToFloat64() float64 {
    switch {
    case isIntegerDerived(a.TypeName):
        f, _ := new(big.Float).SetInt(a.Value.(*big.Int)).Float64()
        return f
    case a.TypeName == TypeDecimal:
        f, _ := a.Value.(*big.Rat).Float64()
        return f
    case a.TypeName == TypeDouble || a.TypeName == TypeFloat:
        return a.Value.(float64)
    }
}
```

### Constructor Helpers

```go
func SingleInteger(n int64) Sequence          // wraps big.NewInt(n)
func SingleIntegerBig(n *big.Int) Sequence    // wraps directly
func SingleDecimal(r *big.Rat) Sequence
func SingleDecimalFromString(s string) Sequence // parse "3.14" → Rat
func SingleDouble(f float64) Sequence         // unchanged
```

### Arithmetic (`eval.go`)

Replace `promoteToDouble` → `float64` approach with type-preserving dispatch:

```
integer op integer → integer  (big.Int arithmetic)
decimal op integer → decimal  (promote integer to Rat, Rat arithmetic)
integer op decimal → decimal  (promote integer to Rat, Rat arithmetic)
decimal op decimal → decimal  (Rat arithmetic)
float   op X       → float   (if X is not double; float64 arithmetic)
double  op X       → double  (float64 arithmetic)
integer div integer → decimal (big.Rat Quo)
```

Implementation approach — three-tier dispatch:

```go
func evalArithmetic(ec *evalContext, e BinaryExpr) (Sequence, error) {
    // ... atomize left/right ...

    // 1. Both integer? → big.Int arithmetic
    if isIntegerDerived(la.TypeName) && isIntegerDerived(ra.TypeName) {
        return integerArith(e.Op, la.BigInt(), ra.BigInt())
    }
    // 2. Either decimal (or integer promoted to decimal)? → big.Rat arithmetic
    if needsDecimalArith(la.TypeName, ra.TypeName) {
        return decimalArith(e.Op, toRat(la), toRat(ra))
    }
    // 3. Otherwise → float64 arithmetic (double/float)
    return floatArith(e.Op, la, ra)
}

func integerArith(op TokenType, a, b *big.Int) (Sequence, error) {
    result := new(big.Int)
    switch op {
    case TokenPlus:  result.Add(a, b)
    case TokenMinus: result.Sub(a, b)
    case TokenStar:  result.Mul(a, b)
    case TokenDiv:
        // integer / integer → decimal
        r := new(big.Rat).SetFrac(a, b) // exact
        return SingleDecimal(r), nil
    case TokenIdiv:
        if b.Sign() == 0 { return error FOAR0002 }
        result.Quo(a, b) // truncates toward zero
    case TokenMod:
        if b.Sign() == 0 { return error FOAR0002 }
        result.Rem(a, b)
    }
    return SingleIntegerBig(result), nil
}

func decimalArith(op TokenType, a, b *big.Rat) (Sequence, error) {
    result := new(big.Rat)
    switch op {
    case TokenPlus:  result.Add(a, b)
    case TokenMinus: result.Sub(a, b)
    case TokenStar:  result.Mul(a, b)
    case TokenDiv:
        if b.Sign() == 0 { return error FOAR0002 }
        result.Quo(a, b)
    case TokenIdiv:
        if b.Sign() == 0 { return error FOAR0002 }
        // truncate: convert Quo to integer
        q := new(big.Rat).Quo(a, b)
        f, _ := q.Float64()
        return SingleIntegerBig(big.NewInt(int64(math.Trunc(f)))), nil
        // Better: use Num/Denom integer division
    case TokenMod:
        if b.Sign() == 0 { return error FOAR0002 }
        // a mod b = a - (a idiv b) * b
        q := new(big.Int).Quo(ratToInt(a), ratToInt(b)) // Quo truncates
        qr := new(big.Rat).SetInt(q)
        result.Sub(a, new(big.Rat).Mul(qr, b))
    }
    return SingleDecimal(result), nil
}
```

### Comparison (`compare.go`)

Replace `compareFloats(promoteToDouble(a), promoteToDouble(b))` with:

```go
func compareNumeric(op TokenType, a, b AtomicValue) (bool, error) {
    // Both integer → big.Int.Cmp
    // Either decimal → promote to Rat, Rat.Cmp
    // Otherwise → float64 comparison (handles NaN, ±Inf, -0)
}
```

### Casting (`cast.go`)

Key changes:
- `CastFromString` for integer: `new(big.Int).SetString(s, 10)`
- `CastFromString` for decimal: `new(big.Rat).SetString(s)`
- `castToDouble` from integer: `new(big.Float).SetInt(bi).Float64()`
- `castToDouble` from decimal: `rat.Float64()`
- `castToInteger` from double: check NaN/Inf/overflow → `big.NewInt(int64(math.Trunc(f)))`
- `castToInteger` from decimal: `rat.Num().Quo(rat.Num(), rat.Denom())`
- Derived integer range validation: compare `*big.Int` against `big.NewInt(min)` / `big.NewInt(max)`

### Serialization (`atomicToString`)

- Integer: `bi.String()` (base-10, handles arbitrarily large values)
- Decimal: need to determine output precision
  - XSD canonical: no trailing zeros except one after decimal point
  - `rat.FloatString(prec)` where `prec` = number of decimal digits needed
  - Strategy: `rat.RatString()` gives `p/q`; compute exact decimal expansion
  - Simpler: `rat.FloatString(20)` then trim trailing zeros

```go
func decimalToString(r *big.Rat) string {
    if r.IsInt() {
        return r.Num().String() // e.g. "42"
    }
    // Determine precision needed for exact representation
    // or use a high-precision float string and trim
    s := r.FloatString(20)
    s = strings.TrimRight(s, "0")
    if strings.HasSuffix(s, ".") {
        s += "0" // keep at least one decimal digit: "3.0"
    }
    return s
}
```

### Map Key Normalization

```go
func normalizeMapKey(key AtomicValue) mapKey {
    switch v := key.Value.(type) {
    case *big.Int:
        return mapKey{typeName: key.TypeName, value: v.String()}
    case *big.Rat:
        return mapKey{typeName: key.TypeName, value: v.RatString()}
    // ... existing cases ...
    }
}
```

### EBV (Effective Boolean Value)

- Integer: `bi.Sign() != 0`
- Decimal: `rat.Sign() != 0`
- Double/float: unchanged

### Aggregate Functions

- `sum` for integers: accumulate with `big.Int.Add`
- `sum` for decimals: accumulate with `big.Rat.Add`
- `avg`: sum with Rat, then divide by count Rat
- `min`/`max`: use `Cmp` methods
- `count`: unchanged

### Math Functions

`math:sin`, `math:cos`, etc. always return `xs:double`. They just need `ToFloat64()` which works for both `*big.Int` and `*big.Rat`.

Rounding functions (`round`, `floor`, `ceiling`, `round-half-to-even`) need type-preserving implementations:
- Integer input → return as-is
- Decimal input → round the Rat, return decimal
- Float/double input → `math.Round` etc., return same type

### Numeric Functions

- `abs` for integer: `new(big.Int).Abs(bi)`
- `abs` for decimal: `new(big.Rat).Abs(rat)`
- `ceiling`/`floor` for decimal: convert to float and back, or compute with integer division
- `round-half-to-even` for decimal: needs careful Rat-based implementation

## Migration Plan

### Phase 1: Change backing types (breaking)
1. Change `AtomicValue.Value` to hold `*big.Int` / `*big.Rat`
2. Update accessor methods
3. Update constructor helpers
4. **Everything breaks here** — this is the big-bang step

### Phase 2: Fix arithmetic
1. Replace `promoteToDouble`-based arithmetic with three-tier dispatch
2. Integer arithmetic with `big.Int`
3. Decimal arithmetic with `big.Rat`
4. Float/double arithmetic unchanged

### Phase 3: Fix comparison
1. Type-preserving numeric comparison
2. Integer uses `big.Int.Cmp`
3. Decimal uses `big.Rat.Cmp`

### Phase 4: Fix casting
1. Update all `CastAtomic` / `CastFromString` paths
2. String ↔ integer via `big.Int.SetString` / `String()`
3. String ↔ decimal via `big.Rat.SetString` / `decimalToString`
4. Cross-type casts (int↔decimal↔double)

### Phase 5: Fix serialization and misc
1. `atomicToString` for integer and decimal
2. Map key normalization
3. EBV
4. All remaining accessor sites (~70 total per audit)

### Phase 6: Fix functions
1. Aggregate functions (sum, avg, min, max)
2. Numeric functions (abs, ceiling, floor, round, round-half-to-even)
3. Math functions (just need ToFloat64)
4. Sequence/position functions (need int extraction from big.Int)

## Touchpoint Count (from audit)

| Category | Sites |
|----------|-------|
| `int64` type assertions | ~20 |
| `float64` type assertions | ~25 |
| `string` (decimal) type assertions | ~12 |
| AtomicValue constructions | ~70 |
| Arithmetic operations | ~20 |
| Comparison operations | ~10 |
| Cast functions | ~15 |
| Aggregate functions | ~15 |
| Math/numeric functions | ~20 |
| Special value handling (NaN/Inf/zero) | ~35 |
| Map key / EBV / misc | ~10 |
| **Total** | **~250** |

## Files Affected

Primary (heavy changes):
- `types.go` — AtomicValue definition, accessors, MapItem key normalization
- `eval.go` — arithmetic, unary, range, predicate position matching
- `cast.go` — all cast functions, CastFromString, atomicToString
- `compare.go` — numeric comparison
- `sequence.go` — EBV, constructor helpers
- `functions_aggregate.go` — sum, avg, min, max
- `functions_numeric.go` — abs, ceiling, floor, round, round-half-to-even

Secondary (lighter changes):
- `functions_math.go` — just ToFloat64 calls
- `functions_string.go` — codepoint extraction
- `functions_array.go` — index extraction
- `functions_hof.go` — arity extraction
- `functions_sequence.go` — position extraction
- `functions_constructors.go` — integer constructor validation
- `functions_helpers.go` — numeric extraction
- `arithmetic_datetime.go` — duration × number
- `xpath3.go` — NewNumericProperty

## Risks

1. **Performance**: `*big.Int` allocation for every integer literal. Mitigate with small-value cache (`sync.Pool` or preallocated 0–100).
2. **Pointer semantics**: `*big.Int` is mutable via pointer. Must `new(big.Int).Set(x)` on every write to avoid aliasing bugs. Consider a wrapper type.
3. **Decimal precision**: `*big.Rat` can represent repeating decimals (1/3) that XSD decimal cannot. Need to limit precision on output. `FloatString(maxPrecision)` with trailing zero trimming.
4. **idiv with big.Rat**: Need integer truncation of rational. Use `Num/Denom` integer division.
5. **NaN/Inf**: Only exist for double/float. Integer and decimal arithmetic never produce NaN/Inf — they error instead (FOAR0002).
