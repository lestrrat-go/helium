package xpath3

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
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
		Code:    "XPTY0004",
		Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, targetType),
	}
}

// formatXSDTimezone returns the timezone suffix for an XSD date/time value.
func formatXSDTimezone(t time.Time) string {
	if t.Location() == time.UTC {
		return ""
	}
	_, offset := t.Zone()
	if offset == 0 {
		return "Z"
	}
	h := offset / 3600
	m := (offset % 3600) / 60
	if m < 0 {
		m = -m
	}
	return fmt.Sprintf("%+03d:%02d", h, m)
}

func parseXSDDate(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02Z07:00",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return ensureExplicitTZ(t, s), nil
		}
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
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, target); err == nil {
			t = ensureExplicitTZ(t, s)
			if isMidnight24 {
				t = t.AddDate(0, 0, 1) // advance to next day
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:dateTime: %q", s)
}

func parseXSDTime(s string) (time.Time, error) {
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
			return ensureExplicitTZ(t, s), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xs:time: %q", s)
}

// normalizeMidnight24DateTime checks if a dateTime string has T24:00:00 and
// replaces it with T00:00:00. Returns the normalized string and true if it was
// a midnight-24 value. The caller must advance the date by one day.
// 24:00:00.xxx (with fractional seconds) is NOT valid per XSD.
func normalizeMidnight24DateTime(s string) (string, bool) {
	idx := strings.Index(s, "T24:00:00")
	if idx < 0 {
		return s, false
	}
	// Check that minutes and seconds are exactly 00:00 (no fractional seconds allowed with hour 24)
	rest := s[idx+len("T24:00:00"):]
	if len(rest) > 0 && rest[0] == '.' {
		return s, false // 24:00:00.xxx is invalid
	}
	return s[:idx] + "T00:00:00" + rest, true
}

// normalizeMidnight24Time checks if a time string starts with 24:00:00 and
// replaces it with 00:00:00. For xs:time, 24:00:00 equals 00:00:00 (no date rollover).
func normalizeMidnight24Time(s string) (string, bool) {
	if !strings.HasPrefix(s, "24:00:00") {
		return s, false
	}
	rest := s[len("24:00:00"):]
	if len(rest) > 0 && rest[0] == '.' {
		return s, false // 24:00:00.xxx is invalid
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
	return t
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

	inTime := false
	for i < len(s) {
		if s[i] == 'T' {
			inTime = true
			i++
			continue
		}

		numStart := i
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
			i++
		}
		if i == numStart || i >= len(s) {
			return Duration{}, fmt.Errorf("invalid duration: %q", s)
		}

		numStr := s[numStart:i]
		designator := s[i]
		i++

		if !inTime {
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			switch designator {
			case 'Y':
				d.Months += n * 12
			case 'M':
				d.Months += n
			case 'D':
				d.Seconds += float64(n) * 86400
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
		} else {
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return Duration{}, fmt.Errorf("invalid duration number: %q", numStr)
			}
			switch designator {
			case 'H':
				d.Seconds += f * 3600
			case 'M':
				d.Seconds += f * 60
			case 'S':
				d.Seconds += f
			default:
				return Duration{}, fmt.Errorf("invalid duration designator: %c", designator)
			}
		}
	}

	return d, nil
}

// formatDuration formats a Duration as an XSD duration string.
func formatDuration(d Duration) string {
	var b strings.Builder
	if d.Negative {
		b.WriteByte('-')
	}
	b.WriteByte('P')

	years := d.Months / 12
	months := d.Months % 12
	if years != 0 {
		fmt.Fprintf(&b, "%dY", years)
	}
	if months != 0 {
		fmt.Fprintf(&b, "%dM", months)
	}

	totalMicro := int64(math.Round(d.Seconds * 1e6))
	days := totalMicro / (86400 * 1e6)
	totalMicro -= days * 86400 * 1e6
	hours := totalMicro / (3600 * 1e6)
	totalMicro -= hours * 3600 * 1e6
	mins := totalMicro / (60 * 1e6)
	totalMicro -= mins * 60 * 1e6
	wholeSecs := totalMicro / 1e6
	fracMicro := totalMicro % 1e6

	if days != 0 {
		fmt.Fprintf(&b, "%dD", days)
	}
	hasSecs := wholeSecs != 0 || fracMicro != 0
	if hours != 0 || mins != 0 || hasSecs {
		b.WriteByte('T')
		if hours != 0 {
			fmt.Fprintf(&b, "%dH", hours)
		}
		if mins != 0 {
			fmt.Fprintf(&b, "%dM", mins)
		}
		if hasSecs {
			if fracMicro == 0 {
				fmt.Fprintf(&b, "%dS", wholeSecs)
			} else {
				frac := fmt.Sprintf("%06d", fracMicro)
				frac = strings.TrimRight(frac, "0")
				fmt.Fprintf(&b, "%d.%sS", wholeSecs, frac)
			}
		}
	}

	if b.Len() == 1 || (d.Negative && b.Len() == 2) {
		b.WriteString("T0S")
	}

	return b.String()
}
