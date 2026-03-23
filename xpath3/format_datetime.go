package xpath3

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func fnFormatDateTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return formatDateTimeCommon(ctx, args, TypeDateTime)
}

func fnFormatDate(ctx context.Context, args []Sequence) (Sequence, error) {
	return formatDateTimeCommon(ctx, args, TypeDate)
}

func fnFormatTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return formatDateTimeCommon(ctx, args, TypeTime)
}

func formatDateTimeCommon(ctx context.Context, args []Sequence, typeName string) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return SingleString(""), nil
	}

	valAtom, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}

	// Cast to the appropriate type if needed
	if valAtom.TypeName != typeName {
		valAtom, err = CastAtomic(valAtom, typeName)
		if err != nil {
			return nil, err
		}
	}

	t := valAtom.TimeVal()

	picture, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}

	lang := "en"
	if ec := getFnContext(ctx); ec != nil {
		lang = ec.getDefaultLanguage()
	}
	calendar := ""
	if len(args) > 2 && seqLen(args[2]) > 0 {
		lang, err = coerceArgToString(args[2])
		if err != nil {
			return nil, err
		}
		if lang == "" {
			return nil, &XPathError{Code: errCodeFOFD1340, Message: "format-dateTime: language argument must not be empty"}
		}
	}
	if len(args) > 3 && seqLen(args[3]) > 0 {
		calendar, err = coerceArgToString(args[3])
		if err != nil {
			return nil, err
		}
		if err := validateCalendarName(calendar); err != nil {
			return nil, err
		}
	}

	if len(args) > 4 && seqLen(args[4]) > 0 {
		place, err := coerceArgToString(args[4])
		if err != nil {
			return nil, err
		}
		if place != "" {
			loc, err := time.LoadLocation(place)
			if err != nil {
				return nil, &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid place: %s", place)}
			}
			if t.Location() == noTZLocation {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
			} else {
				t = t.In(loc)
			}
		}
	}

	result, err := formatDateTimePicture(t, picture, lang, calendar, typeName)
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

// formatDateTimePicture formats a time.Time value using an XPath picture string.
func formatDateTimePicture(t time.Time, picture, lang, calendar, typeName string) (string, error) {
	var b strings.Builder

	// Language fallback: if the requested language is not "en", fall back
	// and prepend [Language: en] per XPath F&O §9.8.4.
	effectiveLang := lang
	if effectiveLang == "" {
		effectiveLang = "en"
	}
	if !isSupportedLanguage(effectiveLang) {
		b.WriteString("[Language: en]")
		effectiveLang = "en"
	}

	// Calendar fallback: if a non-ISO calendar is requested, fall back
	// to the Gregorian/AD calendar and prepend [Calendar: AD].
	effectiveCalendar := calendar
	if effectiveCalendar != "" && !isISOCalendar(effectiveCalendar) {
		b.WriteString("[Calendar: AD]")
		effectiveCalendar = "ISO"
	}

	i := 0
	runes := []rune(picture)

	for i < len(runes) {
		if runes[i] == '[' {
			if i+1 < len(runes) && runes[i+1] == '[' {
				// Escaped '[['  → literal '['
				b.WriteRune('[')
				i += 2
				continue
			}
			// Find matching ']'
			end := -1
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == ']' {
					end = j
					break
				}
			}
			if end < 0 {
				return "", &XPathError{Code: errCodeFOFD1340, Message: "unclosed '[' in picture string"}
			}
			component := string(runes[i+1 : end])
			formatted, err := formatComponent(t, component, effectiveLang, effectiveCalendar, typeName)
			if err != nil {
				return "", err
			}
			b.WriteString(formatted)
			i = end + 1
		} else if runes[i] == ']' {
			if i+1 < len(runes) && runes[i+1] == ']' {
				b.WriteRune(']')
				i += 2
				continue
			}
			b.WriteRune(runes[i])
			i++
		} else {
			b.WriteRune(runes[i])
			i++
		}
	}

	return b.String(), nil
}

// isSupportedLanguage returns true if the language is supported for formatting.
func isSupportedLanguage(lang string) bool {
	switch lang {
	case "en", "de":
		return true
	default:
		return false
	}
}

