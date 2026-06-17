package value

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Compare dispatches to type-specific comparison.
// Returns (cmp, ok) where cmp is -1/0/+1 and ok is false when comparison
// is undefined (NaN, incomparable durations, parse failures).
func Compare(a, b, builtinLocal string) (int, bool) {
	switch builtinLocal {
	case "boolean":
		return compareBoolean(a, b)
	case "float", "double":
		return compareFloat(a, b)
	case "dateTime":
		return compareDateTime(a, b)
	case "date":
		return compareDate(a, b)
	case "time":
		return compareTime(a, b)
	case "gYear":
		return compareGYear(a, b)
	case "gYearMonth":
		return compareGYearMonth(a, b)
	case "gMonth":
		return compareGMonth(a, b)
	case "gDay":
		return compareGDay(a, b)
	case "gMonthDay":
		return compareGMonthDay(a, b)
	case "duration":
		return compareDuration(a, b)
	case "hexBinary":
		return compareHexBinary(a, b)
	case "base64Binary":
		return compareBase64Binary(a, b)
	default:
		cmp := CompareDecimal(a, b)
		if cmp == -2 {
			return 0, false
		}
		return cmp, true
	}
}

// CanonicalKey maps a lexical value to a value-space canonical string for the
// given builtin type, so lexically-distinct but value-equal inputs (e.g. "5"
// and "+5" for xs:integer) produce the same key. It returns (key, true) when a
// canonical form is defined; otherwise (collapsed-whitespace value, false) for
// string-family types and anything unrecognized or unparsable.
func CanonicalKey(s, builtinLocal string) (string, bool) {
	switch builtinLocal {
	case "boolean":
		trimmed := strings.TrimSpace(s)
		if trimmed == "true" || trimmed == "1" {
			return "1", true
		}
		if trimmed == "false" || trimmed == "0" {
			return "0", true
		}
		return trimmed, false
	case "float":
		return canonicalFloatKey(s, 32) // xs:float is 32-bit IEEE-754
	case "double":
		return canonicalFloatKey(s, 64)
	case "dateTime", "date", "time", "gYear", "gYearMonth", "gMonth", "gDay", "gMonthDay":
		return canonicalDateTimeKey(strings.TrimSpace(s), builtinLocal)
	case "decimal", "integer",
		"nonPositiveInteger", "negativeInteger", "long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger":
		trimmed := strings.TrimSpace(s)
		r, ok := new(big.Rat).SetString(trimmed)
		if !ok {
			return trimmed, false
		}
		return r.RatString(), true
	case "string":
		// whiteSpace=preserve: the value space is the exact lexical string, so
		// leading/trailing/internal whitespace is significant. Do not alter it.
		return s, false
	case "normalizedString":
		// whiteSpace=replace: tab/newline/carriage-return become spaces.
		return whitespaceReplace(s), false
	case "NMTOKENS", "IDREFS", "ENTITIES":
		// List types: collapse internal whitespace so token sequences that
		// differ only in separator whitespace are value-equal.
		return strings.Join(strings.Fields(s), " "), false
	default:
		// Remaining string-derived types (token, NMTOKEN, Name, NCName, ID,
		// IDREF, ENTITY, language, anyURI, …) have whiteSpace=collapse.
		return strings.Join(strings.Fields(s), " "), false
	}
}

// canonicalFloatKey canonicalizes an xs:float/xs:double value at the given IEEE
// precision (bitSize 32 for xs:float, 64 for xs:double). Using the correct
// precision ensures values that are equal in xs:float's 32-bit value space (but
// distinct as 64-bit doubles) map to the same key, and vice versa.
func canonicalFloatKey(s string, bitSize int) (string, bool) {
	trimmed := strings.TrimSpace(s)
	f, ok := parseXSDFloat(trimmed)
	if !ok {
		return trimmed, false
	}
	if math.IsNaN(f) {
		return "NaN", true
	}
	if math.IsInf(f, 1) {
		return "INF", true
	}
	if math.IsInf(f, -1) {
		return "-INF", true
	}
	if bitSize == 32 {
		f = float64(float32(f)) // round to xs:float precision
	}
	if f == 0 {
		f = 0 // normalize -0 to +0; they are equal in the value space
	}
	return strconv.FormatFloat(f, 'g', -1, bitSize), true
}

