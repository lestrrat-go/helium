package xpath3_test

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestIntegerMultiplicationBoundaries(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile("$a * $b")
	require.NoError(t, err)
	values := []int64{math.MinInt64, math.MinInt64 + 1, -3037000500, -3037000499, -2, -1, 0, 1, 2, 3037000499, 3037000500, math.MaxInt64 - 1, math.MaxInt64}
	for _, a := range values {
		for _, b := range values {
			t.Run(fmt.Sprintf("%d * %d", a, b), func(t *testing.T) {
				result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
					Variables(map[string]xpath3.Sequence{
						"a": xpath3.SingleInteger(a),
						"b": xpath3.SingleInteger(b),
					}).Evaluate(t.Context(), compiled, nil)
				require.NoError(t, err)
				require.Equal(t, 1, result.Sequence().Len())
				value, ok := result.Sequence().Get(0).(xpath3.AtomicValue)
				require.True(t, ok)
				want := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
				require.Equal(t, want.String(), value.BigInt().String())
			})
		}
	}
}
