package xpath3

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
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

// CastAtomic casts an AtomicValue to the target type.
// Returns an error if the cast is not supported or the value is invalid.
func CastAtomic(v AtomicValue, targetType string) (AtomicValue, error) {
	if v.TypeName == targetType {
		return v, nil
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
			return AtomicValue{TypeName: TypeAnyURI, Value: v.StringVal()}, nil
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
	case TypeTime:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeTime)
		}
		if v.TypeName == TypeDateTime {
			t := v.TimeVal()
			return AtomicValue{TypeName: TypeTime, Value: time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())}, nil
		}
	case TypeDayTimeDuration:
		if v.TypeName == TypeString {
			return CastFromString(v.StringVal(), TypeDayTimeDuration)
		}
		if v.TypeName == TypeDuration || v.TypeName == TypeYearMonthDuration {
			d := v.DurationVal()
			return AtomicValue{TypeName: TypeDayTimeDuration, Value: Duration{Seconds: d.Seconds, Negative: d.Negative}}, nil
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
	}

	return AtomicValue{}, &XPathError{
		Code:    "XPTY0004",
		Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, targetType),
	}
}

// CastFromString casts a string value to the target atomic type.
func CastFromString(s string, targetType string) (AtomicValue, error) {
	s = strings.TrimSpace(s)
	switch targetType {
	case TypeString:
		return AtomicValue{TypeName: TypeString, Value: s}, nil
	case TypeUntypedAtomic:
		return AtomicValue{TypeName: TypeUntypedAtomic, Value: s}, nil
	case TypeInteger:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeInteger, Value: n}, nil
	case TypeDecimal:
		// Validate it looks like a decimal
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDecimal, Value: s}, nil
	case TypeDouble:
		f, err := parseXPathDouble(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDouble, Value: f}, nil
	case TypeFloat:
		f, err := parseXPathDouble(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeFloat, Value: f}, nil
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
		return AtomicValue{TypeName: TypeAnyURI, Value: s}, nil
	case TypeDate:
		t, err := parseXSDDate(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDate, Value: t}, nil
	case TypeDateTime:
		t, err := parseXSDDateTime(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeDateTime, Value: t}, nil
	case TypeTime:
		t, err := parseXSDTime(s)
		if err != nil {
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
		return AtomicValue{TypeName: TypeDayTimeDuration, Value: Duration{Seconds: d.Seconds, Negative: d.Negative}}, nil
	case TypeYearMonthDuration:
		d, err := parseXSDDuration(s)
		if err != nil {
			return AtomicValue{}, castError(s, targetType)
		}
		return AtomicValue{TypeName: TypeYearMonthDuration, Value: Duration{Months: d.Months, Negative: d.Negative}}, nil
	case TypeBase64Binary:
		b, err := base64.StdEncoding.DecodeString(s)
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
	}
	return AtomicValue{}, &XPathError{
		Code:    "XPTY0004",
		Message: fmt.Sprintf("cannot cast string to %s", targetType),
	}
}

func castError(value string, targetType string) *XPathError {
	return &XPathError{
		Code:    "FORG0001",
		Message: fmt.Sprintf("cannot cast %q to %s", value, targetType),
	}
}

func castToString(v AtomicValue) (AtomicValue, error) {
	s, err := atomicToString(v)
	if err != nil {
		return AtomicValue{}, err
	}
	return AtomicValue{TypeName: TypeString, Value: s}, nil
}

