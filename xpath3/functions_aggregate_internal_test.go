package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestFnAvgLexicalDecimal(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`avg((3.0, 4.0, 5.0))`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 4.0, n)
}
