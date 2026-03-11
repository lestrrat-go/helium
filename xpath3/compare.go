package xpath3

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// evalGeneralComparison implements general comparison (= != < <= > >=).
// Per XPath 3.1 Section 3.7.1: atomize both operands, then existentially
// quantify — true if ANY pair satisfies the value comparison.
func evalGeneralComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	result, err := GeneralCompare(e.Op, left, right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(result), nil
}

// evalValueComparison implements value comparison (eq ne lt le gt ge).
// Per XPath 3.1 Section 3.7.2: both operands must be single atomic values.
func evalValueComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	// Atomize operands (handles arrays, nodes, etc.)
	leftAtoms, err := AtomizeSequence(left)
	if err != nil {
		return nil, err
	}
	rightAtoms, err := AtomizeSequence(right)
	if err != nil {
		return nil, err
	}
	// Empty sequence yields empty sequence
	if len(leftAtoms) == 0 || len(rightAtoms) == 0 {
		return nil, nil
	}
	if len(leftAtoms) > 1 || len(rightAtoms) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "value comparison requires singletons"}
	}
	la := leftAtoms[0]
	ra := rightAtoms[0]
	result, err := ValueCompare(e.Op, la, ra)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(result), nil
}

func evalNodeComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	// Empty sequence yields empty sequence
	if len(left) == 0 || len(right) == 0 {
		return nil, nil
	}
	if len(left) > 1 || len(right) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires singletons"}
	}
	ln, ok := left[0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires node operands"}
	}
	rn, ok := right[0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires node operands"}
	}
	switch e.Op {
	case TokenIs:
		return SingleBoolean(ln.Node == rn.Node), nil
	case TokenNodePre:
		lp := ec.docOrder.Position(ln.Node)
		rp := ec.docOrder.Position(rn.Node)
		if lp < 0 || rp < 0 {
			ec.docOrder.BuildFrom(ixpath.DocumentRoot(ln.Node))
			lp = ec.docOrder.Position(ln.Node)
			rp = ec.docOrder.Position(rn.Node)
		}
		return SingleBoolean(lp < rp), nil
	case TokenNodeFol:
		lp := ec.docOrder.Position(ln.Node)
		rp := ec.docOrder.Position(rn.Node)
		if lp < 0 || rp < 0 {
			ec.docOrder.BuildFrom(ixpath.DocumentRoot(ln.Node))
			lp = ec.docOrder.Position(ln.Node)
			rp = ec.docOrder.Position(rn.Node)
		}
		return SingleBoolean(lp > rp), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

// GeneralCompare performs a general comparison between two sequences.
// Atomizes both sides and returns true if any pair of atomic values
// satisfies the operator.
func GeneralCompare(op TokenType, left, right Sequence) (bool, error) {
	leftAtoms, err := AtomizeSequence(left)
	if err != nil {
		return false, err
	}
	rightAtoms, err := AtomizeSequence(right)
	if err != nil {
		return false, err
	}
	for _, la := range leftAtoms {
		for _, ra := range rightAtoms {
			pa, pb, err := promoteForGeneralComparison(la, ra)
			if err != nil {
				return false, err
			}
			match, err := compareAtomic(op, pa, pb)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
	}
	return false, nil
}

// ValueCompare performs a value comparison between two atomic values.
func ValueCompare(op TokenType, a, b AtomicValue) (bool, error) {
	pa, pb := promoteForValueComparison(a, b)
	return compareAtomic(op, pa, pb)
}

// valueCompareThreeWay compares two atomic values and returns -1, 0, or 1.
// If coll is non-nil, it is used for string comparisons instead of codepoint order.
func valueCompareThreeWay(a, b AtomicValue, coll *collationImpl) (int, error) {
	pa, pb := promoteForValueComparison(a, b)

	// If both are strings and a collation is provided, use it
	aStr := isStringDerived(pa.TypeName) || pa.TypeName == TypeAnyURI
	bStr := isStringDerived(pb.TypeName) || pb.TypeName == TypeAnyURI
	if aStr && bStr && coll != nil {
		sa := stringFromAtomic(pa)
		sb := stringFromAtomic(pb)
		return coll.compare(sa, sb), nil
	}

	// Fall back to standard comparison
	less, err := compareAtomic(TokenLt, pa, pb)
	if err != nil {
		return 0, err
	}
	if less {
		return -1, nil
	}
	greater, err := compareAtomic(TokenGt, pa, pb)
	if err != nil {
		return 0, err
	}
	if greater {
		return 1, nil
	}
	return 0, nil
}

// comparisonFamily returns a type family string for comparison compatibility checking.
func comparisonFamily(typeName string) string {
	if isIntegerDerived(typeName) {
		return "numeric"
	}
	switch typeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return "numeric"
	case TypeString, TypeAnyURI:
		return "string"
	case TypeBoolean:
		return "boolean"
	case TypeDate:
		return "date"
	case TypeDateTime:
		return "dateTime"
	case TypeTime:
		return "time"
	case TypeDuration:
		return "duration"
	case TypeYearMonthDuration:
		return "duration:YM"
	case TypeDayTimeDuration:
		return "duration:DT"
	case TypeBase64Binary:
		return "base64Binary"
	case TypeHexBinary:
		return "hexBinary"
	case TypeQName:
		return "QName"
	case TypeGDay:
		return "gDay"
	case TypeGMonth:
		return "gMonth"
	case TypeGMonthDay:
		return "gMonthDay"
	case TypeGYear:
		return "gYear"
	case TypeGYearMonth:
		return "gYearMonth"
	}
	return ""
}

// promoteForValueComparison applies type promotion rules for value comparison (eq/ne/lt/gt/le/ge).
// Per XPath 3.1 Section 3.7.2: xs:untypedAtomic is always cast to xs:string.
// This differs from general comparison where untypedAtomic is cast to the other operand's type.
func promoteForValueComparison(a, b AtomicValue) (AtomicValue, AtomicValue) {
	if a.TypeName == TypeUntypedAtomic {
		a = AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)}
	}
	if b.TypeName == TypeUntypedAtomic {
		b = AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
	}
	return a, b
}

