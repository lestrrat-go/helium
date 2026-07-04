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
// (min/maxInclusive, min/maxExclusive) in the value space identified by
// builtinLocal. The range facets are defined ONLY on types whose primitive value
// space is ordered (value.Orderable); for every other builtin — a string-family
// type, boolean, the binary types, anyURI, QName/NOTATION, or a non-atomic
// (list/union) carrier with an empty/unknown local — the facet is INAPPLICABLE
// and this returns (0, false), which the caller treats as the bound being
// satisfied rather than coercing the value into a spurious comparison.
//
// For the ordered types the actual ordering is deferred to value.Compare, which
// orders numeric and date/time/duration value spaces and itself returns ok=false
// when an operand fails the strict lexical space. Note value.Compare also returns
// a deterministic order for boolean and the binary types (so enumeration can use
// cmp==0); those are NOT orderable, so a range facet can never fire on them here.
//
// There is deliberately NO empty-local fallback. A genuine ordered atomic value
// always reaches this function with its concrete ordered builtinLocal; an empty
// builtinLocal only arises for a NON-atomic carrier (an intermediate union or a
// list active member), which is not in an ordered value space, so the gate below
// rejects it. This is what stops the prior empty-local decimal fallback from
// mis-firing on a list active member of a numeric-looking union (e.g.
// union(list(xs:int)) with minInclusive), wrongly rejecting a valid list instance.
func compareForRangeFacet(v, bound, builtinLocal string) (int, bool) {
	if !value.Orderable(builtinLocal) {
		return 0, false
	}
	return value.Compare(v, bound, builtinLocal)
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
	lexicon.TypeDecimal: {}, lexicon.TypeInteger: {}, lexicon.TypeNonPositiveInteger: {}, lexicon.TypeNegativeInteger: {},
	lexicon.TypeLong: {}, lexicon.TypeInt: {}, lexicon.TypeShort: {}, lexicon.TypeByte: {},
	lexicon.TypeNonNegativeInteger: {}, lexicon.TypeUnsignedLong: {}, lexicon.TypeUnsignedInt: {},
	lexicon.TypeUnsignedShort: {}, lexicon.TypeUnsignedByte: {}, lexicon.TypePositiveInteger: {},
	// Floating point.
	lexicon.TypeFloat: {}, lexicon.TypeDouble: {},
	// Boolean.
	"boolean": {},
	// Date/time/duration.
	"dateTime": {}, "date": {}, "time": {}, "duration": {},
	"gYear": {}, "gYearMonth": {}, "gMonth": {}, "gDay": {}, "gMonthDay": {},
	// XSD 1.1 date/time/duration subtypes (compared in their primitive value space).
	lexicon.TypeDateTimeStamp: {}, lexicon.TypeDayTimeDuration: {}, lexicon.TypeYearMonthDuration: {},
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
	if builtinLocal == lexicon.TypeFloat || builtinLocal == lexicon.TypeDouble {
		if value.IsFloatNaN(v) && value.IsFloatNaN(ev) {
			return true
		}
	}
	cmp, ok := value.Compare(v, ev, builtinLocal)
	return ok && cmp == 0
}

