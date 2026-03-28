package xpath3

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
)

// isIntegerDerived returns true if the type is xs:integer or one of its derived types.
func isIntegerDerived(typeName string) bool {
	switch typeName {
	case TypeInteger,
		TypeLong, TypeInt, TypeShort, TypeByte,
		TypeUnsignedLong, TypeUnsignedInt, TypeUnsignedShort, TypeUnsignedByte,
		TypeNonNegativeInteger, TypeNonPositiveInteger,
		TypePositiveInteger, TypeNegativeInteger:
		return true
	}
	return false
}

// isFloatOrDouble returns true if the type is xs:float or xs:double.
func isFloatOrDouble(typeName string) bool {
	return typeName == TypeFloat || typeName == TypeDouble
}

// isAbstractCastTarget returns true if the type cannot be used as a cast/castable target.
func isAbstractCastTarget(typeName string) bool {
	return typeName == TypeAnyAtomicType ||
		typeName == "xs:anySimpleType" ||
		typeName == "xs:anyType" ||
		typeName == TypeNOTATION
}

// CastAtomic casts an AtomicValue to the target type.
func CastAtomic(v AtomicValue, targetType string) (AtomicValue, error) {
	if isAbstractCastTarget(targetType) {
		return AtomicValue{}, &XPathError{
			Code:    errCodeXPST0080,
			Message: "cannot cast to abstract type " + targetType,
		}
	}

	if v.TypeName == targetType {
		return v, nil
	}

	// Derived integer target types — cast to integer first, then validate range
	if isIntegerDerived(targetType) && targetType != TypeInteger {
		iv, err := CastAtomic(v, TypeInteger)
		if err != nil {
			return AtomicValue{}, err
		}
		n := iv.BigInt()
		minVal, maxVal := integerTypeRange(targetType)
		if minVal != nil && n.Cmp(minVal) < 0 {
			return AtomicValue{}, &XPathError{
				Code:    errCodeFORG0001,
				Message: fmt.Sprintf("value %s out of range for %s", n.String(), targetType),
			}
		}
		if maxVal != nil && n.Cmp(maxVal) > 0 {
			return AtomicValue{}, &XPathError{
				Code:    errCodeFORG0001,
				Message: fmt.Sprintf("value %s out of range for %s", n.String(), targetType),
			}
		}
		return AtomicValue{TypeName: targetType, Value: n}, nil
	}

	// Normalize derived integer types to xs:integer for casting purposes
	if isIntegerDerived(v.TypeName) && v.TypeName != TypeInteger {
		v = AtomicValue{TypeName: TypeInteger, Value: v.Value}
	}

	// xs:untypedAtomic → any type goes through string-based casting
	if v.TypeName == TypeUntypedAtomic {
		return CastFromString(v.StringVal(), targetType)
	}

	switch targetType {
	case TypeString:
		return castToString(v)
	case TypeDouble:
		return castToDouble(v)
	case TypeFloat:
		return castToFloat(v)
	case TypeInteger:
		return castToInteger(v)
	case TypeDecimal:
		return castToDecimal(v)
	case TypeBoolean:
		return castToBoolean(v)
	case TypeUntypedAtomic:
		s, err := atomicToString(v)
		if err != nil {
			return AtomicValue{}, err
		}
		return AtomicValue{TypeName: TypeUntypedAtomic, Value: s}, nil
	case TypeAnyURI:
		if v.TypeName == TypeString {
			return AtomicValue{TypeName: TypeAnyURI, Value: collapseWhitespace(v.StringVal())}, nil
		}
	case TypeBase64Binary:
		return castToBase64Binary(v)
	case TypeHexBinary:
		return castToHexBinary(v)
	case TypeDate:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeDate)
		}
		if v.TypeName == TypeDateTime {
			t := v.TimeVal()
			return AtomicValue{TypeName: TypeDate, Value: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())}, nil
		}
	case TypeDateTime:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeDateTime)
		}
		if v.TypeName == TypeDate {
			t := v.TimeVal()
			return AtomicValue{TypeName: TypeDateTime, Value: t}, nil
		}
	case TypeDateTimeStamp:
		// xs:dateTimeStamp is xs:dateTime with a required timezone
		dt, err := CastAtomic(v, TypeDateTime)
		if err != nil {
			return AtomicValue{}, err
		}
		if !HasTimezone(dt.TimeVal()) {
			return AtomicValue{}, &XPathError{Code: errCodeFORG0001, Message: "xs:dateTimeStamp requires a timezone"}
		}
		dt.TypeName = TypeDateTimeStamp
		return dt, nil
	case TypeTime:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeTime)
		}
		if v.TypeName == TypeDateTime {
			t := v.TimeVal()
			loc := t.Location()
			if !HasTimezone(t) {
				loc = noTZLocation
			}
			return AtomicValue{TypeName: TypeTime, Value: time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)}, nil
		}
	case TypeDayTimeDuration:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeDayTimeDuration)
		}
		if v.TypeName == TypeDuration || v.TypeName == TypeYearMonthDuration {
			d := v.DurationVal()
			return AtomicValue{TypeName: TypeDayTimeDuration, Value: Duration{Seconds: d.Seconds, FracSec: d.FracSec, Negative: d.Negative}}, nil
		}
	case TypeYearMonthDuration:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeYearMonthDuration)
		}
		if v.TypeName == TypeDuration || v.TypeName == TypeDayTimeDuration {
			d := v.DurationVal()
			return AtomicValue{TypeName: TypeYearMonthDuration, Value: Duration{Months: d.Months, Negative: d.Negative}}, nil
		}
	case TypeDuration:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeDuration)
		}
		if v.TypeName == TypeDayTimeDuration || v.TypeName == TypeYearMonthDuration {
			return AtomicValue{TypeName: TypeDuration, Value: v.DurationVal()}, nil
		}
	case TypeGDay:
		return castToGType(v, targetType, func(t time.Time) string {
			return fmt.Sprintf("---%02d%s", t.Day(), formatXSDTimezone(t))
		})
	case TypeGMonth:
		return castToGType(v, targetType, func(t time.Time) string {
			return fmt.Sprintf("--%02d%s", t.Month(), formatXSDTimezone(t))
		})
	case TypeGMonthDay:
		return castToGType(v, targetType, func(t time.Time) string {
			return fmt.Sprintf("--%02d-%02d%s", t.Month(), t.Day(), formatXSDTimezone(t))
		})
	case TypeGYear:
		return castToGType(v, targetType, func(t time.Time) string {
			return fmt.Sprintf("%s%s", formatXSDYear(t.Year()), formatXSDTimezone(t))
		})
	case TypeGYearMonth:
		return castToGType(v, targetType, func(t time.Time) string {
			return fmt.Sprintf("%s-%02d%s", formatXSDYear(t.Year()), t.Month(), formatXSDTimezone(t))
		})
	case TypeNormalizedString:
		s, err := atomicToString(v)
		if err != nil {
			return AtomicValue{}, err
		}
		return AtomicValue{TypeName: targetType, Value: normalizeWhitespace(s)}, nil
	case TypeToken, TypeLanguage, TypeName, TypeNCName,
		TypeNMTOKEN, TypeNMTOKENS, TypeENTITY, TypeID, TypeIDREF, TypeIDREFS:
		s, err := atomicToString(v)
		if err != nil {
			return AtomicValue{}, err
		}
		s = collapseWhitespace(s)
		if err := validateStringDerivedType(s, targetType); err != nil {
			return AtomicValue{}, err
		}
		return AtomicValue{TypeName: targetType, Value: s}, nil
	case TypeQName:
		// NOTATION → QName: NOTATION values are QName-like (same value model).
		// Schema-derived NOTATION types also hold QNameValue values.
		if v.TypeName == TypeNOTATION {
			return AtomicValue{TypeName: TypeQName, Value: v.Value}, nil
		}
		if _, ok := v.Value.(QNameValue); ok {
			return AtomicValue{TypeName: TypeQName, Value: v.Value}, nil
		}
		// QName → QName is handled by identity check above.
		// String/untypedAtomic → QName requires namespace context and is
		// handled by evalCastExpr or the xs:QName constructor function.
		// If we reach here without context, report an appropriate error.
		return AtomicValue{}, &XPathError{
			Code:    errCodeXPTY0004,
			Message: fmt.Sprintf("cannot cast %s to %s (requires namespace context)", v.TypeName, TypeQName),
		}
	case TypeNOTATION:
		if v.TypeName == TypeString || v.TypeName == TypeUntypedAtomic {
			return CastFromString(v.StringVal(), TypeNOTATION)
		}
		// QName → NOTATION
		if v.TypeName == TypeQName {
			return AtomicValue{TypeName: TypeNOTATION, Value: v.Value}, nil
		}
	}

	return AtomicValue{}, &XPathError{
		Code:    errCodeXPTY0004,
		Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, targetType),
	}
}

