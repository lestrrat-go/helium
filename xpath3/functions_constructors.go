package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
)

func init() {
	// Base XSD types — delegate to CastAtomic
	for _, entry := range []struct {
		name       string
		targetType string
	}{
		{"string", TypeString},
		{"boolean", TypeBoolean},
		{"decimal", TypeDecimal},
		{"double", TypeDouble},
		{"float", TypeFloat},
		{"integer", TypeInteger},
		{"date", TypeDate},
		{"dateTime", TypeDateTime},
		{"time", TypeTime},
		{"duration", TypeDuration},
		{"dayTimeDuration", TypeDayTimeDuration},
		{"yearMonthDuration", TypeYearMonthDuration},
		{"anyURI", TypeAnyURI},
		{"base64Binary", TypeBase64Binary},
		{"hexBinary", TypeHexBinary},
		{"untypedAtomic", TypeUntypedAtomic},
		{"QName", TypeQName},
	} {
		registerNS(NSXS, entry.name, 1, 1, makeXSConstructor(entry.targetType))
	}

	// Derived integer types — cast to integer, then validate range
	for _, entry := range []struct {
		name     string
		typeName string
		min      int64
		max      int64
	}{
		{"long", TypeLong, math.MinInt64, math.MaxInt64},
		{"int", TypeInt, -2147483648, 2147483647},
		{"short", TypeShort, -32768, 32767},
		{"byte", TypeByte, -128, 127},
		{"unsignedLong", TypeUnsignedLong, 0, math.MaxInt64}, // XSD max is 18446744073709551615 but we use int64
		{"unsignedInt", TypeUnsignedInt, 0, 4294967295},
		{"unsignedShort", TypeUnsignedShort, 0, 65535},
		{"unsignedByte", TypeUnsignedByte, 0, 255},
		{"nonNegativeInteger", TypeNonNegativeInteger, 0, math.MaxInt64},
		{"nonPositiveInteger", TypeNonPositiveInteger, math.MinInt64, 0},
		{"positiveInteger", TypePositiveInteger, 1, math.MaxInt64},
		{"negativeInteger", TypeNegativeInteger, math.MinInt64, -1},
	} {
		registerNS(NSXS, entry.name, 1, 1, makeXSIntegerRange(entry.typeName, entry.min, entry.max))
	}

	// Derived string types — cast to string, validate, store with derived type name
	for _, entry := range []struct {
		name     string
		typeName string
		re       *regexp.Regexp // nil = no validation, just string cast
	}{
		{"normalizedString", TypeNormalizedString, nil},
		{"token", TypeToken, nil},
		{"language", TypeLanguage, reLang},
		{"Name", TypeName, reName},
		{"NCName", TypeNCName, reNCName},
		{"NMTOKEN", TypeNMTOKEN, reNMTOKEN},
		{"ENTITY", TypeENTITY, reNCName},
		{"ID", TypeID, reNCName},
		{"IDREF", TypeIDREF, reNCName},
	} {
		registerNS(NSXS, entry.name, 1, 1, makeXSStringRestriction(entry.typeName, entry.re))
	}

	// List types — split by whitespace, validate each token
	registerNS(NSXS, "NMTOKENS", 1, 1, makeXSTokenList(TypeNMTOKENS, reNMTOKEN))
	registerNS(NSXS, "IDREFS", 1, 1, makeXSTokenList(TypeIDREFS, reNCName))
	registerNS(NSXS, "ENTITIES", 1, 1, makeXSTokenList(TypeENTITIES, reNCName))

	// Gregorian date part types
	for _, entry := range []struct {
		name     string
		typeName string
		re       *regexp.Regexp
	}{
		{"gDay", TypeGDay, reGDay},
		{"gMonth", TypeGMonth, reGMonth},
		{"gMonthDay", TypeGMonthDay, reGMonthDay},
		{"gYear", TypeGYear, reGYear},
		{"gYearMonth", TypeGYearMonth, reGYearMonth},
	} {
		registerNS(NSXS, entry.name, 1, 1, makeXSGregorian(entry.typeName, entry.re))
	}

	// xs:dateTimeStamp — dateTime with required timezone
	registerNS(NSXS, "dateTimeStamp", 1, 1, makeXSDateTimeStamp())

	// xs:error — always raises an error (used for type checking tests)
	registerNS(NSXS, "error", 1, 1, fnXSError)

	// xs:numeric — cast to double (xs:numeric is an abstract union type)
	registerNS(NSXS, "numeric", 1, 1, makeXSConstructor(TypeDouble))
}

// makeXSConstructor returns a constructor function for a base XSD type.
func makeXSConstructor(targetType string) func(context.Context, []Sequence) (Sequence, error) {
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		result, err := CastAtomic(a, targetType)
		if err != nil {
			return nil, err
		}
		return SingleAtomic(result), nil
	}
}

// makeXSIntegerRange returns a constructor for a derived integer type with range validation.
func makeXSIntegerRange(typeName string, minVal, maxVal int64) func(context.Context, []Sequence) (Sequence, error) {
	minBig := big.NewInt(minVal)
	maxBig := big.NewInt(maxVal)
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		iv, err := CastAtomic(a, TypeInteger)
		if err != nil {
			return nil, err
		}
		n := iv.BigInt()
		if n.Cmp(minBig) < 0 || n.Cmp(maxBig) > 0 {
			return nil, &XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("value %s out of range for %s", n.String(), typeName),
			}
		}
		return SingleAtomic(AtomicValue{TypeName: typeName, Value: n}), nil
	}
}

