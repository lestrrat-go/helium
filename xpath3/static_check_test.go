package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestUndeclaredPrefixErrorOnEvaluate(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`foo:bar`)
	require.NoError(t, err, "compilation should succeed; prefix validation is deferred to evaluation")

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		StrictPrefixes().
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "foo")
}

func TestDeclaredPrefixResolvesOnEvaluate(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`foo:bar`)
	require.NoError(t, err)

	// Providing the namespace binding should pass prefix validation
	// (evaluation may still fail because there is no context node,
	// but the error should NOT be about an undeclared prefix).
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		StrictPrefixes().
		Namespaces(map[string]string{"foo": "urn:test"}).
		Evaluate(t.Context(), compiled, nil)
	if err != nil {
		require.NotContains(t, err.Error(), "undeclared namespace prefix")
	}
}