// whitespaceReplace applies the XSD whiteSpace="replace" normalization: each
// of the four XSD whitespace characters tab (#x9), newline (#xA), and carriage
// return (#xD) becomes a single space (#x20). Per the XSD datatype spec only
// those ASCII whitespace characters are affected; Unicode whitespace such as
// NBSP (U+00A0) is left untouched.
func whitespaceReplace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)
}

// whitespaceCollapse applies the XSD whiteSpace="collapse" normalization:
// replace tab/newline/CR with space (ASCII-only, like whitespaceReplace), then
// collapse runs of spaces and trim leading/trailing spaces. Only the four XSD
// whitespace characters (#x20, #x9, #xD, #xA) are treated as whitespace; Unicode
// whitespace such as NBSP (U+00A0) is preserved, so an invalid value containing
// it remains invalid under subsequent lexical validation.
func whitespaceCollapse(s string) string {
	replaced := whitespaceReplace(s)
	var b strings.Builder
	b.Grow(len(replaced))
	inSpace := true // treat start as space to trim leading
	for i := range len(replaced) {
		if replaced[i] == ' ' {
			inSpace = true
			continue
		}
		if inSpace && b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(replaced[i])
		inSpace = false
	}
	return b.String()
}

// WhiteSpace returns the effective XSD whiteSpace facet ("preserve", "replace",
// or "collapse") for a builtin datatype's local name. xs:string preserves,
// xs:normalizedString replaces, and every other builtin (token, the integer and
// date/time families, boolean, the NCName/Name/NMTOKEN family, list types,
// anyURI, …) collapses, per the XSD 1.1 datatype spec. Unknown names default to
// "collapse" so callers normalize conservatively.
func WhiteSpace(builtinLocal string) string {
	switch builtinLocal {
	case "string":
		return "preserve"
	case "normalizedString":
		return "replace"
	default:
		return "collapse"
	}
}

// Normalize applies the XSD whiteSpace facet of the named builtin datatype to a
// lexical value, returning the whitespace-processed form that must be used
// before lexical validation (ValidateBuiltin) or value comparison. "preserve"
// leaves the value untouched, "replace" turns each tab/newline/CR into a space,
// and "collapse" additionally collapses runs of spaces and trims the ends.
func Normalize(s, builtinLocal string) string {
	switch WhiteSpace(builtinLocal) {
	case "preserve":
		return s
	case "replace":
		return whitespaceReplace(s)
	default: // collapse
		return whitespaceCollapse(s)
	}
}

func canonicalDateTimeKey(s, builtinLocal string) (string, bool) {
	var dt xsdDateTime
	var ok bool
	switch builtinLocal {
	case "dateTime":
		dt, ok = parseXSDDateTime(s)
	case "date":
		dt, ok = parseXSDDate(s)
	case "time":
		dt, ok = parseXSDTime(s)
	case "gYear":
		dt, ok = parseXSDGYear(s)
	case "gYearMonth":
		dt, ok = parseXSDGYearMonth(s)
	case "gMonth":
		dt, ok = parseXSDGMonth(s)
	case "gDay":
		dt, ok = parseXSDGDay(s)
	case "gMonthDay":
		dt, ok = parseXSDGMonthDay(s)
	}
	if !ok {
		return s, false
	}
	if dt.hasTZ {
		dt = dt.normalizeToUTC()
	}
	return fmt.Sprintf("%d|%d|%d|%d|%d|%g|%t", dt.year, dt.month, dt.day, dt.hour, dt.min, dt.sec, dt.hasTZ), true
}

// CompareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func CompareDecimal(a, b string) int {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return -2
	}
	return ra.Cmp(rb)
}

// parseXSDBoolean canonicalizes an xs:boolean lexical form. "true"/"1" map to
// true and "false"/"0" map to false. Any other input is not a valid boolean.
func parseXSDBoolean(s string) (bool, bool) {
	switch s {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	}
	return false, false
}

