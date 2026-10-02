package xpath3_test

import (
	"math"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// pastInt32 is just past 2^32, so it truncates to 5 when converted to a 32-bit
// int. A test using it fails wherever a length is narrowed to int before it is
// compared, on every platform's CI.
const pastInt32 = int64(1)<<32 + 5

// TestNewRangeSequenceLen verifies that a range's length is never wrapped by the
// int conversion: a range longer than math.MaxInt reports math.MaxInt.
func TestNewRangeSequenceLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		start, end int64
		want       int64
	}{
		{"single item", 5, 5, 1},
		{"past 2^32", 1, pastInt32, min(pastInt32, math.MaxInt)},
		{"whole int64 range", math.MinInt64, math.MaxInt64, math.MaxInt},
		{"negative bounds", -10, -1, 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			seq := xpath3.NewRangeSequence(tc.start, tc.end)
			require.Equal(t, tc.want, int64(seq.Len()))
			require.Equal(t, xpath3.SingleInteger(tc.start).Get(0), seq.Get(0))
		})
	}
}

// TestRangeSequencePastInt32HonorsLimit verifies that a lazy range just past
// 2^32 items still trips the node-set limit. A length truncated to 32 bits
// would read as 5 items and pass a limit of 1000.
func TestRangeSequencePastInt32HonorsLimit(t *testing.T) {
	t.Parallel()

	vars := map[string]xpath3.Sequence{"r": xpath3.NewRangeSequence(1, pastInt32)}
	compiled, err := xpath3.NewCompiler().Compile(`for-each(1 to 3, function($x) { $r })`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		MaxNodesForTesting(1000).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}
