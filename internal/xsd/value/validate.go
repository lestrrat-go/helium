package value

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// ValidateBuiltin validates a value against a builtin XSD type's lexical space.
func ValidateBuiltin(value, builtinLocal string) error {
	switch builtinLocal {
	case "decimal":
		return validateDecimal(value)
	case "integer":
		return validateInteger(value)
	case "nonPositiveInteger", "negativeInteger",
		"long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger":
		return validateIntegerWithRange(value, builtinLocal)
	case "hexBinary":
		return validateHexBinary(value)
	case "date":
		return validateDate(value)
	case "boolean":
		return validateBoolean(value)
	case "language":
		return validateLanguage(value)
	case "float", "double":
		return validateFloat(value)
	case "dateTime":
		return validateDateTime(value)
	case "time":
		return validateTime(value)
	case "duration":
		return validateDuration(value)
	case "gYear":
		return validateGYear(value)
	case "gYearMonth":
		return validateGYearMonth(value)
	case "gMonth":
		return validateGMonth(value)
	case "gDay":
		return validateGDay(value)
	case "gMonthDay":
		return validateGMonthDay(value)
	case "NCName", "ID", "IDREF", "ENTITY":
		return validateNCName(value)
	case "Name":
		return validateName(value)
	case "NMTOKEN":
		return validateNMTOKEN(value)
	case "QName", "NOTATION":
		return validateQName(value)
	case "base64Binary":
		return validateBase64Binary(value)
	case "normalizedString":
		return validateNormalizedString(value)
	case "token":
		return validateToken(value)
	case "IDREFS", "ENTITIES":
		return validateSpaceSeparatedList(value, validateNCName)
	case "NMTOKENS":
		return validateSpaceSeparatedList(value, validateNMTOKEN)
	case "anyURI":
		return nil
	default:
		return nil
	}
}

// decimalRegex matches the lexical space of xs:decimal.
// Pattern: optional sign, then digits with optional decimal point.
var decimalRegex = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)

func validateDecimal(value string) error {
	if !decimalRegex.MatchString(value) {
		return fmt.Errorf("invalid decimal")
	}
	return nil
}

// integerRegex matches the lexical space of xs:integer.
var integerRegex = regexp.MustCompile(`^[+-]?\d+$`)

func validateInteger(value string) error {
	if !integerRegex.MatchString(value) {
		return fmt.Errorf("invalid integer")
	}
	return nil
}

// integerRange defines inclusive min/max bounds for integer subtypes.
type integerRange struct {
	min *big.Int // nil means no lower bound
	max *big.Int // nil means no upper bound
}

var integerRanges = map[string]integerRange{
	"byte":               {big.NewInt(-128), big.NewInt(127)},
	"short":              {big.NewInt(-32768), big.NewInt(32767)},
	"int":                {big.NewInt(-2147483648), big.NewInt(2147483647)},
	"long":               {newBigInt("-9223372036854775808"), newBigInt("9223372036854775807")},
	"unsignedByte":       {big.NewInt(0), big.NewInt(255)},
	"unsignedShort":      {big.NewInt(0), big.NewInt(65535)},
	"unsignedInt":        {big.NewInt(0), newBigInt("4294967295")},
	"unsignedLong":       {big.NewInt(0), newBigInt("18446744073709551615")},
	"nonNegativeInteger": {big.NewInt(0), nil},
	"nonPositiveInteger": {nil, big.NewInt(0)},
	"positiveInteger":    {big.NewInt(1), nil},
	"negativeInteger":    {nil, big.NewInt(-1)},
}

func newBigInt(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 10)
	return n
}

func validateIntegerWithRange(value, typeName string) error {
	if err := validateInteger(value); err != nil {
		return err
	}
	r, ok := integerRanges[typeName]
	if !ok {
		return nil
	}
	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return fmt.Errorf("invalid integer")
	}
	if r.min != nil && n.Cmp(r.min) < 0 {
		return fmt.Errorf("value %s is out of range for %s", value, typeName)
	}
	if r.max != nil && n.Cmp(r.max) > 0 {
		return fmt.Errorf("value %s is out of range for %s", value, typeName)
	}
	return nil
}

