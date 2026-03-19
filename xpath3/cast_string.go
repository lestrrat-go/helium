package xpath3

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func castToString(v AtomicValue) (AtomicValue, error) {
	s, err := atomicToString(v)
	if err != nil {
		return AtomicValue{}, err
	}
	return AtomicValue{TypeName: TypeString, Value: s}, nil
}

// AtomicToString returns the canonical string representation of an atomic value.
func AtomicToString(v AtomicValue) (string, error) {
	return atomicToString(v)
}

// atomicToString returns the canonical string representation of an atomic value.
func atomicToString(v AtomicValue) (string, error) {
	switch v.TypeName {
	case TypeString, TypeAnyURI, TypeUntypedAtomic,
		TypeNormalizedString, TypeToken, TypeLanguage, TypeName, TypeNCName,
		TypeNMTOKEN, TypeNMTOKENS, TypeENTITY, TypeID, TypeIDREF, TypeIDREFS,
		TypeGDay, TypeGMonth, TypeGMonthDay, TypeGYear, TypeGYearMonth:
		s, ok := v.Value.(string)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected string value for %s", v.TypeName)
		}
		return s, nil
	case TypeInteger,
		TypeLong, TypeInt, TypeShort, TypeByte,
		TypeUnsignedLong, TypeUnsignedInt, TypeUnsignedShort, TypeUnsignedByte,
		TypeNonNegativeInteger, TypeNonPositiveInteger,
		TypePositiveInteger, TypeNegativeInteger:
		n, ok := v.Value.(*big.Int)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected *big.Int value for %s", v.TypeName)
		}
		return n.String(), nil
	case TypeDecimal:
		r, ok := v.Value.(*big.Rat)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected *big.Rat value for %s", v.TypeName)
		}
		return DecimalToString(r), nil
	case TypeDouble:
		return formatXPathDouble(v.FloatVal().Float64()), nil
	case TypeFloat:
		return formatXPathFloat(v.FloatVal().Float64()), nil
	case TypeBoolean:
		b, ok := v.Value.(bool)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected bool value for %s", v.TypeName)
		}
		if b {
			return "true", nil
		}
		return "false", nil
	case TypeDate:
		t, ok := v.Value.(time.Time)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected time.Time value for %s", v.TypeName)
		}
		return fmt.Sprintf("%s-%02d-%02d%s", formatXSDYear(t.Year()), t.Month(), t.Day(), formatXSDTimezone(t)), nil
	case TypeDateTime, TypeDateTimeStamp:
		t, ok := v.Value.(time.Time)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected time.Time value for %s", v.TypeName)
		}
		s := fmt.Sprintf("%s-%02d-%02dT%02d:%02d:%02d", formatXSDYear(t.Year()), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		if ns := t.Nanosecond(); ns > 0 {
			frac := fmt.Sprintf(".%09d", ns)
			s += strings.TrimRight(frac, "0")
		}
		return s + formatXSDTimezone(t), nil
	case TypeTime:
		t, ok := v.Value.(time.Time)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected time.Time value for %s", v.TypeName)
		}
		s := fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
		if ns := t.Nanosecond(); ns > 0 {
			frac := fmt.Sprintf(".%09d", ns)
			s += strings.TrimRight(frac, "0")
		}
		return s + formatXSDTimezone(t), nil
	case TypeDuration, TypeDayTimeDuration, TypeYearMonthDuration:
		d, ok := v.Value.(Duration)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected Duration value for %s", v.TypeName)
		}
		return formatDuration(d, v.TypeName), nil
	case TypeBase64Binary:
		b, ok := v.Value.([]byte)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected []byte value for %s", v.TypeName)
		}
		return base64.StdEncoding.EncodeToString(b), nil
	case TypeHexBinary:
		b, ok := v.Value.([]byte)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected []byte value for %s", v.TypeName)
		}
		return strings.ToUpper(hex.EncodeToString(b)), nil
	case TypeQName:
		q, ok := v.Value.(QNameValue)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected QNameValue for %s", v.TypeName)
		}
		if q.Prefix != "" {
			return q.Prefix + ":" + q.Local, nil
		}
		return q.Local, nil
	}
	// User-defined types: format based on the underlying Go value type.
	switch val := v.Value.(type) {
	case string:
		return val, nil
	case *big.Int:
		return val.String(), nil
	case *big.Rat:
		return DecimalToString(val), nil
	case *FloatValue:
		return formatXPathDouble(val.Float64()), nil
	case float64:
		return formatXPathDouble(val), nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	case time.Time:
		return fmt.Sprintf("%v", val), nil
	case Duration:
		return formatDuration(val, TypeDuration), nil
	}
	return fmt.Sprintf("%v", v.Value), nil
}

