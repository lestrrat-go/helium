package xpath3

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEnvironmentVariableAccess uses the explicit evaluator environment and
// confirms that the two environment functions use the same exact-cased names.
func TestEnvironmentVariableAccess(t *testing.T) {
	const name = "HeliumEnvCaseTest"
	const want = "yes"
	t.Setenv("HELIUM_ENVIRONMENT_TEST_SECRET", "host-only")
	defaultVars, err := fnAvailableEnvVars(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, 0, seqLen(defaultVars), "the default evaluator must not expose process variables")
	defaultValue, err := fnEnvironmentVariable(t.Context(), []Sequence{SingleString("HELIUM_ENVIRONMENT_TEST_SECRET")})
	require.NoError(t, err)
	require.Equal(t, 0, seqLen(defaultValue), "the default evaluator must not read process variables")

	vars := map[string]string{name: want}
	evaluator := NewEvaluator(DefaultEvaluatorOptions).EnvironmentVariables(vars)
	vars[name] = "changed"
	ctx := withFnContext(t.Context(), evaluator.newEvalCtx(nil))

	avail, err := fnAvailableEnvVars(ctx, nil)
	require.NoError(t, err)
	names := make([]string, 0, avail.Len())
	for i := range avail.Len() {
		names = append(names, avail.Get(i).(AtomicValue).Value.(string))
	}
	require.True(t, slices.Contains(names, name), "exact-cased name must be listed")

	// Exact case resolves.
	got, err := fnEnvironmentVariable(ctx, []Sequence{SingleString(name)})
	require.NoError(t, err)
	require.Equal(t, 1, got.Len())
	require.Equal(t, want, got.Get(0).(AtomicValue).Value)

	// A differently-cased name that is NOT listed must NOT resolve, even on a
	// case-insensitive OS. This is the case-consistency guarantee.
	const wrongCase = "heliumenvcasetest"
	require.False(t, slices.Contains(names, wrongCase))
	miss, err := fnEnvironmentVariable(ctx, []Sequence{SingleString(wrongCase)})
	require.NoError(t, err)
	require.Equal(t, 0, seqLen(miss), "case-variant name must not resolve when it is not listed")
}
