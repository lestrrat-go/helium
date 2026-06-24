package xpath3

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEnvironmentVariableCaseSensitive locks the invariant that
// fn:environment-variable and fn:available-environment-variables agree on every
// OS: a name is resolvable by environment-variable() iff it appears verbatim in
// available-environment-variables(). On Windows os.LookupEnv is
// case-insensitive, which previously let environment-variable("path") return a
// value while available-environment-variables() listed only "Path" — breaking
// W3C function-1501. The fix enumerates os.Environ() with an exact match, so the
// two accessors stay in lock step. This test is GOOS-independent.
func TestEnvironmentVariableCaseSensitive(t *testing.T) {
	const name = "HeliumEnvCaseTest"
	const want = "yes"
	t.Setenv(name, want)

	avail, err := fnAvailableEnvVars(t.Context(), nil)
	require.NoError(t, err)
	names := make([]string, 0, avail.Len())
	for i := range avail.Len() {
		names = append(names, avail.Get(i).(AtomicValue).Value.(string))
	}
	require.True(t, slices.Contains(names, name), "exact-cased name must be listed")

	// Exact case resolves.
	got, err := fnEnvironmentVariable(t.Context(), []Sequence{SingleString(name)})
	require.NoError(t, err)
	require.Equal(t, 1, got.Len())
	require.Equal(t, want, got.Get(0).(AtomicValue).Value)

	// A differently-cased name that is NOT listed must NOT resolve, even on a
	// case-insensitive OS. This is the case-consistency guarantee.
	const wrongCase = "heliumenvcasetest"
	if !slices.Contains(names, wrongCase) {
		miss, err := fnEnvironmentVariable(t.Context(), []Sequence{SingleString(wrongCase)})
		require.NoError(t, err)
		require.Equal(t, 0, seqLen(miss), "case-variant name must not resolve when it is not listed")
	}
}
