package xpath3

import (
	"context"
	"math"
)

func init() {
	registerFn("count", 1, 1, fnCount)
	registerFn("avg", 1, 1, fnAvg)
	registerFn("max", 1, 2, fnMax)
	registerFn("min", 1, 2, fnMin)
	registerFn("sum", 1, 2, fnSum)
	registerFn("distinct-values", 1, 2, fnDistinctValues)
}

func fnCount(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleInteger(int64(len(args[0]))), nil
}

func fnAvg(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	var sum float64
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		sum += promoteToDouble(a)
	}
	return SingleDouble(sum / float64(len(args[0]))), nil
}

func fnMax(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	max := math.Inf(-1)
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		v := promoteToDouble(a)
		if math.IsNaN(v) {
			return SingleDouble(math.NaN()), nil
		}
		if v > max {
			max = v
		}
	}
	return SingleDouble(max), nil
}

func fnMin(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	min := math.Inf(1)
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		v := promoteToDouble(a)
		if math.IsNaN(v) {
			return SingleDouble(math.NaN()), nil
		}
		if v < min {
			min = v
		}
	}
	return SingleDouble(min), nil
}

func fnSum(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		if len(args) > 1 {
			return args[1], nil
		}
		return SingleInteger(0), nil
	}
	var sum float64
	allInt := true
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		if !isIntegerDerived(a.TypeName) {
			allInt = false
		}
		sum += promoteToDouble(a)
	}
	if allInt {
		return SingleInteger(int64(sum)), nil
	}
	return SingleDouble(sum), nil
}

func fnDistinctValues(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool)
	var result Sequence
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
		key := a.TypeName + ":" + s
		if !seen[key] {
			seen[key] = true
			result = append(result, a)
		}
	}
	return result, nil
}
