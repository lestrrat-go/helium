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
		return v.Value.(string), nil
	case TypeInteger,
		TypeLong, TypeInt, TypeShort, TypeByte,
		TypeUnsignedLong, TypeUnsignedInt, TypeUnsignedShort, TypeUnsignedByte,
		TypeNonNegativeInteger, TypeNonPositiveInteger,
		TypePositiveInteger, TypeNegativeInteger:
		return v.Value.(*big.Int).String(), nil
	case TypeDecimal:
		return DecimalToString(v.Value.(*big.Rat)), nil
	case TypeDouble:
		return formatXPathDouble(v.Value.(float64)), nil
	case TypeFloat:
		return formatXPathFloat(v.Value.(float64)), nil
	case TypeBoolean:
		if v.Value.(bool) {
			return "true", nil
		}
		return "false", nil
	case TypeDate:
		t := v.Value.(time.Time)
		return fmt.Sprintf("%s-%02d-%02d%s", formatXSDYear(t.Year()), t.Month(), t.Day(), formatXSDTimezone(t)), nil
	case TypeDateTime:
		t := v.Value.(time.Time)
		s := fmt.Sprintf("%s-%02d-%02dT%02d:%02d:%02d", formatXSDYear(t.Year()), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		if ns := t.Nanosecond(); ns > 0 {
			frac := fmt.Sprintf(".%09d", ns)
			s += strings.TrimRight(frac, "0")
		}
		return s + formatXSDTimezone(t), nil
	case TypeTime:
		t := v.Value.(time.Time)
		s := fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
		if ns := t.Nanosecond(); ns > 0 {
			frac := fmt.Sprintf(".%09d", ns)
			s += strings.TrimRight(frac, "0")
		}
		return s + formatXSDTimezone(t), nil
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
	// XSD 1.1 §3.3.5: plain decimal for abs in [1e-6, 1e6), scientific notation
	// for abs >= 1e6 or abs < 1e-6. The strict < excludes 1e6 intentionally.
	if abs > 0.000001 && abs < 1_000_000 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
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
	if abs > 0.000001 && abs < 1_000_000 {
		s := strconv.FormatFloat(float64(f32), 'f', -1, 32)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
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