// promoteForGeneralComparison applies type promotion rules for general comparison (= != < > <= >=).
// Per XPath 3.1 Section 3.7.1 — untypedAtomic is cast to the type of the other operand.
func promoteForGeneralComparison(a, b AtomicValue) (AtomicValue, AtomicValue, error) {
	// untypedAtomic vs untypedAtomic → compare as string
	if a.TypeName == TypeUntypedAtomic && b.TypeName == TypeUntypedAtomic {
		return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)},
			AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}, nil
	}
	// untypedAtomic vs typed → cast untypedAtomic to the other's type
	if a.TypeName == TypeUntypedAtomic {
		castA, err := castUntypedToType(a, b.TypeName)
		if err != nil {
			return AtomicValue{}, AtomicValue{}, err
		}
		return castA, b, nil
	}
	if b.TypeName == TypeUntypedAtomic {
		castB, err := castUntypedToType(b, a.TypeName)
		if err != nil {
			return AtomicValue{}, AtomicValue{}, err
		}
		return a, castB, nil
	}
	return a, b, nil
}

// castUntypedToType casts an untypedAtomic value to the given target type.
// For general comparison, cast failures are errors (not silently ignored).
func castUntypedToType(untyped AtomicValue, targetType string) (AtomicValue, error) {
	// For numeric types, cast to double per spec
	if isIntegerDerived(targetType) || targetType == TypeDecimal || targetType == TypeFloat {
		targetType = TypeDouble
	}
	// String-derived types: cast to string for comparison
	if isStringDerived(targetType) {
		targetType = TypeString
	}
	return CastFromString(stringFromAtomic(untyped), targetType)
}

// isStringDerived returns true if the type derives from xs:string in the type hierarchy.
func isStringDerived(typeName string) bool {
	for typeName != "" && typeName != TypeAnyAtomicType {
		if typeName == TypeString {
			return true
		}
		typeName = xsdTypeParent[typeName]
	}
	return false
}