// formatComponent handles a single [component] specifier.
func formatComponent(t time.Time, spec, lang, calendar, typeName string) (string, error) {
	// Strip whitespace within the spec
	spec = stripSpaces(spec)

	if len(spec) == 0 {
		return "", &XPathError{Code: errCodeFOFD1340, Message: "empty component specifier"}
	}

	// First character is the component letter
	compChar := spec[0]
	rest := spec[1:]

	// Validate component is applicable for the type (FOFD1350)
	if err := validateComponentForType(compChar, typeName); err != nil {
		return "", err
	}

	// Parse presentation modifier and width
	presentation, width := parseDatePresentation(rest)

	// Validate format token and width
	if err := validateDateFormatToken(compChar, presentation, width); err != nil {
		return "", err
	}

	var value int64
	switch compChar {
	case 'Y':
		value = int64(t.Year())
	case 'M':
		value = int64(t.Month())
	case 'D':
		value = int64(t.Day())
	case 'd':
		value = int64(t.YearDay())
	case 'F':
		return formatDayOfWeek(t, presentation, width, lang), nil
	case 'H':
		value = int64(t.Hour())
	case 'h':
		h := t.Hour() % 12
		if h == 0 {
			h = 12
		}
		value = int64(h)
	case 'P':
		return formatAMPM(t, presentation, width), nil
	case 'm':
		value = int64(t.Minute())
	case 's':
		value = int64(t.Second())
	case 'f':
		return formatFractionalSeconds(t, presentation, width), nil
	case 'Z', 'z':
		return formatTimezone(t, compChar, presentation, width), nil
	case 'W':
		_, w := t.ISOWeek()
		value = int64(w)
	case 'w':
		if isISOCalendar(calendar) {
			value = int64(isoWeekOfMonth(t))
		} else {
			value = int64((t.Day()-1)/7 + 1)
		}
	case 'E':
		return formatEra(t.Year(), presentation), nil
	case 'C':
		return "ISO", nil
	default:
		return "", &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("unknown component specifier: %c", compChar)}
	}

	return formatDateTimeValue(value, compChar, presentation, width, lang), nil
}

func stripSpaces(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type dtPresentation struct {
	format        string // the format token (e.g., "01", "1", "Nn", "n", "N", "I", etc.)
	ordinal       bool
	isTraditional bool
	implicit      bool
}

type dtWidth struct {
	minWidth int
	maxWidth int // -1 = unlimited
}

func parseDatePresentation(rest string) (dtPresentation, dtWidth) {
	p := dtPresentation{format: "1", implicit: true}
	w := dtWidth{minWidth: -1, maxWidth: -1}

	if rest == "" {
		return p, w
	}

	// Split on comma for width specifier
	commaIdx := strings.LastIndex(rest, ",")
	formatPart := rest
	widthPart := ""
	if commaIdx >= 0 {
		formatPart = rest[:commaIdx]
		widthPart = rest[commaIdx+1:]
	}

	// Parse format part
	if formatPart != "" {
		// Only the final ';' introduces a presentation modifier; earlier
		// semicolons remain part of the decimal picture as grouping separators.
		semiIdx := strings.LastIndex(formatPart, ";")
		if semiIdx >= 0 && isDatePresentationModifier(formatPart[semiIdx+1:]) {
			modPart := formatPart[semiIdx+1:]
			formatPart = formatPart[:semiIdx]
			for _, c := range modPart {
				switch c {
				case 'o':
					p.ordinal = true
				case 't':
					p.isTraditional = true
				case 'c':
					// Calendar modifier is accepted but not otherwise interpreted here.
				}
			}
		}
		if semiIdx < 0 {
			suffixStart := len(formatPart)
			for suffixStart > 0 {
				switch formatPart[suffixStart-1] {
				case 'o', 't', 'c':
					suffixStart--
				default:
					goto suffixDone
				}
			}
		suffixDone:
			if suffixStart < len(formatPart) && suffixStart > 0 {
				modPart := formatPart[suffixStart:]
				formatPart = formatPart[:suffixStart]
				for _, c := range modPart {
					switch c {
					case 'o':
						p.ordinal = true
					case 't':
						p.isTraditional = true
					case 'c':
					}
				}
			}
		}
		if formatPart != "" {
			p.format = formatPart
			p.implicit = false
		}
	}

	// Parse width part
	if widthPart != "" {
		parts := strings.Split(widthPart, "-")
		if len(parts) == 1 {
			if parts[0] == "*" {
				// No constraint
			} else {
				n := parseSimpleInt(parts[0])
				if n > 0 {
					w.minWidth = n
					w.maxWidth = n
				}
			}
		} else if len(parts) == 2 {
			if parts[0] != "*" {
				w.minWidth = parseSimpleInt(parts[0])
			}
			if parts[1] != "*" {
				w.maxWidth = parseSimpleInt(parts[1])
			}
		}
	}

	return p, w
}

func isDatePresentationModifier(mod string) bool {
	if mod == "" {
		return false
	}
	for _, r := range mod {
		switch r {
		case 'o', 't', 'c':
		default:
			return false
		}
	}
	return true
}

func parseSimpleInt(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		} else {
			break
		}
	}
	return n
}

