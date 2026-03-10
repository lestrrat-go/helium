package xpath3

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func init() {
	registerFn("parse-ietf-date", 1, 1, fnParseIETFDate)
}

func fnParseIETFDate(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])
	t, err := parseIETFDate(s)
	if err != nil {
		return nil, &XPathError{
			Code:    "FORG0010",
			Message: fmt.Sprintf("fn:parse-ietf-date: invalid date string: %s", err),
		}
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: t}), nil
}

// parseIETFDate parses an IETF date string (RFC 2822, RFC 850, asctime, and
// variations) into a time.Time value. This implements the XPath 3.1
// fn:parse-ietf-date function.
//
// Accepted formats (per XPath 3.1 spec section 10.6):
//   - RFC 2822: "Wed, 06 Jun 1994 07:29:35 GMT"
//   - RFC 850:  "Sunday, 06-Nov-94 08:49:37 GMT"
//   - asctime:  "Wed Jun 06 11:54:45 EST 2013"
//   - Various with optional day-of-week, flexible separators
func parseIETFDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	p := &ietfDateParser{input: s, pos: 0}
	return p.parse()
}

type ietfDateParser struct {
	input string
	pos   int
}

func (p *ietfDateParser) parse() (time.Time, error) {
	p.skipWS()

	// Try to detect format by looking for comma (RFC 2822 style) or month first (asctime)
	// Optional day-of-week prefix
	saved := p.pos
	if p.tryDayOfWeek() {
		p.skipWS()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++ // skip comma
			p.skipWS()
		}
		p.skipWS()
	} else {
		p.pos = saved
	}

	// Determine if this is asctime format (month first) or RFC format (day first)
	if p.peekMonth() {
		return p.parseAsctime()
	}
	return p.parseRFC()
}

// parseRFC parses: day [-] month [-] year time [timezone]
func (p *ietfDateParser) parseRFC() (time.Time, error) {
	day, err := p.readInt()
	if err != nil {
		return time.Time{}, fmt.Errorf("expected day: %w", err)
	}
	p.skipSep()

	monthStr, err := p.readAlpha()
	if err != nil {
		return time.Time{}, fmt.Errorf("expected month: %w", err)
	}
	month, err := parseMonth(monthStr)
	if err != nil {
		return time.Time{}, err
	}
	p.skipSep()

	year, err := p.readInt()
	if err != nil {
		return time.Time{}, fmt.Errorf("expected year: %w", err)
	}
	year = normalizeYear(year)
	p.skipWS()

	hour, min, sec, frac, err := p.parseTime()
	if err != nil {
		return time.Time{}, err
	}

	// Timezone may be immediately adjacent to time (no space)
	loc := p.parseTZ()

	if err := validateIETFDate(year, month, day, hour, min, sec); err != nil {
		return time.Time{}, err
	}

	ns := int(frac * 1e9)
	if hour == 24 {
		hour = 0
		// advance to next day
		t := time.Date(year, time.Month(month), day+1, hour, min, sec, ns, loc)
		return t, nil
	}
	t := time.Date(year, time.Month(month), day, hour, min, sec, ns, loc)
	return t, nil
}

// parseAsctime parses: month day time year [timezone]
// Also handles: month[-]day time year
func (p *ietfDateParser) parseAsctime() (time.Time, error) {
	monthStr, err := p.readAlpha()
	if err != nil {
		return time.Time{}, fmt.Errorf("expected month: %w", err)
	}
	month, err := parseMonth(monthStr)
	if err != nil {
		return time.Time{}, err
	}
	// Require at least one separator between month and day
	if !p.hasSep() {
		return time.Time{}, fmt.Errorf("expected separator after month")
	}
	p.skipSep()

	day, err := p.readInt()
	if err != nil {
		return time.Time{}, fmt.Errorf("expected day: %w", err)
	}
	p.skipWS()

	hour, min, sec, frac, err := p.parseTime()
	if err != nil {
		return time.Time{}, err
	}

	p.skipWS()
	// In asctime format, timezone can come before or after year
	var loc *time.Location
	var year int

	if p.peekTZ() {
		// timezone first, then year
		loc = p.parseTZ()
		p.skipWS()
		year, err = p.readInt()
		if err != nil {
			return time.Time{}, fmt.Errorf("expected year: %w", err)
		}
	} else {
		year, err = p.readInt()
		if err != nil {
			return time.Time{}, fmt.Errorf("expected year: %w", err)
		}
		p.skipWS()
		loc = p.parseTZ()
	}
	year = normalizeYear(year)

	if err := validateIETFDate(year, month, day, hour, min, sec); err != nil {
		return time.Time{}, err
	}

	ns := int(frac * 1e9)
	if hour == 24 {
		hour = 0
		t := time.Date(year, time.Month(month), day+1, hour, min, sec, ns, loc)
		return t, nil
	}
	t := time.Date(year, time.Month(month), day, hour, min, sec, ns, loc)
	return t, nil
}

