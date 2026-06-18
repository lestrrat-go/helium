package xpath3

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFnDateTimeDerivedArgType verifies fn:dateTime accepts xs:date / xs:time
// and any type derived from them by restriction (carried via BaseType), while
// still rejecting xs:dateTime, which is a sibling of xs:date (not a subtype).
func TestFnDateTimeDerivedArgType(t *testing.T) {
	d := time.Date(1948, time.November, 2, 0, 0, 0, 0, noTZLocation)
	tm := time.Date(0, time.January, 1, 1, 2, 3, 0, noTZLocation)

	derivedDate := AtomicValue{TypeName: "Q{urn:x}myDate", Value: d, BaseType: TypeDate}
	derivedTime := AtomicValue{TypeName: "Q{urn:x}myTime", Value: tm, BaseType: TypeTime}

	t.Run("user-derived from xs:date and xs:time are accepted", func(t *testing.T) {
		seq, err := fnDateTime(t.Context(), []Sequence{
			SingleAtomic(derivedDate),
			SingleAtomic(derivedTime),
		})
		require.NoError(t, err)
		require.Equal(t, 1, seq.Len())
		require.Equal(t, TypeDateTime, seq.Get(0).(AtomicValue).TypeName)
	})

	t.Run("xs:dateTime as first arg is rejected (sibling, not subtype)", func(t *testing.T) {
		dt := AtomicValue{TypeName: TypeDateTime, Value: d}
		_, err := fnDateTime(t.Context(), []Sequence{
			SingleAtomic(dt),
			SingleAtomic(AtomicValue{TypeName: TypeTime, Value: tm}),
		})
		require.Error(t, err)
		require.ErrorIs(t, err, &XPathError{Code: "XPTY0004"})
	})

	t.Run("plain xs:date and xs:time are accepted", func(t *testing.T) {
		seq, err := fnDateTime(t.Context(), []Sequence{
			SingleAtomic(AtomicValue{TypeName: TypeDate, Value: d}),
			SingleAtomic(AtomicValue{TypeName: TypeTime, Value: tm}),
		})
		require.NoError(t, err)
		require.Equal(t, 1, seq.Len())
	})
}

// TestExtractDurationSchemaDerived verifies the duration accessors and the
// timezone consumers accept a schema-derived duration whose TypeName is a
// user-defined type carrying a built-in BaseType, rather than rejecting it with
// XPTY0004 because the custom TypeName is not a built-in subtype.
func TestExtractDurationSchemaDerived(t *testing.T) {
	dtd, err := parseXSDDuration("P3DT4H5M6S")
	require.NoError(t, err)
	derivedDTD := SingleAtomic(AtomicValue{
		TypeName: "Q{urn:x}myDTD",
		BaseType: TypeDayTimeDuration,
		Value:    dtd,
	})

	ymd, err := parseXSDDuration("P2Y3M")
	require.NoError(t, err)
	derivedYMD := SingleAtomic(AtomicValue{
		TypeName: "Q{urn:x}myYMD",
		BaseType: TypeYearMonthDuration,
		Value:    ymd,
	})

	t.Run("days-from-duration accepts schema-derived dayTimeDuration", func(t *testing.T) {
		seq, err := fnDaysFromDuration(t.Context(), []Sequence{derivedDTD})
		require.NoError(t, err)
		require.Equal(t, int64(3), seq.Get(0).(AtomicValue).IntegerVal())
	})

	t.Run("hours-from-duration accepts schema-derived dayTimeDuration", func(t *testing.T) {
		seq, err := fnHoursFromDuration(t.Context(), []Sequence{derivedDTD})
		require.NoError(t, err)
		require.Equal(t, int64(4), seq.Get(0).(AtomicValue).IntegerVal())
	})

	t.Run("years-from-duration accepts schema-derived yearMonthDuration", func(t *testing.T) {
		seq, err := fnYearsFromDuration(t.Context(), []Sequence{derivedYMD})
		require.NoError(t, err)
		require.Equal(t, int64(2), seq.Get(0).(AtomicValue).IntegerVal())
	})

	t.Run("timezone consumer accepts schema-derived dayTimeDuration", func(t *testing.T) {
		// A schema-derived +PT1H offset must satisfy the timezone-offset path
		// (extractDuration via TypeDayTimeDuration) and validate cleanly.
		offset, err := parseXSDDuration("PT1H")
		require.NoError(t, err)
		_, ok, err := extractDuration(SingleAtomic(AtomicValue{
			TypeName: "Q{urn:x}myTZ",
			BaseType: TypeDayTimeDuration,
			Value:    offset,
		}), TypeDayTimeDuration)
		require.NoError(t, err)
		require.True(t, ok)
		require.NoError(t, validateTimezoneOffset(offset))
	})
}

