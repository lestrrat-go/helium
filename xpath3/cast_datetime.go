package xpath3

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// castToGType casts a date/dateTime value to a Gregorian partial type.
func castToGType(v AtomicValue, targetType string, format func(time.Time) string) (AtomicValue, error) {
	switch v.TypeName {
	case TypeDateTime, TypeDate:
		return AtomicValue{TypeName: targetType, Value: format(v.TimeVal())}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), targetType)
	}
	return AtomicValue{}, &XPathError{
		Code:    lexicon.ErrXPTY0004,
		Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, targetType),
	}
}

// formatXSDTimezone returns the timezone suffix for an XSD date/time value.
func formatXSDTimezone(t time.Time) string {
	if !HasTimezone(t) {
		return ""
	}
	_, offset := t.Zone()
	if offset == 0 {
		return "Z"
	}
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	h := offset / 3600
	m := (offset % 3600) / 60
	return fmt.Sprintf("%c%02d:%02d", sign, h, m)
}

// splitXSDYear splits an XSD date/dateTime string into the year (as int) and
// the remainder starting from the first '-' after the year digits.
// It handles optional leading '-' for negative years and years with more than
// 4 digits (e.g., "-0002-06-01" → year=-2, rest="-06-01";
// "654321-01-01" → year=654321, rest="-01-01").
func splitXSDYear(s string) (int, string, error) {
	i := 0
	neg := false
	if i < len(s) && s[i] == '-' {
		neg = true
		i++
	}
	// Consume year digits (at least 4 required by XSD)
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	digits := i - start
	if digits < 4 {
		return 0, "", fmt.Errorf("year must have at least 4 digits")
	}
	// Per XSD: years with more than 4 digits must not have leading zeros
	if digits > 4 && s[start] == '0' {
		return 0, "", fmt.Errorf("year with leading zeros is not valid")
	}
	yearStr := s[start:i]
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid year: %q", yearStr)
	}
	if neg {
		year = -year
	}
	// Year 0000 is not supported; reject with FODT0001.
	if year == 0 {
		return 0, "", &XPathError{Code: errCodeFODT0001, Message: "year zero is not supported"}
	}
	// Reject years outside Go's time.Time representable range.
	// time.Date wraps silently for extreme years; cap at ±999,999,999.
	if year > 999_999_999 || year < -999_999_999 {
		return 0, "", fmt.Errorf("year %d out of representable range", year)
	}
	return year, s[i:], nil
}

// buildTimeFromParts constructs a time.Time from a parsed year and the
// remaining month-day (and optional time/tz) components. It uses time.Parse
// with a reference year of 2006, then replaces the year with the actual value.
func buildTimeFromParts(year int, rest string, layouts []string, original string) (time.Time, bool) {
	month, day, err := extractDateMonthDay(rest)
	if err != nil {
		return time.Time{}, false
	}
	if err := validateDateComponents(month, day, year); err != nil {
		return time.Time{}, false
	}

	// rest starts with "-MM-DD..." — prepend a synthetic 4-digit year for time.Parse.
	// Go's time.Parse reference year (2006) is not a leap year, so Feb 29 fails.
	// Use a leap year (2000) in the value string. Go layout uses "2006" for year,
	// but the value has "2000" — Go only matches positionally for fixed-width
	// components so we must re-format the layout to accept "2000".
	// Alternative approach: if the actual year is a leap year, temporarily
	// parse as Jan 1 to get the timezone, then construct manually.

	synthetic := "2006" + rest
	for _, layout := range layouts {
		if t, err := time.Parse(layout, synthetic); err == nil {
			t = time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			t = ensureExplicitTZ(t, original)
			return t, true
		}
	}

	// Retry with a leap-year workaround for Feb 29: parse a modified string
	// with day=28, then adjust the day back to the original value.
	if strings.HasPrefix(rest, "-02-29") {
		modRest := "-02-28" + rest[6:]
		synthetic = "2006" + modRest
		for _, layout := range layouts {
			if t, err := time.Parse(layout, synthetic); err == nil {
				t = time.Date(year, t.Month(), 29, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
				t = ensureExplicitTZ(t, original)
				return t, true
			}
		}
	}

	return time.Time{}, false
}

func extractDateMonthDay(rest string) (int, int, error) {
	if len(rest) < 6 || rest[0] != '-' || rest[3] != '-' {
		return 0, 0, fmt.Errorf("invalid month-day segment %q", rest)
	}

	month, err := strconv.Atoi(rest[1:3])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month in %q", rest)
	}

	day, err := strconv.Atoi(rest[4:6])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid day in %q", rest)
	}
	return month, day, nil
}