func (p *ietfDateParser) parseTime() (hour, min, sec int, frac float64, err error) {
	// Handle 24:00:00 as special case
	hour, err = p.readInt()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("expected hour: %w", err)
	}
	if p.pos >= len(p.input) || p.input[p.pos] != ':' {
		return 0, 0, 0, 0, fmt.Errorf("expected ':' after hour")
	}
	p.pos++ // skip ':'

	min, err = p.readInt()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("expected minute: %w", err)
	}

	// Seconds are optional
	if p.pos < len(p.input) && p.input[p.pos] == ':' {
		p.pos++ // skip ':'
		sec, err = p.readInt()
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("expected second: %w", err)
		}
		// Fractional seconds
		if p.pos < len(p.input) && p.input[p.pos] == '.' {
			p.pos++ // skip '.'
			fracStart := p.pos
			for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
				p.pos++
			}
			if p.pos > fracStart {
				frac, _ = strconv.ParseFloat("0."+p.input[fracStart:p.pos], 64)
			}
		}
	}

	return hour, min, sec, frac, nil
}

func (p *ietfDateParser) parseTZ() *time.Location {
	p.skipWS()
	if p.pos >= len(p.input) {
		return time.UTC
	}

	// Check for +/- offset: +0500, -0530, +05:00, +05, -05, -05:
	if p.input[p.pos] == '+' || p.input[p.pos] == '-' {
		sign := 1
		if p.input[p.pos] == '-' {
			sign = -1
		}
		p.pos++
		numStart := p.pos
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == ':') {
			p.pos++
		}
		offsetStr := p.input[numStart:p.pos]
		// Remove trailing colon (e.g., "-05:")
		offsetStr = strings.TrimRight(offsetStr, ":")
		offsetStr = strings.ReplaceAll(offsetStr, ":", "")
		var h, m int
		switch len(offsetStr) {
		case 1, 2:
			h, _ = strconv.Atoi(offsetStr)
		case 3:
			h, _ = strconv.Atoi(offsetStr[:1])
			m, _ = strconv.Atoi(offsetStr[1:3])
		case 4:
			h, _ = strconv.Atoi(offsetStr[:2])
			m, _ = strconv.Atoi(offsetStr[2:4])
		}
		offset := sign * (h*3600 + m*60)
		p.skipComment()
		return time.FixedZone("", offset)
	}

	// Named timezone
	if p.peekAlpha() {
		name, _ := p.readAlpha()
		name = strings.ToUpper(name)
		if offset, ok := ietfTimezones[name]; ok {
			p.skipComment()
			if offset == 0 && (name == "GMT" || name == "UTC" || name == "UT" || name == "Z") {
				return time.FixedZone("", 0) // explicit UTC
			}
			return time.FixedZone("", offset)
		}
	}

	return time.UTC
}

// skipComment skips an optional RFC 822 comment in parentheses: "(EST)"
func (p *ietfDateParser) skipComment() {
	p.skipWS()
	if p.pos < len(p.input) && p.input[p.pos] == '(' {
		depth := 1
		p.pos++
		for p.pos < len(p.input) && depth > 0 {
			switch p.input[p.pos] {
			case '(':
				depth++
			case ')':
				depth--
			}
			p.pos++
		}
	}
}

func (p *ietfDateParser) tryDayOfWeek() bool {
	saved := p.pos
	s, err := p.readAlpha()
	if err != nil {
		p.pos = saved
		return false
	}
	_, ok := dayOfWeekNames[strings.ToLower(s)]
	if !ok {
		p.pos = saved
		return false
	}
	return true
}

