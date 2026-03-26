package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/lestrrat-go/helium/internal/icu"
)

func init() {
	registerFn("abs", 1, 1, fnAbs)
	registerFn("ceiling", 1, 1, fnCeiling)
	registerFn("floor", 1, 1, fnFloor)
	registerFn("round", 1, 2, fnRound)
	registerFn("round-half-to-even", 1, 2, fnRoundHalfToEven)
	registerFn("format-integer", 2, 3, fnFormatInteger)
	registerFn("format-number", 2, 3, fnFormatNumber)
}

// promoteToNumeric atomizes an item and ensures it is numeric.
// xs:untypedAtomic is cast to xs:double per XPath spec.
// Non-numeric types raise XPTY0004.
func promoteToNumeric(item Item) (AtomicValue, error) {
	a, err := AtomizeItem(item)
	if err != nil {
		return a, err
	}
	if a.TypeName == TypeUntypedAtomic {
		return castToDouble(a)
	}
	if !a.IsNumeric() {
		return a, &XPathError{Code: errCodeXPTY0004, Message: "expected numeric type, got " + a.TypeName}
	}
	// User-defined types: promote to the built-in numeric base type.
	a = PromoteSchemaType(a)
	return a, nil
}

func fnAbs(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		if v, ok := a.Value.(int64); ok {
			if v >= 0 {
				return SingleInteger(v), nil
			}
			if v != math.MinInt64 {
				return SingleInteger(-v), nil
			}
		}
		return SingleIntegerBig(new(big.Int).Abs(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(new(big.Rat).Abs(a.BigRat())), nil
	}
	return SingleAtomic(makeFloatResult(a.TypeName, math.Abs(a.ToFloat64()))), nil
}

func fnCeiling(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		if v, ok := a.Value.(int64); ok {
			return SingleInteger(v), nil
		}
		return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(ratCeiling(a.BigRat())), nil
	}
	f := a.ToFloat64()
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return SingleAtomic(a), nil
	}
	return SingleAtomic(makeFloatResult(a.TypeName, math.Ceil(f))), nil
}

func fnFloor(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		if v, ok := a.Value.(int64); ok {
			return SingleInteger(v), nil
		}
		return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(ratFloor(a.BigRat())), nil
	}
	f := a.ToFloat64()
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return SingleAtomic(a), nil
	}
	return SingleAtomic(makeFloatResult(a.TypeName, math.Floor(f))), nil
}

func fnRound(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	precision := 0
	if len(args) > 1 && seqLen(args[1]) > 0 {
		pa, err := AtomizeItem(args[1].Get(0))
		if err != nil {
			return nil, err
		}
		pa, err = CastAtomic(pa, TypeInteger)
		if err != nil {
			return nil, err
		}
		precision = int(pa.IntegerVal())
	}
	if isIntegerDerived(a.TypeName) {
		if precision >= 0 {
			if v, ok := a.Value.(int64); ok {
				return SingleInteger(v), nil
			}
			return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
		}
		return SingleIntegerBig(roundIntegerHalfUp(a.BigInt(), -precision)), nil
	}
	if a.TypeName == TypeDecimal {
		if precision == 0 {
			return SingleDecimal(ratRound(a.BigRat())), nil
		}
		return SingleDecimal(ratRoundPrecision(a.BigRat(), precision)), nil
	}
	n := a.ToFloat64()
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleAtomic(a), nil
	}
	if precision > 308 {
		return SingleAtomic(a), nil
	}
	// Use big.Float to preserve the exact IEEE 754 value during scaling
	r := roundHalfUpFloat(n, precision)
	if r == 0 && math.Signbit(n) {
		r = math.Copysign(0, -1)
	}
	return SingleAtomic(makeFloatResult(a.TypeName, r)), nil
}

