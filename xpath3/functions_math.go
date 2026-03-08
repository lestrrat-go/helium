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

func mathUnary(args []Sequence, fn func(float64) float64) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	return SingleDouble(fn(promoteToDouble(a))), nil
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
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	b, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	return SingleDouble(math.Pow(promoteToDouble(a), promoteToDouble(b))), nil
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
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	b, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	return SingleDouble(math.Atan2(promoteToDouble(a), promoteToDouble(b))), nil
}
