package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestDerivedContextPreservesVariablesAcrossScalarUpdate(t *testing.T) {
	ctx := xpath3.WithVariables(t.Context(), map[string]xpath3.Sequence{
		"x": xpath3.SingleInteger(1),
	})

	ctx = xpath3.WithOpLimit(ctx, 10)

	result, err := xpath3.Evaluate(ctx, nil, `$x`)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, int64(1), atomics[0].IntegerVal())
}

func TestDerivedContextUsesCopyOnWriteBindings(t *testing.T) {
	base := xpath3.WithNamespaces(t.Context(), map[string]string{
		"a": "urn:a",
	})
	base = xpath3.WithVariables(base, map[string]xpath3.Sequence{
		"x": xpath3.SingleInteger(1),
	})

	derived := xpath3.WithAdditionalNamespaces(base, map[string]string{
		"p": "urn:b",
	})
	derived = xpath3.WithAdditionalVariables(derived, map[string]xpath3.Sequence{
		"p:y": xpath3.SingleInteger(2),
	})

	result, err := xpath3.Evaluate(base, nil, `$x`)
	require.NoError(t, err)
	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, int64(1), atomics[0].IntegerVal())

	_, err = xpath3.Evaluate(base, nil, `$p:y`)
	require.Error(t, err)

	result, err = xpath3.Evaluate(derived, nil, `$p:y`)
	require.NoError(t, err)
	atomics, err = result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, int64(2), atomics[0].IntegerVal())
}
