package xpath3

import (
	"context"
	"math"
	"math/big"
	"time"
)

func init() {
	// Constructors
	registerFn("dateTime", 2, 2, fnDateTime)

	// dateTime accessors
	registerFn("year-from-dateTime", 1, 1, fnYearFromDateTime)
	registerFn("month-from-dateTime", 1, 1, fnMonthFromDateTime)
	registerFn("day-from-dateTime", 1, 1, fnDayFromDateTime)
	registerFn("hours-from-dateTime", 1, 1, fnHoursFromDateTime)
	registerFn("minutes-from-dateTime", 1, 1, fnMinutesFromDateTime)
	registerFn("seconds-from-dateTime", 1, 1, fnSecondsFromDateTime)
	registerFn("timezone-from-dateTime", 1, 1, fnTimezoneFromDateTime)

	// date accessors
	registerFn("year-from-date", 1, 1, fnYearFromDate)
	registerFn("month-from-date", 1, 1, fnMonthFromDate)
	registerFn("day-from-date", 1, 1, fnDayFromDate)
	registerFn("timezone-from-date", 1, 1, fnTimezoneFromDate)

	// time accessors
	registerFn("hours-from-time", 1, 1, fnHoursFromTime)
	registerFn("minutes-from-time", 1, 1, fnMinutesFromTime)
	registerFn("seconds-from-time", 1, 1, fnSecondsFromTime)
	registerFn("timezone-from-time", 1, 1, fnTimezoneFromTime)

	// duration accessors
	registerFn("years-from-duration", 1, 1, fnYearsFromDuration)
	registerFn("months-from-duration", 1, 1, fnMonthsFromDuration)
	registerFn("days-from-duration", 1, 1, fnDaysFromDuration)
	registerFn("hours-from-duration", 1, 1, fnHoursFromDuration)
	registerFn("minutes-from-duration", 1, 1, fnMinutesFromDuration)
	registerFn("seconds-from-duration", 1, 1, fnSecondsFromDuration)

	// timezone adjustment
	registerFn("adjust-dateTime-to-timezone", 1, 2, fnAdjustDateTimeToTimezone)
	registerFn("adjust-date-to-timezone", 1, 2, fnAdjustDateToTimezone)
	registerFn("adjust-time-to-timezone", 1, 2, fnAdjustTimeToTimezone)

	// Formatting stubs
	registerFn("format-dateTime", 2, 5, fnFormatDateTime)
	registerFn("format-date", 2, 5, fnFormatDate)
	registerFn("format-time", 2, 5, fnFormatTime)
}

func extractTime(seq Sequence, allowedTypes ...string) (time.Time, bool, error) {
	if len(seq) == 0 {
		return time.Time{}, false, nil
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return time.Time{}, false, err
	}
	t, ok := a.Value.(time.Time)
	if !ok {
		return time.Time{}, false, &XPathError{Code: errCodeXPTY0004, Message: "expected " + allowedTypes[0] + ", got " + a.TypeName}
	}
	if len(allowedTypes) > 0 {
		matched := false
		for _, at := range allowedTypes {
			if isSubtypeOf(a.TypeName, at) {
				matched = true
				break
			}
		}
		if !matched {
			return time.Time{}, false, &XPathError{Code: errCodeXPTY0004, Message: "expected " + allowedTypes[0] + ", got " + a.TypeName}
		}
	}
	return t, true, nil
}

func extractDuration(seq Sequence, allowedTypes ...string) (Duration, bool, error) {
	if len(seq) == 0 {
		return Duration{}, false, nil
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return Duration{}, false, err
	}
	d, ok := a.Value.(Duration)
	if !ok {
		return Duration{}, false, &XPathError{Code: errCodeXPTY0004, Message: "expected duration type, got " + a.TypeName}
	}
	if len(allowedTypes) > 0 {
		matched := false
		for _, at := range allowedTypes {
			if isSubtypeOf(a.TypeName, at) {
				matched = true
				break
			}
		}
		if !matched {
			return Duration{}, false, &XPathError{Code: errCodeXPTY0004, Message: "expected " + allowedTypes[0] + ", got " + a.TypeName}
		}
	}
	return d, true, nil
}

// --- Constructors ---

func fnDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	dateA, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	timeA, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	// Coerce xs:untypedAtomic to xs:date / xs:time.
	if dateA.TypeName == TypeUntypedAtomic || dateA.TypeName == TypeString {
		dateA, err = CastAtomic(dateA, TypeDate)
		if err != nil {
			return nil, err
		}
	}
	if timeA.TypeName == TypeUntypedAtomic || timeA.TypeName == TypeString {
		timeA, err = CastAtomic(timeA, TypeTime)
		if err != nil {
			return nil, err
		}
	}
	d, ok := dateA.Value.(time.Time)
	if !ok || dateA.TypeName != TypeDate {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "first arg must be xs:date, got " + dateA.TypeName}
	}
	t, ok := timeA.Value.(time.Time)
	if !ok || timeA.TypeName != TypeTime {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "second arg must be xs:time, got " + timeA.TypeName}
	}

	// Per XPath F&O 3.0 §5.2.1: determine timezone from arguments.
	// If both have timezones, they must be equal. If one has a timezone, use it.
	// If neither has a timezone, the result has none.
	dateHasTZ := HasTimezone(d)
	timeHasTZ := HasTimezone(t)
	var loc *time.Location
	switch {
	case dateHasTZ && timeHasTZ:
		_, doff := d.Zone()
		_, toff := t.Zone()
		if doff != toff {
			return nil, &XPathError{Code: "FORG0008", Message: "date and time timezone values are not equal"}
		}
		loc = d.Location()
	case dateHasTZ:
		loc = d.Location()
	case timeHasTZ:
		loc = t.Location()
	default:
		loc = noTZLocation // no timezone
	}

	combined := time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: combined}), nil
}