// formatXPathDouble formats a float64 using XPath canonical representation.
func formatXPathDouble(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "INF"
	}
	if math.IsInf(f, -1) {
		return "-INF"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}

	abs := math.Abs(f)
	// XPath F&O: plain decimal for abs in [1e-6, 1e6), scientific notation otherwise.
	if abs >= 0.000001 && abs < 1_000_000 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}

	s := strconv.FormatFloat(f, 'E', -1, 64)
	if idx := strings.Index(s, "E"); idx >= 0 {
		mantissa := s[:idx]
		expPart := s[idx+1:]
		if !strings.Contains(mantissa, ".") {
			mantissa += ".0"
		}
		expPart = strings.TrimPrefix(expPart, "+")
		if strings.HasPrefix(expPart, "-") {
			inner := strings.TrimLeft(expPart[1:], "0")
			if inner == "" {
				inner = "0"
			}
			expPart = "-" + inner
		} else {
			expPart = strings.TrimLeft(expPart, "0")
			if expPart == "" {
				expPart = "0"
			}
		}
		s = mantissa + "E" + expPart
	}
	return s
}

// formatXPathFloat formats a float64 (representing an xs:float) using
// single-precision XPath canonical representation with 32-bit precision.
func formatXPathFloat(f float64) string {
	f32 := float32(f)
	if math.IsNaN(float64(f32)) {
		return "NaN"
	}
	if math.IsInf(float64(f32), 1) {
		return "INF"
	}
	if math.IsInf(float64(f32), -1) {
		return "-INF"
	}
	if f32 == 0 {
		if math.Signbit(float64(f32)) {
			return "-0"
		}
		return "0"
	}

	abs := math.Abs(float64(f32))
	if abs >= 0.000001 && abs < 1_000_000 {
		return strconv.FormatFloat(float64(f32), 'f', -1, 32)
	}

	s := strconv.FormatFloat(float64(f32), 'E', -1, 32)
	if idx := strings.Index(s, "E"); idx >= 0 {
		mantissa := s[:idx]
		expPart := s[idx+1:]
		if !strings.Contains(mantissa, ".") {
			mantissa += ".0"
		}
		expPart = strings.TrimPrefix(expPart, "+")
		if strings.HasPrefix(expPart, "-") {
			inner := strings.TrimLeft(expPart[1:], "0")
			if inner == "" {
				inner = "0"
			}
			expPart = "-" + inner
		} else {
			expPart = strings.TrimLeft(expPart, "0")
			if expPart == "" {
				expPart = "0"
			}
		}
		s = mantissa + "E" + expPart
	}
	return s
}

// isValidDecimalString checks if a string is a valid xs:decimal literal.
func isValidDecimalString(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	if i >= len(s) {
		return false
	}
	hasDigit := false
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		hasDigit = true
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			hasDigit = true
			i++
		}
	}
	return hasDigit && i == len(s)
}

// normalizeWhitespace replaces #x9, #xA, #xD with #x20 (for xs:normalizedString).
func normalizeWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\t', '\n', '\r':
			return ' '
		default:
			return r
		}
	}, s)
}

// collapseWhitespace normalizes then collapses whitespace (for xs:token and derived).
// Replaces #x9, #xA, #xD with #x20, then collapses runs of #x20 to a single space,
// and strips leading/trailing spaces.
func collapseWhitespace(s string) string {
	s = normalizeWhitespace(s)
	var b strings.Builder
	b.Grow(len(s))
	inSpace := true // treat leading spaces as collapsible
	for _, r := range s {
		if r == ' ' {
			if !inSpace {
				inSpace = true
				b.WriteByte(' ')
			}
		} else {
			inSpace = false
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > 0 && result[len(result)-1] == ' ' {
		return result[:len(result)-1]
	}
	return result
}

// formatXSDYear formats a year per XSD canonical rules:
// - At least 4 digits, zero-padded
// - Negative years get a leading '-'
// - Years > 9999 use as many digits as needed
func formatXSDYear(year int) string {
	if year < 0 {
		return fmt.Sprintf("-%04d", -year)
	}
	return fmt.Sprintf("%04d", year)
}

// parseXPathDouble parses s as an xs:double using XSD 1.1 lexical rules.
// Valid special values: "INF", "+INF", "-INF", "NaN".
func parseXPathDouble(s string) (float64, error) {
	switch s {
	case "INF", "+INF":
		return math.Inf(1), nil
	case "-INF":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}
	// Reject case-insensitive nan/inf variants that strconv.ParseFloat accepts
	// but XSD 1.1 does not (only exact "NaN", "INF", "+INF", "-INF" are valid).
	lower := strings.ToLower(s)
	if lower == "nan" || lower == "inf" || lower == "+inf" || lower == "-inf" {
		return 0, fmt.Errorf("invalid xs:double value: %s", s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	// Reject infinity from ParseFloat — only whitelisted forms above are valid
	if math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid xs:double value: %s", s)
	}
	return f, nil
}
