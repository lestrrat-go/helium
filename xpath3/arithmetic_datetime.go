package xpath3

import (
	"math"
	"math/big"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

func isDurationType(typeName string) bool {
	return typeName == TypeDuration || typeName == TypeYearMonthDuration || typeName == TypeDayTimeDuration
}

func isDateTimeType(typeName string) bool {
	return typeName == TypeDate || typeName == TypeDateTime || typeName == TypeTime
}

// arithmeticType returns the built-in type that drives date/time/duration
// arithmetic for an AtomicValue. A schema-derived value (e.g. a restriction of
// xs:dayTimeDuration) carries a custom TypeName whose BaseType names the
// built-in ancestor; arithmetic must treat it as that built-in type.
func arithmeticType(a AtomicValue) string {
	if isDurationType(a.TypeName) || isDateTimeType(a.TypeName) {
		return a.TypeName
	}
	if isDurationType(a.BaseType) || isDateTimeType(a.BaseType) {
		return a.BaseType
	}
	return a.TypeName
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
	// Promote schema-derived duration/date/time operands to their built-in base
	// type BEFORE classifying, so a restriction of e.g. xs:dayTimeDuration is
	// recognized as a duration and the result carries the built-in type. The
	// promotion only rewrites TypeName; the backing Go value is unchanged.
	if at := arithmeticType(la); at != la.TypeName {
		la.TypeName = at
	}
	if at := arithmeticType(ra); at != ra.TypeName {
		ra.TypeName = at
	}

	lDur := isDurationType(la.TypeName)
	rDur := isDurationType(ra.TypeName)
	lDT := isDateTimeType(la.TypeName)
	rDT := isDateTimeType(ra.TypeName)

	// duration + duration → duration
	// - duration → duration
	// / duration → decimal
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
	// - duration → date/dateTime/time
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
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "incompatible types for arithmetic"}
	}

	return nil, false, nil
}

// arithmeticDurationDuration handles duration ± duration.
func arithmeticDurationDuration(op TokenType, la, ra AtomicValue) (Sequence, bool, error) {
	if op != TokenPlus && op != TokenMinus {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "invalid operator for duration arithmetic"}
	}

	// Only same-kind durations can be added/subtracted:
	// xs:yearMonthDuration ± xs:yearMonthDuration → OK
	// xs:dayTimeDuration ± xs:dayTimeDuration → OK
	// xs:duration ± anything → XPTY0004 (xs:duration is not directly arithmetic)
	// mixing YMD and DTD → XPTY0004
	if la.TypeName == TypeDuration || ra.TypeName == TypeDuration {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "xs:duration cannot be used in arithmetic"}
	}
	if la.TypeName != ra.TypeName {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "cannot mix xs:yearMonthDuration and xs:dayTimeDuration in arithmetic"}
	}

	ld := la.DurationVal()
	rd := ra.DurationVal()

	// Determine result type
	typeName := resultDurationType(la.TypeName, ra.TypeName)

	// yearMonthDuration ± yearMonthDuration operates purely on the integer
	// month component.
	if typeName == TypeYearMonthDuration {
		lm := ld.Months
		if ld.Negative {
			lm = -lm
		}
		rm := rd.Months
		if rd.Negative {
			rm = -rm
		}
		var resMonths int
		if op == TokenPlus {
			resMonths = lm + rm
		} else {
			resMonths = lm - rm
		}
		negative := resMonths < 0
		if negative {
			resMonths = -resMonths
		}
		return SingleAtomic(AtomicValue{
			TypeName: typeName,
			Value:    Duration{Months: resMonths, Negative: negative},
		}), true, nil
	}

	// dayTimeDuration ± dayTimeDuration: compute the result in exact rational
	// seconds so that fractional seconds (e.g. PT0.05S + PT0.05S) canonicalize
	// identically to a parsed PT0.1S.
	lRat := durationToRat(ld, false)
	rRat := durationToRat(rd, false)
	var resRat *big.Rat
	if op == TokenPlus {
		resRat = new(big.Rat).Add(lRat, rRat)
	} else {
		resRat = new(big.Rat).Sub(lRat, rRat)
	}

	negative := resRat.Sign() < 0
	absRat := resRat
	if negative {
		absRat = new(big.Rat).Neg(resRat)
	}

	secs, frac := durationFromRatSeconds(absRat)
	return SingleAtomic(AtomicValue{
		TypeName: typeName,
		Value:    Duration{Seconds: secs, FracSec: frac, SecRat: absRat, Negative: negative},
	}), true, nil
}

