package xsd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// compareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func compareDecimal(a, b string) int {
	return value.CompareDecimal(a, b)
}

// checkMinInclusive compares value >= bound using type-aware comparison.
func checkMinInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp >= 0
}

// checkMaxInclusive compares value <= bound using type-aware comparison.
func checkMaxInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp <= 0
}

// checkMinExclusive compares value > bound using type-aware comparison.
func checkMinExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp > 0
}

// checkMaxExclusive compares value < bound using type-aware comparison.
func checkMaxExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp < 0
}

func checkFacets(value string, valueNS map[string]string, fs *FacetSet, builtinLocal, elemName, filename string, line int, vc *validationContext) error {
	var anyErr error

	// Enumeration.
	if len(fs.Enumeration) > 0 {
		found := false
		if builtinLocal == "QName" || builtinLocal == "NOTATION" {
			valueQN, err := resolveLexicalQName(value, valueNS)
			if err == nil {
				for i, ev := range fs.Enumeration {
					var enumNS map[string]string
					if i < len(fs.EnumerationNS) {
						enumNS = fs.EnumerationNS[i]
					}
					enumQN, enumErr := resolveLexicalQName(ev, enumNS)
					if enumErr == nil && valueQN == enumQN {
						found = true
						break
					}
				}
			}
		} else {
			for _, ev := range fs.Enumeration {
				if value == ev {
					found = true
					break
				}
			}
		}
		if !found {
			set := "'" + strings.Join(fs.Enumeration, "', '") + "'"
			msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", value, set)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("enumeration")
		}
	}

	// minInclusive.
	if fs.MinInclusive != nil {
		if !checkMinInclusive(value, *fs.MinInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minInclusive'] The value '%s' is less than the minimum value allowed ('%s').", value, *fs.MinInclusive)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("minInclusive")
		}
	}

	// maxInclusive.
	if fs.MaxInclusive != nil {
		if !checkMaxInclusive(value, *fs.MaxInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxInclusive'] The value '%s' is greater than the maximum value allowed ('%s').", value, *fs.MaxInclusive)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxInclusive")
		}
	}

	// minExclusive.
	if fs.MinExclusive != nil {
		if !checkMinExclusive(value, *fs.MinExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minExclusive'] The value '%s' must be greater than '%s'.", value, *fs.MinExclusive)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("minExclusive")
		}
	}

	// maxExclusive.
	if fs.MaxExclusive != nil {
		if !checkMaxExclusive(value, *fs.MaxExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxExclusive'] The value '%s' must be less than '%s'.", value, *fs.MaxExclusive)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxExclusive")
		}
	}

	// totalDigits.
	if fs.TotalDigits != nil {
		digits := countTotalDigits(value)
		if digits > *fs.TotalDigits {
			msg := fmt.Sprintf("[facet 'totalDigits'] The value '%s' has more digits than are allowed ('%d').", value, *fs.TotalDigits)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("totalDigits")
		}
	}

	// fractionDigits.
	if fs.FractionDigits != nil {
		frac := countFractionDigits(value)
		if frac > *fs.FractionDigits {
			msg := fmt.Sprintf("[facet 'fractionDigits'] The value '%s' has more fractional digits than are allowed ('%d').", value, *fs.FractionDigits)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("fractionDigits")
		}
	}

	// Length facets — interpretation depends on the builtin base type.
	valueLen := facetLength(value, builtinLocal)

	if fs.Length != nil {
		if valueLen != *fs.Length {
			msg := fmt.Sprintf("[facet 'length'] The value has a length of '%d'; this differs from the allowed length of '%d'.", valueLen, *fs.Length)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("length")
		}
	}

	if fs.MinLength != nil {
		if valueLen < *fs.MinLength {
			msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", valueLen, *fs.MinLength)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("minLength")
		}
	}

	if fs.MaxLength != nil {
		if valueLen > *fs.MaxLength {
			msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", valueLen, *fs.MaxLength)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxLength")
		}
	}

	// Pattern.
	if fs.Pattern != nil {
		re, err := regexp.Compile("^(?:" + *fs.Pattern + ")$")
		if err == nil && !re.MatchString(value) {
			msg := fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the pattern '%s'.", value, *fs.Pattern)
			vc.reportValidityError(filename, line, elemName, msg)
			anyErr = fmt.Errorf("pattern")
		}
	}

	return anyErr
}

func resolveLexicalQName(value string, ns map[string]string) (QName, error) {
	if err := validateQName(value); err != nil {
		return QName{}, err
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 1 {
		return QName{Local: value, NS: ns[""]}, nil
	}
	uri, ok := ns[parts[0]]
	if !ok {
		return QName{}, fmt.Errorf("undeclared prefix %q", parts[0])
	}
	return QName{Local: parts[1], NS: uri}, nil
}

// countTotalDigits counts the total number of significant digits in a decimal value.
// Per XML Schema spec: strip sign, then count digits in the numeral excluding
// leading zeros before the integer part and trailing zeros after the fraction.
// Examples: "0.123" → 3, "0.023" → 3, "123" → 3, "12.3" → 3, "0.0" → 1
func countTotalDigits(value string) int {
	// Strip sign.
	s := value
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}

	dotIdx := strings.Index(s, ".")
	if dotIdx < 0 {
		// No decimal point — count digits in integer, stripping leading zeros.
		s = strings.TrimLeft(s, "0")
		if s == "" {
			return 1
		}
		return len(s)
	}

	// Has decimal point. Integer part = s[:dotIdx], fraction part = s[dotIdx+1:]
	intPart := strings.TrimLeft(s[:dotIdx], "0")
	fracPart := strings.TrimRight(s[dotIdx+1:], "0")

	total := len(intPart) + len(fracPart)
	if total == 0 {
		return 1 // "0.0" has 1 digit
	}
	return total
}

// countFractionDigits counts the number of digits after the decimal point.
// If there is no decimal point, returns 0.
// Trailing zeros are significant: "1.20" → 2, "1.0" → 1.
func countFractionDigits(value string) int {
	s := value
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	dotIdx := strings.Index(s, ".")
	if dotIdx < 0 {
		return 0
	}
	return len(s) - dotIdx - 1
}

// facetLength returns the effective length of a value for facet checking.
// The interpretation depends on the builtin base type.
func facetLength(value, builtinLocal string) int {
	switch builtinLocal {
	case "hexBinary":
		// Length in octets (bytes) = len(hexString) / 2.
		return len(value) / 2
	case "base64Binary":
		// Length in octets — simplified.
		s := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, value)
		s = strings.TrimRight(s, "=")
		return len(s) * 3 / 4
	default:
		// String types: length in characters.
		return len([]rune(value))
	}
}