// stringFromAtomic extracts a string from an atomic value.
func stringFromAtomic(a AtomicValue) string {
	if s, ok := a.Value.(string); ok {
		return s
	}
	s, _ := atomicToString(a)
	return s
}

// compareAtomic compares two (already promoted) atomic values.
// Returns XPTY0004 if types are not comparable.
func compareAtomic(op TokenType, a, b AtomicValue) (bool, error) {
	// Map general comparison operators to value comparison operators
	op = normalizeCompareOp(op)

	// String comparison (includes string-derived types and anyURI)
	aStr := isStringDerived(a.TypeName) || a.TypeName == TypeAnyURI
	bStr := isStringDerived(b.TypeName) || b.TypeName == TypeAnyURI
	if aStr && bStr {
		sa := stringFromAtomic(a)
		sb := stringFromAtomic(b)
		cmp := strings.Compare(sa, sb)
		return applyCompare(op, cmp), nil
	}

	// Boolean comparison
	if a.TypeName == TypeBoolean && b.TypeName == TypeBoolean {
		return compareBooleans(op, a.Value.(bool), b.Value.(bool)), nil
	}

	// Numeric comparison — type-preserving
	if a.IsNumeric() && b.IsNumeric() {
		return compareNumeric(op, a, b)
	}

	// Date/time comparisons (same type only)
	if a.TypeName == b.TypeName {
		switch a.TypeName {
		case TypeDate:
			ta := a.Value.(time.Time)
			tb := b.Value.(time.Time)
			return compareDate(op, ta, tb), nil
		case TypeDateTime:
			ta := a.Value.(time.Time)
			tb := b.Value.(time.Time)
			return compareTime(op, ta, tb), nil
		case TypeTime:
			ta := a.Value.(time.Time)
			tb := b.Value.(time.Time)
			return compareTimeOfDay(op, ta, tb), nil
		case TypeYearMonthDuration, TypeDayTimeDuration:
			return compareDuration(op, a.DurationVal(), b.DurationVal())
		case TypeDuration:
			// xs:duration supports eq/ne only, not ordering
			if op != TokenEq && op != TokenNe {
				return false, &XPathError{Code: "XPTY0004", Message: "xs:duration values are not orderable"}
			}
			return compareDuration(op, a.DurationVal(), b.DurationVal())
		case TypeBase64Binary:
			return compareBinary(op, a.Value.([]byte), b.Value.([]byte))
		case TypeHexBinary:
			return compareBinary(op, a.Value.([]byte), b.Value.([]byte))
		case TypeQName:
			return compareQName(op, a.Value.(QNameValue), b.Value.(QNameValue))
		case TypeGDay, TypeGMonth, TypeGMonthDay, TypeGYear, TypeGYearMonth:
			ta, okA := parseGTypeToTime(a.TypeName, stringFromAtomic(a))
			tb, okB := parseGTypeToTime(b.TypeName, stringFromAtomic(b))
			if !okA || !okB {
				// Fallback to string comparison
				sa := normalizeGTZ(stringFromAtomic(a))
				sb := normalizeGTZ(stringFromAtomic(b))
				return applyCompare(op, strings.Compare(sa, sb)), nil
			}
			return compareTime(op, ta, tb), nil
		}
	}

	// Duration comparison
	famA := comparisonFamily(a.TypeName)
	famB := comparisonFamily(b.TypeName)
	if strings.HasPrefix(famA, "duration") && strings.HasPrefix(famB, "duration") {
		// xs:duration supports eq/ne only (not ordering)
		if a.TypeName == TypeDuration || b.TypeName == TypeDuration {
			if op != TokenEq && op != TokenNe {
				return false, &XPathError{Code: "XPTY0004", Message: "xs:duration values are not orderable"}
			}
		}
		// Cross-subtype (YMD vs DTD): eq/ne only, ordering is XPTY0004
		if a.TypeName != b.TypeName && a.TypeName != TypeDuration && b.TypeName != TypeDuration {
			if op != TokenEq && op != TokenNe {
				return false, &XPathError{Code: "XPTY0004", Message: "cannot order xs:yearMonthDuration and xs:dayTimeDuration"}
			}
		}
		return compareDuration(op, a.DurationVal(), b.DurationVal())
	}

	// Types are not comparable
	return false, &XPathError{
		Code:    "XPTY0004",
		Message: fmt.Sprintf("cannot compare %s with %s", a.TypeName, b.TypeName),
	}
}