// arithmeticDurationNumber handles duration * number and duration / number.
func arithmeticDurationNumber(op TokenType, dur, num AtomicValue) (Sequence, bool, error) {
	if op != TokenStar && op != TokenDiv {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "invalid operator for duration*number"}
	}

	// xs:duration (generic) does not support arithmetic — only YMD and DTD do
	if dur.TypeName == TypeDuration {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "xs:duration cannot be used in arithmetic"}
	}

	d := dur.DurationVal()
	n := num.ToFloat64()

	if math.IsNaN(n) {
		return nil, true, &XPathError{Code: "FOCA0005", Message: "NaN in duration arithmetic"}
	}

	if op == TokenDiv && n == 0 {
		return nil, true, &XPathError{Code: errCodeFODT0002, Message: "division of duration by zero"}
	}

	// dayTimeDuration * / number: compute the result in exact rational seconds
	// when the multiplier/divisor is itself an exact rational (xs:integer or
	// xs:decimal). This makes e.g. PT11S * 0.1 canonicalize identically to a
	// parsed PT1.1S. (xs:double/xs:float multipliers are binary-imprecise and
	// fall through to the float path below.)
	//
	// When d.SecRat is present the EXACT total-seconds magnitude is authoritative,
	// so drive the arithmetic entirely from it with NO float64 2^53 cap — large
	// whole-second durations (e.g. PT9223372036854775808S) compute exactly. Only a
	// legacy float-only duration (no SecRat) is gated on the exact float64 range,
	// since a rational built from an already-imprecise float would be misleading.
	const maxExactDayTimeSecs = float64(1 << 53)
	exactSecs := d.SecRat != nil || math.Abs(d.Seconds) <= maxExactDayTimeSecs
	if dur.TypeName == TypeDayTimeDuration && exactSecs {
		nRat, ok := numericToRat(num)
		if ok {
			secsRat := durationToRat(d, false)
			var resRat *big.Rat
			if op == TokenStar {
				resRat = new(big.Rat).Mul(secsRat, nRat)
			} else {
				resRat = new(big.Rat).Quo(secsRat, nRat)
			}
			negative := resRat.Sign() < 0
			absRat := resRat
			if negative {
				absRat = new(big.Rat).Neg(resRat)
			}
			rsecs, frac := durationFromRatSeconds(absRat)
			return SingleAtomic(AtomicValue{
				TypeName: dur.TypeName,
				Value:    Duration{Seconds: rsecs, FracSec: frac, SecRat: absRat, Negative: negative},
			}), true, nil
		}
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
		months /= n
		secs /= n
	}

	if math.IsInf(months, 0) || math.IsInf(secs, 0) {
		return nil, true, &XPathError{Code: errCodeFODT0002, Message: "duration overflow"}
	}
	// Detect precision loss for very large values
	const maxExactFloat64 = 1 << 53
	absSecs := math.Abs(secs)
	absMonths := math.Abs(months)
	if absSecs > maxExactFloat64 || absMonths > maxExactFloat64 {
		return nil, true, &XPathError{Code: errCodeFODT0002, Message: "duration overflow"}
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
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "invalid operator for date/time ± duration"}
	}

	// xs:time can only be combined with xs:dayTimeDuration, not xs:yearMonthDuration
	if dt.TypeName == TypeTime && (dur.TypeName == TypeYearMonthDuration || dur.TypeName == TypeDuration) {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "cannot combine xs:time with xs:yearMonthDuration"}
	}

	t := dt.TimeVal()
	d := dur.DurationVal()

	// Normalize duration sign
	months := d.Months
	if d.Negative {
		months = -months
	}
	if op == TokenMinus {
		months = -months
	}

	// secsRat is the EXACT total dayTime seconds to add (signed), read from the
	// SecRat-aware rational so a fraction arbitrarily close to a whole second is
	// not rounded up via float64.
	secsRat := durationToRat(d, false)
	if op == TokenMinus {
		secsRat = new(big.Rat).Neg(secsRat)
	}

	// Add months
	if months != 0 {
		t = addMonths(t, months)
	}

	// Add seconds exactly: split into whole seconds and a sub-second nanosecond
	// remainder. Whole seconds are added as days + sub-day seconds via AddDate so
	// large magnitudes never overflow time.Duration's int64-nanosecond range.
	if secsRat.Sign() != 0 {
		abs := secsRat
		neg := false
		if abs.Sign() < 0 {
			abs = new(big.Rat).Neg(abs)
			neg = true
		}
		whole := new(big.Int).Quo(abs.Num(), abs.Denom())
		frac := new(big.Rat).Sub(abs, new(big.Rat).SetInt(whole))
		// Nanosecond remainder (truncated; sub-nanosecond precision is below the
		// representable resolution of time.Time).
		nanoRat := new(big.Rat).Mul(frac, big.NewRat(1e9, 1))
		nanos := new(big.Int).Quo(nanoRat.Num(), nanoRat.Denom()).Int64()

		days := new(big.Int)
		remSecs := new(big.Int)
		days.QuoRem(whole, big.NewInt(86400), remSecs)
		sign := 1
		if neg {
			sign = -1
		}
		t = t.AddDate(0, 0, sign*int(days.Int64()))
		t = t.Add(time.Duration(sign) * time.Duration(remSecs.Int64()) * time.Second)
		t = t.Add(time.Duration(sign) * time.Duration(nanos) * time.Nanosecond)
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
	absSecs := totalSecs
	if negative {
		absSecs = -totalSecs
	}

	// Build the exact fractional-seconds component from the nanosecond
	// difference so that sub-second results canonicalize as exact rationals
	// (matching a parsed dayTimeDuration). The only sub-second contribution is
	// the nanosecond field; all other components are whole seconds.
	absNs := int64(ta.Nanosecond()) - int64(tb.Nanosecond())
	if negative {
		absNs = -absNs
	}
	// Normalize the nanosecond fraction into [0,1) seconds.
	absNs %= 1e9
	if absNs < 0 {
		absNs += 1e9
	}
	var frac *big.Rat
	if absNs != 0 {
		frac = new(big.Rat).SetFrac(big.NewInt(absNs), big.NewInt(1e9))
	}

	// Build the exact total-seconds magnitude: the whole-second part of absSecs
	// (which carries no sub-second component beyond nanoseconds, already captured
	// in frac) plus the exact fractional rational.
	wholeSecs := math.Round(absSecs - math.Mod(absSecs, 1))
	secRat := new(big.Rat).SetInt64(int64(wholeSecs))
	if frac != nil {
		secRat.Add(secRat, frac)
	}

	return SingleAtomic(AtomicValue{
		TypeName: TypeDayTimeDuration,
		Value:    Duration{Seconds: absSecs, FracSec: frac, SecRat: secRat, Negative: negative},
	}), true, nil
}