// CastFromString casts a string value to the target atomic type.
func CastFromString(s string, targetType string) (AtomicValue, error) {
	switch targetType {
	case TypeString:
		return AtomicValue{TypeName: TypeString, Value: s}, nil
	case TypeUntypedAtomic:
		return AtomicValue{TypeName: TypeUntypedAtomic, Value: s}, nil
	case "xs:anyType", "xs:untyped":
		return AtomicValue{TypeName: TypeUntypedAtomic, Value: s}, nil
	}
	// Whitespace trimming applies to all types except string/untypedAtomic
	s = strings.TrimSpace(s)
	switch targetType {
	case TypeInteger:
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeInteger, Value: n}, nil
	case TypeDecimal:
		if !isValidDecimalString(s) {
			return AtomicValue{}, castError(s, targetType)
		}
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDecimal, Value: r}, nil
	case TypeDouble:
		f, err := parseXPathDouble(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		// parseXPathDouble already rejects invalid infinity forms (e.g. "+INF")
		// and overflow to infinity from numeric strings
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
	case TypeFloat:
		return castStringToFloat(s)
	case TypeBoolean:
		switch s {
		case "true", "1":
			return AtomicValue{TypeName: TypeBoolean, Value: true}, nil
		case "false", "0":
			return AtomicValue{TypeName: TypeBoolean, Value: false}, nil
		default:
			return AtomicValue{}, castError(s, targetType)
		}
	case TypeAnyURI:
		return AtomicValue{TypeName: TypeAnyURI, Value: collapseWhitespace(s)}, nil
	case TypeDate:
		t, err := parseXSDDate(s)
		if err != nil {
			var xe *XPathError
			if errors.As(err, &xe) {
				return AtomicValue{}, xe
			}
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDate, Value: t}, nil
	case TypeDateTime:
		t, err := parseXSDDateTime(s)
		if err != nil {
			var xe *XPathError
			if errors.As(err, &xe) {
				return AtomicValue{}, xe
			}
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDateTime, Value: t}, nil
	case TypeTime:
		t, err := parseXSDTime(s)
		if err != nil {
			var xe *XPathError
			if errors.As(err, &xe) {
				return AtomicValue{}, xe
			}
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeTime, Value: t}, nil
	case TypeDuration:
		d, err := parseXSDDuration(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDuration, Value: d}, nil
	case TypeDayTimeDuration:
		d, err := parseXSDDuration(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		if d.Months != 0 {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDayTimeDuration, Value: Duration{Seconds: d.Seconds, FracSec: d.FracSec, Negative: d.Negative}}, nil
	case TypeYearMonthDuration:
		d, err := parseXSDDuration(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		if d.Seconds != 0 {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeYearMonthDuration, Value: Duration{Months: d.Months, Negative: d.Negative}}, nil
	case TypeBase64Binary:
		b, err := decodeXSDBase64(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeBase64Binary, Value: b}, nil
	case TypeHexBinary:
		b, err := hex.DecodeString(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeHexBinary, Value: b}, nil
	case TypeGDay:
		if !reGDay.MatchString(s) || !validateGregorianValue(TypeGDay, s) {
			return AtomicValue{}, castError(s, targetType)
		}
		s = normalizeZeroTimezoneLexical(s)
		return AtomicValue{TypeName: TypeGDay, Value: s}, nil
	case TypeGMonth:
		if !reGMonth.MatchString(s) || !validateGregorianValue(TypeGMonth, s) {
			return AtomicValue{}, castError(s, targetType)
		}
		s = normalizeZeroTimezoneLexical(s)
		return AtomicValue{TypeName: TypeGMonth, Value: s}, nil
	case TypeGMonthDay:
		if !reGMonthDay.MatchString(s) || !validateGregorianValue(TypeGMonthDay, s) {
			return AtomicValue{}, castError(s, targetType)
		}
		s = normalizeZeroTimezoneLexical(s)
		return AtomicValue{TypeName: TypeGMonthDay, Value: s}, nil
	case TypeGYear:
		if !reGYear.MatchString(s) || !validateGregorianValue(TypeGYear, s) {
			return AtomicValue{}, castError(s, targetType)
		}
		s = normalizeNegZeroYear(s)
		s = normalizeZeroTimezoneLexical(s)
		return AtomicValue{TypeName: TypeGYear, Value: s}, nil
	case TypeGYearMonth:
		if !reGYearMonth.MatchString(s) || !validateGregorianValue(TypeGYearMonth, s) {
			return AtomicValue{}, castError(s, targetType)
		}
		s = normalizeNegZeroYear(s)
		s = normalizeZeroTimezoneLexical(s)
		return AtomicValue{TypeName: TypeGYearMonth, Value: s}, nil
	case TypeNormalizedString:
		// xs:normalizedString: replace #x9, #xA, #xD with #x20
		s = normalizeWhitespace(s)
		return AtomicValue{TypeName: targetType, Value: s}, nil
	case TypeToken, TypeLanguage, TypeName, TypeNCName,
		TypeNMTOKEN, TypeNMTOKENS, TypeENTITY, TypeID, TypeIDREF, TypeIDREFS:
		// xs:token and derived: normalize + collapse whitespace
		s = collapseWhitespace(s)
		if err := validateStringDerivedType(s, targetType); err != nil {
			return AtomicValue{}, err
		}
		return AtomicValue{TypeName: targetType, Value: s}, nil
	case TypeNOTATION:
		// xs:NOTATION is QName-like; store the lexical value as-is.
		return AtomicValue{TypeName: TypeNOTATION, Value: s}, nil
	}
	// XSD integer subtypes: parse as integer, validate range, preserve specific type
	if isIntegerSubtype(targetType) {
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return AtomicValue{}, castError(s, targetType)
		}
		if err := validateIntegerRange(n, targetType); err != nil {
			return AtomicValue{}, err
		}
		return AtomicValue{TypeName: targetType, Value: n}, nil
	}
	return AtomicValue{}, &XPathError{
		Code:    errCodeXPTY0004,
		Message: "cannot cast string to " + targetType,
	}
}

