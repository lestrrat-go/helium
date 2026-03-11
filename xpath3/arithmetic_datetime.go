package xpath3

import (
	"math"
	"math/big"
	"time"
)

func isDurationType(typeName string) bool {
	return typeName == TypeDuration || typeName == TypeYearMonthDuration || typeName == TypeDayTimeDuration
}

func isDateTimeType(typeName string) bool {
	return typeName == TypeDate || typeName == TypeDateTime || typeName == TypeTime
}

// julianDayNumber computes a continuous day count for a Gregorian calendar date,
// using the standard Julian Day Number algorithm. Works for negative years.
func julianDayNumber(year, month, day int) int64 {
	// Algorithm from Meeus, Astronomical Algorithms
	a := int64(14-month) / 12
	y := int64(year) + 4800 - a
	m := int64(month) + 12*a - 3
	return int64(day) + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
}

// evalDateTimeArithmetic handles arithmetic involving durations and date/time values.
// Returns (result, handled, error). If handled is false, the caller should fall through
// to numeric arithmetic.
func evalDateTimeArithmetic(ec *evalContext, op TokenType, la, ra AtomicValue) (Sequence, bool, error) {
	lDur := isDurationType(la.TypeName)
	rDur := isDurationType(ra.TypeName)
	lDT := isDateTimeType(la.TypeName)
	rDT := isDateTimeType(ra.TypeName)

	// duration + duration → duration
	// duration - duration → duration
	// duration / duration → decimal
	if lDur && rDur {
		if op == TokenDiv {
			return arithmeticDurationDivDuration(la, ra)
		}
		return arithmeticDurationDuration(op, la, ra)
	}

	// duration * number → duration
	if lDur && ra.IsNumeric() {
		return arithmeticDurationNumber(op, la, ra)
	}
	// number * duration → duration
	if la.IsNumeric() && rDur {
		if op == TokenStar {
			return arithmeticDurationNumber(op, ra, la)
		}
		return nil, false, nil
	}

	// date/dateTime/time + duration → date/dateTime/time
	// date/dateTime/time - duration → date/dateTime/time
	if lDT && rDur {
		return arithmeticDateTimeDuration(op, la, ra)
	}

	// duration + date/dateTime/time → date/dateTime/time (commutative addition only)
	if lDur && rDT && op == TokenPlus {
		return arithmeticDateTimeDuration(op, ra, la)
	}

	// date - date → dayTimeDuration
	// dateTime - dateTime → dayTimeDuration
	// time - time → dayTimeDuration
	if lDT && rDT && la.TypeName == ra.TypeName && op == TokenMinus {
		return arithmeticDateTimeDatetime(ec, la, ra)
	}

	// Not a date/time/duration operation
	if lDur || rDur || (lDT && !la.IsNumeric()) {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "incompatible types for arithmetic"}
	}

	return nil, false, nil
}

// arithmeticDurationDuration handles duration ± duration.
func arithmeticDurationDuration(op TokenType, la, ra AtomicValue) (Sequence, bool, error) {
	if op != TokenPlus && op != TokenMinus {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "invalid operator for duration arithmetic"}
	}

	// Only same-kind durations can be added/subtracted:
	// xs:yearMonthDuration ± xs:yearMonthDuration → OK
	// xs:dayTimeDuration ± xs:dayTimeDuration → OK
	// xs:duration ± anything → XPTY0004 (xs:duration is not directly arithmetic)
	// mixing YMD and DTD → XPTY0004
	if la.TypeName == TypeDuration || ra.TypeName == TypeDuration {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "xs:duration cannot be used in arithmetic"}
	}
	if la.TypeName != ra.TypeName {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "cannot mix xs:yearMonthDuration and xs:dayTimeDuration in arithmetic"}
	}

	ld := la.DurationVal()
	rd := ra.DurationVal()

	// Normalize to signed values
	lm, ls := ld.Months, ld.Seconds
	if ld.Negative {
		lm, ls = -lm, -ls
	}
	rm, rs := rd.Months, rd.Seconds
	if rd.Negative {
		rm, rs = -rm, -rs
	}

	var resMonths int
	var resSecs float64
	if op == TokenPlus {
		resMonths = lm + rm
		resSecs = ls + rs
	} else {
		resMonths = lm - rm
		resSecs = ls - rs
	}

	// Normalize sign: both components must agree in sign
	if resMonths > 0 && resSecs < 0 {
		resMonths--
		resSecs += 86400 * 30 // approximate month in seconds
	} else if resMonths < 0 && resSecs > 0 {
		resMonths++
		resSecs -= 86400 * 30
	}

	negative := resMonths < 0 || (resMonths == 0 && resSecs < 0)
	if negative {
		resMonths = -resMonths
		resSecs = -resSecs
	}

	// Determine result type
	typeName := resultDurationType(la.TypeName, ra.TypeName)

	return SingleAtomic(AtomicValue{
		TypeName: typeName,
		Value:    Duration{Months: resMonths, Seconds: resSecs, Negative: negative},
	}), true, nil
}