// arithmeticDurationDivDuration handles duration / duration → decimal.
func arithmeticDurationDivDuration(la, ra AtomicValue) (Sequence, bool, error) {
	// xs:duration (generic) does not support arithmetic
	if la.TypeName == TypeDuration || ra.TypeName == TypeDuration {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "xs:duration cannot be used in arithmetic"}
	}
	// Cannot mix YMD and DTD
	if la.TypeName != ra.TypeName {
		return nil, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "cannot divide different duration subtypes"}
	}

	ld := la.DurationVal()
	rd := ra.DurationVal()

	isYM := la.TypeName == TypeYearMonthDuration

	// Legacy float-only durations (no SecRat) cannot exactly represent seconds
	// beyond 2^53, so reject them. When SecRat is present the exact magnitude is
	// authoritative and durationToRat reads it directly, so no cap applies.
	if !isYM {
		const maxExactFloat64 = 1 << 53
		if ld.SecRat == nil && ld.Seconds > maxExactFloat64 {
			return nil, true, &XPathError{Code: errCodeFODT0002, Message: "dayTimeDuration value too large for exact division"}
		}
		if rd.SecRat == nil && rd.Seconds > maxExactFloat64 {
			return nil, true, &XPathError{Code: errCodeFODT0002, Message: "dayTimeDuration value too large for exact division"}
		}
	}

	lRat := durationToRat(ld, isYM)
	rRat := durationToRat(rd, ra.TypeName == TypeYearMonthDuration)

	if rRat.Sign() == 0 {
		return nil, true, &XPathError{Code: errCodeFOAR0002, Message: "division of duration by zero duration"}
	}

	r := new(big.Rat).Quo(lRat, rRat)
	return SingleDecimal(r), true, nil
}

