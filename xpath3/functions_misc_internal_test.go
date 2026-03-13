package xpath3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFnImplicitTimezoneUsesEvaluationSnapshot(t *testing.T) {
	loc, err := time.LoadLocation("Pacific/Apia")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}

	snapshot := time.Date(2010, time.July, 1, 12, 0, 0, 0, loc)
	_, wantOffset := snapshot.Zone()
	_, currentOffset := time.Now().In(loc).Zone()
	if wantOffset == currentOffset {
		t.Skip("timezone database does not expose a historical offset change for this location")
	}

	ec := &evalContext{
		currentTime:      &snapshot,
		implicitTimezone: loc,
	}

	seq, err := fnImplicitTimezone(withFnContext(context.Background(), ec), nil)
	require.NoError(t, err)
	require.Len(t, seq, 1)

	av, ok := seq[0].(AtomicValue)
	require.True(t, ok)
	require.Equal(t, TypeDayTimeDuration, av.TypeName)

	duration := av.DurationVal()
	gotOffset := int(duration.Seconds)
	if duration.Negative {
		gotOffset = -gotOffset
	}

	require.Equal(t, wantOffset, gotOffset)
}