// atomicToString returns the canonical string representation of an atomic value.
func atomicToString(v AtomicValue) (string, error) {
	switch v.TypeName {
	case TypeString, TypeAnyURI, TypeUntypedAtomic,
		TypeNormalizedString, TypeToken, TypeLanguage, TypeName, TypeNCName,
		TypeNMTOKEN, TypeNMTOKENS, TypeENTITY, TypeID, TypeIDREF, TypeIDREFS,
		TypeGDay, TypeGMonth, TypeGMonthDay, TypeGYear, TypeGYearMonth:
		return v.Value.(string), nil
	case TypeInteger,
		TypeLong, TypeInt, TypeShort, TypeByte,
		TypeUnsignedLong, TypeUnsignedInt, TypeUnsignedShort, TypeUnsignedByte,
		TypeNonNegativeInteger, TypeNonPositiveInteger,
		TypePositiveInteger, TypeNegativeInteger:
		return strconv.FormatInt(v.Value.(int64), 10), nil
	case TypeDecimal:
		return v.Value.(string), nil
	case TypeDouble, TypeFloat:
		f := v.Value.(float64)
		if math.IsNaN(f) {
			return "NaN", nil
		}
		if math.IsInf(f, 1) {
			return "INF", nil
		}
		if math.IsInf(f, -1) {
			return "-INF", nil
		}
		if f == 0 {
			if math.Signbit(f) {
				return "-0", nil
			}
			return "0", nil
		}
		return strconv.FormatFloat(f, 'G', -1, 64), nil
	case TypeBoolean:
		if v.Value.(bool) {
			return "true", nil
		}
		return "false", nil
	case TypeDate:
		return v.Value.(time.Time).Format("2006-01-02"), nil
	case TypeDateTime:
		return v.Value.(time.Time).Format("2006-01-02T15:04:05"), nil
	case TypeTime:
		return v.Value.(time.Time).Format("15:04:05"), nil
	case TypeDuration, TypeDayTimeDuration, TypeYearMonthDuration:
		return formatDuration(v.Value.(Duration)), nil
	case TypeBase64Binary:
		return base64.StdEncoding.EncodeToString(v.Value.([]byte)), nil
	case TypeHexBinary:
		return strings.ToUpper(hex.EncodeToString(v.Value.([]byte))), nil
	case TypeQName:
		q := v.Value.(QNameValue)
		if q.Prefix != "" {
			return q.Prefix + ":" + q.Local, nil
		}
		return q.Local, nil
	}
	return fmt.Sprintf("%v", v.Value), nil
}

func castToDouble(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeInteger:
		return AtomicValue{TypeName: TypeDouble, Value: float64(v.IntegerVal())}, nil
	case TypeDecimal:
		f, _ := strconv.ParseFloat(v.StringVal(), 64)
		return AtomicValue{TypeName: TypeDouble, Value: f}, nil
	case TypeFloat:
		return AtomicValue{TypeName: TypeDouble, Value: v.DoubleVal()}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeDouble, Value: float64(1)}, nil
		}
		return AtomicValue{TypeName: TypeDouble, Value: float64(0)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeDouble)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:double", v.TypeName)}
}

func castToFloat(v AtomicValue) (AtomicValue, error) {
	result, err := castToDouble(v)
	if err != nil {
		return AtomicValue{}, err
	}
	result.TypeName = TypeFloat
	return result, nil
}

func castToInteger(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return AtomicValue{}, &XPathError{Code: "FOCA0002", Message: "cannot cast NaN/INF to xs:integer"}
		}
		return AtomicValue{TypeName: TypeInteger, Value: int64(f)}, nil
	case TypeDecimal:
		f, _ := strconv.ParseFloat(v.StringVal(), 64)
		return AtomicValue{TypeName: TypeInteger, Value: int64(f)}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeInteger, Value: int64(1)}, nil
		}
		return AtomicValue{TypeName: TypeInteger, Value: int64(0)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeInteger)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:integer", v.TypeName)}
}

func castToDecimal(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeInteger:
		return AtomicValue{TypeName: TypeDecimal, Value: strconv.FormatInt(v.IntegerVal(), 10)}, nil
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return AtomicValue{}, &XPathError{Code: "FOCA0002", Message: "cannot cast NaN/INF to xs:decimal"}
		}
		return AtomicValue{TypeName: TypeDecimal, Value: strconv.FormatFloat(f, 'f', -1, 64)}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeDecimal, Value: "1"}, nil
		}
		return AtomicValue{TypeName: TypeDecimal, Value: "0"}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeDecimal)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:decimal", v.TypeName)}
}

