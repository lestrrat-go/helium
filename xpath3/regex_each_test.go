package xpath3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// EachSubmatchIndex must stream exactly the same successive matches as the
// accumulating FindAllSubmatchIndex, for both the RE2 (std) and backtracking
// (regexp2) engines. Anchored, capturing, and empty-matching patterns are all
// covered so the empty-match advancement rule is exercised.
func TestRegexEachSubmatchIndexParity(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		pattern string
		flags   string
		input   string
	}{
		{name: "digits", pattern: "[0-9]", input: "a1b2c3"},
		{name: "empty match star", pattern: "x*", input: "aaa"},
		{name: "greedy with empties", pattern: "a*", input: "aabaa"},
		{name: "anchored start", pattern: "^a", input: "aaa"},
		{name: "anchored end", pattern: "a$", input: "aaa"},
		{name: "capturing groups", pattern: "(a)(b)?", input: "abac"},
		{name: "word", pattern: "\\c+", input: "foo bar baz"},
		// Leading-context (full-context) patterns: these stream through the
		// regexp2 twin and must match std's FindAll exactly. A multi-line "^"
		// matches at every line start (the amplification vector), so it exercises
		// the streamed-with-context path, including empty matches.
		{name: "multiline anchor", pattern: "^", flags: "m", input: "a\nb\nc"},
		{name: "multiline anchor empty lines", pattern: "^", flags: "m", input: "\n\n\n"},
		{name: "multiline anchor trailing newline", pattern: "^", flags: "m", input: "x\n"},
		{name: "multiline anchor capture", pattern: "(^)", flags: "m", input: "a\nb"},
		{name: "multiline anchored line", pattern: "^\\c*", flags: "m", input: "ab\ncd\n"},
		{name: "multiline anchored empty line", pattern: "^$", flags: "m", input: "a\n\nb"},
		// Backreference forces the regexp2 backtracking engine.
		{name: "backref backtrack", pattern: "(a)\\1", input: "aa-aa-b-aa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			re, err := xpath3.CompileRegex(tc.pattern, tc.flags)
			require.NoError(t, err)

			want, err := re.FindAllSubmatchIndex(tc.input, -1)
			require.NoError(t, err)

			var got [][]int
			err = re.EachSubmatchIndex(tc.input, func(m []int) bool {
				got = append(got, append([]int(nil), m...))
				return true
			})
			require.NoError(t, err)
			require.Equal(t, want, got,
				"EachSubmatchIndex must yield the same matches as FindAllSubmatchIndex")
		})
	}
}

// EachSubmatchIndex must stop as soon as the caller returns false, so a caller
// enforcing a match-count budget never enumerates (nor allocates) work
// proportional to the input size. With an empty-matching regex over a large
// input, an L-char string would otherwise yield ~L matches; capping the caller
// at a small budget must visit only budget+1 matches regardless of L.
func TestRegexEachSubmatchIndexEarlyStopIsBounded(t *testing.T) {
	t.Parallel()

	re, err := xpath3.CompileRegex("x*", "")
	require.NoError(t, err)

	const budget = 100
	for _, size := range []int{1_000, 100_000} {
		input := strings.Repeat("a", size)
		visited := 0
		err := re.EachSubmatchIndex(input, func(_ []int) bool {
			visited++
			return visited <= budget // stop one past the budget
		})
		require.NoError(t, err)
		require.Equal(t, budget+1, visited,
			"early-stop must bound enumeration to the budget, independent of input size %d", size)
	}
}

// A leading-context (full-context) anchor like a multi-line "^" matches at every
// line start, so an input of N newlines yields ~N matches. The full-context path
// must still honor the caller's early stop BEFORE enumerating (and allocating)
// work proportional to the match count — otherwise the cap is defeated.
func TestRegexEachSubmatchIndexMultilineAnchorIsBounded(t *testing.T) {
	t.Parallel()

	re, err := xpath3.CompileRegex("^", "m")
	require.NoError(t, err)

	const budget = 100
	for _, lines := range []int{1_000, 100_000} {
		input := strings.Repeat("\n", lines)
		visited := 0
		err := re.EachSubmatchIndex(input, func(_ []int) bool {
			visited++
			return visited <= budget // stop one past the budget
		})
		require.NoError(t, err)
		require.Equal(t, budget+1, visited,
			"multiline-anchor early-stop must bound enumeration to the budget, independent of line count %d", lines)
	}
}
