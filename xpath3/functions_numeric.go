package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/lestrrat-go/helium/internal/icu"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
		return a, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected numeric type, got " + a.TypeName}
	}
	// User-defined types: promote to the built-in numeric base type.
	a = PromoteSchemaType(a)
	return a, nil
}

// promoteSeqToNumeric coerces an xs:numeric? argument. Empty sequences yield
// (_, false, nil); a single item is promoted via promoteToNumeric; multi-item
// sequences are rejected with XPTY0004.
func promoteSeqToNumeric(seq Sequence) (AtomicValue, bool, error) {
	switch seqLen(seq) {
	case 0:
		return AtomicValue{}, false, nil
	case 1:
		a, err := promoteToNumeric(seq.Get(0))
		if err != nil {
			return AtomicValue{}, false, err
		}
		return a, true, nil
	default:
		return AtomicValue{}, false, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected single numeric value, got sequence of length > 1"}
	}
}

func fnAbs(_ context.Context, args []Sequence) (Sequence, error) {
	a, ok, err := promoteSeqToNumeric(args[0])
	if err != nil {
		return nil, err
	}
	if !ok {
		return validNilSequence, nil
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
	a, ok, err := promoteSeqToNumeric(args[0])
	if err != nil {
		return nil, err
	}
	if !ok {
		return validNilSequence, nil
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
	a, ok, err := promoteSeqToNumeric(args[0])
	if err != nil {
		return nil, err
	}
	if !ok {
		return validNilSequence, nil
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

// roundDecision is the scale-aware outcome of resolving a $precision argument
// against an operand's own base-10 scale. See resolveRoundScale.
type roundDecision int

const (
	// roundCompute: the precision is within range of the operand's scale, so
	// the result must be computed with the (bounded) scale exponent.
	roundCompute roundDecision = iota
	// roundUnchanged: the precision is finer than the operand's fractional
	// scale; the result equals the operand unchanged. No Exp needed.
	roundUnchanged
	// roundTrivialZero: the precision is coarser than the operand's magnitude
	// by more than one decade, so every digit rounds away. The result is 0
	// (the boundary/tie case s == magnitude is handled by roundCompute, since
	// its scale exponent is bounded by the operand's own digit count). No Exp
	// needed.
	roundTrivialZero
)

// roundPrecisionArg extracts the rounding precision from the second argument
// under the XPath 3.1 function-conversion rules for a required
// "$precision as xs:integer" parameter: the argument must be exactly one item;
// xs:integer (and its subtypes) is accepted as-is, xs:untypedAtomic is cast to
// xs:integer, and every other type (xs:boolean, xs:decimal, xs:float, xs:double,
// xs:string, ...) is rejected with XPTY0004 — there is no implicit numeric
// truncation. An empty sequence violates the exactly-one cardinality and is a
// type error too. The precision is returned as a *big.Int (XPath integers are
// arbitrary precision); callers pair it with the operand's scale via
// resolveRoundScale so an astronomically large 10^|precision| is never
// materialised for cases whose result is determined trivially.
func roundPrecisionArg(arg Sequence) (*big.Int, error) {
	pa, err := extractSingleAtomicArg(arg, "round")
	if err != nil {
		return nil, err
	}
	pa, err = coerceToInteger(pa)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(pa.BigInt()), nil
}

// resolveRoundScale decides, scale-aware, how to round an operand to the given
// $precision. intDigits is the number of base-10 digits in the operand's
// integer part (>= 1; use 1 for |operand| < 1). fracDigits is the number of
// fractional decimal digits the operand actually has (0 for integers). The
// returned scale is the bounded magnitude of 10 to raise (always >= 0) and is
// only meaningful when the decision is roundCompute.
//
// For coarse (negative) precision the scale is s = -precision: when s exceeds
// the operand's integer magnitude by more than one decade no tie is possible
// and the result is 0; otherwise s <= intDigits so 10^s is bounded by the
// operand. For fine (positive) precision the scale is p = precision: when p is
// at least the operand's fractional-digit count the operand is already
// representable and is returned unchanged; otherwise 10^p is bounded by the
// operand's fractional scale.
//
// The one exception is a non-terminating rational, whose fracDigits is the large
// ratFracDigitNonTerminating sentinel. It must never be treated as "fully
// representable" — that is handled BEFORE the positive-precision unchanged test,
// so a precision at or above the sentinel (e.g. 1<<30) cannot short-circuit to
// roundUnchanged and bypass the cap. Such an operand has no terminating decimal
// expansion, so its value is well-defined at any requested fractional precision;
// honour the request up to roundMaxComputeScale (so 10^p stays bounded), and
// beyond that cap raise an error rather than silently rounding at a lower scale
// than asked. Silently clamping would return an observably wrong (lower-precision)
// value; erroring keeps the computation DoS-safe without ever lying about the
// result.
func resolveRoundScale(precision *big.Int, intDigits, fracDigits int) (int, roundDecision, error) {
	if precision.Sign() < 0 {
		// scale = -precision, as a non-negative big.Int
		s := new(big.Int).Neg(precision)
		// If -precision > intDigits, the scale is at least a full decade above
		// the operand's magnitude, so |operand| < 10^s / 2 and it rounds to 0.
		if s.Cmp(big.NewInt(int64(intDigits))) > 0 {
			return 0, roundTrivialZero, nil
		}
		// Here s <= intDigits, which fits in int and bounds 10^s.
		return int(s.Int64()), roundCompute, nil
	}
	// Non-negative precision. A non-terminating rational has no terminating
	// decimal expansion, so fracDigits is the ratFracDigitNonTerminating
	// sentinel rather than a true scale: it must NEVER be treated as "fully
	// representable" (which would short-circuit to roundUnchanged and return the
	// repeating operand). Handle it first — honour the request up to
	// roundMaxComputeScale so 10^p stays bounded, and beyond that refuse with
	// FOAR0002 rather than silently returning a lower-precision value. This must
	// precede the precision >= fracDigits test below, because a precision at or
	// above the sentinel (e.g. 1<<30) would otherwise wrongly select
	// roundUnchanged and bypass the cap entirely.
	if fracDigits == ratFracDigitNonTerminating {
		if precision.Cmp(big.NewInt(roundMaxComputeScale)) > 0 {
			return 0, roundCompute, &XPathError{
				Code: errCodeFOAR0002,
				Message: fmt.Sprintf(
					"round precision %s exceeds the maximum of %d fractional digits for a non-terminating decimal",
					precision.String(), roundMaxComputeScale),
			}
		}
		return int(precision.Int64()), roundCompute, nil
	}
	// Terminating decimal. If precision is at least the operand's fractional-digit
	// count the operand is unchanged; this also covers all integers (fracDigits
	// == 0) and any precision >= the operand's scale.
	if precision.Cmp(big.NewInt(int64(fracDigits))) >= 0 {
		return 0, roundUnchanged, nil
	}
	// Here precision < fracDigits, and fracDigits is the operand's true scale, so
	// 10^precision is bounded by the operand's own size (the accepted invariant).
	return int(precision.Int64()), roundCompute, nil
}

// roundMaxComputeScale bounds the positive (fine) compute scale handed to Exp for
// a non-terminating decimal operand (whose fractional-digit count is the
// ratFracDigitNonTerminating sentinel). A precision up to this is computed
// exactly; a larger precision raises FOAR0002 rather than silently rounding at a
// lower scale. It comfortably exceeds the fractional precision of any value that
// can arise from terminating xs:decimal or float operands (a float64 has at most
// ~324 fractional digits), so it never alters a representable result; it only
// bounds the otherwise-unbounded non-terminating case. 10^(1<<20) is a ~315 KB
// big.Int — cheap for Exp — while a precision past a million fractional digits on
// a repeating decimal is far outside any legitimate use.
const roundMaxComputeScale = 1 << 20

// intDigitCount returns the number of base-10 digits in |n|, with 0 counted as
// a single digit. Used as the integer-magnitude input to resolveRoundScale.
func intDigitCount(n *big.Int) int {
	if n.Sign() == 0 {
		return 1
	}
	return len(new(big.Int).Abs(n).Text(10))
}

// ratIntDigitCount returns the number of base-10 digits in the integer part of
// |r| (>= 1).
func ratIntDigitCount(r *big.Rat) int {
	return intDigitCount(ratFloorInt(new(big.Rat).Abs(r)))
}

// ratFracDigitNonTerminating is returned by ratFracDigitCount for a rational
// whose decimal expansion does not terminate (e.g. 1/3, common when an
// xs:decimal is produced by integer division). It is large enough that no
// practical fine precision will be treated as "finer than the operand's scale",
// so such values always fall through to roundCompute. Because the operand itself
// no longer bounds the scale in this case, resolveRoundScale caps the positive
// compute scale at roundMaxComputeScale so the precision argument can never feed
// an astronomically large exponent to Exp.
const ratFracDigitNonTerminating = 1 << 30

// ratFracDigitCount returns the number of fractional decimal digits of r in its
// terminating decimal expansion: when r's reduced denominator is 2^a * 5^b this
// is max(a, b). For a non-terminating rational (a denominator with other prime
// factors, which an xs:decimal division result can be) it returns
// ratFracDigitNonTerminating so any finite precision triggers real rounding.
func ratFracDigitCount(r *big.Rat) int {
	d := new(big.Int).Set(r.Denom())
	two := big.NewInt(2)
	five := big.NewInt(5)
	a := 0
	for d.Cmp(big.NewInt(1)) != 0 {
		q, rem := new(big.Int).QuoRem(d, two, new(big.Int))
		if rem.Sign() != 0 {
			break
		}
		d = q
		a++
	}
	b := 0
	for d.Cmp(big.NewInt(1)) != 0 {
		q, rem := new(big.Int).QuoRem(d, five, new(big.Int))
		if rem.Sign() != 0 {
			break
		}
		d = q
		b++
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		// Residual factor other than 2 or 5: the decimal expansion repeats.
		return ratFracDigitNonTerminating
	}
	if a > b {
		return a
	}
	return b
}

func fnRound(_ context.Context, args []Sequence) (Sequence, error) {
	return roundImpl(args, false)
}

func fnRoundHalfToEven(_ context.Context, args []Sequence) (Sequence, error) {
	return roundImpl(args, true)
}

// roundImpl is the shared core of fn:round (halfToEven == false, half towards
// +infinity) and fn:round-half-to-even (halfToEven == true). The $precision
// argument is resolved scale-aware against each operand so an astronomically
// large 10^|precision| is never materialised: cases whose result is determined
// trivially (rounds to 0, or operand unchanged) short-circuit before any Exp.
func roundImpl(args []Sequence, halfToEven bool) (Sequence, error) {
	// When the 2-arg form is used, $precision is a required "exactly one
	// xs:integer" parameter and must be validated whether or not $arg is empty:
	// the empty first argument does not excuse an absent or ill-typed precision.
	// round((), "bad") and round(1, ()) both raise XPTY0004; round((), 3) and
	// round((), 1) are () only because the precision itself is valid.
	precision := big.NewInt(0)
	if len(args) > 1 {
		p, err := roundPrecisionArg(args[1])
		if err != nil {
			return nil, err
		}
		precision = p
	}

	a, ok, err := promoteSeqToNumeric(args[0])
	if err != nil {
		return nil, err
	}
	if !ok {
		return validNilSequence, nil
	}

	if isIntegerDerived(a.TypeName) {
		scale, decision, err := resolveRoundScale(precision, intDigitCount(a.BigInt()), 0)
		if err != nil {
			return nil, err
		}
		if decision == roundUnchanged {
			if v, ok := a.Value.(int64); ok {
				return SingleInteger(v), nil
			}
			return SingleIntegerBig(new(big.Int).Set(a.BigInt())), nil
		}
		if decision == roundTrivialZero {
			return SingleInteger(0), nil
		}
		if halfToEven {
			return SingleIntegerBig(roundIntegerHalfToEven(a.BigInt(), scale)), nil
		}
		return SingleIntegerBig(roundIntegerHalfUp(a.BigInt(), scale)), nil
	}

	if a.TypeName == TypeDecimal {
		r := a.BigRat()
		scale, decision, err := resolveRoundScale(precision, ratIntDigitCount(r), ratFracDigitCount(r))
		if err != nil {
			return nil, err
		}
		if decision == roundUnchanged {
			return SingleDecimal(new(big.Rat).Set(r)), nil
		}
		if decision == roundTrivialZero {
			return SingleDecimal(new(big.Rat)), nil
		}
		p := scale
		if precision.Sign() < 0 {
			p = -scale
		}
		if halfToEven {
			return SingleDecimal(ratRoundPrecisionHalfToEven(r, p)), nil
		}
		if p == 0 {
			return SingleDecimal(ratRound(r)), nil
		}
		return SingleDecimal(ratRoundPrecision(r, p)), nil
	}

	n := a.ToFloat64()
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleAtomic(a), nil
	}
	// A float64 has at most ~309 integer digits and ~324 fractional digits, so
	// these bounds make resolveRoundScale short-circuit every extreme precision
	// without ever computing a huge scale.
	intDigits := floatIntDigitCount(n)
	scale, decision, err := resolveRoundScale(precision, intDigits, 324)
	if err != nil {
		return nil, err
	}
	if decision == roundUnchanged {
		return SingleAtomic(a), nil
	}
	if decision == roundTrivialZero {
		r := 0.0
		if math.Signbit(n) {
			r = math.Copysign(0, -1)
		}
		return SingleAtomic(makeFloatResult(a.TypeName, r)), nil
	}
	p := scale
	if precision.Sign() < 0 {
		p = -scale
	}
	if halfToEven {
		result := roundHalfToEvenFloat(n, p)
		return SingleAtomic(makeFloatResult(a.TypeName, result)), nil
	}
	r := roundHalfUpFloat(n, p)
	if r == 0 && math.Signbit(n) {
		r = math.Copysign(0, -1)
	}
	return SingleAtomic(makeFloatResult(a.TypeName, r)), nil
}

// floatIntDigitCount returns the number of base-10 digits in the integer part
// of |n| (>= 1), used as the magnitude input to resolveRoundScale. It derives
// the magnitude from the exact value via big.Float to avoid the off-by-one that
// math.Log10 exhibits near powers of ten.
func floatIntDigitCount(n float64) int {
	m := math.Abs(n)
	if m < 1 {
		return 1
	}
	intPart, _ := new(big.Float).SetPrec(256).SetFloat64(m).Int(nil)
	return intDigitCount(intPart)
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

// compatArgString applies the XPath 1.0 function-conversion rule for an
// xs:string argument: fn:string of its first item. Used for format-number's
// picture and decimal-format-name arguments under compatibility mode.
func compatArgString(arg Sequence) (string, error) {
	sv, err := xpath10CompatStringItem(arg)
	if err != nil {
		return "", err
	}
	s, _ := sv.Value.(string)
	return s, nil
}

func fnFormatNumber(ctx context.Context, args []Sequence) (Sequence, error) {
	compat := getFnContext(ctx).xpath10CompatMode()

	// Value argument (xs:double?). An empty value formats as NaN (F&O); compat
	// mode converts a non-empty value with fn:number (first item, non-numeric →
	// NaN) instead of requiring a singleton numeric.
	var a AtomicValue
	switch {
	case seqLen(args[0]) == 0:
		a = AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}
	case compat:
		var err error
		if a, err = xpath10CompatNumberItem(args[0]); err != nil {
			return nil, err
		}
	default:
		if seqLen(args[0]) != 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "format-number() first argument must be a singleton numeric value"}
		}
		var err error
		if a, err = AtomizeItem(args[0].Get(0)); err != nil {
			return nil, err
		}
		if a.TypeName == TypeUntypedAtomic {
			if a, err = CastAtomic(a, TypeDouble); err != nil {
				return nil, err
			}
		}
		if !isSubtypeOf(a.TypeName, TypeNumeric) {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("format-number() first argument must be numeric, got %s", a.TypeName)}
		}
	}

	// Picture argument (xs:string); compat mode applies fn:string to its first item.
	var picture string
	var err error
	if compat {
		picture, err = compatArgString(args[1])
	} else {
		picture, err = coerceArgToStringRequired(ctx, args[1])
	}
	if err != nil {
		return nil, err
	}

	// Optional decimal-format-name argument (xs:string), coerced the same way.
	df := defaultDecimalFormat(ctx)
	if len(args) > 2 && seqLen(args[2]) > 0 {
		var formatName string
		if compat {
			formatName, err = compatArgString(args[2])
		} else {
			formatName, err = coerceArgToString(ctx, args[2])
		}
		if err != nil {
			return nil, err
		}
		if df, err = resolveDecimalFormat(ctx, formatName); err != nil {
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
	// Negative precision: round to 10^(-precision). Operate on the rational
	// directly (divide, round-half-towards-+∞, multiply back) rather than
	// flooring to an integer first — flooring discards the fractional part and
	// can push the value past the rounding boundary (e.g. -249.9 floors to -250
	// or even -251, landing on the wrong multiple of 100).
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-precision)), nil)
	scaleRat := new(big.Rat).SetInt(scale)
	divided := new(big.Rat).Quo(r, scaleRat)
	rounded := ratRound(divided)
	return new(big.Rat).Mul(rounded, scaleRat)
}