// compareBoolean compares two xs:boolean values in value space. xs:boolean has
// no order relation in XSD, so callers should rely only on equality (cmp == 0).
// For a total, deterministic result this orders false < true; equal values
// return 0.
func compareBoolean(a, b string) (int, bool) {
	ba, ok1 := parseXSDBoolean(a)
	bb, ok2 := parseXSDBoolean(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	if ba == bb {
		return 0, true
	}
	if !ba {
		return -1, true
	}
	return 1, true
}

// compareHexBinary compares two xs:hexBinary values in value space (the decoded
// octet sequence), so lexically distinct forms that decode to the same bytes are
// equal (e.g. "0A" == "0a"). XSD does not order hexBinary, but a deterministic,
// antisymmetric total order (bytes.Compare of the decoded octets) is returned so
// the result is a well-behaved comparator; enumeration only relies on cmp == 0.
// Returns ok=false if either operand is not valid hexBinary.
func compareHexBinary(a, b string) (int, bool) {
	da, err1 := hex.DecodeString(a)
	db, err2 := hex.DecodeString(b)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return bytes.Compare(da, db), true
}

// compareBase64Binary compares two xs:base64Binary values in value space (the
// decoded octet sequence), ignoring the whitespace permitted in the lexical
// form. As with hexBinary, a deterministic bytes.Compare total order is returned
// rather than a bare equality flag. Returns ok=false if either operand is not
// valid base64Binary.
func compareBase64Binary(a, b string) (int, bool) {
	da, ok1 := decodeBase64Binary(a)
	db, ok2 := decodeBase64Binary(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return bytes.Compare(da, db), true
}

func decodeBase64Binary(s string) ([]byte, bool) {
	stripped := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
	if decoded, err := base64.StdEncoding.DecodeString(stripped); err == nil {
		return decoded, true
	}
	// validateBase64Binary is regex-only, so it admits unpadded (and partially
	// padded) forms that StdEncoding rejects. Fall back to RawStdEncoding after
	// dropping any partial padding, so a value-space comparison still succeeds
	// for a value the lexical validator accepted (e.g. "TQ" == "TQ==").
	if decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(stripped, "=")); err == nil {
		return decoded, true
	}
	return nil, false
}

func parseXSDFloat(s string) (float64, bool) {
	switch s {
	case "INF", "+INF":
		return math.Inf(1), true
	case "-INF":
		return math.Inf(-1), true
	// The float lexical validator (floatRegex) accepts an optional leading sign
	// on NaN, so accept the signed forms here too for consistency. The sign is
	// meaningless for NaN.
	case "NaN", "+NaN", "-NaN":
		return math.NaN(), true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// IsFloatNaN reports whether s is a valid xs:float/xs:double lexical form that
// denotes NaN (including the sign-prefixed forms the lexical validator accepts).
func IsFloatNaN(s string) bool {
	f, ok := parseXSDFloat(s)
	return ok && math.IsNaN(f)
}

func compareFloat(a, b string) (int, bool) {
	fa, ok1 := parseXSDFloat(a)
	fb, ok2 := parseXSDFloat(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	if math.IsNaN(fa) || math.IsNaN(fb) {
		return 0, false
	}
	if fa < fb {
		return -1, true
	}
	if fa > fb {
		return 1, true
	}
	return 0, true
}

type xsdDateTime struct {
	year, month, day int
	hour, min        int
	sec              float64
	hasTZ            bool
	tzMin            int
}

func parseTZ(s string) (bool, int) {
	if s == "" {
		return false, 0
	}
	if s[0] == 'Z' || s[0] == 'z' {
		return true, 0
	}
	if (s[0] == '+' || s[0] == '-') && len(s) >= 6 && s[3] == ':' {
		hh, err1 := strconv.Atoi(s[1:3])
		mm, err2 := strconv.Atoi(s[4:6])
		if err1 != nil || err2 != nil {
			return false, 0
		}
		offset := hh*60 + mm
		if s[0] == '-' {
			offset = -offset
		}
		return true, offset
	}
	return false, 0
}

func parseXSDDateTime(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find 'T' separator.
	datePart, timePart, found := strings.Cut(s, "T")
	if !found {
		return dt, false
	}

	// Parse date: YYYY-MM-DD
	dParts := strings.SplitN(datePart, "-", 3)
	if len(dParts) != 3 {
		return dt, false
	}
	year, err := strconv.Atoi(dParts[0])
	if err != nil {
		return dt, false
	}
	month, err := strconv.Atoi(dParts[1])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(dParts[2])
	if err != nil {
		return dt, false
	}
	if neg {
		year = -year
	}
	dt.year = year
	dt.month = month
	dt.day = day

	// Parse time: HH:MM:SS[.frac][TZ]
	if !parseTimeInto(&dt, timePart) {
		return dt, false
	}
	return dt, true
}

func parseTimeFields(s string) (int, int, float64, string, bool) {
	if len(s) < 8 || s[2] != ':' || s[5] != ':' {
		return 0, 0, 0, "", false
	}
	hh, err1 := strconv.Atoi(s[0:2])
	mm, err2 := strconv.Atoi(s[3:5])
	if err1 != nil || err2 != nil {
		return 0, 0, 0, "", false
	}
	// Seconds may have fractional part.
	rest := s[6:]
	secEnd := 0
	for secEnd < len(rest) {
		c := rest[secEnd]
		if (c >= '0' && c <= '9') || c == '.' {
			secEnd++
		} else {
			break
		}
	}
	sec, err := strconv.ParseFloat(rest[:secEnd], 64)
	if err != nil {
		return 0, 0, 0, "", false
	}
	return hh, mm, sec, rest[secEnd:], true
}

func parseTimeInto(dt *xsdDateTime, s string) bool {
	hh, mm, sec, rest, ok := parseTimeFields(s)
	if !ok {
		return false
	}
	dt.hour = hh
	dt.min = mm
	dt.sec = sec
	hasTZ, tzOff := parseTZ(rest)
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return true
}

func parseXSDDate(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// YYYY-MM-DD[TZ]
	if len(s) < 10 || s[4] != '-' || s[7] != '-' {
		return dt, false
	}
	// Handle years > 4 digits.
	dashIdx := strings.IndexByte(s, '-')
	if dashIdx < 4 {
		return dt, false
	}
	year, err := strconv.Atoi(s[:dashIdx])
	if err != nil {
		return dt, false
	}
	rest := s[dashIdx+1:]
	if len(rest) < 5 || rest[2] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(rest[0:2])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(rest[3:5])
	if err != nil {
		return dt, false
	}
	if neg {
		year = -year
	}
	dt.year = year
	dt.month = month
	dt.day = day
	hasTZ, tzOff := parseTZ(rest[5:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDTime(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if !parseTimeInto(&dt, s) {
		return dt, false
	}
	return dt, true
}

func parseXSDGYear(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find end of digit run.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < 4 {
		return dt, false
	}
	year, err := strconv.Atoi(s[:i])
	if err != nil {
		return dt, false
	}
	if neg {
		year = -year
	}
	dt.year = year
	hasTZ, tzOff := parseTZ(s[i:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGYearMonth(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find first dash after year digits.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < 4 || i >= len(s) || s[i] != '-' {
		return dt, false
	}
	year, err := strconv.Atoi(s[:i])
	if err != nil {
		return dt, false
	}
	rest := s[i+1:]
	if len(rest) < 2 {
		return dt, false
	}
	month, err := strconv.Atoi(rest[:2])
	if err != nil {
		return dt, false
	}
	if neg {
		year = -year
	}
	dt.year = year
	dt.month = month
	hasTZ, tzOff := parseTZ(rest[2:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGMonth(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 4 || s[0] != '-' || s[1] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(s[2:4])
	if err != nil {
		return dt, false
	}
	dt.month = month
	hasTZ, tzOff := parseTZ(s[4:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGDay(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 5 || s[0] != '-' || s[1] != '-' || s[2] != '-' {
		return dt, false
	}
	day, err := strconv.Atoi(s[3:5])
	if err != nil {
		return dt, false
	}
	dt.day = day
	hasTZ, tzOff := parseTZ(s[5:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGMonthDay(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 7 || s[0] != '-' || s[1] != '-' || s[4] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(s[2:4])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(s[5:7])
	if err != nil {
		return dt, false
	}
	dt.month = month
	dt.day = day
	hasTZ, tzOff := parseTZ(s[7:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
			return 29
		}
		return 28
	}
	return 30
}

func (dt xsdDateTime) normalizeToUTC() xsdDateTime {
	if !dt.hasTZ || dt.tzMin == 0 {
		return dt
	}
	r := dt
	r.min -= r.tzMin
	r.tzMin = 0

	// Propagate minutes overflow.
	for r.min < 0 {
		r.min += 60
		r.hour--
	}
	for r.min >= 60 {
		r.min -= 60
		r.hour++
	}

	// Propagate hours overflow.
	for r.hour < 0 {
		r.hour += 24
		r.day--
	}
	for r.hour >= 24 {
		r.hour -= 24
		r.day++
	}

	// Propagate day overflow.
	for r.day < 1 {
		r.month--
		if r.month < 1 {
			r.month = 12
			r.year--
		}
		r.day += daysInMonth(r.year, r.month)
	}
	for r.month > 0 && r.day > daysInMonth(r.year, r.month) {
		r.day -= daysInMonth(r.year, r.month)
		r.month++
		if r.month > 12 {
			r.month = 1
			r.year++
		}
	}

	return r
}

func compareDateTimeFields(a, b xsdDateTime) int {
	if a.year != b.year {
		if a.year < b.year {
			return -1
		}
		return 1
	}
	if a.month != b.month {
		if a.month < b.month {
			return -1
		}
		return 1
	}
	if a.day != b.day {
		if a.day < b.day {
			return -1
		}
		return 1
	}
	if a.hour != b.hour {
		if a.hour < b.hour {
			return -1
		}
		return 1
	}
	if a.min != b.min {
		if a.min < b.min {
			return -1
		}
		return 1
	}
	if a.sec < b.sec {
		return -1
	}
	if a.sec > b.sec {
		return 1
	}
	return 0
}

func compareDateTimeParsed(a, b xsdDateTime) (int, bool) {
	if a.hasTZ != b.hasTZ {
		return compareDateTimeMixedTZ(a, b)
	}
	if a.hasTZ {
		a = a.normalizeToUTC()
		b = b.normalizeToUTC()
	}
	return compareDateTimeFields(a, b), true
}

// compareDateTimeMixedTZ compares two date/time values when exactly one carries
// a timezone, applying the XSD 1.0 order relation (3.2.7.4). A non-timezoned
// value denotes the instant interval [v-14:00, v+14:00]; if that whole interval
// lies on one side of the timezoned operand the result is determinate. Only an
// overlapping interval is indeterminate.
func compareDateTimeMixedTZ(a, b xsdDateTime) (int, bool) {
	// The determinate rule normalizes a synthetic ±14:00 offset across day
	// boundaries, which requires a full calendar date (year, month, day). The
	// partial gregorian types leave some of those components zero — gYear
	// (month=0, day=0), gYearMonth (day=0), gMonth (year=0, day=0), gDay (year=0,
	// month=0), and gMonthDay (year=0). Applying the offset to a zero field makes
	// normalizeToUTC borrow into a neighbouring period and yield a determinately
	// wrong result (e.g. gYear "2020" rolling back to 2019), so those types stay
	// indeterminate, as they were before this rule existed. (xs:time is not in
	// this set: compareTime assigns a reference date before comparing, so it has
	// a full calendar date and flows through the determinate path correctly. The
	// only loss is a literal year-0000 dateTime, an XSD 1.1 edge case, which
	// falls back to indeterminate rather than wrong.)
	if a.year == 0 || b.year == 0 || a.month < 1 || b.month < 1 || a.day < 1 || b.day < 1 {
		return 0, false
	}

	// Orient so that `tz` is the timezoned operand and `plain` has no timezone.
	tz, plain := a, b
	swapped := false
	if !a.hasTZ {
		tz, plain = b, a
		swapped = true
	}

	tz = tz.normalizeToUTC()

	// Interpret the non-timezoned operand under its two extreme timezones.
	// +14:00 yields its earliest instant (largest subtraction from UTC),
	// -14:00 yields its latest instant. We compare plain against tz, so we
	// build the UTC-normalized plain value at each extreme.
	low := plain
	low.hasTZ = true
	low.tzMin = 14 * 60
	low = low.normalizeToUTC()

	high := plain
	high.hasTZ = true
	high.tzMin = -14 * 60
	high = high.normalizeToUTC()

	cmpLow := compareDateTimeFields(low, tz)
	cmpHigh := compareDateTimeFields(high, tz)

	// Both extremes on the same side → determinate result for `plain` vs `tz`.
	// orient converts that into the result for the original `a` vs `b` order:
	// when the operands were not swapped, `a` is `tz` and `b` is `plain`, so
	// the sign must be inverted.
	orient := func(cmp int) int {
		if swapped {
			return cmp
		}
		return -cmp
	}
	if cmpLow > 0 && cmpHigh > 0 {
		return orient(1), true
	}
	if cmpLow < 0 && cmpHigh < 0 {
		return orient(-1), true
	}
	return 0, false
}

func compareDateTime(a, b string) (int, bool) {
	da, ok1 := parseXSDDateTime(a)
	db, ok2 := parseXSDDateTime(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareDate(a, b string) (int, bool) {
	da, ok1 := parseXSDDate(a)
	db, ok2 := parseXSDDate(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareTime(a, b string) (int, bool) {
	da, ok1 := parseXSDTime(a)
	db, ok2 := parseXSDTime(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	// Set a reference date so TZ normalization day overflow works.
	da.year, da.month, da.day = 2000, 1, 15
	db.year, db.month, db.day = 2000, 1, 15
	return compareDateTimeParsed(da, db)
}

func compareGYear(a, b string) (int, bool) {
	da, ok1 := parseXSDGYear(a)
	db, ok2 := parseXSDGYear(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGYearMonth(a, b string) (int, bool) {
	da, ok1 := parseXSDGYearMonth(a)
	db, ok2 := parseXSDGYearMonth(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGMonth(a, b string) (int, bool) {
	da, ok1 := parseXSDGMonth(a)
	db, ok2 := parseXSDGMonth(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGDay(a, b string) (int, bool) {
	da, ok1 := parseXSDGDay(a)
	db, ok2 := parseXSDGDay(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGMonthDay(a, b string) (int, bool) {
	da, ok1 := parseXSDGMonthDay(a)
	db, ok2 := parseXSDGMonthDay(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

type xsdDuration struct {
	negative bool
	months   int
	seconds  float64
}

func parseXSDDurationValue(s string) (xsdDuration, bool) {
	var d xsdDuration
	if len(s) == 0 {
		return d, false
	}
	if s[0] == '-' {
		d.negative = true
		s = s[1:]
	}
	if len(s) == 0 || s[0] != 'P' {
		return d, false
	}
	s = s[1:]
	if s == "" || s == "T" {
		return d, false
	}

	inTime := false
	for len(s) > 0 {
		if s[0] == 'T' {
			inTime = true
			s = s[1:]
			continue
		}
		// Read number (may have fractional part for seconds).
		numEnd := 0
		for numEnd < len(s) && ((s[numEnd] >= '0' && s[numEnd] <= '9') || s[numEnd] == '.') {
			numEnd++
		}
		if numEnd == 0 || numEnd >= len(s) {
			return d, false
		}
		numStr := s[:numEnd]
		designator := s[numEnd]
		s = s[numEnd+1:]

		if !inTime {
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return d, false
			}
			switch designator {
			case 'Y':
				d.months += n * 12
			case 'M':
				d.months += n
			case 'D':
				d.seconds += float64(n) * 86400
			default:
				return d, false
			}
		} else {
			switch designator {
			case 'H':
				n, err := strconv.Atoi(numStr)
				if err != nil {
					return d, false
				}
				d.seconds += float64(n) * 3600
			case 'M':
				n, err := strconv.Atoi(numStr)
				if err != nil {
					return d, false
				}
				d.seconds += float64(n) * 60
			case 'S':
				f, err := strconv.ParseFloat(numStr, 64)
				if err != nil {
					return d, false
				}
				d.seconds += f
			default:
				return d, false
			}
		}
	}
	return d, true
}

func compareDuration(a, b string) (int, bool) {
	da, ok1 := parseXSDDurationValue(a)
	db, ok2 := parseXSDDurationValue(b)
	if !ok1 || !ok2 {
		return 0, false
	}

	// Apply sign.
	am, as := da.months, da.seconds
	if da.negative {
		am, as = -am, -as
	}
	bm, bs := db.months, db.seconds
	if db.negative {
		bm, bs = -bm, -bs
	}

	// Compare month and seconds components independently.
	monthCmp := intCmp(am, bm)
	secCmp := floatCmp(as, bs)

	if monthCmp == secCmp {
		return monthCmp, true
	}
	// One is zero, the other determines.
	if monthCmp == 0 {
		return secCmp, true
	}
	if secCmp == 0 {
		return monthCmp, true
	}
	// Components disagree — indeterminate.
	return 0, false
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func floatCmp(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
