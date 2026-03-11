package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
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
	} {
		registerNS(NSXS, entry.name, 1, 1, makeXSConstructor(entry.targetType))
	}

	// xs:QName constructor needs namespace context to resolve prefixes
	registerNS(NSXS, "QName", 1, 1, makeXSQNameConstructor())

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

	// xs:unsignedLong needs big.Int max since MaxUint64 exceeds int64
	registerNS(NSXS, "unsignedLong", 1, 1, makeXSIntegerRangeBig(TypeUnsignedLong, big.NewInt(0), new(big.Int).SetUint64(math.MaxUint64)))

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

	// List types — split by whitespace, produce sequence of item type values
	registerNS(NSXS, "NMTOKENS", 1, 1, makeXSTokenList(TypeNMTOKEN, reNMTOKEN))
	registerNS(NSXS, "IDREFS", 1, 1, makeXSTokenList(TypeIDREF, reNCName))
	registerNS(NSXS, "ENTITIES", 1, 1, makeXSTokenList(TypeENTITY, reNCName))

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

// makeXSQNameConstructor returns a constructor for xs:QName that resolves
// prefixes using the namespace context from the evaluator.
func makeXSQNameConstructor() func(context.Context, []Sequence) (Sequence, error) {
	return func(ctx context.Context, args []Sequence) (Sequence, error) {
		if len(args[0]) == 0 {
			return nil, nil
		}
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		// If already a QName, return as-is
		if a.TypeName == TypeQName {
			return SingleAtomic(a), nil
		}
		// Only string and untypedAtomic can be cast to QName
		if a.TypeName != TypeString && a.TypeName != TypeUntypedAtomic {
			return nil, &XPathError{
				Code:    errCodeXPTY0004,
				Message: fmt.Sprintf("cannot cast %s to %s", a.TypeName, TypeQName),
			}
		}
		s, err := atomicToString(a)
		if err != nil {
			return nil, err
		}
		s = strings.TrimSpace(s)

		prefix := ""
		local := s
		if idx := strings.IndexByte(s, ':'); idx >= 0 {
			prefix = s[:idx]
			local = s[idx+1:]
		}

		uri := ""
		if prefix != "" {
			// Look up the prefix in the evaluation context's namespace bindings
			resolved := false
			ec := getFnContext(ctx)
			if ec != nil && ec.namespaces != nil {
				if ns, ok := ec.namespaces[prefix]; ok {
					uri = ns
					resolved = true
				}
			}
			if !resolved {
				// Fall back to default prefix mappings
				if ns, ok := defaultPrefixNS[prefix]; ok {
					uri = ns
					resolved = true
				}
			}
			if !resolved {
				return nil, &XPathError{
					Code:    "FONS0004",
					Message: fmt.Sprintf("no namespace binding for prefix %q", prefix),
				}
			}
		} else {
			// No prefix: check default namespace in context
			ec := getFnContext(ctx)
			if ec != nil && ec.namespaces != nil {
				if ns, ok := ec.namespaces[""]; ok {
					uri = ns
				}
			}
		}

		return SingleAtomic(AtomicValue{
			TypeName: TypeQName,
			Value:    QNameValue{Prefix: prefix, Local: local, URI: uri},
		}), nil
	}
}

// makeXSIntegerRange returns a constructor for a derived integer type with range validation.
func makeXSIntegerRange(typeName string, minVal, maxVal int64) func(context.Context, []Sequence) (Sequence, error) {
	return makeXSIntegerRangeBig(typeName, big.NewInt(minVal), big.NewInt(maxVal))
}