// arithmeticDurationNumber handles duration * number and duration / number.
func arithmeticDurationNumber(op TokenType, dur, num AtomicValue) (Sequence, bool, error) {
	if op != TokenStar && op != TokenDiv {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "invalid operator for duration*number"}
	}

	// xs:duration (generic) does not support arithmetic — only YMD and DTD do
	if dur.TypeName == TypeDuration {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "xs:duration cannot be used in arithmetic"}
	}

	d := dur.DurationVal()
	n := num.ToFloat64()

	if math.IsNaN(n) {
		return nil, true, &XPathError{Code: "FOCA0005", Message: "NaN in duration arithmetic"}
	}

	// Normalize duration to signed
	months := float64(d.Months)
	secs := d.Seconds
	if d.Negative {
		months, secs = -months, -secs
	}

	if op == TokenStar {
		months *= n
		secs *= n
	} else {
		if n == 0 {
			return nil, true, &XPathError{Code: "FODT0002", Message: "division of duration by zero"}
		}
		months /= n
		secs /= n
	}

	if math.IsInf(months, 0) || math.IsInf(secs, 0) {
		return nil, true, &XPathError{Code: "FODT0002", Message: "duration overflow"}
	}

	// Per XPath F&O spec: months are rounded "half towards positive infinity"
	// i.e. math.Floor(months + 0.5)
	resMonths := int(math.Floor(months + 0.5))
	resSecs := secs
	negative := resMonths < 0 || (resMonths == 0 && resSecs < 0)
	if negative {
		resMonths = -resMonths
		resSecs = -resSecs
	}

	return SingleAtomic(AtomicValue{
		TypeName: dur.TypeName,
		Value:    Duration{Months: resMonths, Seconds: resSecs, Negative: negative},
	}), true, nil
}

// arithmeticDateTimeDuration handles date/time ± duration.
func arithmeticDateTimeDuration(op TokenType, dt, dur AtomicValue) (Sequence, bool, error) {
	if op != TokenPlus && op != TokenMinus {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "invalid operator for date/time ± duration"}
	}

	// xs:time can only be combined with xs:dayTimeDuration, not xs:yearMonthDuration
	if dt.TypeName == TypeTime && (dur.TypeName == TypeYearMonthDuration || dur.TypeName == TypeDuration) {
		return nil, true, &XPathError{Code: "XPTY0004", Message: "cannot combine xs:time with xs:yearMonthDuration"}
	}

	t := dt.TimeVal()
	d := dur.DurationVal()

	// Normalize duration sign
	months := d.Months
	secs := d.Seconds
	if d.Negative {
		months, secs = -months, -secs
	}
	if op == TokenMinus {
		months, secs = -months, -secs
	}

	// Add months
	if months != 0 {
		t = addMonths(t, months)
	}

	// Add seconds (as time.Duration)
	if secs != 0 {
		t = t.Add(time.Duration(secs * float64(time.Second)))
	}

	// For xs:date results, strip the time component to keep only the date
	if dt.TypeName == TypeDate {
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	return SingleAtomic(AtomicValue{
		TypeName: dt.TypeName,
		Value:    t,
	}), true, nil
}