// TestTimezoneAccessorExactSecRat verifies the timezone accessors return an
// xs:dayTimeDuration whose magnitude is carried exactly in SecRat (in addition
// to the lossy Seconds mirror), so the duration is exact by construction.
func TestTimezoneAccessorExactSecRat(t *testing.T) {
	// +05:30 = 19800 seconds east of UTC.
	posTZ := time.FixedZone("+0530", 5*3600+30*60)
	// -08:00 = 28800 seconds west of UTC.
	negTZ := time.FixedZone("-0800", -8*3600)

	check := func(t *testing.T, seq Sequence, err error, wantSecs int64, wantNeg bool) {
		t.Helper()
		require.NoError(t, err)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(AtomicValue)
		require.Equal(t, TypeDayTimeDuration, av.TypeName)
		d := av.Value.(Duration)
		require.NotNil(t, d.SecRat)
		require.Equal(t, big.NewRat(wantSecs, 1).RatString(), d.SecRat.RatString())
		require.Equal(t, wantNeg, d.Negative)
	}

	t.Run("timezone-from-dateTime positive offset", func(t *testing.T) {
		dt := time.Date(2020, time.January, 1, 0, 0, 0, 0, posTZ)
		seq, err := fnTimezoneFromDateTime(t.Context(), []Sequence{
			SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: dt}),
		})
		check(t, seq, err, 19800, false)
	})

	t.Run("timezone-from-dateTime negative offset", func(t *testing.T) {
		dt := time.Date(2020, time.January, 1, 0, 0, 0, 0, negTZ)
		seq, err := fnTimezoneFromDateTime(t.Context(), []Sequence{
			SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: dt}),
		})
		check(t, seq, err, 28800, true)
	})

	t.Run("timezone-from-date positive offset", func(t *testing.T) {
		d := time.Date(2020, time.January, 1, 0, 0, 0, 0, posTZ)
		seq, err := fnTimezoneFromDate(t.Context(), []Sequence{
			SingleAtomic(AtomicValue{TypeName: TypeDate, Value: d}),
		})
		check(t, seq, err, 19800, false)
	})

	t.Run("timezone-from-time negative offset", func(t *testing.T) {
		tm := time.Date(2020, time.January, 1, 12, 0, 0, 0, negTZ)
		seq, err := fnTimezoneFromTime(t.Context(), []Sequence{
			SingleAtomic(AtomicValue{TypeName: TypeTime, Value: tm}),
		})
		check(t, seq, err, 28800, true)
	})
}

// TestValidateTimezoneOffsetUnderflow verifies that a nonzero offset whose
// seconds underflow float64 (and therefore round to exactly 0.0 in d.Seconds)
// is still rejected via the exact SecRat-aware rational, rather than being
// silently accepted as UTC.
func TestValidateTimezoneOffsetUnderflow(t *testing.T) {
	lex := "PT0." + strings.Repeat("0", 400) + "1S"
	d, err := parseXSDDuration(lex)
	require.NoError(t, err)
	// The lossy float mirror flattens to zero, but the exact rational is nonzero.
	require.Equal(t, float64(0), d.Seconds)

	err = validateTimezoneOffset(d)
	require.Error(t, err)
	require.ErrorIs(t, err, &XPathError{Code: "FODT0003"})
}