// isIntegerSubtype returns true for XSD integer-derived types like xs:int, xs:short, etc.
func isIntegerSubtype(t string) bool {
	switch t {
	case "xs:long", "xs:int", "xs:short", "xs:byte",
		"xs:unsignedLong", "xs:unsignedInt", "xs:unsignedShort", "xs:unsignedByte",
		"xs:positiveInteger", "xs:nonPositiveInteger",
		"xs:negativeInteger", "xs:nonNegativeInteger":
		return true
	}
	return false
}

// validateIntegerRange checks that an integer value is within the range of the target type.
func validateIntegerRange(n *big.Int, targetType string) error {
	var minVal, maxVal *big.Int
	switch targetType {
	case "xs:long":
		minVal, maxVal = big.NewInt(math.MinInt64), big.NewInt(math.MaxInt64)
	case "xs:int":
		minVal, maxVal = big.NewInt(math.MinInt32), big.NewInt(math.MaxInt32)
	case "xs:short":
		minVal, maxVal = big.NewInt(math.MinInt16), big.NewInt(math.MaxInt16)
	case "xs:byte":
		minVal, maxVal = big.NewInt(math.MinInt8), big.NewInt(math.MaxInt8)
	case "xs:unsignedLong":
		minVal = big.NewInt(0)
		maxVal = new(big.Int).SetUint64(math.MaxUint64)
	case "xs:unsignedInt":
		minVal, maxVal = big.NewInt(0), big.NewInt(math.MaxUint32)
	case "xs:unsignedShort":
		minVal, maxVal = big.NewInt(0), big.NewInt(math.MaxUint16)
	case "xs:unsignedByte":
		minVal, maxVal = big.NewInt(0), big.NewInt(math.MaxUint8)
	case "xs:positiveInteger":
		minVal = big.NewInt(1)
	case "xs:nonPositiveInteger":
		maxVal = big.NewInt(0)
	case "xs:negativeInteger":
		maxVal = big.NewInt(-1)
	case "xs:nonNegativeInteger":
		minVal = big.NewInt(0)
	default:
		return nil
	}
	if minVal != nil && n.Cmp(minVal) < 0 {
		return castError(n.String(), targetType)
	}
	if maxVal != nil && n.Cmp(maxVal) > 0 {
		return castError(n.String(), targetType)
	}
	return nil
}