// addMonths adds months to a time.Time, clamping the day per XSD rules.
// Uses time.AddDate for the heavy lifting; detects day overflow (e.g. Jan 31 + 1 month
// normalizing to Mar 3) and clamps to the last day of the target month.
func addMonths(t time.Time, months int) time.Time {
	result := t.AddDate(0, months, 0)
	if result.Day() != t.Day() {
		// Day overflowed — go back to last day of the intended month
		result = t.AddDate(0, months+1, -t.Day())
	}
	return result
}

// arithmeticDateTimeDatetime handles dateTime - dateTime, date - date, time - time.
// Per XPath spec, if either operand lacks a timezone, the implicit timezone is applied.
func arithmeticDateTimeDatetime(ec *evalContext, la, ra AtomicValue) (Sequence, bool, error) {
	ta := la.TimeVal()
	tb := ra.TimeVal()

	// Per XPath spec, apply implicit timezone to operands that lack an explicit timezone.
	// Use attachTimezone (not In) to preserve the local time components.
	implicitTZ := ec.getImplicitTimezone()
	if !HasTimezone(ta) {
		ta = attachTimezone(ta, implicitTZ)
	}
	if !HasTimezone(tb) {
		tb = attachTimezone(tb, implicitTZ)
	}

	// Compute difference as total seconds to avoid time.Duration int64 overflow.
	// Convert both to Julian Day Number and time-of-day seconds.
	aDays := julianDayNumber(ta.Year(), int(ta.Month()), ta.Day())
	bDays := julianDayNumber(tb.Year(), int(tb.Month()), tb.Day())
	aSecs := float64(ta.Hour())*3600 + float64(ta.Minute())*60 + float64(ta.Second()) + float64(ta.Nanosecond())/1e9
	bSecs := float64(tb.Hour())*3600 + float64(tb.Minute())*60 + float64(tb.Second()) + float64(tb.Nanosecond())/1e9

	// Also account for timezone offsets
	_, aOff := ta.Zone()
	_, bOff := tb.Zone()
	aSecs -= float64(aOff)
	bSecs -= float64(bOff)

	totalSecs := float64(aDays-bDays)*86400 + (aSecs - bSecs)
	negative := totalSecs < 0
	if negative {
		totalSecs = -totalSecs
	}

	return SingleAtomic(AtomicValue{
		TypeName: TypeDayTimeDuration,
		Value:    Duration{Seconds: totalSecs, Negative: negative},
	}), true, nil
}

// arithmeticDurationDivDuration handles duration / duration → decimal.
func arithmeticDurationDivDuration(la, ra AtomicValue) (Sequence, bool, error) {
	ld := la.DurationVal()
	rd := ra.DurationVal()

	// Convert to total seconds for dayTimeDuration, total months for yearMonthDuration
	var lVal, rVal float64
	if la.TypeName == TypeYearMonthDuration {
		lVal = float64(ld.Months)
		if ld.Negative {
			lVal = -lVal
		}
		rVal = float64(rd.Months)
		if rd.Negative {
			rVal = -rVal
		}
	} else {
		lVal = ld.Seconds
		if ld.Negative {
			lVal = -lVal
		}
		rVal = rd.Seconds
		if rd.Negative {
			rVal = -rVal
		}
	}

	if rVal == 0 {
		return nil, true, &XPathError{Code: "FOAR0002", Message: "division of duration by zero duration"}
	}

	r := new(big.Rat).SetFloat64(lVal / rVal)
	return SingleDecimal(r), true, nil
}

// attachTimezone sets the timezone on a time value without converting the local
// time components. This is used when the XPath spec says to "apply" the implicit
// timezone to a timezone-less value: the local time stays the same, only the
// timezone label changes. This differs from time.In() which preserves the UTC
// instant and changes the local representation.
func attachTimezone(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// resultDurationType determines the result type for duration ± duration.
func resultDurationType(a, b string) string {
	if a == TypeYearMonthDuration && b == TypeYearMonthDuration {
		return TypeYearMonthDuration
	}
	if a == TypeDayTimeDuration && b == TypeDayTimeDuration {
		return TypeDayTimeDuration
	}
	return TypeDuration
}
