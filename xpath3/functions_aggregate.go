package xpath3

import (
	"context"
	"fmt"
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

// aggregateTypeFamily classifies an atomic type for aggregate type checking.
// Returns "numeric", "string", "duration:YM", "duration:DT", "duration",
// "date", "dateTime", "time", or "" for unsupported types.
func aggregateTypeFamily(typeName string) string {
	if isIntegerDerived(typeName) {
		return "numeric"
	}
	switch typeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return "numeric"
	case TypeUntypedAtomic:
		return "numeric" // untypedAtomic promotes to double
	case TypeString, TypeAnyURI:
		return "string"
	case TypeYearMonthDuration:
		return "duration:YM"
	case TypeDayTimeDuration:
		return "duration:DT"
	case TypeDuration:
		return "duration"
	case TypeDate:
		return "date"
	case TypeDateTime:
		return "dateTime"
	case TypeTime:
		return "time"
	case TypeBoolean:
		return "boolean"
	}
	return ""
}

// checkAggregateType returns FORG0006 if the type is not valid for avg/sum.
func checkSumAvgType(a AtomicValue) error {
	family := aggregateTypeFamily(a.TypeName)
	switch family {
	case "numeric", "duration:YM", "duration:DT":
		return nil
	}
	return &XPathError{
		Code:    "FORG0006",
		Message: fmt.Sprintf("invalid type %s for aggregate function", a.TypeName),
	}
}

// checkAggregateHomogeneity checks that all values are in the same type family
// for avg/sum. Returns the common family or error.
func checkAggregateHomogeneity(family, newFamily string) (string, error) {
	if family == "" {
		return newFamily, nil
	}
	if family == newFamily {
		return family, nil
	}
	// numeric types are compatible with each other
	if family == "numeric" && newFamily == "numeric" {
		return "numeric", nil
	}
	return "", &XPathError{
		Code:    "FORG0006",
		Message: fmt.Sprintf("incompatible types in aggregate: %s and %s", family, newFamily),
	}
}

func fnAvg(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	var family string
	var sum float64
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		if err := checkSumAvgType(a); err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		family, err = checkAggregateHomogeneity(family, newFamily)
		if err != nil {
			return nil, err
		}
		sum += promoteToDouble(a)
	}
	if family == "duration:YM" || family == "duration:DT" {
		// Duration avg: sum the durations, divide by count
		return avgDurations(args[0], family)
	}
	return SingleDouble(sum / float64(len(args[0]))), nil
}

func avgDurations(seq Sequence, family string) (Sequence, error) {
	var totalMonths int
	var totalSeconds float64
	var anyNegative bool
	for _, item := range seq {
		a, _ := AtomizeItem(item)
		d := a.DurationVal()
		if d.Negative {
			totalMonths -= d.Months
			totalSeconds -= d.Seconds
		} else {
			totalMonths += d.Months
			totalSeconds += d.Seconds
		}
		anyNegative = anyNegative || d.Negative
	}
	count := len(seq)
	avgMonths := totalMonths / count
	avgSeconds := totalSeconds / float64(count)
	negative := avgMonths < 0 || avgSeconds < 0
	if negative {
		avgMonths = -avgMonths
		avgSeconds = -avgSeconds
	}
	_ = anyNegative
	typeName := TypeYearMonthDuration
	if family == "duration:DT" {
		typeName = TypeDayTimeDuration
	}
	return SingleAtomic(AtomicValue{
		TypeName: typeName,
		Value:    Duration{Months: avgMonths, Seconds: avgSeconds, Negative: negative},
	}), nil
}

func fnMax(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	var family string
	max := math.Inf(-1)
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		if newFamily == "" {
			return nil, &XPathError{
				Code:    "FORG0006",
				Message: fmt.Sprintf("invalid type %s for fn:max", a.TypeName),
			}
		}
		if family == "" {
			family = newFamily
		} else if family != newFamily {
			return nil, &XPathError{
				Code:    "FORG0006",
				Message: fmt.Sprintf("incompatible types in fn:max: %s and %s", family, newFamily),
			}
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
	var family string
	min := math.Inf(1)
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		if newFamily == "" {
			return nil, &XPathError{
				Code:    "FORG0006",
				Message: fmt.Sprintf("invalid type %s for fn:min", a.TypeName),
			}
		}
		if family == "" {
			family = newFamily
		} else if family != newFamily {
			return nil, &XPathError{
				Code:    "FORG0006",
				Message: fmt.Sprintf("incompatible types in fn:min: %s and %s", family, newFamily),
			}
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
	var family string
	var sum float64
	allInt := true
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		if err := checkSumAvgType(a); err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		family, err = checkAggregateHomogeneity(family, newFamily)
		if err != nil {
			return nil, err
		}
		if !isIntegerDerived(a.TypeName) {
			allInt = false
		}
		sum += promoteToDouble(a)
	}
	if family == "duration:YM" || family == "duration:DT" {
		return sumDurations(args[0], family)
	}
	if allInt {
		return SingleInteger(int64(sum)), nil
	}
	return SingleDouble(sum), nil
}

func sumDurations(seq Sequence, family string) (Sequence, error) {
	var totalMonths int
	var totalSeconds float64
	for _, item := range seq {
		a, _ := AtomizeItem(item)
		d := a.DurationVal()
		if d.Negative {
			totalMonths -= d.Months
			totalSeconds -= d.Seconds
		} else {
			totalMonths += d.Months
			totalSeconds += d.Seconds
		}
	}
	negative := totalMonths < 0 || totalSeconds < 0
	if negative {
		totalMonths = -totalMonths
		totalSeconds = -totalSeconds
	}
	typeName := TypeYearMonthDuration
	if family == "duration:DT" {
		typeName = TypeDayTimeDuration
	}
	return SingleAtomic(AtomicValue{
		TypeName: typeName,
		Value:    Duration{Months: totalMonths, Seconds: totalSeconds, Negative: negative},
	}), nil
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