// hexBinaryRegex matches the lexical space of xs:hexBinary.
// Must be even number of hex digits, or empty.
var hexBinaryRegex = regexp.MustCompile(`^([0-9a-fA-F]{2})*$`)

func validateHexBinary(value string) error {
	if !hexBinaryRegex.MatchString(value) {
		return fmt.Errorf("invalid hexBinary")
	}
	return nil
}

// tzSuffix is the timezone suffix pattern shared by date/time types.
const tzSuffix = `([Zz]|[+-]\d{2}:\d{2})?`

// dateRegex is a basic match for xs:date: YYYY-MM-DD with optional timezone.
var dateRegex = regexp.MustCompile(`^-?\d{4,}-\d{2}-\d{2}` + tzSuffix + `$`)

func validateDate(value string) error {
	if !dateRegex.MatchString(value) {
		return fmt.Errorf("invalid date")
	}
	return validateDateComponents(value)
}

// languageRegex matches the lexical space of xs:language (RFC 3066).
var languageRegex = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

func validateLanguage(value string) error {
	if !languageRegex.MatchString(value) {
		return fmt.Errorf("invalid language")
	}
	return nil
}

func validateBoolean(value string) error {
	switch value {
	case "true", "false", "1", "0":
		return nil
	}
	return fmt.Errorf("invalid boolean")
}

// floatRegex matches xs:float and xs:double.
var floatRegex = regexp.MustCompile(`^[+-]?((\d+\.?\d*|\.\d+)([eE][+-]?\d+)?|INF|NaN)$`)

func validateFloat(value string) error {
	if !floatRegex.MatchString(value) {
		return fmt.Errorf("invalid float")
	}
	return nil
}

// dateTimeRegex matches xs:dateTime.
var dateTimeRegex = regexp.MustCompile(`^-?\d{4,}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?` + tzSuffix + `$`)

func validateDateTime(value string) error {
	if !dateTimeRegex.MatchString(value) {
		return fmt.Errorf("invalid dateTime")
	}
	return validateDateComponents(value)
}

