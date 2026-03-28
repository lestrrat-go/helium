package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
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
	return SingleInteger(int64(seqLen(args[0]))), nil
}

// aggregateTypeFamily classifies an atomic type for aggregate type checking.
func aggregateTypeFamily(typeName string) string {
	if isIntegerDerived(typeName) {
		return "numeric" //nolint:goconst
	}
	if isStringDerived(typeName) {
		return lexicon.TypeString
	}
	switch typeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return "numeric"
	case TypeUntypedAtomic:
		return "numeric" // untypedAtomic promotes to double
	case TypeString, TypeAnyURI:
		return lexicon.TypeString
	case TypeYearMonthDuration:
		return "duration:YM" //nolint:goconst
	case TypeDayTimeDuration:
		return "duration:DT" //nolint:goconst
	case TypeDuration:
		return lexicon.TypeDuration
	case TypeDate:
		return "date"
	case TypeDateTime:
		return "dateTime"
	case TypeTime:
		return "time"
	case TypeBoolean:
		return lexicon.TypeBoolean
	case TypeBase64Binary:
		return "base64Binary"
	case TypeHexBinary:
		return "hexBinary"
	}
	return ""
}

// resolveCollationArg resolves the optional collation argument at args[idx].
// Returns nil collation (use default) if the argument is absent or is the
// codepoint collation.
func resolveCollationArg(args []Sequence, idx int) (*collationImpl, error) {
	if len(args) <= idx || seqLen(args[idx]) == 0 {
		return nil, nil //nolint:nilnil
	}
	uri, err := coerceArgToString(args[idx])
	if err != nil {
		return nil, err
	}
	if uri == lexicon.CollationCodepoint {
		return nil, nil //nolint:nilnil
	}
	return resolveCollation(uri, "")
}

// validateCollationArg checks if the collation argument at args[idx] is supported.
func validateCollationArg(args []Sequence, idx int) error {
	_, err := resolveCollationArg(args, idx)
	return err
}

func checkSumAvgType(a AtomicValue) error {
	family := aggregateTypeFamily(a.TypeName)
	switch family {
	case "numeric", "duration:YM", "duration:DT":
		return nil
	}
	return &XPathError{
		Code:    errCodeFORG0006,
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
		Code:    errCodeFORG0006,
		Message: fmt.Sprintf("incompatible types in aggregate: %s and %s", family, newFamily),
	}
}

func fnAvg(_ context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil //nolint:nilnil
	}
	var family string
	allDecOrInt := true
	sumRat := new(big.Rat)
	var sumFloat float64
	widest := TypeInteger

	for _, a := range atoms {
		a, err = promoteForAggregate(a)
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
		if numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if isIntegerDerived(a.TypeName) {
			if v, ok := a.Value.(int64); ok {
				sumRat.Add(sumRat, new(big.Rat).SetInt64(v))
				sumFloat += float64(v)
			} else {
				sumRat.Add(sumRat, new(big.Rat).SetInt(a.BigInt()))
				sumFloat += a.ToFloat64()
			}
		} else if a.TypeName == TypeDecimal {
			sumRat.Add(sumRat, a.BigRat())
			f, _ := a.BigRat().Float64()
			sumFloat += f
		} else {
			allDecOrInt = false
			sumFloat += a.ToFloat64()
		}
	}
	if family == "duration:YM" || family == "duration:DT" {
		atomSeq := make(ItemSlice, len(atoms))
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
	avg := sumFloat / float64(count)
	if widest == TypeFloat {
		return SingleFloat(avg), nil
	}
	return SingleDouble(avg), nil
}

