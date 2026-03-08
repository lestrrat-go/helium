package xpath3

import (
	"context"
	"math"
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

func fnAbs(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	if a.TypeName == TypeInteger {
		v := a.IntegerVal()
		if v < 0 {
			v = -v
		}
		return SingleInteger(v), nil
	}
	return SingleDouble(math.Abs(promoteToDouble(a))), nil
}

func fnCeiling(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	if a.TypeName == TypeInteger {
		return SingleInteger(a.IntegerVal()), nil
	}
	return SingleDouble(math.Ceil(promoteToDouble(a))), nil
}

func fnFloor(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	if a.TypeName == TypeInteger {
		return SingleInteger(a.IntegerVal()), nil
	}
	return SingleDouble(math.Floor(promoteToDouble(a))), nil
}

func fnRound(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	if a.TypeName == TypeInteger {
		return SingleInteger(a.IntegerVal()), nil
	}
	n := promoteToDouble(a)
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleDouble(n), nil
	}
	// XPath round: round half towards positive infinity
	r := math.Floor(n + 0.5)
	if r == 0 && n < 0 {
		r = math.Copysign(0, -1)
	}
	return SingleDouble(r), nil
}

func fnRoundHalfToEven(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	n := promoteToDouble(a)
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return SingleDouble(n), nil
	}
	precision := 0
	if len(args) > 1 && len(args[1]) > 0 {
		pa, err := AtomizeItem(args[1][0])
		if err != nil {
			return nil, err
		}
		precision = int(promoteToDouble(pa))
	}
	scale := math.Pow(10, float64(precision))
	return SingleDouble(math.RoundToEven(n*scale) / scale), nil
}

func fnFormatInteger(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FOER0000", Message: "format-integer not yet implemented"}
}

func fnFormatNumber(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FOER0000", Message: "format-number not yet implemented"}
}