func (p *ietfDateParser) peekMonth() bool {
	saved := p.pos
	s, err := p.readAlpha()
	p.pos = saved
	if err != nil {
		return false
	}
	_, ok := monthNames[strings.ToLower(s)]
	return ok
}

func (p *ietfDateParser) peekAlpha() bool {
	return p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos]))
}

// peekTZ checks if the next token looks like a timezone (+/- offset or alpha tz name)
func (p *ietfDateParser) peekTZ() bool {
	if p.pos >= len(p.input) {
		return false
	}
	if p.input[p.pos] == '+' || p.input[p.pos] == '-' {
		return true
	}
	if p.peekAlpha() {
		saved := p.pos
		name, err := p.readAlpha()
		p.pos = saved
		if err != nil {
			return false
		}
		_, ok := ietfTimezones[strings.ToUpper(name)]
		return ok
	}
	return false
}

func (p *ietfDateParser) readInt() (int, error) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	if p.pos == start {
		return 0, fmt.Errorf("expected integer at position %d", p.pos)
	}
	return strconv.Atoi(p.input[start:p.pos])
}

func (p *ietfDateParser) readAlpha() (string, error) {
	start := p.pos
	for p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos])) {
		p.pos++
	}
	if p.pos == start {
		return "", fmt.Errorf("expected alphabetic at position %d", p.pos)
	}
	return p.input[start:p.pos], nil
}

func (p *ietfDateParser) skipWS() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

// hasSep checks if the current position has a separator character
func (p *ietfDateParser) hasSep() bool {
	return p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '-')
}

// skipSep skips optional separators: whitespace, '-', ','
func (p *ietfDateParser) skipSep() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '-') {
		p.pos++
	}
}

func parseMonth(s string) (int, error) {
	m, ok := monthNames[strings.ToLower(s)]
	if !ok {
		return 0, fmt.Errorf("unknown month: %q", s)
	}
	return m, nil
}

func normalizeYear(y int) int {
	if y < 100 {
		// 2-digit year: per XPath 3.1 spec, adjusted to range 1970-2069
		// (same as RFC 2822: 00-69 → 2000-2069, 70-99 → 1970-1999)
		if y <= 69 {
			return 2000 + y
		}
		return 1900 + y
	}
	return y
}

func validateIETFDate(year, month, day, hour, min, sec int) error {
	if month < 1 || month > 12 {
		return fmt.Errorf("month %d out of range", month)
	}
	if day < 1 {
		return fmt.Errorf("day %d out of range", day)
	}
	maxDay := daysInMonth(year, month)
	if day > maxDay {
		return fmt.Errorf("day %d out of range for month %d", day, month)
	}
	if hour > 24 || (hour == 24 && (min != 0 || sec != 0)) {
		return fmt.Errorf("hour %d out of range", hour)
	}
	if min < 0 || min > 59 {
		return fmt.Errorf("minute %d out of range", min)
	}
	if sec < 0 || sec > 60 { // 60 for leap seconds
		return fmt.Errorf("second %d out of range", sec)
	}
	return nil
}

func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 31
}

var monthNames = map[string]int{
	"jan": 1, "january": 1,
	"feb": 2, "february": 2,
	"mar": 3, "march": 3,
	"apr": 4, "april": 4,
	"may": 5,
	"jun": 6, "june": 6,
	"jul": 7, "july": 7,
	"aug": 8, "august": 8,
	"sep": 9, "september": 9,
	"oct": 10, "october": 10,
	"nov": 11, "november": 11,
	"dec": 12, "december": 12,
}

var dayOfWeekNames = map[string]bool{
	"mon": true, "monday": true,
	"tue": true, "tuesday": true,
	"wed": true, "wednesday": true,
	"thu": true, "thursday": true,
	"fri": true, "friday": true,
	"sat": true, "saturday": true,
	"sun": true, "sunday": true,
}

// ietfTimezones maps timezone abbreviations to UTC offsets in seconds.
// Per the XPath spec, only the US military-origin abbreviations are recognized.
var ietfTimezones = map[string]int{
	"UT":  0,
	"UTC": 0,
	"GMT": 0,
	"Z":   0,
	"EST": -5 * 3600,
	"EDT": -4 * 3600,
	"CST": -6 * 3600,
	"CDT": -5 * 3600,
	"MST": -7 * 3600,
	"MDT": -6 * 3600,
	"PST": -8 * 3600,
	"PDT": -7 * 3600,
}