// normalizeNegZeroYear strips leading '-' from "-0000" years (e.g. "-0000-05" → "0000-05").
// Per XSD, negative zero is identical to positive zero for year values.
func normalizeNegZeroYear(s string) string {
	if len(s) < 5 || s[0] != '-' {
		return s
	}
	// Check if all year digits are zero: "-0000..." or "-00000..."
	i := 1
	for i < len(s) && s[i] == '0' {
		i++
	}
	// i now points past the zeros; if it hits a non-digit boundary, year is all zeros
	if i < len(s) && s[i] >= '0' && s[i] <= '9' {
		return s // non-zero year digit found
	}
	return s[1:] // strip the '-'
}

func normalizeZeroTimezoneLexical(s string) string {
	switch {
	case strings.HasSuffix(s, "+00:00"), strings.HasSuffix(s, "-00:00"):
		return s[:len(s)-6] + "Z"
	default:
		return s
	}
}

// validateStringDerivedType checks pattern constraints for string-derived types.
func validateStringDerivedType(s, targetType string) error {
	switch targetType {
	case TypeName:
		if !reName.MatchString(s) {
			return castError(s, targetType)
		}
	case TypeNCName, TypeENTITY, TypeID, TypeIDREF:
		if !reNCName.MatchString(s) {
			return castError(s, targetType)
		}
	case TypeNMTOKEN:
		if !reNMTOKEN.MatchString(s) {
			return castError(s, targetType)
		}
	case TypeNMTOKENS:
		// xs:NMTOKENS is a space-separated list of NMTOKEN values; must have at least one
		if s == "" {
			return castError(s, targetType)
		}
		for _, tok := range strings.Fields(s) {
			if !reNMTOKEN.MatchString(tok) {
				return castError(s, targetType)
			}
		}
	case TypeIDREFS:
		// xs:IDREFS is a space-separated list of IDREF (NCName) values; must have at least one
		if s == "" {
			return castError(s, targetType)
		}
		for _, tok := range strings.Fields(s) {
			if !reNCName.MatchString(tok) {
				return castError(s, targetType)
			}
		}
	case TypeLanguage:
		if !reLang.MatchString(s) {
			return castError(s, targetType)
		}
	}
	return nil
}