// --- dateTime accessors ---

func fnYearFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Year())), nil
}

func fnMonthFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Month())), nil
}

func fnDayFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Day())), nil
}

func fnHoursFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Hour())), nil
}

func fnMinutesFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Minute())), nil
}

func fnSecondsFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	return SingleDouble(sec), nil
}

func fnTimezoneFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if !HasTimezone(t) {
		return nil, nil
	}
	_, offset := t.Zone()
	secs := float64(offset)
	neg := secs < 0
	if neg {
		secs = -secs
	}
	d := Duration{Seconds: secs, Negative: neg}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}

// --- date accessors ---

func fnYearFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Year())), nil
}

func fnMonthFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Month())), nil
}

func fnDayFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Day())), nil
}

func fnTimezoneFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if !HasTimezone(t) {
		return nil, nil
	}
	_, offset := t.Zone()
	secs := float64(offset)
	neg := secs < 0
	if neg {
		secs = -secs
	}
	d := Duration{Seconds: secs, Negative: neg}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}

// --- time accessors ---

func fnHoursFromTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Hour())), nil
}

func fnMinutesFromTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Minute())), nil
}

func fnSecondsFromTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	return SingleDouble(sec), nil
}

func fnTimezoneFromTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if !HasTimezone(t) {
		return nil, nil
	}
	_, offset := t.Zone()
	secs := float64(offset)
	neg := secs < 0
	if neg {
		secs = -secs
	}
	d := Duration{Seconds: secs, Negative: neg}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}

// --- duration accessors ---

func fnYearsFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	years := d.Months / 12
	if d.Negative {
		years = -years
	}
	return SingleInteger(int64(years)), nil
}

func fnMonthsFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	months := d.Months % 12
	if d.Negative {
		months = -months
	}
	return SingleInteger(int64(months)), nil
}

func fnDaysFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	days := int(d.Seconds) / 86400
	if d.Negative {
		days = -days
	}
	return SingleInteger(int64(days)), nil
}

func fnHoursFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	hours := (int(d.Seconds) % 86400) / 3600
	if d.Negative {
		hours = -hours
	}
	return SingleInteger(int64(hours)), nil
}

func fnMinutesFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	minutes := (int(d.Seconds) % 3600) / 60
	if d.Negative {
		minutes = -minutes
	}
	return SingleInteger(int64(minutes)), nil
}

func fnSecondsFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok, err := extractDuration(args[0], TypeDuration)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	// Use exact arithmetic: integer total seconds mod 60 + exact fractional part
	intSec := int64(d.Seconds)
	wholeSec := intSec % 60
	result := new(big.Rat).SetInt64(wholeSec)
	if d.FracSec != nil {
		result.Add(result, d.FracSec)
	}
	if d.Negative {
		result.Neg(result)
	}
	return SingleDecimal(result), nil
}

// --- timezone adjustment ---

func fnAdjustDateTimeToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDateTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone: keep local components
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), noTZLocation)
		return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: t}), nil
	}
	loc, err := getTargetTimezone(ctx, args)
	if err != nil {
		return nil, err
	}
	if !HasTimezone(t) {
		// No timezone — attach the target timezone (same local time)
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	} else {
		// Has timezone — convert
		t = t.In(loc)
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: t}), nil
}

func fnAdjustDateToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone: keep local date
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, noTZLocation)
		return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: t}), nil
	}
	loc, err := getTargetTimezone(ctx, args)
	if err != nil {
		return nil, err
	}
	if !HasTimezone(t) {
		// No timezone — attach the target timezone (same local date)
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	} else {
		// Has timezone — convert via dateTime (T00:00:00), then extract date
		dt := t.In(loc)
		t = time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, loc)
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: t}), nil
}

func fnAdjustTimeToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok, err := extractTime(args[0], TypeTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone: keep local time components
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), noTZLocation)
		return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: t}), nil
	}
	loc, err := getTargetTimezone(ctx, args)
	if err != nil {
		return nil, err
	}
	if !HasTimezone(t) {
		// No timezone — attach the target timezone (same local time)
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	} else {
		// Has timezone — convert
		t = t.In(loc)
	}
	return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: t}), nil
}

// getTargetTimezone extracts the target timezone from the second argument (if provided)
// or falls back to the implicit timezone from the dynamic context.
func getTargetTimezone(ctx context.Context, args []Sequence) (*time.Location, error) {
	if len(args) > 1 && len(args[1]) > 0 {
		d, ok, err := extractDuration(args[1], TypeDayTimeDuration)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected dayTimeDuration"}
		}
		if err := validateTimezoneOffset(d); err != nil {
			return nil, err
		}
		return durationToLocation(d), nil
	}
	if ec := getFnContext(ctx); ec != nil {
		return ec.getImplicitTimezone(), nil
	}
	return time.Local, nil
}

// validateTimezoneOffset checks that a duration used as a timezone offset is
// within the allowed range (-PT14H to PT14H) and represents whole minutes.
// Returns FODT0003 if the constraints are violated.
func validateTimezoneOffset(d Duration) error {
	secs := d.Seconds
	if secs < 0 {
		secs = -secs
	}
	if secs > 50400 { // 14 * 3600
		return &XPathError{Code: "FODT0003", Message: "timezone offset out of range (-PT14H to PT14H)"}
	}
	if math.Mod(secs, 60) != 0 {
		return &XPathError{Code: "FODT0003", Message: "timezone offset must be a whole number of minutes"}
	}
	return nil
}

func durationToLocation(d Duration) *time.Location {
	offset := int(d.Seconds)
	if d.Negative {
		offset = -offset
	}
	return time.FixedZone("", offset)
}