// makeXSIntegerRangeBig returns a constructor for a derived integer type with big.Int range validation.
func makeXSIntegerRangeBig(typeName string, minBig, maxBig *big.Int) func(context.Context, []Sequence) (Sequence, error) {
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
		if typeName == TypeNormalizedString {
			s = normalizeWhitespace(s)
		} else {
			s = collapseWhitespace(s)
		}
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
func makeXSTokenList(itemType string, tokenRe *regexp.Regexp) func(context.Context, []Sequence) (Sequence, error) {
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
				Message: fmt.Sprintf("cannot cast empty string to %s", itemType),
			}
		}
		tokens := strings.Fields(s)
		result := make(Sequence, len(tokens))
		for i, tok := range tokens {
			if !tokenRe.MatchString(tok) {
				return nil, &XPathError{
					Code:    "FORG0001",
					Message: fmt.Sprintf("invalid token %q in %s", tok, itemType),
				}
			}
			result[i] = AtomicValue{TypeName: itemType, Value: tok}
		}
		return result, nil
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
	// Validate timezone if present
	if validateTimezoneInString(s) != nil {
		return false
	}
	switch typeName {
	case TypeGDay:
		// ---DD format: day must be 01-31
		day := extractGDayValue(s)
		return day >= 1 && day <= 31
	case TypeGMonth:
		// --MM format: month must be 01-12
		month := extractGMonthValue(s)
		return month >= 1 && month <= 12
	case TypeGMonthDay:
		// --MM-DD format: month 01-12, day valid for month
		month, day := extractGMonthDayValues(s)
		if month < 1 || month > 12 {
			return false
		}
		// Use a leap year for gMonthDay since no year is specified;
		// February allows up to 29 because gMonthDay must accommodate leap years.
		maxDay := daysInMonth(4, month) // year 4 is a leap year
		return day >= 1 && day <= maxDay
	case TypeGYear:
		// XSD 1.1: year 0000 is valid
		y := extractYearDigits(s)
		if len(y) > 9 {
			return false
		}
	case TypeGYearMonth:
		// XSD 1.1: year 0000 is valid, validate month 01-12
		y := extractYearDigits(s)
		if len(y) > 9 {
			return false
		}
		month := extractGYearMonthMonth(s)
		return month >= 1 && month <= 12
	}
	return true
}

// extractGDayValue extracts the day from a ---DD[tz] string.
func extractGDayValue(s string) int {
	// format: ---DD...
	if len(s) < 5 {
		return 0
	}
	d, err := strconv.Atoi(s[3:5])
	if err != nil {
		return 0
	}
	return d
}

// extractGMonthValue extracts the month from a --MM[tz] string.
func extractGMonthValue(s string) int {
	// format: --MM...
	if len(s) < 4 {
		return 0
	}
	m, err := strconv.Atoi(s[2:4])
	if err != nil {
		return 0
	}
	return m
}

// extractGMonthDayValues extracts month and day from a --MM-DD[tz] string.
func extractGMonthDayValues(s string) (int, int) {
	// format: --MM-DD...
	if len(s) < 7 {
		return 0, 0
	}
	m, err := strconv.Atoi(s[2:4])
	if err != nil {
		return 0, 0
	}
	d, err := strconv.Atoi(s[5:7])
	if err != nil {
		return 0, 0
	}
	return m, d
}

// extractGYearMonthMonth extracts the month from a [-]YYYY-MM[tz] string.
func extractGYearMonthMonth(s string) int {
	// Find the last '-' before any timezone
	// The month is right after the year portion: skip optional leading '-', then digits, then '-'
	work := strings.TrimPrefix(s, "-")
	// Skip year digits
	i := 0
	for i < len(work) && work[i] >= '0' && work[i] <= '9' {
		i++
	}
	if i >= len(work) || work[i] != '-' {
		return 0
	}
	// work[i] == '-', month starts at i+1
	if i+3 > len(work) {
		return 0
	}
	m, err := strconv.Atoi(work[i+1 : i+3])
	if err != nil {
		return 0
	}
	return m
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

func fnXSError(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	return nil, &XPathError{
		Code:    "FORG0001",
		Message: "xs:error always fails",
	}
}