// integerTypeRange returns the min/max bounds for a derived integer type.
func integerTypeRange(typeName string) (minVal *big.Int, maxVal *big.Int) {
	switch typeName {
	case TypeLong:
		return big.NewInt(math.MinInt64), big.NewInt(math.MaxInt64)
	case TypeInt:
		return big.NewInt(-2147483648), big.NewInt(2147483647)
	case TypeShort:
		return big.NewInt(-32768), big.NewInt(32767)
	case TypeByte:
		return big.NewInt(-128), big.NewInt(127)
	case TypeUnsignedLong:
		return big.NewInt(0), new(big.Int).SetUint64(math.MaxUint64)
	case TypeUnsignedInt:
		return big.NewInt(0), big.NewInt(4294967295)
	case TypeUnsignedShort:
		return big.NewInt(0), big.NewInt(65535)
	case TypeUnsignedByte:
		return big.NewInt(0), big.NewInt(255)
	case TypeNonNegativeInteger:
		return big.NewInt(0), nil
	case TypeNonPositiveInteger:
		return nil, big.NewInt(0)
	case TypePositiveInteger:
		return big.NewInt(1), nil
	case TypeNegativeInteger:
		return nil, big.NewInt(-1)
	}
	return nil, nil
}

func castError(value string, targetType string) *XPathError {
	return &XPathError{
		Code:    errCodeFORG0001,
		Message: fmt.Sprintf("cannot cast %q to %s", value, targetType),
	}
}