func formatDateTimeValue(value int64, comp byte, p dtPresentation, w dtWidth, lang string) string {
	format := p.format
	numericValue := normalizeDateNumericValue(value, comp, w)

	switch format {
	case "N", "n", "Nn":
		// Named format (for months and days)
		return formatNamedValue(value, comp, format, w, lang)
	case "A":
		return formatAlpha(new(big.Int).SetInt64(numericValue), true)
	case "a":
		return formatAlpha(new(big.Int).SetInt64(numericValue), false)
	case "I", "i":
		// Roman numerals
		n := new(big.Int).SetInt64(numericValue)
		return applyTextWidth(formatRoman(n, format == "I"), w)
	case "W", "w", "Ww":
		n := new(big.Int).SetInt64(numericValue)
		style := "lower"
		switch format {
		case "W":
			style = "upper"
		case "w":
			style = "lower"
		case "Ww":
			style = "title"
		}
		result := formatWords(n, lang, style)
		if p.ordinal {
			result = applyOrdinalWords(result, lang, style, "")
		}
		return result
	}

	// Decimal number formatting
	result := formatDateDecimal(numericValue, p, w, comp)

	if p.ordinal {
		n := new(big.Int).SetInt64(value)
		result = applyOrdinalDecimal(result, n, lang, "")
	}

	return result
}

func formatDateDecimal(value int64, p dtPresentation, w dtWidth, comp byte) string {
	format := p.format
	// Determine min digits from format token
	minDigits := 0
	digitSigns := 0
	zeroDigit := '0'

	runes := []rune(format)
	for _, r := range runes {
		if unicode.IsDigit(r) {
			zeroDigit = unicodeDigitZero(r)
			minDigits++
			digitSigns++
		} else if r == '#' {
			digitSigns++
		}
	}

	if minDigits == 0 {
		minDigits = 1
	}

	// Apply width constraints
	if p.implicit && (comp == 'm' || comp == 's') && w.minWidth < 0 {
		minDigits = 2
	}
	if w.minWidth > 0 {
		minDigits = w.minWidth
	}

	// For year with max width, truncate
	maxWidth := w.maxWidth
	if maxWidth < 0 && (comp == 'Y' || comp == 'E') && digitSigns > 1 {
		maxWidth = digitSigns
	}

	abs := value
	neg := false
	if abs < 0 {
		abs = -abs
		neg = true
	}

	s := fmt.Sprintf("%d", abs)

	// Pad to minimum digits
	for len(s) < minDigits {
		s = "0" + s
	}

	// Truncate to max width (for year, take rightmost digits)
	if maxWidth > 0 && len(s) > maxWidth {
		s = s[len(s)-maxWidth:]
	}

	// Apply grouping separators from format pattern
	s = applyDateGrouping(s, runes, zeroDigit)

	// Translate to target digit set
	if zeroDigit != '0' {
		s = translateDigits(s, zeroDigit)
	}

	if neg {
		s = "-" + s
	}

	return s
}

func normalizeDateNumericValue(value int64, comp byte, w dtWidth) int64 {
	if comp != 'Y' && comp != 'E' {
		return value
	}

	if value < 0 {
		value = -value
	}

	if w.maxWidth > 0 {
		mod := int64(1)
		for i := 0; i < w.maxWidth; i++ {
			mod *= 10
		}
		value %= mod
	}
	return value
}

func applyTextWidth(s string, w dtWidth) string {
	if w.minWidth <= 0 {
		return s
	}
	for utf8.RuneCountInString(s) < w.minWidth {
		s += " "
	}
	return s
}

