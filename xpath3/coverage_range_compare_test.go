package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// General comparisons against a range expression (1 to N) take an optimized
// path through compareSingletonAgainstRange / compareRangeBounds /
// compareRangeBoundsInt64 across all six comparison operators and both operand
// orders.
func TestGeneralComparison_AgainstRange(t *testing.T) {
	cases := []struct {
		expr   string
		expect bool
	}{
		// singleton (op) range
		{`5 = (1 to 10)`, true},
		{`50 = (1 to 10)`, false},
		{`5 != (1 to 10)`, true},
		{`0 < (1 to 10)`, true},
		{`11 < (1 to 10)`, false},
		{`1 <= (1 to 10)`, true},
		{`11 > (1 to 10)`, true},
		{`0 > (1 to 10)`, false},
		{`10 >= (1 to 10)`, true},
		// range (op) singleton (rangeOnLeft)
		{`(1 to 10) = 5`, true},
		{`(1 to 10) < 11`, true},
		{`(1 to 10) <= 10`, true},
		{`(1 to 10) > 0`, true},
		{`(1 to 10) >= 1`, true},
		{`(1 to 10) != 5`, true},
		// large bounds exceeding int64 fast path force the big.Int comparator.
		{`5 = (1 to 100000000000000000000)`, true},
		{`0 = (1 to 100000000000000000000)`, false},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		b, ok := r.IsBoolean()
		require.True(t, ok, tc.expr)
		require.Equal(t, tc.expect, b, tc.expr)
	}
}
