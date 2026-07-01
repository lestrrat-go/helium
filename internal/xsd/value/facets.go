package value

import (
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// orderedRangeFacetTypes is the set of builtin base types whose PRIMITIVE value
// space is ORDERED, so the range facets (min/maxInclusive, min/maxExclusive) may
// apply to them. Per XSD 1.1 (§4.2.x, the {ordered} fundamental facet), the
// ordered primitives are the numeric types (decimal and its derived integers,
// float, double) and the date/time/duration family (duration,
// dayTimeDuration, yearMonthDuration, dateTime, dateTimeStamp, time, date, and
// the gregorian g-types). Every other primitive — string-family,
// boolean, hexBinary, base64Binary, anyURI, QName, NOTATION — is {ordered}=false,
// so a range facet is INAPPLICABLE to it and the bound is treated as satisfied.
//
// Compare can return a deterministic total order for some of these non-ordered
// types (boolean, hexBinary, base64Binary) purely so enumeration can rely on
// cmp==0; that order is NOT the XSD value-space order and must never be used to
// fire a range facet. Gating on Orderable keeps the range facets off those types
// regardless of what Compare would return.
//
// The xs:decimal family is reused from numericComparableTypes (the single source
// of that set); the float/double and date/time/duration members are added on
// top.
var orderedRangeFacetTypes = newOrderedRangeFacetTypes()

func newOrderedRangeFacetTypes() map[string]struct{} {
	m := make(map[string]struct{}, len(numericComparableTypes)+14)
	for k := range numericComparableTypes {
		m[k] = struct{}{}
	}
	for _, t := range []string{
		lexicon.TypeFloat, lexicon.TypeDouble,
		lexicon.TypeDateTime, lexicon.TypeDateTimeStamp, lexicon.TypeDate, lexicon.TypeTime,
		lexicon.TypeDuration, lexicon.TypeDayTimeDuration, lexicon.TypeYearMonthDuration,
		lexicon.TypeGYear, lexicon.TypeGYearMonth, lexicon.TypeGMonth, lexicon.TypeGDay, lexicon.TypeGMonthDay,
	} {
		m[t] = struct{}{}
	}
	return m
}

// Orderable reports whether builtinLocal's primitive value space is ordered, so
// the range facets (min/maxInclusive, min/maxExclusive) may apply to it. A caller
// gating on this keeps range facets off the non-ordered primitives even though
// Compare returns a deterministic order for boolean and the binary types.
func Orderable(builtinLocal string) bool {
	_, ok := orderedRangeFacetTypes[builtinLocal]
	return ok
}

// lengthApplicableTypes is the set of builtin locals on whose atomic value space
// the length facets (length, minLength, maxLength) are applicable. Per XSD §3.16
// length measures characters (string family), octets (the binary types) or
// characters of the lexical/canonical form for anyURI, QName and NOTATION. It is
// INAPPLICABLE to the numeric/decimal family, boolean, float, double and the
// date/time/duration family. The string-derived family is enumerated explicitly
// so the gate does not depend on primitive collapsing.
var lengthApplicableTypes = map[string]struct{}{
	lexicon.TypeString: {}, lexicon.TypeNormalizedString: {}, lexicon.TypeToken: {}, "language": {},
	"Name": {}, "NCName": {}, "ID": {}, lexicon.TypeIDREF: {}, "IDREFS": {},
	"ENTITY": {}, "ENTITIES": {}, "NMTOKEN": {}, "NMTOKENS": {},
	lexicon.TypeAnyURI: {}, lexicon.TypeQName: {}, lexicon.TypeNotation: {},
	"hexBinary": {}, "base64Binary": {},
}

// LengthApplicable reports whether the length facets (length, minLength,
// maxLength) are applicable to builtinLocal's atomic value space. Callers gating
// on this keep the length facets off the numeric/decimal family, boolean, float,
// double and the date/time/duration family, where XSD declares them inapplicable.
//
// On the applicable types — including xs:QName and xs:NOTATION — the length
// facets are CONSTRAINING (XSD 1.0 / libxml2 parity): a value whose length (rune
// count for the string family / QName / NOTATION / anyURI, octet count for the
// binary types) violates the bound is rejected, the same way the xsd package
// enforces it (see xsd's facetLength). This keeps relaxng and xsd consistent.
func LengthApplicable(builtinLocal string) bool {
	_, ok := lengthApplicableTypes[builtinLocal]
	return ok
}

// IsDecimalFamily reports whether builtinLocal is in the xs:decimal family
// (xs:decimal and its integer derivations). These are the ONLY types on which the
// digit facets (totalDigits, fractionDigits) are applicable; the facets are
// meaningless on float/double (no decimal digit notion in their value space) and
// on every non-numeric primitive.
func IsDecimalFamily(builtinLocal string) bool {
	_, ok := numericComparableTypes[builtinLocal]
	return ok
}

// CountTotalDigits counts the total number of significant digits in a decimal
// value. Per the XSD datatype spec: strip the sign, then count digits in the
// numeral excluding leading zeros before the integer part and trailing zeros
// after the fraction. Examples: "0.123" → 3, "0.023" → 3, "123" → 3, "12.3" → 3,
// "0.0" → 1.
func CountTotalDigits(value string) int {
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

// CountFractionDigits counts the number of significant digits after the decimal
// point. The fractionDigits facet constrains the value, not the lexical form, so
// trailing zeros are not significant: "1.20" → 1, "2.00" → 0, "1.0" → 0. If there
// is no decimal point, it returns 0.
func CountFractionDigits(value string) int {
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