func fnRoundHalfToEven(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	precision := 0
	if len(args) > 1 && seqLen(args[1]) > 0 {
		pa, err := AtomizeItem(args[1].Get(0))
		if err != nil {
			return nil, err
		}
		pa, err = CastAtomic(pa, TypeInteger)
		if err != nil {
			return nil, err
		}
		precision = int(pa.IntegerVal())
	}
	if isIntegerDerived(a.TypeName) {
		if precision >= 0 {
			if v, ok := a.Value.(int64); ok {
				return SingleInteger(v), nil
			}
			return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
		}
		// Negative precision: round to 10^(-precision)
		return SingleIntegerBig(roundIntegerHalfToEven(a.BigInt(), -precision)), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(icu.RatRoundHalfToEven(a.BigRat(), precision)), nil
	}
	n := a.ToFloat64()
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleAtomic(a), nil
	}
	// Clamp precision for float64: beyond ~15 significant digits, no rounding effect
	if precision > 308 {
		return SingleAtomic(a), nil
	}
	// Use big.Float to preserve the exact double value during scaling,
	// avoiding precision loss that can flip the rounding direction.
	result := roundHalfToEvenFloat(n, precision)
	return SingleAtomic(makeFloatResult(a.TypeName, result)), nil
}

// roundHalfToEvenFloat rounds a float64 to the given precision using
// half-to-even (banker's rounding). It uses big.Float to scale the value
// floatToBigFloat converts a float64 to a big.Float using its shortest
// decimal representation (matching Java's BigDecimal.valueOf behaviour).
// This avoids rounding artifacts from the exact IEEE 754 binary value.
func floatToBigFloat(n float64) *big.Float {
	s := strconv.FormatFloat(n, 'G', -1, 64)
	bf, _, err := new(big.Float).SetPrec(256).Parse(s, 10)
	if err != nil {
		return new(big.Float).SetPrec(256).SetFloat64(n)
	}
	return bf
}

// roundHalfUpFloat rounds a float64 using "half towards positive infinity"
// semantics (XPath fn:round). Uses big.Float to preserve precision.
func roundHalfUpFloat(n float64, precision int) float64 {
	if precision >= 0 {
		bf := new(big.Float).SetPrec(256).SetFloat64(n)
		scale := bigPow10(precision)
		scaled := new(big.Float).SetPrec(256).Mul(bf, scale)

		// XPath round: floor(x + 0.5) — rounds half towards +∞
		half := new(big.Float).SetPrec(256).SetFloat64(0.5)
		shifted := new(big.Float).SetPrec(256).Add(scaled, half)

		intPart := bigFloatFloor(shifted)
		result := new(big.Float).SetPrec(256).SetInt(intPart)
		result.Quo(result, scale)
		f, _ := result.Float64()
		return f
	}

	// Negative precision: use the shortest decimal representation to avoid
	// artifacts from the exact IEEE 754 binary value at extreme magnitudes.
	bf := floatToBigFloat(n)
	divisor := bigPow10(-precision)
	scaled := new(big.Float).SetPrec(256).Quo(bf, divisor)

	half := new(big.Float).SetPrec(256).SetFloat64(0.5)
	shifted := new(big.Float).SetPrec(256).Add(scaled, half)

	intPart := bigFloatFloor(shifted)
	result := new(big.Float).SetPrec(256).SetInt(intPart)
	result.Mul(result, divisor)
	f, _ := result.Float64()
	return f
}

// bigPow10 returns 10^n as a big.Float with 256-bit precision.
func bigPow10(n int) *big.Float {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	return new(big.Float).SetPrec(256).SetInt(p)
}

// bigFloatFloor returns the floor of a big.Float as a big.Int.
func bigFloatFloor(bf *big.Float) *big.Int {
	intPart, _ := bf.Int(nil)
	if bf.Sign() < 0 {
		frac := new(big.Float).SetPrec(256).Sub(bf, new(big.Float).SetPrec(256).SetInt(intPart))
		if frac.Sign() != 0 {
			intPart.Sub(intPart, big.NewInt(1))
		}
	}
	return intPart
}

