package xpath3

import (
	"context"
	"math"
	"math/big"

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
		return a, &XPathError{Code: "XPTY0004", Message: "expected numeric type, got " + a.TypeName}
	}
	return a, nil
}

func fnAbs(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0][0])
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		return SingleIntegerBig(new(big.Int).Abs(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(new(big.Rat).Abs(a.BigRat())), nil
	}
	f := a.ToFloat64()
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return SingleAtomic(a), nil
	}
	return SingleAtomic(makeFloatResult(a.TypeName, math.Abs(f))), nil
}

func fnCeiling(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0][0])
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
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
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0][0])
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
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
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0][0])
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(ratRound(a.BigRat())), nil
	}
	n := a.ToFloat64()
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleAtomic(a), nil
	}
	// XPath round: round half towards positive infinity
	r := math.Floor(n + 0.5)
	if r == 0 && n < 0 {
		r = math.Copysign(0, -1)
	}
	return SingleAtomic(makeFloatResult(a.TypeName, r)), nil
}

func fnRoundHalfToEven(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := promoteToNumeric(args[0][0])
	if err != nil {
		return nil, err
	}
	precision := 0
	if len(args) > 1 && len(args[1]) > 0 {
		pa, err := AtomizeItem(args[1][0])
		if err != nil {
			return nil, err
		}
		pa, err = CastAtomic(pa, TypeInteger)
		if err != nil {
			return nil, err
		}
		precision = int(pa.BigInt().Int64())
	}
	if isIntegerDerived(a.TypeName) {
		if precision >= 0 {
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
	scale := math.Pow(10, float64(precision))
	return SingleAtomic(makeFloatResult(a.TypeName, math.RoundToEven(n*scale)/scale)), nil
}

func fnFormatNumber(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	picture := seqToString(args[1])

	// Default decimal format
	df := defaultDecimalFormat()

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
