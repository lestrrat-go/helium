package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
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
func aggregateTypeFamily(typeName string) string {
	if isIntegerDerived(typeName) {
		return "numeric"
	}
	if isStringDerived(typeName) {
		return "string"
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
	case TypeBase64Binary:
		return "base64Binary"
	case TypeHexBinary:
		return "hexBinary"
	}
	return ""
}

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

func checkAggregateHomogeneity(family, newFamily string) (string, error) {
	if family == "" {
		return newFamily, nil
	}
	if family == newFamily {
		return family, nil
	}
	if family == "numeric" && newFamily == "numeric" {
		return "numeric", nil
	}
	return "", &XPathError{
		Code:    "FORG0006",
		Message: fmt.Sprintf("incompatible types in aggregate: %s and %s", family, newFamily),
	}
}

func fnAvg(_ context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil
	}
	var family string
	// Track whether all values are integer/decimal for type-preserving avg
	allInt := true
	allDecOrInt := true
	sumInt := new(big.Int)
	sumRat := new(big.Rat)
	var sumFloat float64
	widest := TypeInteger

	for _, a := range atoms {
		if err := checkSumAvgType(a); err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		family, err = checkAggregateHomogeneity(family, newFamily)
		if err != nil {
			return nil, err
		}
		if numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if isIntegerDerived(a.TypeName) {
			sumInt.Add(sumInt, a.BigInt())
			sumRat.Add(sumRat, new(big.Rat).SetInt(a.BigInt()))
		} else if a.TypeName == TypeDecimal {
			allInt = false
			sumRat.Add(sumRat, a.BigRat())
		} else {
			allInt = false
			allDecOrInt = false
			sumFloat += a.ToFloat64()
		}
	}
	if family == "duration:YM" || family == "duration:DT" {
		atomSeq := make(Sequence, len(atoms))
		for i, a := range atoms {
			atomSeq[i] = a
		}
		return avgDurations(atomSeq, family)
	}
	count := len(atoms)
	if allDecOrInt {
		// avg returns decimal for integer/decimal inputs
		countRat := new(big.Rat).SetInt64(int64(count))
		return SingleDecimal(new(big.Rat).Quo(sumRat, countRat)), nil
	}
	// Float/double path
	if allInt {
		f, _ := new(big.Float).SetInt(sumInt).Float64()
		sumFloat = f
	}
	avg := sumFloat / float64(count)
	if widest == TypeFloat {
		return SingleFloat(avg), nil
	}
	return SingleDouble(avg), nil
}

func avgDurations(seq Sequence, family string) (Sequence, error) {
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
	count := len(seq)
	avgMonths := totalMonths / count
	avgSeconds := totalSeconds / float64(count)
	negative := avgMonths < 0 || avgSeconds < 0
	if negative {
		avgMonths = -avgMonths
		avgSeconds = -avgSeconds
	}
	typeName := TypeYearMonthDuration
	if family == "duration:DT" {
		typeName = TypeDayTimeDuration
	}
	return SingleAtomic(AtomicValue{
		TypeName: typeName,
		Value:    Duration{Months: avgMonths, Seconds: avgSeconds, Negative: negative},
	}), nil
}

// promoteForAggregate promotes an atomic value for aggregate operations.
func promoteForAggregate(a AtomicValue) AtomicValue {
	if a.TypeName == TypeUntypedAtomic {
		f, err := castToDouble(a)
		if err != nil {
			return AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}
		}
		return f
	}
	if isIntegerDerived(a.TypeName) && a.TypeName != TypeInteger {
		return AtomicValue{TypeName: TypeInteger, Value: a.BigInt()}
	}
	return a
}

// promoteResult promotes the result of fn:max/fn:min to the widest numeric type.
func promoteResult(best AtomicValue, widest string) AtomicValue {
	if best.TypeName == widest {
		return best
	}
	switch widest {
	case TypeDouble:
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(best.ToFloat64())}
	case TypeFloat:
		return AtomicValue{TypeName: TypeFloat, Value: NewFloat(best.ToFloat64())}
	case TypeDecimal:
		if isIntegerDerived(best.TypeName) {
			return AtomicValue{TypeName: TypeDecimal, Value: new(big.Rat).SetInt(best.BigInt())}
		}
	}
	return best
}

func numericTypeWidth(typeName string) int {
	switch typeName {
	case TypeDouble:
		return 4
	case TypeFloat:
		return 3
	case TypeDecimal:
		return 2
	default:
		return 1
	}
}

func fnMax(_ context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil
	}
	var family string
	var best AtomicValue
	widest := TypeInteger
	first := true
	for _, a := range atoms {
		a = promoteForAggregate(a)
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
		if family == "numeric" && numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if (a.TypeName == TypeDouble || a.TypeName == TypeFloat) && a.FloatVal().IsNaN() {
			return SingleAtomic(AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}), nil
		}
		if first {
			best = a
			first = false
			continue
		}
		gt, err := ValueCompare(TokenGt, a, best)
		if err != nil {
			return nil, err
		}
		if gt {
			best = a
		}
	}
	if family == "numeric" {
		best = promoteResult(best, widest)
	}
	return SingleAtomic(best), nil
}

func fnMin(_ context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil
	}
	var family string
	var best AtomicValue
	widest := TypeInteger
	first := true
	for _, a := range atoms {
		a = promoteForAggregate(a)
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
		if family == "numeric" && numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if (a.TypeName == TypeDouble || a.TypeName == TypeFloat) && a.FloatVal().IsNaN() {
			return SingleAtomic(AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}), nil
		}
		if first {
			best = a
			first = false
			continue
		}
		lt, err := ValueCompare(TokenLt, a, best)
		if err != nil {
			return nil, err
		}
		if lt {
			best = a
		}
	}
	if family == "numeric" {
		best = promoteResult(best, widest)
	}
	return SingleAtomic(best), nil
}

func fnSum(_ context.Context, args []Sequence) (Sequence, error) {
	// Atomize to handle arrays: sum([1,2,3]) should flatten the array
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		if len(args) > 1 {
			return args[1], nil
		}
		return SingleInteger(0), nil
	}
	var family string
	allInt := true
	allDecOrInt := true
	sumInt := new(big.Int)
	sumRat := new(big.Rat)
	var sumFloat float64
	widest := TypeInteger

	for _, a := range atoms {
		a = promoteForAggregate(a)
		if err := checkSumAvgType(a); err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		family, err = checkAggregateHomogeneity(family, newFamily)
		if err != nil {
			return nil, err
		}
		if numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if isIntegerDerived(a.TypeName) {
			sumInt.Add(sumInt, a.BigInt())
			sumRat.Add(sumRat, new(big.Rat).SetInt(a.BigInt()))
		} else if a.TypeName == TypeDecimal {
			allInt = false
			sumRat.Add(sumRat, a.BigRat())
		} else {
			allInt = false
			allDecOrInt = false
			sumFloat += a.ToFloat64()
		}
	}
	if family == "duration:YM" || family == "duration:DT" {
		atomSeq := make(Sequence, len(atoms))
		for i, a := range atoms {
			atomSeq[i] = a
		}
		return sumDurations(atomSeq, family)
	}
	if allInt {
		return SingleIntegerBig(sumInt), nil
	}
	if allDecOrInt {
		return SingleDecimal(sumRat), nil
	}
	if widest == TypeFloat {
		return SingleFloat(sumFloat), nil
	}
	return SingleDouble(sumFloat), nil
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