// roundHalfToEvenFloat rounds a float64 to the given precision using
// half-to-even (banker's rounding). It uses big.Float to scale the value
// so that the exact IEEE 754 value is preserved — naive float64 arithmetic
// can lose the "above/below midpoint" information when multiplying by the
// scale factor.
func roundHalfToEvenFloat(n float64, precision int) float64 {
	if precision >= 0 {
		bf := new(big.Float).SetPrec(256).SetFloat64(n)
		scale := bigPow10(precision)
		scaled := new(big.Float).SetPrec(256).Mul(bf, scale)

		// Determine floor and the fractional part
		intPart, _ := scaled.Int(nil) // truncates towards zero
		if scaled.Sign() < 0 {
			// For negative numbers, Int truncates towards zero (ceiling);
			// we want the floor.
			frac := new(big.Float).SetPrec(256).Sub(scaled, new(big.Float).SetPrec(256).SetInt(intPart))
			if frac.Sign() != 0 {
				intPart.Sub(intPart, big.NewInt(1))
			}
		}

		floor := new(big.Float).SetPrec(256).SetInt(intPart)
		frac := new(big.Float).SetPrec(256).Sub(scaled, floor)
		half := new(big.Float).SetPrec(256).SetFloat64(0.5)

		var rounded *big.Int
		cmp := frac.Cmp(half)
		if cmp < 0 {
			rounded = new(big.Int).Set(intPart)
		} else if cmp > 0 {
			rounded = new(big.Int).Add(intPart, big.NewInt(1))
		} else {
			// Exactly 0.5: round to even
			rem := new(big.Int).Mod(new(big.Int).Abs(intPart), big.NewInt(2))
			if rem.Sign() == 0 {
				rounded = new(big.Int).Set(intPart) // already even
			} else {
				rounded = new(big.Int).Add(intPart, big.NewInt(1))
			}
		}

		result := new(big.Float).SetPrec(256).SetInt(rounded)
		result.Quo(result, scale)
		f, _ := result.Float64()
		// Preserve negative zero per IEEE 754
		if f == 0 && math.Signbit(n) {
			return math.Copysign(0, -1)
		}
		return f
	}

	// Negative precision: round to 10^(-precision)
	bf := new(big.Float).SetPrec(256).SetFloat64(n)
	divisor := bigPow10(-precision)
	scaled := new(big.Float).SetPrec(256).Quo(bf, divisor)

	intPart, _ := scaled.Int(nil)
	if scaled.Sign() < 0 {
		frac := new(big.Float).SetPrec(256).Sub(scaled, new(big.Float).SetPrec(256).SetInt(intPart))
		if frac.Sign() != 0 {
			intPart.Sub(intPart, big.NewInt(1))
		}
	}

	floor := new(big.Float).SetPrec(256).SetInt(intPart)
	frac := new(big.Float).SetPrec(256).Sub(scaled, floor)
	half := new(big.Float).SetPrec(256).SetFloat64(0.5)

	var rounded *big.Int
	cmp := frac.Cmp(half)
	if cmp < 0 {
		rounded = new(big.Int).Set(intPart)
	} else if cmp > 0 {
		rounded = new(big.Int).Add(intPart, big.NewInt(1))
	} else {
		rem := new(big.Int).Mod(new(big.Int).Abs(intPart), big.NewInt(2))
		if rem.Sign() == 0 {
			rounded = new(big.Int).Set(intPart)
		} else {
			rounded = new(big.Int).Add(intPart, big.NewInt(1))
		}
	}

	result := new(big.Float).SetPrec(256).SetInt(rounded)
	result.Mul(result, divisor)
	f, _ := result.Float64()
	// Preserve negative zero per IEEE 754
	if f == 0 && math.Signbit(n) {
		return math.Copysign(0, -1)
	}
	return f
}

func fnFormatNumber(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		// Per F&O: empty sequence is treated as NaN for formatting, result is xs:string
		a := AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}
		picture, err := coerceArgToStringRequired(args[1])
		if err != nil {
			return nil, err
		}
		df := defaultDecimalFormat(ctx)
		if len(args) > 2 && seqLen(args[2]) > 0 {
			formatName, fErr := coerceArgToString(args[2])
			if fErr != nil {
				return nil, fErr
			}
			df, err = resolveDecimalFormat(ctx, formatName)
			if err != nil {
				return nil, err
			}
		}
		s, err := formatNumber(a, picture, df)
		if err != nil {
			return nil, err
		}
		return SingleString(s), nil
	}
	if seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "format-number() first argument must be a singleton numeric value"}
	}
	a, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	if a.TypeName == TypeUntypedAtomic {
		a, err = CastAtomic(a, TypeDouble)
		if err != nil {
			return nil, err
		}
	}
	if !isSubtypeOf(a.TypeName, TypeNumeric) {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("format-number() first argument must be numeric, got %s", a.TypeName)}
	}

	picture, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}

	df := defaultDecimalFormat(ctx)
	if len(args) > 2 && seqLen(args[2]) > 0 {
		formatName, err := coerceArgToString(args[2])
		if err != nil {
			return nil, err
		}
		df, err = resolveDecimalFormat(ctx, formatName)
		if err != nil {
			return nil, err
		}
	}

	result, err := formatNumber(a, picture, df)
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

// --- Rat rounding helpers ---