// compareDate compares xs:date values by their UTC instants.
// Applies implicit timezone for values without explicit timezone.
func compareDate(op TokenType, a, b time.Time) bool {
	return compareTime(op, applyImplicitTZ(a), applyImplicitTZ(b))
}

// compareTimeOfDay compares xs:time values per XPath F&O 3.0 §10.4.4:
// https://www.w3.org/TR/xpath-functions-30/#func-time-equal
// Times are converted to xs:dateTime using the reference date 1972-12-31,
// then compared as UTC instants. This correctly handles date-wrap from timezone offsets.
// When a time has no explicit timezone (Location == time.UTC), the implicit
// timezone is applied per spec.
func compareTimeOfDay(op TokenType, a, b time.Time) bool {
	ra := timeToReferenceDateTime(applyImplicitTZ(a))
	rb := timeToReferenceDateTime(applyImplicitTZ(b))
	return compareTime(op, ra, rb)
}

// timeToReferenceDateTime converts an xs:time to an xs:dateTime using the
// XPath reference date 1972-12-31, preserving the timezone offset.
func timeToReferenceDateTime(t time.Time) time.Time {
	_, offset := t.Zone()
	loc := time.FixedZone("", offset)
	return time.Date(1972, 12, 31, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// applyImplicitTZ applies the system's implicit timezone to a time that has
// no explicit timezone (Location == time.UTC). Times with explicit timezones
// (Location is a FixedZone) are returned as-is.
func applyImplicitTZ(t time.Time) time.Time {
	if t.Location() != time.UTC {
		return t // has explicit timezone
	}
	// No timezone — apply implicit timezone (system local).
	// Use time.Local directly rather than time.Now().Zone() for consistency.
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
}

// normalizeGTZ normalizes timezone suffixes in g* type string values
// so that -00:00, +00:00, and Z are all treated as equivalent.
func normalizeGTZ(s string) string {
	if strings.HasSuffix(s, "+00:00") || strings.HasSuffix(s, "-00:00") {
		return s[:len(s)-6] + "Z"
	}
	return s
}

// parseGTypeToTime converts a g* type string value into a time.Time using
// reference dates from the XSD specification for comparison purposes.
// Reference dates: gDay → 1972-12-DD, gMonth → 1972-MM-01,
// gMonthDay → 1972-MM-DD, gYear → YYYY-01-01, gYearMonth → YYYY-MM-01.
func parseGTypeToTime(typeName, s string) (time.Time, bool) {
	// Extract timezone suffix
	loc, rest := parseGTimezone(s)

	switch typeName {
	case TypeGDay:
		// Format: ---DD
		if len(rest) < 5 {
			return time.Time{}, false
		}
		day := parseInt(rest[3:5])
		return time.Date(1972, 12, day, 0, 0, 0, 0, loc), true
	case TypeGMonth:
		// Format: --MM
		if len(rest) < 4 {
			return time.Time{}, false
		}
		month := parseInt(rest[2:4])
		return time.Date(1972, time.Month(month), 1, 0, 0, 0, 0, loc), true
	case TypeGMonthDay:
		// Format: --MM-DD
		if len(rest) < 7 {
			return time.Time{}, false
		}
		month := parseInt(rest[2:4])
		day := parseInt(rest[5:7])
		return time.Date(1972, time.Month(month), day, 0, 0, 0, 0, loc), true
	case TypeGYear:
		// Format: [-]YYYY
		year, endIdx := parseGYear(rest)
		_ = endIdx
		return time.Date(year, 1, 1, 0, 0, 0, 0, loc), true
	case TypeGYearMonth:
		// Format: [-]YYYY-MM
		year, endIdx := parseGYear(rest)
		if endIdx+3 > len(rest) {
			return time.Time{}, false
		}
		month := parseInt(rest[endIdx+1 : endIdx+3])
		return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc), true
	}
	return time.Time{}, false
}

// parseGTimezone extracts the timezone from a g* string value, returning the
// location and the string with timezone removed.
func parseGTimezone(s string) (*time.Location, string) {
	if strings.HasSuffix(s, "Z") {
		return time.UTC, s[:len(s)-1]
	}
	if len(s) >= 6 {
		tz := s[len(s)-6:]
		if (tz[0] == '+' || tz[0] == '-') && tz[3] == ':' {
			hours := parseInt(tz[1:3])
			mins := parseInt(tz[4:6])
			offset := hours*3600 + mins*60
			if tz[0] == '-' {
				offset = -offset
			}
			return time.FixedZone("", offset), s[:len(s)-6]
		}
	}
	// No timezone — use noTZLocation sentinel
	return noTZLocation, s
}

// parseInt parses a string of digits as an int (no error handling — input is pre-validated by regex).
func parseInt(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// parseGYear parses the year from a g* value string, returning the year and
// the index after the year digits.
func parseGYear(s string) (int, int) {
	neg := false
	start := 0
	if len(s) > 0 && s[0] == '-' {
		neg = true
		start = 1
	}
	end := start
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	year := parseInt(s[start:end])
	if neg {
		year = -year
	}
	return year, end
}

func compareTime(op TokenType, a, b time.Time) bool {
	switch op {
	case TokenEq:
		return a.Equal(b)
	case TokenNe:
		return !a.Equal(b)
	case TokenLt:
		return a.Before(b)
	case TokenLe:
		return a.Before(b) || a.Equal(b)
	case TokenGt:
		return a.After(b)
	case TokenGe:
		return a.After(b) || a.Equal(b)
	}
	return false
}

func compareDuration(op TokenType, a, b Duration) (bool, error) {
	// Normalize: convert negative to signed months/seconds
	aMonths, aSecs := a.Months, a.Seconds
	if a.Negative {
		aMonths, aSecs = -aMonths, -aSecs
	}
	bMonths, bSecs := b.Months, b.Seconds
	if b.Negative {
		bMonths, bSecs = -bMonths, -bSecs
	}

	// Use epsilon comparison for seconds to handle floating-point arithmetic drift
	const secEps = 1e-9
	eq := aMonths == bMonths && math.Abs(aSecs-bSecs) < secEps

	switch op {
	case TokenEq:
		return eq, nil
	case TokenNe:
		return !eq, nil
	case TokenLt, TokenLe, TokenGt, TokenGe:
		// Ordering is only defined for yearMonthDuration and dayTimeDuration (not mixed)
		if aMonths != 0 && aSecs != 0 {
			return false, &XPathError{Code: "XPTY0004", Message: "xs:duration values are not orderable"}
		}
		if bMonths != 0 && bSecs != 0 {
			return false, &XPathError{Code: "XPTY0004", Message: "xs:duration values are not orderable"}
		}
		// Compare months-only or seconds-only
		if aMonths != 0 || bMonths != 0 {
			cmp := aMonths - bMonths
			return applyCompareInt(op, cmp), nil
		}
		return compareFloats(op, aSecs, bSecs), nil
	}
	return false, nil
}

func compareBinary(op TokenType, a, b []byte) (bool, error) {
	cmp := bytes.Compare(a, b)
	return applyCompare(op, cmp), nil
}

func compareQName(op TokenType, a, b QNameValue) (bool, error) {
	switch op {
	case TokenEq:
		return a.URI == b.URI && a.Local == b.Local, nil
	case TokenNe:
		return a.URI != b.URI || a.Local != b.Local, nil
	default:
		return false, &XPathError{Code: "XPTY0004", Message: "QName values only support eq/ne"}
	}
}

func applyCompareInt(op TokenType, cmp int) bool {
	switch op {
	case TokenLt:
		return cmp < 0
	case TokenLe:
		return cmp <= 0
	case TokenGt:
		return cmp > 0
	case TokenGe:
		return cmp >= 0
	}
	return false
}

// normalizeCompareOp maps general comparison operators to value comparison operators.
func normalizeCompareOp(op TokenType) TokenType {
	switch op {
	case TokenEquals:
		return TokenEq
	case TokenNotEquals:
		return TokenNe
	case TokenLess:
		return TokenLt
	case TokenLessEq:
		return TokenLe
	case TokenGreater:
		return TokenGt
	case TokenGreaterEq:
		return TokenGe
	}
	return op
}

func applyCompare(op TokenType, cmp int) bool {
	switch op {
	case TokenEq:
		return cmp == 0
	case TokenNe:
		return cmp != 0
	case TokenLt:
		return cmp < 0
	case TokenLe:
		return cmp <= 0
	case TokenGt:
		return cmp > 0
	case TokenGe:
		return cmp >= 0
	}
	return false
}

// compareNumeric performs type-preserving numeric comparison.
// Both integer → big.Int.Cmp; either decimal → big.Rat.Cmp; otherwise → float64.
func compareNumeric(op TokenType, a, b AtomicValue) (bool, error) {
	// Both integer → big.Int comparison
	if isIntegerDerived(a.TypeName) && isIntegerDerived(b.TypeName) {
		cmp := a.BigInt().Cmp(b.BigInt())
		return applyCompare(op, cmp), nil
	}
	// Either decimal (and other is integer or decimal) → big.Rat comparison
	aDecOrInt := a.TypeName == TypeDecimal || isIntegerDerived(a.TypeName)
	bDecOrInt := b.TypeName == TypeDecimal || isIntegerDerived(b.TypeName)
	if aDecOrInt && bDecOrInt {
		ar := toRatForCompare(a)
		br := toRatForCompare(b)
		cmp := ar.Cmp(br)
		return applyCompare(op, cmp), nil
	}
	// When one operand is xs:float and the other is decimal/integer,
	// promote to xs:float (float32) per XPath type promotion rules.
	// This avoids float32→float64 precision artifacts (e.g., float32(1.2) != float64(1.2)).
	aIsFloat := a.TypeName == TypeFloat
	bIsFloat := b.TypeName == TypeFloat
	if (aIsFloat || bIsFloat) && a.TypeName != TypeDouble && b.TypeName != TypeDouble {
		return compareFloats(op, float64(float32(a.ToFloat64())), float64(float32(b.ToFloat64()))), nil
	}
	// Otherwise → float64 (handles double, float, NaN, ±Inf)
	return compareFloats(op, a.ToFloat64(), b.ToFloat64()), nil
}

// toRatForCompare converts integer or decimal AtomicValue to *big.Rat for comparison.
func toRatForCompare(a AtomicValue) *big.Rat {
	if isIntegerDerived(a.TypeName) {
		return new(big.Rat).SetInt(a.BigInt())
	}
	return a.BigRat()
}

func compareFloats(op TokenType, a, b float64) bool {
	switch op {
	case TokenEq:
		return a == b
	case TokenNe:
		// NaN != NaN is true per IEEE 754
		if math.IsNaN(a) || math.IsNaN(b) {
			return true
		}
		return a != b
	case TokenLt:
		return a < b
	case TokenLe:
		return a <= b
	case TokenGt:
		return a > b
	case TokenGe:
		return a >= b
	}
	return false
}

func compareBooleans(op TokenType, a, b bool) bool {
	ai, bi := 0, 0
	if a {
		ai = 1
	}
	if b {
		bi = 1
	}
	switch op {
	case TokenEq:
		return ai == bi
	case TokenNe:
		return ai != bi
	case TokenLt:
		return ai < bi
	case TokenLe:
		return ai <= bi
	case TokenGt:
		return ai > bi
	case TokenGe:
		return ai >= bi
	}
	return false
}
