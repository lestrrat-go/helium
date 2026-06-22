package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Patterns containing backreferences force the regexp2 backtracking engine
// (r.backtrack != nil), exercising the Split / FindAllStringSubmatchIndex /
// ReplaceAllString / NumSubexp branches that the std regexp path skips.
// These are reached through fn:tokenize, fn:replace, and fn:analyze-string.

func evalString(t *testing.T, expr string) string {
	t.Helper()
	r, err := evaluate(t.Context(), nil, expr)
	require.NoError(t, err)
	s, ok := r.IsString()
	require.True(t, ok, "expected single string result for %q", expr)
	return s
}

func TestRegexBacktrack_Tokenize(t *testing.T) {
	// Backreference \1 forces the backtracking engine; tokenize -> Split.
	r, err := evaluate(t.Context(), nil, `fn:tokenize("aXXbYYc", "(.)\1")`)
	require.NoError(t, err)
	atoms, err := r.Atomics()
	require.NoError(t, err)
	require.Len(t, atoms, 3)
}

func TestRegexBacktrack_Replace(t *testing.T) {
	// Backreference in the pattern forces backtracking; replace -> ReplaceAllString.
	got := evalString(t, `fn:replace("aabbcc", "(.)\1", "$1")`)
	require.Equal(t, "abc", got)
}

func TestRegexBacktrack_AnalyzeString(t *testing.T) {
	// analyze-string with a backreference pattern exercises
	// FindAllStringSubmatchIndex and NumSubexp on the backtracking path.
	r, err := evaluate(t.Context(), nil,
		`fn:analyze-string("aabb", "(.)\1")//*:match => count()`)
	require.NoError(t, err)
	n, ok := r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)
}