func checkFacets(ctx context.Context, val string, valueNS map[string]string, fs *FacetSet, builtinLocal, whiteSpace, elemName, filename string, line int, vc *validationContext) error {
	var anyErr error

	// Enumeration.
	if len(fs.Enumeration) > 0 {
		found := false
		if builtinLocal == lexicon.TypeQName || builtinLocal == lexicon.TypeNotation {
			// For xs:NOTATION an UNPREFIXED value — the instance value here and each
			// enumeration literal below — picks up its own in-scope DEFAULT namespace,
			// the same rule the compile-time declared-notation lookup (check_facets.go)
			// applies to the enumeration literal, so the enum's value space is
			// consistent between compile and validation. resolveNotationOrQNameValue
			// centralizes that (each side resolves against ITS OWN default — the
			// instance's valueNS, the literal's captured enumNS); xs:QName keeps the
			// no-namespace value-space rule.
			valueQN, err := resolveNotationOrQNameValue(val, builtinLocal, valueNS)
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
					enumLex := normalizeWhiteSpace(ev, whiteSpace)
					enumQN, enumErr := resolveNotationOrQNameValue(enumLex, builtinLocal, enumNS)
					if enumErr != nil {
						continue
					}
					if valueQN == enumQN {
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
				if nev == val || enumerationValueEqual(val, nev, builtinLocal) {
					found = true
					break
				}
			}
		}
		if !found {
			set := "'" + strings.Join(fs.Enumeration, "', '") + "'"
			msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", val, set)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("enumeration")
		}
	}

	// minInclusive.
	if fs.MinInclusive != nil {
		if !checkMinInclusive(val, *fs.MinInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minInclusive'] The value '%s' is less than the minimum value allowed ('%s').", val, *fs.MinInclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minInclusive")
		}
	}

	// maxInclusive.
	if fs.MaxInclusive != nil {
		if !checkMaxInclusive(val, *fs.MaxInclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxInclusive'] The value '%s' is greater than the maximum value allowed ('%s').", val, *fs.MaxInclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxInclusive")
		}
	}

	// minExclusive.
	if fs.MinExclusive != nil {
		if !checkMinExclusive(val, *fs.MinExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'minExclusive'] The value '%s' must be greater than '%s'.", val, *fs.MinExclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minExclusive")
		}
	}

	// maxExclusive.
	if fs.MaxExclusive != nil {
		if !checkMaxExclusive(val, *fs.MaxExclusive, builtinLocal) {
			msg := fmt.Sprintf("[facet 'maxExclusive'] The value '%s' must be less than '%s'.", val, *fs.MaxExclusive)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxExclusive")
		}
	}

	// totalDigits.
	if fs.TotalDigits != nil {
		digits := value.CountTotalDigits(val)
		if digits > *fs.TotalDigits {
			msg := fmt.Sprintf("[facet 'totalDigits'] The value '%s' has more digits than are allowed ('%d').", val, *fs.TotalDigits)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("totalDigits")
		}
	}

	// fractionDigits.
	if fs.FractionDigits != nil {
		frac := value.CountFractionDigits(val)
		if frac > *fs.FractionDigits {
			msg := fmt.Sprintf("[facet 'fractionDigits'] The value '%s' has more fractional digits than are allowed ('%d').", val, *fs.FractionDigits)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("fractionDigits")
		}
	}

	// Length facets — interpretation depends on the builtin base type.
	//
	// length/minLength/maxLength do NOT apply to xs:QName or xs:NOTATION: their
	// length is undefined (a QName's value is an (namespace, local) pair, not a
	// string), so per XSD Part 2 (W3C Schema errata, bug 4009) these facets are
	// VACUOUSLY SATISFIED by every value. A schema may still declare them, but they
	// never constrain a value — enforcing a lexical rune-count here wrongly rejects
	// e.g. an `xs:QName` restricted to `length="7"` holding the value `a`.
	// Version-independent.
	lengthApplies := builtinLocal != lexicon.TypeQName && builtinLocal != lexicon.TypeNotation
	valueLen := facetLength(val, builtinLocal)

	if fs.Length != nil && lengthApplies {
		if valueLen != *fs.Length {
			msg := fmt.Sprintf("[facet 'length'] The value has a length of '%d'; this differs from the allowed length of '%d'.", valueLen, *fs.Length)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("length")
		}
	}

	if fs.MinLength != nil && lengthApplies {
		if valueLen < *fs.MinLength {
			msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", valueLen, *fs.MinLength)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("minLength")
		}
	}

	if fs.MaxLength != nil && lengthApplies {
		if valueLen > *fs.MaxLength {
			msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", valueLen, *fs.MaxLength)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("maxLength")
		}
	}

	if fs.ExplicitTimezone != nil {
		hasTimezone := hasExplicitTimezone(val)
		switch *fs.ExplicitTimezone {
		case attrValRequired:
			if !hasTimezone {
				msg := fmt.Sprintf("[facet 'explicitTimezone'] The value '%s' must have an explicit timezone.", val)
				vc.reportValidityError(ctx, filename, line, elemName, msg)
				anyErr = fmt.Errorf("explicitTimezone")
			}
		case attrValProhibited:
			if hasTimezone {
				msg := fmt.Sprintf("[facet 'explicitTimezone'] The value '%s' must not have an explicit timezone.", val)
				vc.reportValidityError(ctx, filename, line, elemName, msg)
				anyErr = fmt.Errorf("explicitTimezone")
			}
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
			if re.MatchString(val) {
				matched = true
				break
			}
		}
		if anyValid && !matched {
			var msg string
			if len(fs.Patterns) == 1 {
				msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the pattern '%s'.", val, fs.Patterns[0])
			} else {
				msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the patterns '%s'.", val, strings.Join(fs.Patterns, "', '"))
			}
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			anyErr = fmt.Errorf("pattern")
		}
	}

	return anyErr
}

