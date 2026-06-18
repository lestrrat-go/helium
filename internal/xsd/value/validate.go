package value

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
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
	if tz, ok := splitTimezone(value); ok {
		if err := validateTimezone(tz); err != nil {
			return err
		}
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
	if err := validateDateComponents(value); err != nil {
		return err
	}
	// The time portion follows the 'T' separator.
	_, timePart, found := strings.Cut(value, "T")
	if !found {
		return fmt.Errorf("invalid dateTime")
	}
	return validateTimeComponents(timePart)
}

// timeRegex matches xs:time.
var timeRegex = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?` + tzSuffix + `$`)

func validateTime(value string) error {
	if !timeRegex.MatchString(value) {
		return fmt.Errorf("invalid time")
	}
	return validateTimeComponents(value)
}

// validateTimeComponents enforces the value-space ranges of the time portion
// of xs:time / xs:dateTime: hour 00-23 (or exactly 24:00:00 with zero minutes,
// seconds and fractional seconds), minute 00-59, second 00-59 (XSD does not
// allow leap seconds), plus the timezone offset (±14:00, with minutes 00-59).
// The input must already have passed the relevant lexical regex.
func validateTimeComponents(value string) error {
	// Split off the timezone designator, if any.
	timeOnly := value
	if tz, ok := splitTimezone(value); ok {
		timeOnly = value[:len(value)-len(tz)]
		if err := validateTimezone(tz); err != nil {
			return err
		}
	}

	// timeOnly is HH:MM:SS or HH:MM:SS.fff (lexical shape guaranteed by regex).
	hourStr, rest, found := strings.Cut(timeOnly, ":")
	if !found {
		return fmt.Errorf("invalid time")
	}
	minStr, secStr, found := strings.Cut(rest, ":")
	if !found {
		return fmt.Errorf("invalid time")
	}

	var hour, minute, sec int
	if _, err := fmt.Sscanf(hourStr, "%d", &hour); err != nil {
		return fmt.Errorf("invalid time")
	}
	if _, err := fmt.Sscanf(minStr, "%d", &minute); err != nil {
		return fmt.Errorf("invalid time")
	}
	intSecStr, fracStr, _ := strings.Cut(secStr, ".")
	if _, err := fmt.Sscanf(intSecStr, "%d", &sec); err != nil {
		return fmt.Errorf("invalid time")
	}

	if minute < 0 || minute > 59 {
		return fmt.Errorf("invalid time: minute %d out of range", minute)
	}
	if sec < 0 || sec > 59 {
		return fmt.Errorf("invalid time: second %d out of range", sec)
	}
	if hour < 0 || hour > 24 {
		return fmt.Errorf("invalid time: hour %d out of range", hour)
	}
	// 24:00:00 (with zero minutes, seconds, and fractional seconds) is the only
	// permitted use of hour 24; it denotes midnight at the end of the day.
	if hour == 24 && (minute != 0 || sec != 0 || !isZeroFraction(fracStr)) {
		return fmt.Errorf("invalid time: 24:00:00 is the only allowed value with hour 24")
	}
	return nil
}

// isZeroFraction reports whether a fractional-seconds string (the part after
// the '.', without the dot) represents zero. An empty string means no fraction
// was present, which is also zero.
func isZeroFraction(frac string) bool {
	return strings.Trim(frac, "0") == ""
}

// splitTimezone returns the trailing timezone designator (e.g. "Z", "+09:00",
// "-05:00") and true if present, otherwise "" and false.
func splitTimezone(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	last := value[len(value)-1]
	if last == 'Z' || last == 'z' {
		return value[len(value)-1:], true
	}
	// A numeric offset is "±HH:MM" — 6 trailing chars whose 6th-from-last is a
	// sign and whose 3rd-from-last is the ':' separator. The colon check is
	// what distinguishes a real offset from the '-' that separates date fields
	// (e.g. the "-05-20" tail of "1996-05-20").
	if len(value) >= 6 {
		if c := value[len(value)-6]; (c == '+' || c == '-') && value[len(value)-3] == ':' {
			return value[len(value)-6:], true
		}
	}
	return "", false
}

// validateTimezone enforces XSD timezone offset ranges: total offset within
// ±14:00, minutes 00-59, and when hours are 14 the minutes must be 00.
func validateTimezone(tz string) error {
	if tz == "Z" || tz == "z" {
		return nil
	}
	// tz is "±HH:MM" (lexical shape guaranteed by regex).
	hhmm := tz[1:]
	hourStr, minStr, found := strings.Cut(hhmm, ":")
	if !found {
		return fmt.Errorf("invalid timezone")
	}
	var hour, minute int
	if _, err := fmt.Sscanf(hourStr, "%d", &hour); err != nil {
		return fmt.Errorf("invalid timezone")
	}
	if _, err := fmt.Sscanf(minStr, "%d", &minute); err != nil {
		return fmt.Errorf("invalid timezone")
	}
	if minute < 0 || minute > 59 {
		return fmt.Errorf("invalid timezone: minute %d out of range", minute)
	}
	if hour < 0 || hour > 14 {
		return fmt.Errorf("invalid timezone: hour %d out of range", hour)
	}
	if hour == 14 && minute != 0 {
		return fmt.Errorf("invalid timezone: offset exceeds 14:00")
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
	yearStr, rest, found := strings.Cut(s, "-")
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

	var year, month, day int
	if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
		return fmt.Errorf("invalid date")
	}
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
		return fmt.Errorf("invalid date")
	}
	if _, err := fmt.Sscanf(dayStr, "%d", &day); err != nil {
		return fmt.Errorf("invalid date")
	}

	if month < 1 || month > 12 {
		return fmt.Errorf("invalid date: month %d out of range", month)
	}
	// Maximum days per month. February's length depends on whether the year is
	// a leap year in the proleptic Gregorian calendar.
	maxDays := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month == 2 && isLeapYear(year) {
		maxDays[2] = 29
	}
	if day < 1 || day > maxDays[month] {
		return fmt.Errorf("invalid date: day %d out of range for month %d", day, month)
	}
	return nil
}

// isLeapYear reports whether the given (non-negative) Gregorian year is a leap
// year. The year value here is the absolute magnitude parsed from the lexical
// form; leap-year status is symmetric for the BCE side of the proleptic
// Gregorian calendar as used by XSD.
func isLeapYear(year int) bool {
	if year%400 == 0 {
		return true
	}
	if year%100 == 0 {
		return false
	}
	return year%4 == 0
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
	if _, err := stripAndCheckTimezone(value); err != nil {
		return err
	}
	return nil
}

// stripAndCheckTimezone removes any trailing timezone designator from value,
// validating its range, and returns the remaining (timezone-free) body. The
// input is assumed to have already passed its lexical regex.
func stripAndCheckTimezone(value string) (string, error) {
	tz, ok := splitTimezone(value)
	if !ok {
		return value, nil
	}
	if err := validateTimezone(tz); err != nil {
		return "", err
	}
	return value[:len(value)-len(tz)], nil
}

// gMonthRange checks that a two-digit month string is within 1-12.
func gMonthRange(s, typeName string) error {
	var month int
	if _, err := fmt.Sscanf(s, "%d", &month); err != nil {
		return fmt.Errorf("invalid %s", typeName)
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("invalid %s: month %d out of range", typeName, month)
	}
	return nil
}

// gDayRange checks that a two-digit day string is within 1-31.
func gDayRange(s, typeName string) error {
	var day int
	if _, err := fmt.Sscanf(s, "%d", &day); err != nil {
		return fmt.Errorf("invalid %s", typeName)
	}
	if day < 1 || day > 31 {
		return fmt.Errorf("invalid %s: day %d out of range", typeName, day)
	}
	return nil
}

// gYearMonthRegex matches xs:gYearMonth.
var gYearMonthRegex = regexp.MustCompile(`^-?\d{4,}-\d{2}` + tzSuffix + `$`)

func validateGYearMonth(value string) error {
	if !gYearMonthRegex.MatchString(value) {
		return fmt.Errorf("invalid gYearMonth")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	// body is "[-]YYYY...-MM": the month is the last two characters.
	return gMonthRange(body[len(body)-2:], "gYearMonth")
}

// gMonthRegex matches xs:gMonth.
var gMonthRegex = regexp.MustCompile(`^--\d{2}` + tzSuffix + `$`)

func validateGMonth(value string) error {
	if !gMonthRegex.MatchString(value) {
		return fmt.Errorf("invalid gMonth")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	// body is "--MM".
	return gMonthRange(body[2:4], "gMonth")
}

// gDayRegex matches xs:gDay.
var gDayRegex = regexp.MustCompile(`^---\d{2}` + tzSuffix + `$`)

func validateGDay(value string) error {
	if !gDayRegex.MatchString(value) {
		return fmt.Errorf("invalid gDay")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	// body is "---DD".
	return gDayRange(body[3:5], "gDay")
}

// gMonthDayRegex matches xs:gMonthDay.
var gMonthDayRegex = regexp.MustCompile(`^--\d{2}-\d{2}` + tzSuffix + `$`)

func validateGMonthDay(value string) error {
	if !gMonthDayRegex.MatchString(value) {
		return fmt.Errorf("invalid gMonthDay")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	// body is "--MM-DD".
	if err := gMonthRange(body[2:4], "gMonthDay"); err != nil {
		return err
	}
	if err := gDayRange(body[5:7], "gMonthDay"); err != nil {
		return err
	}
	// Reject month/day combinations that never occur (e.g. --02-30, --04-31).
	var month, day int
	if _, err := fmt.Sscanf(body[2:4], "%d", &month); err != nil {
		return fmt.Errorf("invalid gMonthDay")
	}
	if _, err := fmt.Sscanf(body[5:7], "%d", &day); err != nil {
		return fmt.Errorf("invalid gMonthDay")
	}
	// gMonthDay is year-agnostic, so February permits its maximal length (29).
	maxDays := [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day > maxDays[month] {
		return fmt.Errorf("invalid gMonthDay: day %d out of range for month %d", day, month)
	}
	return nil
}

// isNameStartChar reports whether r is a valid XML 1.0 NameStartChar.
// (https://www.w3.org/TR/xml/#NT-NameStartChar) The colon is handled by the
// caller, since NCName forbids it while Name allows it.
func isNameStartChar(r rune) bool {
	switch {
	case r == '_':
		return true
	case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		return true
	case r >= 0xC0 && r <= 0xD6, r >= 0xD8 && r <= 0xF6, r >= 0xF8 && r <= 0x2FF:
		return true
	case r >= 0x370 && r <= 0x37D, r >= 0x37F && r <= 0x1FFF:
		return true
	case r >= 0x200C && r <= 0x200D, r >= 0x2070 && r <= 0x218F:
		return true
	case r >= 0x2C00 && r <= 0x2FEF, r >= 0x3001 && r <= 0xD7FF:
		return true
	case r >= 0xF900 && r <= 0xFDCF, r >= 0xFDF0 && r <= 0xFFFD:
		return true
	case r >= 0x10000 && r <= 0xEFFFF:
		return true
	}
	return false
}

// isNameChar reports whether r is a valid XML 1.0 NameChar (excluding the
// colon, which the caller decides on per type).
// (https://www.w3.org/TR/xml/#NT-NameChar)
func isNameChar(r rune) bool {
	switch {
	case isNameStartChar(r):
		return true
	case r == '-', r == '.':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == 0xB7:
		return true
	case r >= 0x0300 && r <= 0x036F:
		return true
	case r >= 0x203F && r <= 0x2040:
		return true
	}
	return false
}

// isXMLName reports whether value is a valid XML Name (or NCName when
// allowColon is false), per the XML 1.0 NameStartChar/NameChar productions.
func isXMLName(value string, allowColon bool) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		colon := r == ':'
		if i == 0 {
			if (colon && allowColon) || isNameStartChar(r) {
				continue
			}
			return false
		}
		if (colon && allowColon) || isNameChar(r) {
			continue
		}
		return false
	}
	return true
}

func validateNCName(value string) error {
	if !isXMLName(value, false) {
		return fmt.Errorf("invalid NCName")
	}
	return nil
}

func validateName(value string) error {
	if !isXMLName(value, true) {
		return fmt.Errorf("invalid Name")
	}
	return nil
}

// isNMTOKEN reports whether value is a valid XML Nmtoken: one or more NameChar
// (colon allowed).
func isNMTOKEN(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == ':' || isNameChar(r) {
			continue
		}
		return false
	}
	return true
}

func validateNMTOKEN(value string) error {
	if !isNMTOKEN(value) {
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

// base64Regex restricts the character repertoire of xs:base64Binary; the
// grammar (correct quad/padding structure) is enforced by attempting a decode.
var base64Regex = regexp.MustCompile(`^[A-Za-z0-9+/=\s]*$`)

func validateBase64Binary(value string) error {
	if !base64Regex.MatchString(value) {
		return fmt.Errorf("invalid base64Binary")
	}
	// xs:base64Binary has whiteSpace=collapse and permits whitespace between
	// characters; strip all whitespace then enforce the grammar via a strict
	// (length/padding-aware) decode. This rejects malformed input such as
	// "====", a lone "A", or "AAA" that the character regex alone would pass.
	var b strings.Builder
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	if _, err := base64.StdEncoding.DecodeString(b.String()); err != nil {
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
