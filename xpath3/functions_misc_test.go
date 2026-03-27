package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestFnImplicitTimezoneReturnsDuration(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`implicit-timezone()`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	require.True(t, result.IsAtomic())

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, xpath3.TypeDayTimeDuration, atomics[0].TypeName)
}
