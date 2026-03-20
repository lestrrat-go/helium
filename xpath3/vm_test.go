package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestPartialApplicationPlaceholderVM(t *testing.T) {
	result, err := xpath3.Evaluate(t.Context(), nil, `let $f := concat("a", ?, "c") return $f("b")`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "abc", value)
}

func TestGeneralComparisonAgainstLargeRangeVM(t *testing.T) {
	result, err := xpath3.Evaluate(t.Context(), nil, `1000000000000000020001 = 1000000000000000000000 to 1000000000000010000003`)
	require.NoError(t, err)

	value, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, value)
}
