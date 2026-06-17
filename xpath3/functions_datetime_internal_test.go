package xpath3

import (
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