func avgDurations(seq Sequence, family string) (Sequence, error) {
	totalMonths := new(big.Int)
	var totalSeconds float64
	for item := range seqItems(seq) {
		a, _ := AtomizeItem(item)
		d := a.DurationVal()
		if d.Negative {
			totalMonths.Sub(totalMonths, big.NewInt(int64(d.Months)))
			totalSeconds -= d.Seconds
		} else {
			totalMonths.Add(totalMonths, big.NewInt(int64(d.Months)))
			totalSeconds += d.Seconds
		}
	}
	if !totalMonths.IsInt64() {
		return nil, &XPathError{Code: errCodeFODT0002, Message: "duration overflow"}
	}
	count := seqLen(seq)
	// Per XPath F&O spec: months are rounded "half towards positive infinity"
	// i.e. math.Floor(months + 0.5), matching op:divide-yearMonthDuration behavior.
	monthsF := float64(totalMonths.Int64()) / float64(count)
	avgMonths := int(math.Floor(monthsF + 0.5))
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
func promoteForAggregate(a AtomicValue) (AtomicValue, error) {
	if a.TypeName == TypeDecimal {
		if s, ok := a.Value.(string); ok {
			dec, err := CastFromString(s, TypeDecimal)
			if err != nil {
				return AtomicValue{}, &XPathError{
					Code:    errCodeFORG0001,
					Message: fmt.Sprintf("cannot promote %q to xs:decimal", s),
				}
			}
			return dec, nil
		}
	}
	if a.TypeName == TypeUntypedAtomic {
		f, err := castToDouble(a)
		if err != nil {
			return AtomicValue{}, &XPathError{
				Code:    errCodeFORG0001,
				Message: fmt.Sprintf("cannot promote %q to xs:double", a.StringVal()),
			}
		}
		return f, nil
	}
	if isIntegerDerived(a.TypeName) && a.TypeName != TypeInteger {
		return AtomicValue{TypeName: TypeInteger, Value: a.Value}, nil
	}
	// User-defined schema types: promote based on the underlying Go value.
	if !IsKnownXSDType(a.TypeName) && a.TypeName != "" {
		switch a.Value.(type) {
		case int64, *big.Int:
			return AtomicValue{TypeName: TypeInteger, Value: a.Value}, nil
		case *big.Rat:
			return AtomicValue{TypeName: TypeDecimal, Value: a.BigRat()}, nil
		case float64, *FloatValue:
			return AtomicValue{TypeName: TypeDouble, Value: a.Value}, nil
		case float32:
			return AtomicValue{TypeName: TypeFloat, Value: a.Value}, nil
		}
	}
	return a, nil
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
			if v, ok := best.Value.(int64); ok {
				return AtomicValue{TypeName: TypeDecimal, Value: new(big.Rat).SetInt64(v)}
			}
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

func fnMax(ctx context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil //nolint:nilnil
	}
	coll, err := getCollation(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	if coll == codepointCollation {
		coll = nil
	}
	return maxMinCommon(atoms, true, coll)
}

func fnMin(ctx context.Context, args []Sequence) (Sequence, error) {
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, nil //nolint:nilnil
	}
	coll, err := getCollation(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	if coll == codepointCollation {
		coll = nil
	}
	return maxMinCommon(atoms, false, coll)
}

func maxMinCommon(atoms []AtomicValue, isMax bool, coll *collationImpl) (Sequence, error) {
	fnName := "fn:min"
	if isMax {
		fnName = "fn:max"
	}
	// Single-item case: validate type but preserve derived type
	if len(atoms) == 1 {
		a := atoms[0]
		if a.TypeName == TypeUntypedAtomic {
			var err error
			a, err = promoteForAggregate(a)
			if err != nil {
				return nil, err
			}
		}
		family := aggregateTypeFamily(a.TypeName)
		if family == "" || family == lexicon.TypeDuration {
			return nil, &XPathError{
				Code:    errCodeFORG0006,
				Message: fmt.Sprintf("invalid type %s for %s", a.TypeName, fnName),
			}
		}
		return SingleAtomic(a), nil
	}
	var family string
	var best AtomicValue
	widest := TypeInteger
	first := true
	hasNaN := false
	var err error
	for _, a := range atoms {
		a, err = promoteForAggregate(a)
		if err != nil {
			return nil, err
		}
		newFamily := aggregateTypeFamily(a.TypeName)
		if newFamily == "" || newFamily == "duration" {
			return nil, &XPathError{
				Code:    errCodeFORG0006,
				Message: fmt.Sprintf("invalid type %s for %s", a.TypeName, fnName),
			}
		}
		if family == "" {
			family = newFamily
		} else if family != newFamily {
			return nil, &XPathError{
				Code:    errCodeFORG0006,
				Message: fmt.Sprintf("incompatible types in %s: %s and %s", fnName, family, newFamily),
			}
		}
		if family == "numeric" && numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if (a.TypeName == TypeDouble || a.TypeName == TypeFloat) && a.FloatVal().IsNaN() {
			hasNaN = true
			continue
		}
		if first {
			best = a
			first = false
			continue
		}
		var cmp bool
		if coll != nil && family == lexicon.TypeString {
			r := coll.compare(a.StringVal(), best.StringVal())
			cmp = (isMax && r > 0) || (!isMax && r < 0)
		} else {
			if isMax {
				cmp, err = ValueCompare(TokenGt, a, best)
			} else {
				cmp, err = ValueCompare(TokenLt, a, best)
			}
			if err != nil {
				return nil, err
			}
		}
		if cmp {
			best = a
		}
	}
	if hasNaN {
		nanType := widest
		if nanType != TypeFloat {
			nanType = TypeDouble
		}
		return SingleAtomic(AtomicValue{TypeName: nanType, Value: NewDouble(math.NaN())}), nil
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
	if len(atoms) == 1 {
		a := atoms[0]
		// Promote untypedAtomic to double per spec, but preserve derived integer types
		if a.TypeName == TypeUntypedAtomic {
			a, err = promoteForAggregate(a)
			if err != nil {
				return nil, err
			}
		}
		if err := checkSumAvgType(a); err != nil {
			return nil, err
		}
		return SingleAtomic(a), nil
	}
	var family string
	allInt := true
	allDecOrInt := true
	sumInt := new(big.Int)
	sumRat := new(big.Rat)
	var sumFloat float64
	widest := TypeInteger

	for _, a := range atoms {
		a, err = promoteForAggregate(a)
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
		if numericTypeWidth(a.TypeName) > numericTypeWidth(widest) {
			widest = a.TypeName
		}
		if isIntegerDerived(a.TypeName) {
			if v, ok := a.Value.(int64); ok {
				sumInt.Add(sumInt, big.NewInt(v))
				sumRat.Add(sumRat, new(big.Rat).SetInt64(v))
				sumFloat += float64(v)
			} else {
				sumInt.Add(sumInt, a.BigInt())
				sumRat.Add(sumRat, new(big.Rat).SetInt(a.BigInt()))
				sumFloat += a.ToFloat64()
			}
		} else if a.TypeName == TypeDecimal {
			allInt = false
			sumRat.Add(sumRat, a.BigRat())
			f, _ := a.BigRat().Float64()
			sumFloat += f
		} else {
			allInt = false
			allDecOrInt = false
			sumFloat += a.ToFloat64()
		}
	}
	if family == "duration:YM" || family == "duration:DT" {
		atomSeq := make(ItemSlice, len(atoms))
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
	for item := range seqItems(seq) {
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

func fnDistinctValues(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil //nolint:nilnil
	}
	if err := validateCollationArg(args, 1); err != nil {
		return nil, err
	}
	coll, err := getCollation(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	var implicitTZ *time.Location
	if ec := getFnContext(ctx); ec != nil {
		implicitTZ = ec.getImplicitTimezone()
	}
	// Atomize the entire sequence so list-typed nodes are
	// decomposed into individual atomic values.
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	var result []AtomicValue
	seenFast := make(map[string]struct{})
	var numericDecInt []AtomicValue
	var numericFloat []AtomicValue
	var numericDouble []AtomicValue
	seenNaN := false
	for _, a := range atoms {
		// Promote untypedAtomic to string for comparison per spec
		if a.TypeName == TypeUntypedAtomic {
			a = AtomicValue{TypeName: TypeString, Value: a.StringVal()}
		}
		// NaN handling: op:is-same-key treats all NaN values as equal
		if isAtomicNaN(a) {
			if seenNaN {
				continue
			}
			seenNaN = true
			result = append(result, a)
			continue
		}
		if group, key, ok := distinctValueFastKey(a); ok {
			// When a non-codepoint collation is active, string fast
			// keys must go through the collation's key function so
			// that collation-equal strings share the same bucket.
			if coll != nil && group == distinctGroupString {
				key = "s:" + string(coll.key(a.StringVal()))
			}
			if _, exists := seenFast[key]; exists {
				continue
			}
			found, err := distinctValueSeenInOtherNumericGroups(a, group, coll, implicitTZ, numericDecInt, numericFloat, numericDouble)
			if err != nil {
				return nil, err
			}
			if found {
				continue
			}
			seenFast[key] = struct{}{}
			result = append(result, a)
			switch group {
			case distinctGroupDecimalInt:
				numericDecInt = append(numericDecInt, a)
			case distinctGroupFloat:
				numericFloat = append(numericFloat, a)
			case distinctGroupDouble:
				numericDouble = append(numericDouble, a)
			}
			continue
		}
		found := false
		for _, existing := range result {
			if isAtomicNaN(existing) {
				continue
			}
			eq, err := distinctValueEqual(a, existing, coll, implicitTZ)
			if err != nil {
				// Incomparable types are considered distinct
				continue
			}
			if eq {
				found = true
				break
			}
		}
		if !found {
			result = append(result, a)
		}
	}
	seq := make(ItemSlice, len(result))
	for i, a := range result {
		seq[i] = a
	}
	return seq, nil
}

type distinctGroup uint8

const (
	distinctGroupUnknown distinctGroup = iota
	distinctGroupString
	distinctGroupBoolean
	distinctGroupDecimalInt
	distinctGroupFloat
	distinctGroupDouble
)

func distinctValueFastKey(a AtomicValue) (distinctGroup, string, bool) {
	switch {
	case isStringDerived(a.TypeName):
		return distinctGroupString, "s:" + a.StringVal(), true
	case a.TypeName == TypeAnyURI:
		return distinctGroupString, "s:" + stringFromAtomic(a), true
	case a.TypeName == TypeBoolean:
		return distinctGroupBoolean, "b:" + strconv.FormatBool(a.BooleanVal()), true
	case isIntegerDerived(a.TypeName) || a.TypeName == TypeDecimal:
		return distinctGroupDecimalInt, "n:" + toRatForCompare(a).RatString(), true
	case a.TypeName == TypeFloat:
		f := float32(a.ToFloat64())
		if f == 0 {
			f = 0
		}
		return distinctGroupFloat, "f:" + strconv.FormatUint(uint64(math.Float32bits(f)), 16), true
	case a.TypeName == TypeDouble:
		f := a.ToFloat64()
		if f == 0 {
			f = 0
		}
		return distinctGroupDouble, "d:" + strconv.FormatUint(math.Float64bits(f), 16), true
	default:
		return distinctGroupUnknown, "", false
	}
}

func distinctValueSeenInOtherNumericGroups(
	a AtomicValue,
	group distinctGroup,
	coll *collationImpl,
	implicitTZ *time.Location,
	decimalInts []AtomicValue,
	floats []AtomicValue,
	doubles []AtomicValue,
) (bool, error) {
	switch group {
	case distinctGroupDecimalInt:
		return distinctValueSeenInSet(a, coll, implicitTZ, floats, doubles)
	case distinctGroupFloat:
		return distinctValueSeenInSet(a, coll, implicitTZ, decimalInts, doubles)
	case distinctGroupDouble:
		return distinctValueSeenInSet(a, coll, implicitTZ, decimalInts, floats)
	default:
		return false, nil
	}
}

func distinctValueSeenInSet(
	a AtomicValue,
	coll *collationImpl,
	implicitTZ *time.Location,
	sets ...[]AtomicValue,
) (bool, error) {
	for _, set := range sets {
		for _, existing := range set {
			eq, err := distinctValueEqual(a, existing, coll, implicitTZ)
			if err != nil {
				continue
			}
			if eq {
				return true, nil
			}
		}
	}
	return false, nil
}

func distinctValueEqual(a, b AtomicValue, coll *collationImpl, implicitTZ *time.Location) (bool, error) {
	if coll != nil {
		aStr := isStringDerived(a.TypeName) || a.TypeName == TypeAnyURI
		bStr := isStringDerived(b.TypeName) || b.TypeName == TypeAnyURI
		if aStr && bStr {
			return coll.compare(stringFromAtomic(a), stringFromAtomic(b)) == 0, nil
		}
	}
	return ValueCompareWithImplicitTimezone(TokenEq, a, b, implicitTZ)
}

// isAtomicNaN returns true if the atomic value is a float or double NaN.
func isAtomicNaN(a AtomicValue) bool {
	if a.TypeName == TypeDouble || a.TypeName == TypeFloat {
		return a.FloatVal().IsNaN()
	}
	return false
}