// ratRoundPrecisionHalfToEven rounds a *big.Rat to the given precision (number
// of decimal places) using half-to-even rounding. Unlike the icu helper it
// applies no coarse magnitude clamp: callers pass a precision whose scale has
// already been bounded by the operand's own digit count (via resolveRoundScale),
// so 10^|precision| is bounded by the operand and never astronomically large.
func ratRoundPrecisionHalfToEven(r *big.Rat, precision int) *big.Rat {
	if precision >= 0 {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
		scaleRat := new(big.Rat).SetInt(scale)
		shifted := new(big.Rat).Mul(r, scaleRat)
		rounded := icu.RatRoundHalfToEvenInt(shifted)
		return new(big.Rat).SetFrac(rounded, new(big.Int).Set(scale))
	}
	// Negative precision: round to 10^(-precision). Operate on the rational
	// directly (divide, round-half-to-even, multiply back) rather than flooring
	// to an integer first — flooring would discard the fractional part that can
	// break a tie at the rounding boundary (e.g. ...896.5...123 must round up,
	// not be treated as an exact half that lands on the even digit).
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-precision)), nil)
	scaleRat := new(big.Rat).SetInt(scale)
	divided := new(big.Rat).Quo(r, scaleRat)
	rounded := icu.RatRoundHalfToEvenInt(divided)
	return new(big.Rat).Mul(new(big.Rat).SetInt(rounded), scaleRat)
}

// ratFloorInt returns floor of a rational as big.Int.
func ratFloorInt(r *big.Rat) *big.Int {
	q := new(big.Int).Div(r.Num(), r.Denom())
	if r.Sign() < 0 && new(big.Int).Mul(q, r.Denom()).Cmp(r.Num()) != 0 {
		q.Sub(q, big.NewInt(1))
	}
	return q
}
