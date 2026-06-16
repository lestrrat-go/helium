package xpath3

import (
	"context"
	"math"
)

func init() {
	registerNS(NSMath, "pi", 0, 0, fnMathPi)
	registerNS(NSMath, "exp", 1, 1, fnMathExp)
	registerNS(NSMath, "exp10", 1, 1, fnMathExp10)
	registerNS(NSMath, "log", 1, 1, fnMathLog)
	registerNS(NSMath, "log10", 1, 1, fnMathLog10)
	registerNS(NSMath, "pow", 2, 2, fnMathPow)
	registerNS(NSMath, "sqrt", 1, 1, fnMathSqrt)
	registerNS(NSMath, "sin", 1, 1, fnMathSin)
	registerNS(NSMath, "cos", 1, 1, fnMathCos)
	registerNS(NSMath, "tan", 1, 1, fnMathTan)
	registerNS(NSMath, "asin", 1, 1, fnMathAsin)
	registerNS(NSMath, "acos", 1, 1, fnMathAcos)
	registerNS(NSMath, "atan", 1, 1, fnMathAtan)
	registerNS(NSMath, "atan2", 2, 2, fnMathAtan2)
}

func fnMathPi(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleDouble(math.Pi), nil
}

// coerceMathArg coerces an xs:double? math argument. Empty sequences yield
// (0, false, nil) so the caller can return an empty result; otherwise the
// argument is required to be a single numeric/untypedAtomic item, rejecting
// multi-item and non-numeric values with XPTY0004.
func coerceMathArg(seq Sequence) (float64, bool, error) {
	if seqLen(seq) == 0 {
		return 0, false, nil
	}
	f, err := coerceArgToDoubleRequired(seq)
	if err != nil {
		return 0, false, err
	}
	return f, true, nil
}

func mathUnary(args []Sequence, fn func(float64) float64) (Sequence, error) {
	a, ok, err := coerceMathArg(args[0])
	if err != nil {
		return nil, err
	}
	if !ok {
		return validNilSequence, nil
	}
	return SingleDouble(fn(a)), nil
}

func fnMathExp(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Exp)
}

func fnMathExp10(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, func(x float64) float64 { return math.Pow(10, x) })
}

func fnMathLog(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Log)
}

func fnMathLog10(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Log10)
}

func fnMathPow(_ context.Context, args []Sequence) (Sequence, error) {
	// math:pow($x as xs:double?, $y as xs:numeric): empty $x yields empty, but
	// only once $y has been validated. $y is a required single numeric value,
	// so a malformed or empty $y must raise XPTY0004 even when $x is empty —
	// function-conversion validates each argument independently of the others.
	b, err := coerceArgToDoubleRequired(args[1])
	if err != nil {
		return nil, err
	}
	a, aOK, err := coerceMathArg(args[0])
	if err != nil {
		return nil, err
	}
	if !aOK {
		return validNilSequence, nil
	}
	return SingleDouble(math.Pow(a, b)), nil
}

func fnMathSqrt(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Sqrt)
}

func fnMathSin(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Sin)
}

func fnMathCos(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Cos)
}

func fnMathTan(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Tan)
}

func fnMathAsin(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Asin)
}

func fnMathAcos(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Acos)
}

func fnMathAtan(_ context.Context, args []Sequence) (Sequence, error) {
	return mathUnary(args, math.Atan)
}

func fnMathAtan2(_ context.Context, args []Sequence) (Sequence, error) {
	// math:atan2($y as xs:double, $x as xs:double): both required, exactly one
	// numeric value each.
	a, err := coerceArgToDoubleRequired(args[0])
	if err != nil {
		return nil, err
	}
	b, err := coerceArgToDoubleRequired(args[1])
	if err != nil {
		return nil, err
	}
	return SingleDouble(math.Atan2(a, b)), nil
}
