package value

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// Version selects the XSD specification version for the version-sensitive
// lexical rules. The zero value is Version10 (strict XSD 1.0), so callers that
// have no version concept default to 1.0 behavior.
type Version int

const (
	// Version10 targets XML Schema 1.0 lexical rules: "+INF" is NOT a valid
	// xs:float/xs:double form and year "0000" is NOT a valid date year.
	Version10 Version = iota
	// Version11 targets XML Schema 1.1 lexical rules: "+INF" is a valid
	// xs:float/xs:double form and year "0000" is a valid date year.
	Version11
)

// ValidateBuiltin validates a value against a builtin XSD type's lexical space.
// version selects the 1.0-vs-1.1 lexical rules for the version-sensitive types
// (xs:float/xs:double "+INF", and year "0000" on the date types).
func ValidateBuiltin(value, builtinLocal string, version Version) error {
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
		return validateDate(value, version)
	case "boolean":
		return validateBoolean(value)
	case "language":
		return validateLanguage(value)
	case lexicon.TypeFloat, lexicon.TypeDouble:
		return validateFloat(value, version)
	case "dateTime":
		return validateDateTime(value, version)
	case lexicon.TypeDateTimeStamp:
		return validateDateTimeStamp(value, version)
	case "time":
		return validateTime(value)
	case "duration":
		return validateDuration(value)
	case lexicon.TypeDayTimeDuration:
		return validateDayTimeDuration(value)
	case lexicon.TypeYearMonthDuration:
		return validateYearMonthDuration(value)
	case lexicon.TypeAnyAtomicType:
		// xs:anyAtomicType is abstract: it has no direct lexical constraints (its
		// abstractness is enforced where types are used, not here).
		return nil
	case lexicon.TypeError:
		// xs:error has an empty value space: no literal is ever valid.
		return fmt.Errorf("xs:error has an empty value space")
	case "gYear":
		return validateGYear(value, version)
	case "gYearMonth":
		return validateGYearMonth(value, version)
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
		return validateAnyURI(value, version)
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
// XSD permits only the uppercase 'Z' designator, never lowercase 'z'.
const tzSuffix = `(Z|[+-]\d{2}:\d{2})?`

// yearFrag matches the calendar-year fragment used by xs:date, xs:dateTime,
// xs:gYear, and xs:gYearMonth. Expanded years have more than four digits and
// must not have leading zeroes; year zero itself is written as exactly 0000.
const yearFrag = `-?(?:\d{4}|[1-9]\d{4,})`

// dateRegex is a basic match for xs:date: YYYY-MM-DD with optional timezone.
var dateRegex = regexp.MustCompile(`^` + yearFrag + `-\d{2}-\d{2}` + tzSuffix + `$`)

func validateDate(value string, version Version) error {
	if !dateRegex.MatchString(value) {
		return fmt.Errorf("invalid date")
	}
	if tz, ok := splitTimezone(value); ok {
		if err := validateTimezone(tz); err != nil {
			return err
		}
	}
	return validateDateComponents(value, version)
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

// floatRegex matches xs:float and xs:double under XSD 1.1. The optional sign
// applies to the numeric forms and INF, but NaN must be bare: the valid special
// lexical forms are INF, +INF, -INF and NaN — +NaN and -NaN are not valid.
var floatRegex = regexp.MustCompile(`^([+-]?((\d+\.?\d*|\.\d+)([eE][+-]?\d+)?|INF)|NaN)$`)

// floatRegex10 matches xs:float and xs:double under XSD 1.0, which differs from
// 1.1 only in the infinity forms: 1.0 allows INF and -INF but NOT +INF. The
// sign on the numeric forms is unchanged; only the INF alternative drops the
// leading '+'.
var floatRegex10 = regexp.MustCompile(`^(([+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?)|(-?INF)|NaN)$`)

func validateFloat(value string, version Version) error {
	re := floatRegex
	if version == Version10 {
		re = floatRegex10
	}
	if !re.MatchString(value) {
		return fmt.Errorf("invalid float")
	}
	return nil
}

// dateTimeRegex matches xs:dateTime.
var dateTimeRegex = regexp.MustCompile(`^` + yearFrag + `-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?` + tzSuffix + `$`)

func validateDateTime(value string, version Version) error {
	if !dateTimeRegex.MatchString(value) {
		return fmt.Errorf("invalid dateTime")
	}
	if err := validateDateComponents(value, version); err != nil {
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
	if last == 'Z' {
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
	if tz == "Z" {
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
// yearForbiddenInXSD10 reports whether yearStr (the digits-only year field, sign
// already stripped) is the year "0000". XSD 1.0 has no year zero; XSD 1.1 adds
// it. yearStr is assumed to have passed its lexical regex (all digits).
func yearForbiddenInXSD10(yearStr string) bool {
	return strings.Trim(yearStr, "0") == ""
}

func validateDateComponents(value string, version Version) error {
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
	if version == Version10 && yearForbiddenInXSD10(yearStr) {
		return fmt.Errorf("invalid date: year 0000 is not valid in XSD 1.0")
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

	// The year is variable-length and may be an arbitrarily large expanded year
	// (e.g. "999999999999999999999999"), so do not parse it into a fixed-width
	// int; leap-year status is computed from the digit string directly.
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
	// Maximum days per month. February's length depends on whether the year is
	// a leap year in the proleptic Gregorian calendar.
	maxDays := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month == 2 && isLeapYearStr(yearStr) {
		maxDays[2] = 29
	}
	if day < 1 || day > maxDays[month] {
		return fmt.Errorf("invalid date: day %d out of range for month %d", day, month)
	}
	return nil
}

// isLeapYearStr reports whether the Gregorian year given as a (sign-stripped)
// decimal digit string is a leap year. The year may be an arbitrarily large
// expanded year, so leap-year status is computed via modulo arithmetic over the
// digit string rather than by parsing into a fixed-width int (which would
// overflow): a year is leap iff divisible by 4 and (not divisible by 100, or
// divisible by 400).
func isLeapYearStr(year string) bool {
	mod := func(s string, n int) int {
		v := 0
		for _, r := range s {
			v = (v*10 + int(r-'0')) % n
		}
		return v
	}
	if mod(year, 4) != 0 {
		return false
	}
	if mod(year, 100) != 0 {
		return true
	}
	return mod(year, 400) == 0
}

// durationRegex matches xs:duration. When the time designator 'T' is present it
// must be followed by at least one of H/M/S; a dangling 'T' (e.g. "P1YT") is
// rejected by requiring at least one time component within the T group.
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
	// A time designator 'T' must be followed by at least one H/M/S component.
	if _, after, ok := strings.Cut(s, "T"); ok {
		if !strings.ContainsAny(after, "HMS") {
			return fmt.Errorf("invalid duration")
		}
	}
	return nil
}

// dateTimeStampRegex matches xs:dateTimeStamp (XSD 1.1): an xs:dateTime whose
// explicitTimezone is "required", i.e. the timezone designator is mandatory
// (the tzSuffix group is not optional).
var dateTimeStampRegex = regexp.MustCompile(`^` + yearFrag + `-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)

// validateDateTimeStamp validates xs:dateTimeStamp: an xs:dateTime value that
// must carry a timezone. Year-0000 gating follows the dateTime rules.
func validateDateTimeStamp(value string, version Version) error {
	if !dateTimeStampRegex.MatchString(value) {
		return fmt.Errorf("invalid dateTimeStamp")
	}
	return validateDateTime(value, version)
}

// dayTimeDurationRegex matches xs:dayTimeDuration (XSD 1.1): a duration with
// only day and time components (no year or month). At least one component must
// be present.
var dayTimeDurationRegex = regexp.MustCompile(`^-?P(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`)

func validateDayTimeDuration(value string) error {
	if !dayTimeDurationRegex.MatchString(value) {
		return fmt.Errorf("invalid dayTimeDuration")
	}
	// Reuse the shared duration body checks (at least one component, no dangling
	// 'T'); they additionally reject "P"/"PT".
	return validateDuration(value)
}

// yearMonthDurationRegex matches xs:yearMonthDuration (XSD 1.1): a duration with
// only year and month components (no day or time). At least one component must
// be present.
var yearMonthDurationRegex = regexp.MustCompile(`^-?P(\d+Y)?(\d+M)?$`)

func validateYearMonthDuration(value string) error {
	if !yearMonthDurationRegex.MatchString(value) {
		return fmt.Errorf("invalid yearMonthDuration")
	}
	// "P" with no component is invalid.
	if value == "P" || value == "-P" {
		return fmt.Errorf("invalid yearMonthDuration")
	}
	return validateDuration(value)
}

// gYearRegex matches xs:gYear.
var gYearRegex = regexp.MustCompile(`^` + yearFrag + tzSuffix + `$`)

func validateGYear(value string, version Version) error {
	if !gYearRegex.MatchString(value) {
		return fmt.Errorf("invalid gYear")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	if version == Version10 && yearForbiddenInXSD10(strings.TrimPrefix(body, "-")) {
		return fmt.Errorf("invalid gYear: year 0000 is not valid in XSD 1.0")
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
var gYearMonthRegex = regexp.MustCompile(`^` + yearFrag + `-\d{2}` + tzSuffix + `$`)

func validateGYearMonth(value string, version Version) error {
	if !gYearMonthRegex.MatchString(value) {
		return fmt.Errorf("invalid gYearMonth")
	}
	body, err := stripAndCheckTimezone(value)
	if err != nil {
		return err
	}
	// body is "[-]YYYY...-MM": the year is everything before the final "-MM".
	if version == Version10 {
		yearStr, _, _ := strings.Cut(strings.TrimPrefix(body, "-"), "-")
		if yearForbiddenInXSD10(yearStr) {
			return fmt.Errorf("invalid gYearMonth: year 0000 is not valid in XSD 1.0")
		}
	}
	// the month is the last two characters.
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

func validateNCName(value string) error {
	if !xmlchar.IsValidNCName(value) {
		return fmt.Errorf("invalid NCName")
	}
	return nil
}

func validateName(value string) error {
	if !xmlchar.IsValidName(value) {
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
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r == ':' || xmlchar.IsNCNameChar(r) {
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
	if value != trimXSDSpace(value) {
		return fmt.Errorf("invalid token")
	}
	if strings.Contains(value, "  ") {
		return fmt.Errorf("invalid token")
	}
	return nil
}

func validateAnyURI(value string, version Version) error {
	if version == Version11 {
		return nil
	}
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, ":") || strings.HasSuffix(value, ":") {
		return fmt.Errorf("invalid anyURI")
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= utf8.RuneSelf {
			continue
		}
		if ch < ' ' || ch == 0x7f || strings.ContainsRune("\\^`", rune(ch)) {
			return fmt.Errorf("invalid anyURI")
		}
		if ch == '%' {
			if i+2 >= len(value) || !isHexDigit(value[i+1]) || !isHexDigit(value[i+2]) {
				return fmt.Errorf("invalid anyURI")
			}
			i += 2
		}
	}
	return nil
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('A' <= ch && ch <= 'F') || ('a' <= ch && ch <= 'f')
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
// XSD whitespace is exactly space, tab, CR, and LF, so the repertoire excludes
// other Unicode whitespace such as form-feed.
var base64Regex = regexp.MustCompile("^[A-Za-z0-9+/= \t\r\n]*$")

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
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			continue
		}
		b.WriteRune(r)
	}
	// Strict() rejects padded forms whose unused trailing bits are non-zero
	// (e.g. "TR==", "AAB="), which are not valid xs:base64Binary lexical forms.
	if _, err := base64.StdEncoding.Strict().DecodeString(b.String()); err != nil {
		return fmt.Errorf("invalid base64Binary")
	}
	return nil
}

func validateSpaceSeparatedList(value string, validateItem func(string) error) error {
	if value == "" {
		return fmt.Errorf("empty list")
	}
	// Split on XSD whitespace only (space, tab, CR, LF). strings.Fields would
	// also split on NBSP and other Unicode whitespace, so a token containing
	// NBSP would be wrongly broken into valid pieces; XSD-only splitting keeps
	// it as one token that per-item validation then rejects.
	items := xsdFields(value)
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