func hasExplicitTimezone(value string) bool {
	if strings.HasSuffix(value, "Z") {
		return true
	}
	n := len(value)
	if n < len("+00:00") {
		return false
	}
	sign := value[n-6]
	if sign != '+' && sign != '-' {
		return false
	}
	return value[n-3] == ':' &&
		value[n-5] >= '0' && value[n-5] <= '9' &&
		value[n-4] >= '0' && value[n-4] <= '9' &&
		value[n-2] >= '0' && value[n-2] <= '9' &&
		value[n-1] >= '0' && value[n-1] <= '9'
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

// resolveNotationOrQNameValue resolves a QName/NOTATION lexical value against its
// in-scope namespaces for value-comparison (enumeration/fixed) purposes. It is the
// single bottom every comparison path funnels through, so the atomic, list-item,
// and union-member paths cannot diverge.
//
// For xs:NOTATION an UNPREFIXED value picks up the in-scope DEFAULT namespace
// (ns[""]) — the same rule the compile-time declared-notation lookup
// (check_facets.go) applies to an enumeration literal — so an unprefixed literal
// and an unprefixed instance value each name {default}local, and compile-time and
// validation agree the enum names {ns}local. builtinLocal gates this: xs:QName
// keeps the value-space rule (an unprefixed QName value has NO namespace — there
// is no compile-time QName default-namespace resolution to agree with), so it is
// resolved by resolveLexicalQName unchanged.
func resolveNotationOrQNameValue(lexical, builtinLocal string, ns map[string]string) (QName, error) {
	qn, err := resolveLexicalQName(lexical, ns)
	if err != nil {
		return QName{}, err
	}
	if builtinLocal == lexicon.TypeNotation && strings.IndexByte(lexical, ':') < 0 {
		qn.NS = ns[""]
	}
	return qn, nil
}

// facetLength returns the effective length of a value for facet checking.
// The interpretation depends on the builtin base type.
//
// Deliberately NOT shared with relaxng's same-named facetLength
// (relaxng/validate.go): this approximates hexBinary length as len/2 and
// base64Binary as len*3/4, whereas relaxng strict-decodes the binary value (and
// also handles list types). They are kept separate because adopting the strict
// variant here would change xsd's golden-validated facet error output.
func facetLength(val, builtinLocal string) int {
	switch builtinLocal {
	case typeIDRefs, typeEntities, typeNMTokens:
		// The built-in LIST datatypes: length/minLength/maxLength count the number
		// of whitespace-separated LIST ITEMS (XSD Part 2 §3.16 / cvc-length), not
		// characters. An empty value has zero items. value.XSDFields splits on XSD
		// whitespace only (space/tab/CR/LF), matching validateListValue.
		if val == "" {
			return 0
		}
		return len(value.XSDFields(val))
	case "hexBinary":
		// Length in octets (bytes) = len(hexString) / 2.
		return len(val) / 2
	case "base64Binary":
		// Length in octets — simplified.
		s := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, val)
		s = strings.TrimRight(s, "=")
		return len(s) * 3 / 4
	default:
		// String types: length in characters.
		return len([]rune(val))
	}
}