func applyDateGrouping(s string, formatRunes []rune, zeroDigit rune) string {
	// Extract grouping separator positions from the format pattern
	var sepPositions []int // positions from the right
	var sepChars []rune
	digitCount := 0
	for i := len(formatRunes) - 1; i >= 0; i-- {
		r := formatRunes[i]
		if r == '#' || isDecimalDigitInRange(r, zeroDigit) {
			digitCount++
		} else {
			if digitCount > 0 {
				sepPositions = append(sepPositions, digitCount)
				sepChars = append(sepChars, r)
			}
		}
	}

	if len(sepPositions) == 0 {
		return s
	}

	// Insert separators
	sRunes := []rune(s)
	sepAt := make(map[int]rune)
	for i, pos := range sepPositions {
		sepAt[pos] = sepChars[i]
	}

	var result []rune
	for i := len(sRunes) - 1; i >= 0; i-- {
		digitPos := len(sRunes) - 1 - i
		if sep, ok := sepAt[digitPos]; ok {
			result = append(result, sep)
		}
		result = append(result, sRunes[i])
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func formatNamedValue(value int64, comp byte, format string, w dtWidth, lang string) string {
	var name string

	switch comp {
	case 'M':
		if value >= 1 && value <= 12 {
			months := monthNamesForLang(lang)
			name = months[value-1]
		}
	case 'F':
		if value >= 0 && value <= 6 {
			days := dayNamesForLang(lang)
			name = days[value]
		}
	default:
		return fmt.Sprintf("%d", value)
	}

	if name == "" {
		return fmt.Sprintf("%d", value)
	}
	name = applyNameWidth(name, comp, w)

	switch format {
	case "N":
		return strings.ToUpper(name)
	case "n":
		return strings.ToLower(name)
	case "Nn":
		r, size := utf8.DecodeRuneInString(name)
		return string(unicode.ToUpper(r)) + strings.ToLower(name[size:])
	}
	return name
}

func monthNamesForLang(lang string) []string {
	switch strings.ToLower(lang) {
	case "de":
		return []string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	default:
		return []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	}
}

func dayNamesForLang(lang string) []string {
	switch strings.ToLower(lang) {
	case "de":
		return []string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
	default:
		return []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	}
}

func applyNameWidth(name string, comp byte, w dtWidth) string {
	if w.maxWidth <= 0 && w.minWidth <= 0 {
		return name
	}
	target := w.maxWidth
	if w.minWidth > 0 {
		target = w.minWidth
	}
	if target <= 0 {
		return name
	}
	runes := []rune(name)
	if len(runes) <= target {
		return name
	}
	return string(runes[:target])
}

func formatDayOfWeek(t time.Time, p dtPresentation, w dtWidth, lang string) string {
	dow := int64(t.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday
	format := p.format

	switch format {
	case "N", "n", "Nn":
		return formatNamedValue(dow, 'F', format, w, lang)
	default:
		// Numeric day of week: 1=Monday, ..., 7=Sunday per ISO
		isoDow := int64(t.Weekday())
		if isoDow == 0 {
			isoDow = 7
		}
		return formatDateTimeValue(isoDow, 'F', p, w, lang)
	}
}

func formatAMPM(t time.Time, p dtPresentation, w dtWidth) string {
	var s string
	if t.Hour() < 12 {
		s = "am"
	} else {
		s = "pm"
	}
	switch p.format {
	case "N":
		s = strings.ToUpper(s)
	case "n":
		s = strings.ToLower(s)
	case "Nn":
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	// Apply maximum width constraint: truncate to maxWidth characters
	if w.maxWidth > 0 && len(s) > w.maxWidth {
		s = s[:w.maxWidth]
	}
	return s
}

func formatFractionalSeconds(t time.Time, p dtPresentation, w dtWidth) string {
	ns := t.Nanosecond()

	// Determine the zero digit from presentation format
	zeroDigit := rune('0')
	for _, r := range p.format {
		if unicode.IsDigit(r) {
			zeroDigit = unicodeDigitZero(r)
			break
		}
	}

	// Count mandatory digits and optional (#) digits, and find grouping separators
	mandatoryDigits := 0
	optionalDigits := 0
	totalDigits := 0
	type groupInfo struct {
		pos int  // position from left (after how many digits)
		sep rune // separator character
	}
	var groups []groupInfo
	for _, r := range p.format {
		if isDecimalDigitInRange(r, zeroDigit) {
			mandatoryDigits++
			totalDigits++
		} else if r == '#' {
			optionalDigits++
			totalDigits++
		} else {
			if totalDigits > 0 {
				groups = append(groups, groupInfo{pos: totalDigits, sep: r})
			}
		}
	}
	if mandatoryDigits == 0 && optionalDigits == 0 {
		mandatoryDigits = 1
		totalDigits = 1
	}

	// For fractional seconds:
	// - Single digit format token ("1") → min=1, max=9 (show all significant)
	// - Multi-digit format token → exact precision (min=max=digitCount)
	// - # digits → optional trailing (increase max)
	// - Width modifier overrides
	minPlaces := mandatoryDigits
	var maxPlaces int
	if mandatoryDigits <= 1 && optionalDigits == 0 {
		// Single-digit or default: unbounded max (show all significant)
		maxPlaces = 9
	} else {
		maxPlaces = totalDigits
	}

	if w.minWidth > 0 && w.minWidth > minPlaces {
		minPlaces = w.minWidth
	}
	if w.maxWidth > 0 {
		maxPlaces = w.maxWidth
	}
	if maxPlaces < minPlaces {
		maxPlaces = minPlaces
	}
	if minPlaces < 1 {
		minPlaces = 1
	}
	if maxPlaces > 9 {
		maxPlaces = 9
	}

	// Format as decimal fraction
	s := fmt.Sprintf("%09d", ns)
	s = s[:maxPlaces]

	// Trim trailing zeros down to minPlaces
	for len(s) > minPlaces && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}

	// Apply grouping separators from format token
	if len(groups) > 0 {
		var b strings.Builder
		gi := 0
		for i, r := range s {
			if gi < len(groups) && i == groups[gi].pos {
				b.WriteRune(groups[gi].sep)
				gi++
			}
			b.WriteRune(rune(r))
		}
		s = b.String()
	}

	// Translate to target digit set
	if zeroDigit != '0' {
		s = translateDigits(s, zeroDigit)
	}

	return s
}

func formatTimezone(t time.Time, comp byte, p dtPresentation, w dtWidth) string {
	// Check if the time has no explicit timezone
	if !HasTimezone(t) {
		if p.format == "Z" && !p.implicit {
			return "J"
		}
		return ""
	}
	_, offset := t.Zone()

	if p.format == "N" || p.format == "n" || p.format == "Nn" {
		if name := formatTimezoneName(t, p.format); name != "" {
			return name
		}
	}
	if p.format == "Z" && !p.implicit {
		if code, ok := militaryTimezoneCode(offset); ok {
			return code
		}
	}

	spec := parseTimezonePicture(comp, p)
	// When a width modifier is present and limits the width to just the hour
	// portion, show minutes only when non-zero (XSLT 2.0 Erratum E29 behavior).
	if w.maxWidth > 0 && p.implicit && w.maxWidth <= spec.hourWidth {
		spec.showMinutes = false
	}
	return renderTimezoneOffset(offset, spec)
}

func isoWeekOfMonth(t time.Time) int {
	loc := t.Location()
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	weekStart := isoWeekStart(dayStart)
	monthWeek1 := isoMonthWeekOneStart(dayStart)
	if weekStart.Before(monthWeek1) {
		prevMonth := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, loc)
		return isoWeekOfMonth(prevMonth)
	}
	return int(weekStart.Sub(monthWeek1)/(7*24*time.Hour)) + 1
}

func formatEra(year int, p dtPresentation) string {
	era := "AD"
	if year < 0 {
		era = "BC"
	}
	switch p.format {
	case "n":
		return strings.ToLower(era)
	case "Nn":
		return strings.ToUpper(era[:1]) + strings.ToLower(era[1:])
	default:
		return era
	}
}

type timezonePicture struct {
	prefix       string
	hourWidth    int
	minuteWidth  int
	separator    string
	showMinutes  bool
	zeroAsZ      bool
	militaryCode bool
	zeroDigit    rune
}

func parseTimezonePicture(comp byte, p dtPresentation) timezonePicture {
	spec := timezonePicture{
		hourWidth:   2,
		minuteWidth: 2,
		separator:   ":",
		showMinutes: true,
		zeroDigit:   '0',
	}
	if comp == 'z' {
		spec.prefix = "GMT"
	}
	if p.implicit {
		return spec
	}

	format := p.format
	if strings.HasSuffix(format, "t") {
		spec.zeroAsZ = true
		format = strings.TrimSuffix(format, "t")
	} else if p.isTraditional {
		spec.zeroAsZ = true
	}
	if format == "Z" {
		spec.militaryCode = true
		return spec
	}
	if format == "N" || format == "n" || format == "Nn" {
		return spec
	}

	formatRunes := []rune(format)
	firstNonDigit := len(formatRunes)
	for i, r := range formatRunes {
		if !unicode.IsDigit(r) {
			firstNonDigit = i
			break
		}
		spec.zeroDigit = unicodeDigitZero(r)
	}
	prefixDigits := formatRunes[:firstNonDigit]
	if len(prefixDigits) > 0 {
		if firstNonDigit == len(formatRunes) && len(prefixDigits) > 2 {
			spec.hourWidth = len(prefixDigits) - 2
			spec.minuteWidth = 2
			spec.separator = ""
			spec.showMinutes = true
			return spec
		}
		spec.hourWidth = len(prefixDigits)
	}
	restRunes := formatRunes[firstNonDigit:]
	if len(restRunes) == 0 {
		spec.showMinutes = false
		return spec
	}

	sepEnd := 0
	for sepEnd < len(restRunes) && !unicode.IsDigit(restRunes[sepEnd]) {
		sepEnd++
	}
	spec.separator = string(restRunes[:sepEnd])
	spec.showMinutes = true
	if sepEnd < len(restRunes) {
		spec.minuteWidth = len(restRunes[sepEnd:])
	}
	if spec.minuteWidth == 0 {
		spec.minuteWidth = 2
	}
	return spec
}

func renderTimezoneOffset(offset int, spec timezonePicture) string {
	if spec.militaryCode {
		if code, ok := militaryTimezoneCode(offset); ok {
			return code
		}
	}
	if spec.zeroAsZ && offset == 0 {
		return "Z"
	}

	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60

	hourFmt := fmt.Sprintf("%%0%dd", spec.hourWidth)
	body := sign + fmt.Sprintf(hourFmt, hours)
	if spec.showMinutes {
		minuteFmt := fmt.Sprintf("%%0%dd", spec.minuteWidth)
		body += spec.separator + fmt.Sprintf(minuteFmt, minutes)
	} else if minutes != 0 {
		minuteFmt := fmt.Sprintf("%%0%dd", spec.minuteWidth)
		body += ":" + fmt.Sprintf(minuteFmt, minutes)
	}
	if spec.zeroDigit != '0' {
		body = translateDigits(body, spec.zeroDigit)
	}
	return spec.prefix + body
}

func formatTimezoneName(t time.Time, format string) string {
	name, _ := t.Zone()
	if name == "" {
		return ""
	}
	switch format {
	case "N":
		return strings.ToUpper(name)
	case "n":
		return strings.ToLower(name)
	default:
		return name
	}
}

func militaryTimezoneCode(offset int) (string, bool) {
	if offset%3600 != 0 {
		return "", false
	}
	hours := offset / 3600
	switch {
	case hours == 0:
		return "Z", true
	case hours >= 1 && hours <= 12:
		letters := "ABCDEFGHIKLM"
		return string(letters[hours-1]), true
	case hours <= -1 && hours >= -12:
		letters := "NOPQRSTUVWXY"
		return string(letters[-hours-1]), true
	default:
		return "", false
	}
}

func isISOCalendar(calendar string) bool {
	switch calendar {
	case "ISO", "Q{}ISO", "AD", "Q{}AD":
		return true
	default:
		return false
	}
}

func validateCalendarName(calendar string) error {
	if calendar == "" {
		return nil
	}
	if isKnownCalendarName(calendar) {
		return nil
	}
	if strings.HasPrefix(calendar, "Q{") {
		end := strings.Index(calendar, "}")
		if end < 0 || end == len(calendar)-1 {
			return &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid calendar: %s", calendar)}
		}
		uri := calendar[2:end]
		local := calendar[end+1:]
		if !isNCName(local) {
			return &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid calendar: %s", calendar)}
		}
		if uri == "" && !isKnownCalendarName(local) {
			return &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid calendar: %s", calendar)}
		}
		return nil
	}
	if strings.Contains(calendar, ":") {
		parts := strings.Split(calendar, ":")
		if len(parts) != 2 || !isNCName(parts[0]) || !isNCName(parts[1]) {
			return &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid calendar: %s", calendar)}
		}
		return nil
	}
	return &XPathError{Code: errCodeFOFD1340, Message: fmt.Sprintf("format-dateTime: invalid calendar: %s", calendar)}
}

func isNCName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

func isoWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

func isoMonthWeekOneStart(t time.Time) time.Time {
	loc := t.Location()
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	firstWeekday := int(first.Weekday())
	if firstWeekday == 0 {
		firstWeekday = 7
	}
	offsetToThursday := (4 - firstWeekday + 7) % 7
	firstThursday := first.AddDate(0, 0, offsetToThursday)
	return isoWeekStart(firstThursday)
}

func isKnownCalendarName(calendar string) bool {
	switch calendar {
	case "ISO", "AD", "AH", "AM", "AP", "AS", "BE", "CB", "CE", "CL",
		"CS", "EE", "FE", "JE", "KE", "KY", "ME", "MS", "NS",
		"OS", "RS", "SE", "SH", "SS", "TE", "US":
		return true
	default:
		return false
	}
}

// validateComponentForType checks that a component specifier is valid for the given type.
// Per XPath F&O §9.8.4: raises FOFD1350 if the component is not available.
func validateComponentForType(comp byte, typeName string) error {
	// Date components: Y, M, D, d, F, W, w, E, C
	// Time components: H, h, P, m, s, f
	// Both: Z, z
	switch typeName {
	case TypeTime:
		switch comp {
		case 'H', 'h', 'P', 'm', 's', 'f', 'Z', 'z':
			return nil
		default:
			return &XPathError{Code: errCodeFOFD1350, Message: fmt.Sprintf("component [%c] is not available for xs:time", comp)}
		}
	case TypeDate:
		switch comp {
		case 'Y', 'M', 'D', 'd', 'F', 'W', 'w', 'E', 'C', 'Z', 'z':
			return nil
		default:
			return &XPathError{Code: errCodeFOFD1350, Message: fmt.Sprintf("component [%c] is not available for xs:date", comp)}
		}
	}
	// xs:dateTime allows all components
	return nil
}

// validateDateFormatToken validates the format token and width for FOFD1340 errors.
func validateDateFormatToken(comp byte, p dtPresentation, w dtWidth) error {
	format := p.format

	// Check for mixed Unicode digit families
	var firstZero rune
	hasDigit := false
	for _, r := range format {
		if unicode.IsDigit(r) {
			z := unicodeDigitZero(r)
			if hasDigit && z != firstZero {
				return &XPathError{Code: errCodeFOFD1340, Message: "mixed Unicode digit families in format token"}
			}
			firstZero = z
			hasDigit = true
		}
	}

	// For non-fractional components: '#' after digits is invalid
	// For fractional seconds ('f'): '#' after digits means optional trailing digits
	if comp != 'f' {
		seenDigit := false
		for _, r := range format {
			if unicode.IsDigit(r) {
				seenDigit = true
			} else if r == '#' {
				if seenDigit {
					return &XPathError{Code: errCodeFOFD1340, Message: "# after digit in format token"}
				}
			}
		}
	} else {
		// For fractional seconds: '#' BEFORE digits is invalid
		seenHash := false
		for _, r := range format {
			if r == '#' {
				seenHash = true
			} else if unicode.IsDigit(r) {
				if seenHash {
					return &XPathError{Code: errCodeFOFD1340, Message: "# before digit in fractional seconds format"}
				}
			}
		}
	}

	// Validate width modifier
	if w.minWidth == 0 {
		return &XPathError{Code: errCodeFOFD1340, Message: "minimum width cannot be zero"}
	}
	if w.maxWidth == 0 {
		return &XPathError{Code: errCodeFOFD1340, Message: "maximum width cannot be zero"}
	}
	if w.minWidth > 0 && w.maxWidth > 0 && w.minWidth > w.maxWidth {
		return &XPathError{Code: errCodeFOFD1340, Message: "minimum width exceeds maximum width"}
	}

	return nil
}