// timeRegex matches xs:time.
var timeRegex = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?` + tzSuffix + `$`)

func validateTime(value string) error {
	if !timeRegex.MatchString(value) {
		return fmt.Errorf("invalid time")
	}
	return nil
}

// validateDateComponents parses YYYY-MM-DD from a date/dateTime string and
// checks month (1-12) and day (1-maxDay) ranges. The value must already have
// passed the regex check.
func validateDateComponents(value string) error {
	s := value
	// Skip optional leading '-' for negative years.
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	// Find year-month-day fields: YYYY-MM-DD...
	// Year is variable-length (4+ digits), so find second '-' after first.
	_, rest, found := strings.Cut(s, "-")
	if !found {
		return fmt.Errorf("invalid date")
	}
	monthStr, dayStr, found := strings.Cut(rest, "-")
	if !found {
		return fmt.Errorf("invalid date")
	}
	// dayStr may have trailing 'T...' or timezone; take first 2 chars.
	if len(dayStr) < 2 {
		return fmt.Errorf("invalid date")
	}
	dayStr = dayStr[:2]

	var month, day int
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
		return fmt.Errorf("invalid date")
	}
	if _, err := fmt.Sscanf(dayStr, "%d", &day); err != nil {
		return fmt.Errorf("invalid date")
	}

	if month < 1 || month > 12 {
		return fmt.Errorf("invalid date: month %d out of range", month)
	}
	// Maximum days per month (February gets 29 for simplicity; the XML Schema
	// spec allows Feb 29 in every year, or rather doesn't restrict by year for
	// the xs:date datatype — but it does require valid Gregorian days).
	maxDays := [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day < 1 || day > maxDays[month] {
		return fmt.Errorf("invalid date: day %d out of range for month %d", day, month)
	}
	return nil
}

// durationRegex matches xs:duration.
var durationRegex = regexp.MustCompile(`^-?P(\d+Y)?(\d+M)?(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`)

func validateDuration(value string) error {
	if !durationRegex.MatchString(value) {
		return fmt.Errorf("invalid duration")
	}
	// At least one component must be present after P.
	s := value
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	s = s[1:] // remove 'P'
	if s == "" || s == "T" {
		return fmt.Errorf("invalid duration")
	}
	return nil
}

// gYearRegex matches xs:gYear.
var gYearRegex = regexp.MustCompile(`^-?\d{4,}` + tzSuffix + `$`)

func validateGYear(value string) error {
	if !gYearRegex.MatchString(value) {
		return fmt.Errorf("invalid gYear")
	}
	return nil
}

// gYearMonthRegex matches xs:gYearMonth.
var gYearMonthRegex = regexp.MustCompile(`^-?\d{4,}-\d{2}` + tzSuffix + `$`)

func validateGYearMonth(value string) error {
	if !gYearMonthRegex.MatchString(value) {
		return fmt.Errorf("invalid gYearMonth")
	}
	return nil
}

// gMonthRegex matches xs:gMonth.
var gMonthRegex = regexp.MustCompile(`^--\d{2}` + tzSuffix + `$`)

func validateGMonth(value string) error {
	if !gMonthRegex.MatchString(value) {
		return fmt.Errorf("invalid gMonth")
	}
	return nil
}

// gDayRegex matches xs:gDay.
var gDayRegex = regexp.MustCompile(`^---\d{2}` + tzSuffix + `$`)

func validateGDay(value string) error {
	if !gDayRegex.MatchString(value) {
		return fmt.Errorf("invalid gDay")
	}
	return nil
}

// gMonthDayRegex matches xs:gMonthDay.
var gMonthDayRegex = regexp.MustCompile(`^--\d{2}-\d{2}` + tzSuffix + `$`)

func validateGMonthDay(value string) error {
	if !gMonthDayRegex.MatchString(value) {
		return fmt.Errorf("invalid gMonthDay")
	}
	return nil
}

// ncNameRegex matches XML NCName: letter or underscore, then name chars (no colon).
var ncNameRegex = regexp.MustCompile(`^[a-zA-Z_][\w.-]*$`)

func validateNCName(value string) error {
	if !ncNameRegex.MatchString(value) {
		return fmt.Errorf("invalid NCName")
	}
	return nil
}

// nameRegex matches XML Name: like NCName but allows colon.
var nameRegex = regexp.MustCompile(`^[a-zA-Z_:][\w.:-]*$`)

func validateName(value string) error {
	if !nameRegex.MatchString(value) {
		return fmt.Errorf("invalid Name")
	}
	return nil
}

// nmtokenRegex matches XML NMTOKEN: one or more name characters.
var nmtokenRegex = regexp.MustCompile(`^[\w.:-]+$`)

func validateNMTOKEN(value string) error {
	if !nmtokenRegex.MatchString(value) {
		return fmt.Errorf("invalid NMTOKEN")
	}
	return nil
}

func validateNormalizedString(value string) error {
	if strings.ContainsAny(value, "\t\n\r") {
		return fmt.Errorf("invalid normalizedString")
	}
	return nil
}

func validateToken(value string) error {
	if strings.ContainsAny(value, "\t\n\r") {
		return fmt.Errorf("invalid token")
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("invalid token")
	}
	if strings.Contains(value, "  ") {
		return fmt.Errorf("invalid token")
	}
	return nil
}

func validateQName(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 1 {
		return validateNCName(value)
	}
	if err := validateNCName(parts[0]); err != nil {
		return fmt.Errorf("invalid QName")
	}
	if err := validateNCName(parts[1]); err != nil {
		return fmt.Errorf("invalid QName")
	}
	return nil
}

// base64Regex matches the lexical space of xs:base64Binary.
var base64Regex = regexp.MustCompile(`^[A-Za-z0-9+/=\s]*$`)

func validateBase64Binary(value string) error {
	if !base64Regex.MatchString(value) {
		return fmt.Errorf("invalid base64Binary")
	}
	return nil
}

func validateSpaceSeparatedList(value string, validateItem func(string) error) error {
	if value == "" {
		return fmt.Errorf("empty list")
	}
	items := strings.Fields(value)
	if len(items) == 0 {
		return fmt.Errorf("empty list")
	}
	for _, item := range items {
		if err := validateItem(item); err != nil {
			return err
		}
	}
	return nil
}