// ratFloor returns floor(r) as a *big.Rat.
func ratFloor(r *big.Rat) *big.Rat {
	if r.IsInt() {
		return new(big.Rat).Set(r)
	}
	// Quo truncates toward zero; adjust for negative
	i := new(big.Int).Quo(r.Num(), r.Denom())
	if r.Sign() < 0 {
		i.Sub(i, big.NewInt(1))
	}
	return new(big.Rat).SetInt(i)
}

// ratCeiling returns ceiling(r) as a *big.Rat.
func ratCeiling(r *big.Rat) *big.Rat {
	if r.IsInt() {
		return new(big.Rat).Set(r)
	}
	i := new(big.Int).Quo(r.Num(), r.Denom())
	if r.Sign() > 0 {
		i.Add(i, big.NewInt(1))
	}
	return new(big.Rat).SetInt(i)
}

// ratRound rounds a *big.Rat toward positive infinity at the half (XPath round).
func ratRound(r *big.Rat) *big.Rat {
	if r.IsInt() {
		return new(big.Rat).Set(r)
	}
	// Add 0.5, then floor
	half := new(big.Rat).SetFrac64(1, 2)
	shifted := new(big.Rat).Add(r, half)
	return ratFloor(shifted)
}

// roundIntegerHalfToEven rounds a *big.Int to 10^scale using half-to-even.
func roundIntegerHalfToEven(n *big.Int, scale int) *big.Int {
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	q, r := new(big.Int).QuoRem(n, pow, new(big.Int))
	half := new(big.Int).Quo(pow, big.NewInt(2))
	absR := new(big.Int).Abs(r)
	cmp := absR.Cmp(half)
	if cmp < 0 {
		return new(big.Int).Mul(q, pow)
	}
	if cmp > 0 {
		if n.Sign() >= 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
		return new(big.Int).Mul(q, pow)
	}
	// Exactly half — round to even
	if new(big.Int).And(q, big.NewInt(1)).Sign() == 0 {
		return new(big.Int).Mul(q, pow)
	}
	if n.Sign() >= 0 {
		q.Add(q, big.NewInt(1))
	} else {
		q.Sub(q, big.NewInt(1))
	}
	return new(big.Int).Mul(q, pow)
}

// roundIntegerHalfUp rounds a *big.Int to 10^scale using half-up (towards positive infinity).
func roundIntegerHalfUp(n *big.Int, scale int) *big.Int {
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	q, r := new(big.Int).QuoRem(n, pow, new(big.Int))
	half := new(big.Int).Quo(pow, big.NewInt(2))
	absR := new(big.Int).Abs(r)
	cmp := absR.Cmp(half)
	if cmp < 0 {
		return new(big.Int).Mul(q, pow)
	}
	// cmp >= 0: at or past halfway. XPath round: "half towards +infinity".
	if n.Sign() >= 0 {
		// Positive: round up (towards +∞)
		q.Add(q, big.NewInt(1))
	} else if cmp > 0 {
		// Negative and strictly past halfway: round away from zero (towards -∞)
		q.Sub(q, big.NewInt(1))
	}
	// Negative and exactly half: round towards +∞ = towards zero = no change
	return new(big.Int).Mul(q, pow)
}

// ratRoundPrecision rounds a *big.Rat to the given precision (number of decimal places)
// using half-up rounding (towards positive infinity on tie).
func ratRoundPrecision(r *big.Rat, precision int) *big.Rat {
	if precision >= 0 {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
		// Multiply by scale, round, divide by scale
		shifted := new(big.Rat).Mul(r, new(big.Rat).SetInt(scale))
		half := new(big.Rat).SetFrac64(1, 2)
		rounded := ratFloor(new(big.Rat).Add(shifted, half))
		return new(big.Rat).Quo(rounded, new(big.Rat).SetInt(scale))
	}
	// Negative precision: round to 10^(-precision)
	result := roundIntegerHalfUp(ratFloorInt(r), -precision)
	return new(big.Rat).SetInt(result)
}

// ratFloorInt returns floor of a rational as big.Int.
func ratFloorInt(r *big.Rat) *big.Int {
	q := new(big.Int).Div(r.Num(), r.Denom())
	if r.Sign() < 0 && new(big.Int).Mul(q, r.Denom()).Cmp(r.Num()) != 0 {
		q.Sub(q, big.NewInt(1))
	}
	return q
}
