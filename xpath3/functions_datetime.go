package xpath3

import (
	"context"
	"math"
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

func extractTime(seq Sequence) (time.Time, bool) {
	if len(seq) == 0 {
		return time.Time{}, false
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return time.Time{}, false
	}
	t, ok := a.Value.(time.Time)
	return t, ok
}

func extractDuration(seq Sequence) (Duration, bool) {
	if len(seq) == 0 {
		return Duration{}, false
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return Duration{}, false
	}
	d, ok := a.Value.(Duration)
	return d, ok
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
	d, ok := dateA.Value.(time.Time)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "first arg must be xs:date"}
	}
	t, ok := timeA.Value.(time.Time)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "second arg must be xs:time"}
	}

	// Per XPath F&O 3.0 §5.2.1: determine timezone from arguments.
	// If both have timezones, they must be equal. If one has a timezone, use it.
	// If neither has a timezone, the result has none.
	dateHasTZ := d.Location() != time.UTC
	timeHasTZ := t.Location() != time.UTC
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
		loc = time.UTC // no timezone
	}

	combined := time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: combined}), nil
}

// --- dateTime accessors ---

func fnYearFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Year())), nil
}

func fnMonthFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Month())), nil
}

func fnDayFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Day())), nil
}

func fnHoursFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Hour())), nil
}

func fnMinutesFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Minute())), nil
}

func fnSecondsFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	return SingleDouble(sec), nil
}

func fnTimezoneFromDateTime(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	_, offset := t.Zone()
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	d := Duration{Seconds: float64(hours*3600 + minutes*60)}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}

// --- date accessors (reuse dateTime extractors) ---

func fnYearFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Year())), nil
}

func fnMonthFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Month())), nil
}

func fnDayFromDate(_ context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	return SingleInteger(int64(t.Day())), nil
}

func fnTimezoneFromDate(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnTimezoneFromDateTime(ctx, args)
}

// --- time accessors ---

func fnHoursFromTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnHoursFromDateTime(ctx, args)
}

func fnMinutesFromTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnMinutesFromDateTime(ctx, args)
}

func fnSecondsFromTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnSecondsFromDateTime(ctx, args)
}

func fnTimezoneFromTime(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnTimezoneFromDateTime(ctx, args)
}

// --- duration accessors ---

func fnYearsFromDuration(_ context.Context, args []Sequence) (Sequence, error) {
	d, ok := extractDuration(args[0])
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
	d, ok := extractDuration(args[0])
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
	d, ok := extractDuration(args[0])
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
	d, ok := extractDuration(args[0])
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
	d, ok := extractDuration(args[0])
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
	d, ok := extractDuration(args[0])
	if !ok {
		return nil, nil
	}
	sec := math.Mod(d.Seconds, 60)
	if d.Negative {
		sec = -sec
	}
	return SingleDouble(sec), nil
}

// --- timezone adjustment ---

func fnAdjustDateTimeToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: t}), nil
	}
	if len(args) > 1 {
		d, ok := extractDuration(args[1])
		if !ok {
			return nil, &XPathError{Code: "XPTY0004", Message: "expected dayTimeDuration"}
		}
		offset := int(d.Seconds)
		loc := time.FixedZone("", offset)
		t = t.In(loc)
	} else {
		// Adjust to implicit timezone from the dynamic context
		loc := time.Local
		if ec := getFnContext(ctx); ec != nil {
			loc = ec.getImplicitTimezone()
		}
		t = t.In(loc)
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: t}), nil
}

func fnAdjustDateToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone: keep local date
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: t}), nil
	}
	if len(args) > 1 {
		d, ok := extractDuration(args[1])
		if !ok {
			return nil, &XPathError{Code: "XPTY0004", Message: "expected dayTimeDuration"}
		}
		loc := durationToLocation(d)
		if t.Location() == time.UTC {
			// No timezone — just attach the new timezone
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		} else {
			// Has timezone — convert via dateTime (T00:00:00), then extract date
			// Per XPath F&O §10.7.2
			dt := t.In(loc)
			t = time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, loc)
		}
	} else {
		// No second arg — adjust to implicit timezone from dynamic context
		loc := time.Local
		if ec := getFnContext(ctx); ec != nil {
			loc = ec.getImplicitTimezone()
		}
		if t.Location() == time.UTC {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		} else {
			dt := t.In(loc)
			t = time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, loc)
		}
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: t}), nil
}

func fnAdjustTimeToTimezone(ctx context.Context, args []Sequence) (Sequence, error) {
	t, ok := extractTime(args[0])
	if !ok {
		return nil, nil
	}
	if len(args) > 1 && len(args[1]) == 0 {
		// Remove timezone: keep local time components
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: t}), nil
	}
	if len(args) > 1 {
		d, ok := extractDuration(args[1])
		if !ok {
			return nil, &XPathError{Code: "XPTY0004", Message: "expected dayTimeDuration"}
		}
		loc := durationToLocation(d)
		t = t.In(loc)
	} else {
		// Adjust to implicit timezone from the dynamic context
		loc := time.Local
		if ec := getFnContext(ctx); ec != nil {
			loc = ec.getImplicitTimezone()
		}
		t = t.In(loc)
	}
	return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: t}), nil
}

func durationToLocation(d Duration) *time.Location {
	offset := int(d.Seconds)
	if d.Negative {
		offset = -offset
	}
	return time.FixedZone("", offset)
}
