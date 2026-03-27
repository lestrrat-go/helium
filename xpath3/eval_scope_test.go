package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestVariableScopeLookupPrefersNearestBinding(t *testing.T) {
	// let $x := 1 return (let $x := 2 return $x)  → 2  (inner shadows outer)
	compiled, err := xpath3.NewCompiler().Compile(`let $x := 1 return let $x := 2 return $x`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 2.0, n)
}

func TestVariableScopeLookupReachesOuter(t *testing.T) {
	// let $x := 10 return let $y := 20 return $x  → 10  (outer visible)
	compiled, err := xpath3.NewCompiler().Compile(`let $x := 10 return let $y := 20 return $x`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 10.0, n)
}
