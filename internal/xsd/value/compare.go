package value

import (
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
	default:
		cmp := CompareDecimal(a, b)
		if cmp == -2 {
			return 0, false
		}
		return cmp, true
	}
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

func parseXSDFloat(s string) (float64, bool) {
	switch s {
	case "INF", "+INF":
		return math.Inf(1), true
	case "-INF":
		return math.Inf(-1), true
	case "NaN":
		return math.NaN(), true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
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
	tIdx := strings.IndexByte(s, 'T')
	if tIdx < 0 {
		return dt, false
	}
	datePart := s[:tIdx]
	timePart := s[tIdx+1:]

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
		return 0, false // indeterminate
	}
	if a.hasTZ {
		a = a.normalizeToUTC()
		b = b.normalizeToUTC()
	}
	return compareDateTimeFields(a, b), true
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