func castToBoolean(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeInteger:
		return AtomicValue{TypeName: TypeBoolean, Value: v.IntegerVal() != 0}, nil
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		return AtomicValue{TypeName: TypeBoolean, Value: f != 0 && !math.IsNaN(f)}, nil
	case TypeDecimal:
		s := v.StringVal()
		return AtomicValue{TypeName: TypeBoolean, Value: s != "0" && s != "0.0"}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeBoolean)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:boolean", v.TypeName)}
}

func castToBase64Binary(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeHexBinary:
		return AtomicValue{TypeName: TypeBase64Binary, Value: v.BytesVal()}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeBase64Binary)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:base64Binary", v.TypeName)}
}

func castToHexBinary(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeBase64Binary:
		return AtomicValue{TypeName: TypeHexBinary, Value: v.BytesVal()}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeHexBinary)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:hexBinary", v.TypeName)}
}

// --- XSD Parsing Helpers ---

func parseXPathDouble(s string) (float64, error) {
	switch s {
	case "INF":
		return math.Inf(1), nil
	case "-INF":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}
	return strconv.ParseFloat(s, 64)
}

func parseXSDDate(s string) (time.Time, error) {
	// Try with timezone first, then without
	for _, layout := range []string{
		"2006-01-02Z07:00",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:date: %q", s)
}

func parseXSDDateTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q", s)
}

func parseXSDTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"15:04:05.999999999Z07:00",
		"15:04:05Z07:00",
		"15:04:05.999999999",
		"15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:time: %q", s)
}

// parseXSDDuration parses an XSD duration string like "P1Y2M3DT4H5M6S".
func parseXSDDuration(s string) (Duration, error) {
	if len(s) == 0 {
		return Duration{}, fmt.Errorf("empty duration")
	}

	d := Duration{}
	i := 0

	if s[i] == '-' {
		d.Negative = true
		i++
	}

	if i >= len(s) || s[i] != 'P' {
		return Duration{}, fmt.Errorf("invalid duration: %q", s)
	}
	i++

	inTime := false
	for i < len(s) {
		if s[i] == 'T' {
			inTime = true
			i++
			continue
		}

		// Parse number (may be decimal)
		numStart := i
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
			i++
		}
		if i == numStart || i >= len(s) {
			return Duration{}, fmt.Errorf("invalid duration: %q", s)
		}

		numStr := s[numStart:i]
		designator := s[i]
		i++

		if !inTime {
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			switch designator {
			case 'Y':
				d.Months += n * 12
			case 'M':
				d.Months += n
			case 'D':
				d.Seconds += float64(n) * 86400
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
		} else {
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			switch designator {
			case 'H':
				d.Seconds += f * 3600
			case 'M':
				d.Seconds += f * 60
			case 'S':
				d.Seconds += f
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
		}
	}

	return d, nil
}

// formatDuration formats a Duration as an XSD duration string.
func formatDuration(d Duration) string {
	var b strings.Builder
	if d.Negative {
		b.WriteByte('-')
	}
	b.WriteByte('P')

	years := d.Months / 12
	months := d.Months % 12
	if years != 0 {
		fmt.Fprintf(&b, "%dY", years)
	}
	if months != 0 {
		fmt.Fprintf(&b, "%dM", months)
	}

	secs := d.Seconds
	days := int(secs / 86400)
	secs -= float64(days) * 86400
	hours := int(secs / 3600)
	secs -= float64(hours) * 3600
	mins := int(secs / 60)
	secs -= float64(mins) * 60

	if days != 0 {
		fmt.Fprintf(&b, "%dD", days)
	}
	if hours != 0 || mins != 0 || secs != 0 {
		b.WriteByte('T')
		if hours != 0 {
			fmt.Fprintf(&b, "%dH", hours)
		}
		if mins != 0 {
			fmt.Fprintf(&b, "%dM", mins)
		}
		if secs != 0 {
			if secs == float64(int(secs)) {
				fmt.Fprintf(&b, "%dS", int(secs))
			} else {
				fmt.Fprintf(&b, "%gS", secs)
			}
		}
	}

	// Ensure at least "P0D" or "PT0S" for zero duration
	if b.Len() == 1 || (d.Negative && b.Len() == 2) {
		b.WriteString("T0S")
	}

	return b.String()
}
