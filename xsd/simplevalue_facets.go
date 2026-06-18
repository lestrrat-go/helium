package xsd

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// compareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func compareDecimal(a, b string) int {
	return value.CompareDecimal(a, b)
}

// compareForRangeFacet compares two ordered values for a range facet
// (min/maxInclusive, min/maxExclusive). It first tries value.Compare for the
// builtin type. value.Compare is deliberately strict: it returns ok=false for an
// unrecognized or non-value-comparable builtinLocal (e.g. an empty local from an
// anonymous numeric base, or a string-family type). Range facets, however, are
// only meaningful on ordered (numeric/date-time) types. For the empty-local case
// — a numeric value over an anonymous/empty base that lost its builtin name — we
// fall back to a decimal comparison ONLY when BOTH operands genuinely parse as
// decimals (so the leaf is in the numeric value space); a non-numeric value (e.g.
// a string leaf) yields ok=false and the caller treats the facet as inapplicable
// rather than coercing it into a spurious numeric comparison. A non-empty
// builtinLocal that value.Compare could not compare (a string-family or otherwise
// non-numeric type) is likewise left indeterminate.
func compareForRangeFacet(v, bound, builtinLocal string) (int, bool) {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if ok {
		return cmp, true
	}
	if builtinLocal != "" {
		return 0, false
	}
	// Empty builtin local: only enforce the bound when the value is genuinely in
	// the decimal value space. value.Compare(..., "decimal") trims XSD whitespace
	// and validates both operands against the decimal lexical space, returning
	// ok=false for a non-numeric leaf so the range facet stays inapplicable.
	return value.Compare(v, bound, "decimal")
}

// checkMinInclusive compares value >= bound using type-aware comparison.
func checkMinInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := compareForRangeFacet(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp >= 0
}

// checkMaxInclusive compares value <= bound using type-aware comparison.
func checkMaxInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := compareForRangeFacet(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp <= 0
}

// checkMinExclusive compares value > bound using type-aware comparison.
func checkMinExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := compareForRangeFacet(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp > 0
}

// checkMaxExclusive compares value < bound using type-aware comparison.
func checkMaxExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := compareForRangeFacet(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp < 0
}

// enumValueSpaceTypes is the set of builtin base types for which value.Compare
// implements correct value-space equality, so the enumeration facet may accept a
// member that is value-equal but lexically distinct. It deliberately excludes
// string-family and anyURI types: value.Compare falls back to decimal comparison
// for any unrecognized builtin, which would wrongly treat numeric-looking
// lexicals as equal (e.g. xs:string enumeration "5" accepting "5.0"). Those
// types stay lexical-only, which is correct because their value space equals
// their (whitespace-processed) lexical space.
var enumValueSpaceTypes = map[string]struct{}{
	// Numeric (decimal-derived).
	"decimal": {}, "integer": {}, "nonPositiveInteger": {}, "negativeInteger": {},
	"long": {}, "int": {}, "short": {}, "byte": {},
	"nonNegativeInteger": {}, "unsignedLong": {}, "unsignedInt": {},
	"unsignedShort": {}, "unsignedByte": {}, "positiveInteger": {},
	// Floating point.
	"float": {}, "double": {},
	// Boolean.
	"boolean": {},
	// Date/time/duration.
	"dateTime": {}, "date": {}, "time": {}, "duration": {},
	"gYear": {}, "gYearMonth": {}, "gMonth": {}, "gDay": {}, "gMonthDay": {},
	// Binary (compared by decoded octets, not lexical text).
	"hexBinary": {}, "base64Binary": {},
}

// enumerationValueEqual reports whether v is value-equal to a member ev for the
// purpose of the enumeration facet. It only performs value-space comparison for
// types in enumValueSpaceTypes (numeric, boolean, date/time, and binary); for
// all others (string-family, anyURI, and the empty/untyped case) enumeration
// stays lexical-only and this returns false. For float/double, XSD treats NaN as
// equal to NaN for enumeration (unlike ordering, where NaN is incomparable),
// handled explicitly here.
func enumerationValueEqual(v, ev, builtinLocal string) bool {
	if _, ok := enumValueSpaceTypes[builtinLocal]; !ok {
		return false
	}
	if builtinLocal == "float" || builtinLocal == "double" {
		if value.IsFloatNaN(v) && value.IsFloatNaN(ev) {
			return true
		}
	}
	cmp, ok := value.Compare(v, ev, builtinLocal)
	return ok && cmp == 0
}