func parseXSDDate(s string) (time.Time, error) {
	year, rest, err := splitXSDYear(s)
	if err != nil {
		var xe *XPathError
		if errors.As(err, &xe) {
			return time.Time{}, xe
		}
		return time.Time{}, fmt.Errorf("invalid xs:date: %q", s)
	}
	if err := validateTimezoneInString(s); err != nil {
		return time.Time{}, fmt.Errorf("invalid xs:date: %q: %w", s, err)
	}
	layouts := []string{
		"2006-01-02Z07:00",
		"2006-01-02",
	}
	if t, ok := buildTimeFromParts(year, rest, layouts, s); ok {
		if err := validateDateComponents(int(t.Month()), t.Day(), year); err != nil {
			return time.Time{}, fmt.Errorf("invalid xs:date: %q: %w", s, err)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid xs:date: %q", s)
}

func parseXSDDateTime(s string) (time.Time, error) {
	// Handle 24:00:00 (end-of-day midnight) — XSD allows this
	normalized, isMidnight24 := normalizeMidnight24DateTime(s)
	target := s
	if isMidnight24 {
		target = normalized
	}
	if err := validateTimezoneInString(s); err != nil {
		return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q: %w", s, err)
	}
	year, rest, err := splitXSDYear(target)
	if err != nil {
		var xe *XPathError
		if errors.As(err, &xe) {
			return time.Time{}, xe
		}
		return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q", s)
	}
	layouts := []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}
	if t, ok := buildTimeFromParts(year, rest, layouts, s); ok {
		if err := validateDateComponents(int(t.Month()), t.Day(), year); err != nil {
			return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q: %w", s, err)
		}
		if !isMidnight24 {
			if err := validateTimeComponents(t.Hour(), t.Minute(), t.Second(), t.Nanosecond()); err != nil {
				return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q: %w", s, err)
			}
		}
		if isMidnight24 {
			t = t.AddDate(0, 0, 1) // advance to next day
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q", s)
}

func parseXSDTime(s string) (time.Time, error) {
	if err := validateTimezoneInString(s); err != nil {
		return time.Time{}, fmt.Errorf("invalid xs:time: %q: %w", s, err)
	}
	// Handle 24:00:00 — XSD allows this as equivalent to 00:00:00
	normalized, isMidnight24 := normalizeMidnight24Time(s)
	target := s
	if isMidnight24 {
		target = normalized
	}
	for _, layout := range []string{
		"15:04:05.999999999Z07:00",
		"15:04:05Z07:00",
		"15:04:05.999999999",
		"15:04:05",
	} {
		if t, err := time.Parse(layout, target); err == nil {
			if !isMidnight24 {
				if err := validateTimeComponents(t.Hour(), t.Minute(), t.Second(), t.Nanosecond()); err != nil {
					return time.Time{}, fmt.Errorf("invalid xs:time: %q: %w", s, err)
				}
			}
			return ensureExplicitTZ(t, s), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:time: %q", s)
}

// normalizeMidnight24DateTime checks if a dateTime string has T24:00:00 and
// replaces it with T00:00:00. Returns the normalized string and true if it was
// a midnight-24 value. The caller must advance the date by one day.
// 24:00:00.nnn is valid only when the fractional part is all zeros.
func normalizeMidnight24DateTime(s string) (string, bool) {
	before, rest, found := strings.Cut(s, "T24:00:00")
	if !found {
		return s, false
	}
	// Allow fractional seconds only if all digits are zero (e.g. .000)
	if len(rest) > 0 && rest[0] == '.' {
		i := 1
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			if rest[i] != '0' {
				return s, false // non-zero fractional part is invalid
			}
			i++
		}
		if i == 1 {
			return s, false // bare "." with no digits is invalid
		}
		// Strip the all-zero fractional part
		rest = rest[i:]
	}
	return before + "T00:00:00" + rest, true
}

// normalizeMidnight24Time checks if a time string starts with 24:00:00 and
// replaces it with 00:00:00. For xs:time, 24:00:00 equals 00:00:00 (no date rollover).
// 24:00:00.nnn is valid only when the fractional part is all zeros.
func normalizeMidnight24Time(s string) (string, bool) {
	if !strings.HasPrefix(s, "24:00:00") {
		return s, false
	}
	rest := s[len("24:00:00"):]
	if len(rest) > 0 && rest[0] == '.' {
		i := 1
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			if rest[i] != '0' {
				return s, false // non-zero fractional part is invalid
			}
			i++
		}
		if i == 1 {
			return s, false
		}
		rest = rest[i:]
	}
	return "00:00:00" + rest, true
}

func ensureExplicitTZ(t time.Time, s string) time.Time {
	if t.Location() != time.UTC {
		return t
	}
	if hasExplicitTimezone(s) {
		return t.In(time.FixedZone("", 0))
	}
	// No explicit timezone — use the sentinel location so we can distinguish
	// "12:00:00" (no TZ) from "12:00:00Z" (explicit UTC).
	return t.In(noTZLocation)
}

func hasExplicitTimezone(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[len(s)-1] == 'Z' {
		return true
	}
	if len(s) >= 6 {
		tail := s[len(s)-6:]
		if (tail[0] == '+' || tail[0] == '-') &&
			tail[1] >= '0' && tail[1] <= '9' &&
			tail[2] >= '0' && tail[2] <= '9' &&
			tail[3] == ':' &&
			tail[4] >= '0' && tail[4] <= '9' &&
			tail[5] >= '0' && tail[5] <= '9' {
			return true
		}
	}
	return false
}

// validateDateComponents checks that month and day are in valid ranges.
// XSD 1.1: year 0000 is valid.
func validateDateComponents(month, day, year int) error {
	if month < 1 || month > 12 {
		return fmt.Errorf("month %d out of range 1-12", month)
	}
	maxDay := daysInMonth(year, month)
	if day < 1 || day > maxDay {
		return fmt.Errorf("day %d out of range for month %d (max %d)", day, month, maxDay)
	}
	return nil
}

// validateTimeComponents checks that hours, minutes, seconds are in valid ranges.
// Note: hour 24 is handled separately (normalizeMidnight24*) before this is called.
func validateTimeComponents(hour, minute, second, nanosecond int) error {
	if hour < 0 || hour > 23 {
		return fmt.Errorf("hour %d out of range 0-23", hour)
	}
	if minute < 0 || minute > 59 {
		return fmt.Errorf("minute %d out of range 0-59", minute)
	}
	if second < 0 || second > 59 {
		return fmt.Errorf("second %d out of range 0-59", second)
	}
	_ = nanosecond // fractional seconds are always valid if parsed
	return nil
}

// validateTimezoneInString checks that any timezone offset in the string is
// within the valid XSD range of -14:00 to +14:00.
func validateTimezoneInString(s string) error {
	if len(s) == 0 {
		return nil
	}
	if s[len(s)-1] == 'Z' {
		return nil // Z is always valid
	}
	if len(s) < 6 {
		return nil // no timezone present
	}
	tail := s[len(s)-6:]
	if (tail[0] != '+' && tail[0] != '-') ||
		tail[3] != ':' {
		return nil // no timezone suffix
	}
	hh := (int(tail[1]-'0') * 10) + int(tail[2]-'0')
	mm := (int(tail[4]-'0') * 10) + int(tail[5]-'0')
	if hh > 14 || (hh == 14 && mm != 0) {
		return fmt.Errorf("timezone offset %s out of range (-14:00 to +14:00)", tail)
	}
	if mm > 59 {
		return fmt.Errorf("timezone offset minutes %d out of range 0-59", mm)
	}
	return nil
}

// addCheckedMonths adds two non-negative month counts, reporting ok=false on
// int overflow. The Duration.Months field is a plain int, so a year/month total
// that exceeds it (e.g. P768614336404564650Y11M) must be rejected BEFORE it
// wraps to an invalid negative lexical form.
func addCheckedMonths(a, b int) (int, bool) {
	if b > math.MaxInt-a {
		return 0, false
	}
	return a + b, true
}

// parseXSDDuration parses an XSD duration string like "P1Y2M3DT4H5M6S".
func parseXSDDuration(s string) (Duration, error) {
	if len(s) == 0 {
		return Duration{}, fmt.Errorf("empty duration")
	}

	d := Duration{}
	i := 0

	if s[i] == '-' {
		d.Negative = true
		i++
	}

	if i >= len(s) || s[i] != 'P' {
		return Duration{}, fmt.Errorf("invalid duration: %q", s)
	}
	i++

	seenComponent := false
	seenTimeComponent := false
	sawTimeMarker := false
	lastOrder := 0
	used := map[byte]struct{}{}

	// Accumulate the dayTime seconds magnitude EXACTLY as a rational so that
	// large whole-second values beyond float64's 2^53 exact range (and exact
	// fractional seconds) are preserved. d.Seconds keeps a float64 mirror for the
	// rest of the Duration machinery; d.SecRat is the authoritative value.
	secRat := new(big.Rat)
	addSecRat := func(num *big.Rat, mult int64) {
		secRat.Add(secRat, new(big.Rat).Mul(num, big.NewRat(mult, 1)))
	}

	inTime := false
	for i < len(s) {
		if s[i] == 'T' {
			if inTime || sawTimeMarker {
				return Duration{}, fmt.Errorf("invalid duration: %q", s)
			}
			if i == len(s)-1 {
				return Duration{}, fmt.Errorf("invalid duration: %q", s)
			}
			inTime = true
			sawTimeMarker = true
			i++
			continue
		}

		numStart := i
		dotPos := -1
		for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
			if s[i] == '.' {
				if dotPos >= 0 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				dotPos = i
			}
			i++
		}
		if i == numStart || i >= len(s) {
			return Duration{}, fmt.Errorf("invalid duration: %q", s)
		}
		if dotPos == numStart || dotPos == i-1 {
			return Duration{}, fmt.Errorf("invalid duration: %q", s)
		}

		numStr := s[numStart:i]
		designator := s[i]
		i++
		key := designator
		if inTime && designator == 'M' {
			key = 'm'
		}
		if _, exists := used[key]; exists {
			return Duration{}, fmt.Errorf("invalid duration: %q", s)
		}
		used[key] = struct{}{}

		if !inTime {
			if dotPos >= 0 {
				return Duration{}, fmt.Errorf("invalid duration: %q", s)
			}
			switch designator {
			case 'Y':
				if lastOrder >= 1 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 1
				n, err := strconv.Atoi(numStr)
				if err != nil {
					return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
				}
				if n > math.MaxInt/12 {
					return Duration{}, fmt.Errorf("duration overflow: %sY", numStr)
				}
				months, ok := addCheckedMonths(d.Months, n*12)
				if !ok {
					return Duration{}, fmt.Errorf("duration overflow: %sY", numStr)
				}
				d.Months = months
			case 'M':
				if lastOrder >= 2 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 2
				n, err := strconv.Atoi(numStr)
				if err != nil {
					return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
				}
				months, ok := addCheckedMonths(d.Months, n)
				if !ok {
					return Duration{}, fmt.Errorf("duration overflow: %sM", numStr)
				}
				d.Months = months
			case 'D':
				if lastOrder >= 3 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 3
				// Days feed dayTime seconds, which are tracked exactly in SecRat.
				// Parse as a big.Int so very large day counts (beyond int64) round
				// -trip; the float64 mirror is best-effort metadata.
				dayInt, ok := new(big.Int).SetString(numStr, 10)
				if !ok {
					return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
				}
				dayFloat, _ := new(big.Float).SetInt(dayInt).Float64()
				d.Seconds += dayFloat * 86400
				addSecRat(new(big.Rat).SetInt(dayInt), 86400)
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
		} else {
			if designator != 'S' && dotPos >= 0 {
				return Duration{}, fmt.Errorf("invalid duration: %q", s)
			}
			// Parse the same lexical number exactly as a rational for SecRat —
			// this is the authoritative value. A very large but VALID whole-second
			// lexical must not be rejected just because float64 cannot hold it.
			numRat, ok := new(big.Rat).SetString(numStr)
			if !ok {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			// The float64 mirror is best-effort metadata. ParseFloat returns an
			// out-of-range error along with a saturated value (±Inf); accept that
			// value rather than rejecting the duration, since SecRat is exact.
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			switch designator {
			case 'H':
				if lastOrder >= 4 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 4
				d.Seconds += f * 3600
				addSecRat(numRat, 3600)
			case 'M':
				if lastOrder >= 5 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 5
				d.Seconds += f * 60
				addSecRat(numRat, 60)
			case 'S':
				if lastOrder >= 6 {
					return Duration{}, fmt.Errorf("invalid duration: %q", s)
				}
				lastOrder = 6
				d.Seconds += f
				addSecRat(numRat, 1)
				// Store exact fractional seconds as big.Rat to avoid float64 precision loss
				if strings.ContainsRune(numStr, '.') {
					// Extract just the fractional part: frac = numRat - floor(numRat)
					intPart := new(big.Int).Div(numRat.Num(), numRat.Denom())
					d.FracSec = new(big.Rat).Sub(numRat, new(big.Rat).SetInt(intPart))
				}
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
			seenTimeComponent = true
		}
		seenComponent = true
	}
	if !seenComponent || (sawTimeMarker && !seenTimeComponent) {
		return Duration{}, fmt.Errorf("invalid duration: %q", s)
	}

	// Record the exact dayTime seconds magnitude. d.Negative carries the sign,
	// so secRat stays non-negative here.
	d.SecRat = secRat

	return d, nil
}

// exactFractionDigits returns the digits AFTER the decimal point for a
// fractional rational in [0,1), with NO rounding for terminating decimals.
//
// When FloatPrec reports the fraction is EXACT (terminating), the full exact
// precision is used with no cap, so an exact value arbitrarily close to 1 (e.g.
// 0.999...9 with hundreds of nines that is exactly representable) is rendered in
// full rather than rounded UP to "1.0...". Only NON-terminating fractions are
// capped, at which point FloatString may legitimately round; the cap value is
// chosen far beyond any precision a real lexical form carries.
//
// Trailing zeros are trimmed. Callers must never see a carried integer part
// (a leading "1."): with the exact-precision path that cannot occur.
func exactFractionDigits(frac *big.Rat) string {
	prec, exact := frac.FloatPrec()
	if exact {
		s := frac.FloatString(prec)
		s = strings.TrimPrefix(s, "0.")
		return strings.TrimRight(s, "0")
	}

	// Non-terminating fraction: emit a capped number of digits via TRUNCATING
	// long division. FloatString would ROUND, which can carry into the integer
	// part (e.g. 0.999... → 1.0) and corrupt the formatted duration; truncation
	// never carries, so the integer part stays 0 and no stray "." is emitted.
	const maxDigits = 40
	num := new(big.Int).Set(frac.Num())
	den := frac.Denom()
	ten := big.NewInt(10)
	var b strings.Builder
	q := new(big.Int)
	for i := 0; i < maxDigits && num.Sign() != 0; i++ {
		num.Mul(num, ten)
		q.QuoRem(num, den, num)
		b.WriteByte(byte('0' + q.Int64()))
	}
	return strings.TrimRight(b.String(), "0")
}

// formatDuration formats a Duration as an XSD duration string.
// typeName is used to select the correct zero representation:
// yearMonthDuration → "P0M", dayTimeDuration → "PT0S", duration → "PT0S".
func formatDuration(d Duration, typeName string) string {
	var b strings.Builder
	secsZero := d.Seconds == 0
	if d.SecRat != nil {
		secsZero = d.SecRat.Sign() == 0
	}
	isZero := d.Months == 0 && secsZero
	if d.Negative && !isZero {
		b.WriteByte('-')
	}
	b.WriteByte('P')

	totalMonths := d.Months
	if d.Negative && totalMonths < 0 {
		totalMonths = -totalMonths
	}

	years := totalMonths / 12
	months := totalMonths % 12
	if years != 0 {
		fmt.Fprintf(&b, "%dY", years)
	}
	if months != 0 {
		fmt.Fprintf(&b, "%dM", months)
	}

	// Decompose the seconds magnitude. Two distinct paths:
	//   - SecRat present: the EXACT total-seconds rational is authoritative.
	//     Split it into a whole-second big.Int (floor) and an exact fractional
	//     rational, bypassing all float64 rounding so a value just below a whole
	//     second never rounds the integer part UP while still carrying a
	//     fraction (which would emit an invalid "PT1.1.S").
	//   - SecRat absent (legacy float path): decompose d.Seconds via integer
	//     arithmetic and round the fraction to microseconds.
	// totalWholeSeconds holds the whole-second count as a big.Int so values above
	// math.MaxInt64 never wrap into malformed negative components.
	totalWholeSeconds := new(big.Int)
	var fracMicro int64
	var fracRat *big.Rat
	if d.SecRat != nil {
		absRat := d.SecRat
		if absRat.Sign() < 0 {
			absRat = new(big.Rat).Neg(absRat)
		}
		whole := new(big.Int).Quo(absRat.Num(), absRat.Denom())
		totalWholeSeconds.Set(whole)
		rem := new(big.Rat).Sub(absRat, new(big.Rat).SetInt(whole))
		if rem.Sign() != 0 {
			fracRat = rem
		}
	} else {
		totalSeconds := d.Seconds
		if d.Negative && totalSeconds < 0 {
			totalSeconds = -totalSeconds
		}
		whole := int64(totalSeconds)
		fracSeconds := totalSeconds - float64(whole)
		// Round fractional part to microseconds
		fracMicro = int64(math.Round(fracSeconds * 1e6))
		if fracMicro >= 1e6 {
			whole++
			fracMicro -= 1e6
		}
		if fracMicro < 0 {
			fracMicro = 0
		}
		totalWholeSeconds.SetInt64(whole)
		// Use FracSec for exact fractional representation if available
		if d.FracSec != nil && d.FracSec.Sign() != 0 {
			fracMicro = 0 // will be formatted from FracSec below
			fracRat = d.FracSec
		}
	}

	// Decompose days/hours/minutes/seconds entirely in big.Int via QuoRem so the
	// days component can exceed int64 without overflow; the sub-day components are
	// bounded and safely fit in int64.
	days := new(big.Int)
	rem := new(big.Int)
	days.QuoRem(totalWholeSeconds, big.NewInt(86400), rem)
	hours := rem.Int64() / 3600
	subHour := rem.Int64() - hours*3600
	mins := subHour / 60
	wholeSecs := subHour - mins*60

	hasFrac := fracMicro != 0 || (fracRat != nil && fracRat.Sign() != 0)
	hasSecs := wholeSecs != 0 || hasFrac
	if days.Sign() != 0 {
		fmt.Fprintf(&b, "%sD", days.String())
	}
	if hours != 0 || mins != 0 || hasSecs {
		b.WriteByte('T')
		if hours != 0 {
			fmt.Fprintf(&b, "%dH", hours)
		}
		if mins != 0 {
			fmt.Fprintf(&b, "%dM", mins)
		}
		if hasSecs {
			if fracRat != nil && fracRat.Sign() != 0 {
				// Use exact fractional representation
				fracStr := exactFractionDigits(fracRat)
				if fracStr == "" {
					fmt.Fprintf(&b, "%dS", wholeSecs)
				} else {
					fmt.Fprintf(&b, "%d.%sS", wholeSecs, fracStr)
				}
			} else if fracMicro == 0 {
				fmt.Fprintf(&b, "%dS", wholeSecs)
			} else {
				frac := fmt.Sprintf("%06d", fracMicro)
				frac = strings.TrimRight(frac, "0")
				fmt.Fprintf(&b, "%d.%sS", wholeSecs, frac)
			}
		}
	}

	if b.Len() == 1 || (d.Negative && b.Len() == 2) {
		// Zero duration: yearMonthDuration → "P0M", all others → "PT0S"
		if typeName == TypeYearMonthDuration {
			b.WriteString("0M")
		} else {
			b.WriteString("T0S")
		}
	}

	return b.String()
}