// numericToRat returns the exact rational value of an integer- or
// decimal-typed numeric AtomicValue. It returns ok=false for xs:double and
// xs:float, whose backing float64 values are binary-imprecise and therefore
// cannot be represented exactly as the intended decimal.
func numericToRat(a AtomicValue) (*big.Rat, bool) {
	// Resolve the effective numeric type. A schema-derived numeric (e.g. a
	// restriction of xs:decimal or xs:integer) carries a custom TypeName whose
	// BaseType names the built-in ancestor; consult BaseType so its exact value
	// is preserved instead of falling through to the imprecise float path.
	tn := a.TypeName
	if !isIntegerDerived(tn) && tn != TypeDecimal {
		if isIntegerDerived(a.BaseType) || a.BaseType == TypeDecimal {
			tn = a.BaseType
		}
	}

	if isIntegerDerived(tn) {
		switch v := a.Value.(type) {
		case int64:
			return new(big.Rat).SetInt64(v), true
		case *big.Int:
			return new(big.Rat).SetInt(v), true
		}
		return nil, false
	}
	if tn == TypeDecimal {
		r, ok := a.Value.(*big.Rat)
		if !ok {
			return nil, false
		}
		return new(big.Rat).Set(r), true
	}
	return nil, false
}

// durationFromRatSeconds splits a non-negative exact total-seconds rational
// into the float64 Seconds field (whole seconds plus fractional part, kept for
// compatibility with the rest of the Duration machinery) and an exact FracSec
// rational holding the fractional part in [0,1). Storing the exact fraction lets
// arithmetic-created durations canonicalize identically to parsed ones.
func durationFromRatSeconds(secsRat *big.Rat) (float64, *big.Rat) {
	// Whole seconds = floor(secsRat) for a non-negative value.
	whole := new(big.Int).Quo(secsRat.Num(), secsRat.Denom())
	frac := new(big.Rat).Sub(secsRat, new(big.Rat).SetInt(whole))

	secs, _ := secsRat.Float64()
	if frac.Sign() == 0 {
		return secs, nil
	}
	return secs, frac
}

// durationToRat converts a Duration to an exact big.Rat value.
// For yearMonthDuration, the value is total months; for dayTimeDuration, total seconds.
func durationToRat(d Duration, isYM bool) *big.Rat {
	var r *big.Rat
	if isYM {
		r = new(big.Rat).SetInt64(int64(d.Months))
	} else if d.SecRat != nil {
		// SecRat holds the EXACT total dayTime seconds magnitude (>=0). Prefer it
		// over the lossy float64 Seconds field so that values beyond 2^53 (e.g.
		// PT9007199254740992S vs PT9007199254740993S) stay distinct.
		r = new(big.Rat).Set(d.SecRat)
	} else {
		// d.Seconds is the total seconds INCLUDING any fractional part, while
		// d.FracSec (when set) holds the exact fractional component in [0,1).
		// Building the rational from d.Seconds and then adding d.FracSec would
		// double-count the fraction. When FracSec is present, derive the
		// whole-second count and add the exact fraction once.
		secs := d.Seconds
		if d.FracSec != nil {
			// Recover the whole-second count by removing the (possibly rounded)
			// fractional part from Seconds before truncation. Subtracting FracSec
			// first prevents float64 rounding from inflating the integer portion
			// for fractions extremely close to 1 (e.g. PT0.999...9S where Seconds
			// rounds up to 1.0).
			fracFloat, _ := d.FracSec.Float64()
			secs = math.Trunc(d.Seconds - fracFloat + 0.5)
		}
		// Use big.Float → big.Rat to handle values exceeding int64 range
		bf := new(big.Float).SetPrec(256).SetFloat64(secs)
		r, _ = bf.Rat(nil)
		if d.FracSec != nil {
			r.Add(r, d.FracSec)
		}
	}
	if d.Negative {
		r.Neg(r)
	}
	return r
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