func checkFacets(ctx context.Context, value string, valueNS map[string]string, fs *FacetSet, builtinLocal, whiteSpace, elemName, filename string, line int, vc *validationContext) error {
	var anyErr error

	// Enumeration.
	if len(fs.Enumeration) > 0 {
		found := false
		if builtinLocal == lexicon.TypeQName || builtinLocal == lexicon.TypeNotation {
			valueQN, err := resolveLexicalQName(value, valueNS)
			if err == nil {
				for i, ev := range fs.Enumeration {
					var enumNS map[string]string
					if i < len(fs.EnumerationNS) {
						enumNS = fs.EnumerationNS[i]
					}
					// The enumeration literal is a value in the constrained type's
					// value space, so it must be whitespace-normalized with the same
					// effective whiteSpace facet the instance value already had
					// applied before its QName is resolved.
					enumQN, enumErr := resolveLexicalQName(normalizeWhiteSpace(ev, whiteSpace), enumNS)
					if enumErr == nil && valueQN == enumQN {
						found = true
						break
					}
				}
			}
		} else {
			// Enumeration is defined on the value space. A lexical match is
			// always sufficient; additionally, for value-space-comparable types
			// (see enumValueSpaceTypes) a lexically distinct value that is
			// value-equal to a member must also be accepted. String-family and
			// other non-comparable types stay lexical-only. Each enumeration
			// literal is whitespace-normalized with the constrained type's
			// effective whiteSpace facet first, mirroring the normalization the
			// instance value already underwent — otherwise a token enumeration
			// "a  b" (two spaces) would never match the collapsed instance "a b".
			for _, ev := range fs.Enumeration {
				nev := normalizeWhiteSpace(ev, whiteSpace)
				if nev == value || enumerationValueEqual(value, nev, builtinLocal) {
					found = true
					break
				}
			}
		}
		if !found {
			set := "'" + strings.Join(fs.Enumeration, "', '") + "'"
			msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", value, set)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("enumeration")
		}
	}

	// minInclusive.
	if fs.MinInclusive != nil {
		if !checkMinInclusive(value, *fs.MinInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minInclusive'] The value '%s' is less than the minimum value allowed ('%s').", value, *fs.MinInclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minInclusive")
		}
	}

	// maxInclusive.
	if fs.MaxInclusive != nil {
		if !checkMaxInclusive(value, *fs.MaxInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxInclusive'] The value '%s' is greater than the maximum value allowed ('%s').", value, *fs.MaxInclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxInclusive")
		}
	}

	// minExclusive.
	if fs.MinExclusive != nil {
		if !checkMinExclusive(value, *fs.MinExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minExclusive'] The value '%s' must be greater than '%s'.", value, *fs.MinExclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minExclusive")
		}
	}

	// maxExclusive.
	if fs.MaxExclusive != nil {
		if !checkMaxExclusive(value, *fs.MaxExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxExclusive'] The value '%s' must be less than '%s'.", value, *fs.MaxExclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxExclusive")
		}
	}

	// totalDigits.
	if fs.TotalDigits != nil {
		digits := countTotalDigits(value)
		if digits > *fs.TotalDigits {
			msg := fmt.Sprintf("[facet 'totalDigits'] The value '%s' has more digits than are allowed ('%d').", value, *fs.TotalDigits)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("totalDigits")
		}
	}

	// fractionDigits.
	if fs.FractionDigits != nil {
		frac := countFractionDigits(value)
		if frac > *fs.FractionDigits {
			msg := fmt.Sprintf("[facet 'fractionDigits'] The value '%s' has more fractional digits than are allowed ('%d').", value, *fs.FractionDigits)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("fractionDigits")
		}
	}

	// Length facets — interpretation depends on the builtin base type.
	valueLen := facetLength(value, builtinLocal)

	if fs.Length != nil {
		if valueLen != *fs.Length {
			msg := fmt.Sprintf("[facet 'length'] The value has a length of '%d'; this differs from the allowed length of '%d'.", valueLen, *fs.Length)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("length")
		}
	}

	if fs.MinLength != nil {
		if valueLen < *fs.MinLength {
			msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", valueLen, *fs.MinLength)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minLength")
		}
	}

	if fs.MaxLength != nil {
		if valueLen > *fs.MaxLength {
			msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", valueLen, *fs.MaxLength)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxLength")
		}
	}

	// Pattern: multiple <xs:pattern> facets in the same restriction step are
	// ORed — the value is valid if it matches any of them. Regexes are compiled
	// once at schema compile time (FacetSet.compiledPatterns); a nil entry means
	// that pattern failed to compile and is skipped.
	if len(fs.Patterns) > 0 {
		matched := false
		anyValid := false
		for _, re := range fs.compiledPatterns {
			if re == nil {
				continue
			}
			anyValid = true
			if re.MatchString(value) {
				matched = true
				break
			}
		}
		if anyValid && !matched {
			var msg string
			if len(fs.Patterns) == 1 {
				msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the pattern '%s'.", value, fs.Patterns[0])
			} else {
				msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the patterns '%s'.", value, strings.Join(fs.Patterns, "', '"))
			}
			vc.reportValidityError(ctx, filename, line, elemName, msg)
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
		// An unprefixed QName/NOTATION *value* does NOT pick up the in-scope
		// default namespace (unlike element/attribute names): per XSD it
		// resolves to no namespace. So never consult ns[""] here.
		return QName{Local: value}, nil
	}
	// The "xml" prefix is predeclared (bound to the XML namespace) and never
	// needs an explicit declaration, so it is always in scope regardless of the
	// instance's collected namespace context.
	if parts[0] == "xml" {
		return QName{Local: parts[1], NS: lexicon.NamespaceXML}, nil
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

// countFractionDigits counts the number of significant digits after the
// decimal point. The fractionDigits facet constrains the value, not the
// lexical form, so trailing zeros are not significant: "1.20" → 1, "2.00" → 0,
// "1.0" → 0. If there is no decimal point, returns 0.
func countFractionDigits(value string) int {
	s := value
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	_, frac, found := strings.Cut(s, ".")
	if !found {
		return 0
	}
	return len(strings.TrimRight(frac, "0"))
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
