package xslt3

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// clampMatchCap guards the xsl:analyze-string one-over-budget probe (cap+1)
// against integer overflow. A resource limit raised above math.MaxInt makes
// clampInt64ToInt return math.MaxInt; without the clamp, cap+1 would wrap to a
// negative value, and FindAllSubmatchIndex treats a negative n as "unbounded",
// silently defeating the cap. The clamped cap must keep cap+1 strictly
// positive while preserving the unbounded (negative) sentinel.
func TestClampMatchCap(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   int
		want int
	}{
		{name: "unbounded sentinel preserved", in: -1, want: -1},
		{name: "small cap unchanged", in: 1000, want: 1000},
		{name: "max-minus-two unchanged", in: math.MaxInt - 2, want: math.MaxInt - 2},
		{name: "max-minus-one unchanged", in: math.MaxInt - 1, want: math.MaxInt - 1},
		{name: "maxint clamped down", in: math.MaxInt, want: math.MaxInt - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := clampMatchCap(tc.in)
			require.Equal(t, tc.want, got)

			// The whole point of the clamp: for a non-negative cap, the
			// one-over-budget probe must stay strictly positive (never wrap
			// negative), so FindAllSubmatchIndex stays bounded.
			if tc.in >= 0 {
				findN := got + 1
				require.Positive(t, findN,
					"clamped cap+1 must stay positive to keep match search bounded")
			}
		})
	}
}
