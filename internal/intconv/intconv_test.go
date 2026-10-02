package intconv_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium/internal/intconv"
	"github.com/stretchr/testify/require"
)

func TestAtoi(t *testing.T) {
	t.Parallel()

	n, err := intconv.Atoi("42")
	require.NoError(t, err)
	require.Equal(t, 42, n)

	// 2^32+5 fits int64; where int is 32 bits it saturates.
	n, err = intconv.Atoi("4294967301")
	require.NoError(t, err)
	require.Equal(t, min(int64(4294967301), math.MaxInt), int64(n))

	n, err = intconv.Atoi("-4294967301")
	require.NoError(t, err)
	require.Equal(t, max(int64(-4294967301), math.MinInt), int64(n))

	_, err = intconv.Atoi("18446744073709551617")
	require.ErrorIs(t, err, strconv.ErrRange)

	_, err = intconv.Atoi("12a")
	require.ErrorIs(t, err, strconv.ErrSyntax)
}