var (
	reNCName  = regexp.MustCompile(`^[a-zA-Z_][\w.-]*$`)
	reName    = regexp.MustCompile(`^[a-zA-Z_:][\w.:-]*$`)
	reNMTOKEN = regexp.MustCompile(`^[\w.:-]+$`)
	reLang    = regexp.MustCompile(`(?i)^[a-z]{1,8}(-[a-z0-9]{1,8})*$`)
)

// makeXSStringRestriction returns a constructor for a derived string type.
func makeXSStringRestriction(typeName string, validate *regexp.Regexp) func(context.Context, []Sequence) (Sequence, error) {
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		s, err := atomicToString(a)
		if err != nil {
			return nil, err
		}
		s = strings.TrimSpace(s)
		if validate != nil && !validate.MatchString(s) {
			return nil, &XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("cannot cast %q to %s", s, typeName),
			}
		}
		return SingleAtomic(AtomicValue{TypeName: typeName, Value: s}), nil
	}
}

// makeXSTokenList returns a constructor for xs:NMTOKENS or xs:IDREFS (whitespace-separated list).
func makeXSTokenList(typeName string, tokenRe *regexp.Regexp) func(context.Context, []Sequence) (Sequence, error) {
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		s, err := atomicToString(a)
		if err != nil {
			return nil, err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, &XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("cannot cast empty string to %s", typeName),
			}
		}
		tokens := strings.Fields(s)
		for _, tok := range tokens {
			if !tokenRe.MatchString(tok) {
				return nil, &XPathError{
					Code:    "FORG0001",
					Message: fmt.Sprintf("invalid token %q in %s", tok, typeName),
				}
			}
		}
		return SingleAtomic(AtomicValue{TypeName: typeName, Value: strings.Join(tokens, " ")}), nil
	}
}

var (
	reGDay       = regexp.MustCompile(`^---(\d{2})(Z|[+-]\d{2}:\d{2})?$`)
	reGMonth     = regexp.MustCompile(`^--(\d{2})(Z|[+-]\d{2}:\d{2})?$`)
	reGMonthDay  = regexp.MustCompile(`^--(\d{2})-(\d{2})(Z|[+-]\d{2}:\d{2})?$`)
	reGYear      = regexp.MustCompile(`^-?(\d{4,})(Z|[+-]\d{2}:\d{2})?$`)
	reGYearMonth = regexp.MustCompile(`^-?(\d{4,})-(\d{2})(Z|[+-]\d{2}:\d{2})?$`)
)

// makeXSGregorian returns a constructor for xs:gDay, xs:gMonth, xs:gMonthDay, xs:gYear, xs:gYearMonth.
func makeXSGregorian(typeName string, re *regexp.Regexp) func(context.Context, []Sequence) (Sequence, error) {
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		s, err := atomicToString(a)
		if err != nil {
			return nil, err
		}
		s = strings.TrimSpace(s)
		if !re.MatchString(s) || !validateGregorianValue(typeName, s) {
			return nil, &XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("cannot cast %q to %s", s, typeName),
			}
		}
		return SingleAtomic(AtomicValue{TypeName: typeName, Value: s}), nil
	}
}

// validateGregorianValue performs additional validation beyond regex matching.
func validateGregorianValue(typeName, s string) bool {
	switch typeName {
	case TypeGYear:
		// xs:gYear: reject 0000 and -0000 (year zero is invalid)
		y := extractYearDigits(s)
		if isAllZero(y) {
			return false
		}
		if len(y) > 9 {
			return false
		}
	case TypeGYearMonth:
		// xs:gYearMonth: reject 0000 but allow -0000 (represents 1 BCE)
		neg := strings.HasPrefix(s, "-")
		y := extractYearDigits(s)
		if !neg && isAllZero(y) {
			return false
		}
		if len(y) > 9 {
			return false
		}
	}
	return true
}

func extractYearDigits(s string) string {
	y := strings.TrimPrefix(s, "-")
	for i, c := range y {
		if c == '-' || c == 'Z' || c == '+' {
			return y[:i]
		}
	}
	return y
}

func isAllZero(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

var reDateTimeStampTZ = regexp.MustCompile(`[+-]\d{2}:\d{2}$`)

func makeXSDateTimeStamp() func(context.Context, []Sequence) (Sequence, error) {
	return func(_ context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		// Cast to dateTime first
		dt, err := CastAtomic(a, TypeDateTime)
		if err != nil {
			return nil, err
		}
		// dateTimeStamp requires a timezone — check the string representation
		s, _ := atomicToString(a)
		if !strings.HasSuffix(s, "Z") && !reDateTimeStampTZ.MatchString(s) {
			return nil, &XPathError{
				Code:    "FORG0001",
				Message: "xs:dateTimeStamp requires a timezone",
			}
		}
		return SingleAtomic(AtomicValue{TypeName: TypeDateTimeStamp, Value: dt.Value}), nil
	}
}

func fnXSError(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{
		Code:    "FORG0001",
		Message: "xs:error always fails",
	}
}
