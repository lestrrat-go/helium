package xpath3_test

import (
	"strings"
	"testing"
	"time"

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
		// Leading-context (full-context) patterns: these are matched against the
		// whole string by a bounded FindAll on Go's RE2 engine and must match
		// std's FindAll exactly. A multi-line "^" matches at every line start (the
		// amplification vector), so it exercises the full-context path, including
		// empty matches.
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
			err = re.EachSubmatchIndex(tc.input, -1, func(m []int) bool {
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
		err := re.EachSubmatchIndex(input, -1, func(_ []int) bool {
			visited++
			return visited <= budget // stop one past the budget
		})
		require.NoError(t, err)
		require.Equal(t, budget+1, visited,
			"early-stop must bound enumeration to the budget, independent of input size %d", size)
	}
}

// A leading-context (full-context) anchor like a multi-line "^" matches at every
// line start, so an input of N newlines yields ~N matches. This pattern cannot be
// streamed incrementally on RE2, so the caller passes limit = budget+1 to bound
// the single FindAll pass to the budget — the enumeration must not allocate (nor
// visit) work proportional to the match count, otherwise the cap is defeated.
func TestRegexEachSubmatchIndexMultilineAnchorIsBounded(t *testing.T) {
	t.Parallel()

	re, err := xpath3.CompileRegex("^", "m")
	require.NoError(t, err)

	const budget = 100
	for _, lines := range []int{1_000, 100_000} {
		input := strings.Repeat("\n", lines)
		visited := 0
		err := re.EachSubmatchIndex(input, budget+1, func(_ []int) bool {
			visited++
			return visited <= budget // stop one past the budget
		})
		require.NoError(t, err)
		require.Equal(t, budget+1, visited,
			"multiline-anchor enumeration must be bounded by limit, independent of line count %d", lines)
	}
}

// A leading-context pattern cannot stream on RE2 and is materialized in one
// FindAll pass. With an uncapped (or larger-than-ceiling) limit, that pass must
// NOT allocate one record per input position: it is bounded to an internal
// allocation ceiling, and an input that exceeds the ceiling is rejected with
// ErrRegexMatchLimit before any match is delivered (rather than silently
// truncated or allowed to amplify a bounded input into millions of records).
func TestRegexEachSubmatchIndexFullContextCeilingIsBounded(t *testing.T) {
	t.Parallel()

	re, err := xpath3.CompileRegex("^", "m")
	require.NoError(t, err)

	// A multi-line "^" matches at every line start, so N newlines yield N+1
	// matches. Exceed the internal ceiling (1<<20) so the uncapped enumeration is
	// rejected instead of materializing a record per line start.
	input := strings.Repeat("\n", (1<<20)+16)
	visited := 0
	err = re.EachSubmatchIndex(input, -1, func(_ []int) bool {
		visited++
		return true
	})
	require.ErrorIs(t, err, xpath3.ErrRegexMatchLimit,
		"a full-context pattern over the allocation ceiling must be rejected, not materialized")
	require.Zero(t, visited,
		"the ceiling breach must be reported before any match is delivered")
}

// A high-capture leading-context pattern must be bounded by the index-CELL
// ceiling, not the match COUNT: each FindAll record holds 2*(NumSubexp()+1)
// ints, so a pattern with many captures gets a proportionally smaller match cap
// and trips ErrRegexMatchLimit on a far smaller input than a capture-free
// pattern would. This guards against bounding match records alone — which would
// let `^()()()...` with `m` allocate far past the intended ceiling before the
// cap fires.
func TestRegexEachSubmatchIndexHighCaptureCeilingIsBounded(t *testing.T) {
	t.Parallel()

	// 63 empty capture groups => 2*(63+1) = 128 ints per match record, so the
	// derived match cap is (1<<20)/128 = 8192 — far below the capture-free 1<<20
	// match ceiling. An input with more line starts than that must be rejected.
	const captures = 63
	pattern := "^" + strings.Repeat("()", captures)
	re, err := xpath3.CompileRegex(pattern, "m")
	require.NoError(t, err)

	const matchCap = (1 << 20) / (2 * (captures + 1)) // 8192
	// Each newline adds one line start; matchCap+16 newlines yield matchCap+17
	// matches — past the cap, so the cell ceiling (not the 1<<20 count) binds.
	input := strings.Repeat("\n", matchCap+16)
	visited := 0
	err = re.EachSubmatchIndex(input, -1, func(_ []int) bool {
		visited++
		return true
	})
	require.ErrorIs(t, err, xpath3.ErrRegexMatchLimit,
		"a high-capture full-context pattern over the cell ceiling must be rejected")
	require.Zero(t, visited,
		"the ceiling breach must be reported before any match is delivered")
}

// The public limit contract (regex.go) must hold uniformly across all three
// enumeration paths: the ordinary RE2 streaming offset-loop, the full-context
// RE2 FindAll path, and the regexp2 backtracking path. A non-positive limit
// (including 0) means uncapped — every match is produced. A positive limit N
// stops iteration after exactly N matches (or all of them, if fewer than N).
func TestRegexEachSubmatchIndexLimitContract(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		pattern string
		flags   string
		input   string
		total   int // number of matches with an uncapped limit
	}{
		// Ordinary RE2 streaming offset-loop.
		{name: "re2 digits", pattern: "[0-9]", input: "1234567", total: 7},
		// Full-context RE2 (multi-line anchor matches at every line start).
		{name: "re2 multiline anchor", pattern: "^", flags: "m", input: "a\nb\nc\nd\ne", total: 5},
		// regexp2 backtracking engine (backreference).
		{name: "regexp2 backref", pattern: "(a)\\1", input: "aa-aa-aa-aa", total: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			re, err := xpath3.CompileRegex(tc.pattern, tc.flags)
			require.NoError(t, err)

			count := func(limit int) int {
				n := 0
				err := re.EachSubmatchIndex(tc.input, limit, func(_ []int) bool {
					n++
					return true
				})
				require.NoError(t, err)
				return n
			}

			// limit == 0 and negative limits mean uncapped: all matches produced.
			require.Equal(t, tc.total, count(0),
				"limit == 0 must be uncapped and yield every match")
			require.Equal(t, tc.total, count(-1),
				"a negative limit must be uncapped and yield every match")

			// A positive limit below the total stops after exactly that many.
			require.Equal(t, 2, count(2),
				"a positive limit must stop after exactly that many matches")

			// A positive limit above the total yields all matches.
			require.Equal(t, tc.total, count(tc.total+5),
				"a limit larger than the match count yields all matches")
		})
	}
}

// A backtracking-shaped but RE2-compatible leading-context pattern such as
// ^(a+)+b stays on Go's linear RE2 engine — it must NOT be routed through the
// backtracking regexp2 engine, where the nested quantifier over a non-matching
// input would explode into catastrophic backtracking and trip the match timeout
// (surfacing as XTDE1140 in xsl:analyze-string). On RE2 it completes promptly
// with no match and no error regardless of input length.
func TestRegexEachSubmatchIndexLeadingContextStaysLinear(t *testing.T) {
	t.Parallel()

	re, err := xpath3.CompileRegex("^(a+)+b", "")
	require.NoError(t, err)

	// 50 'a's with no trailing 'b': a backtracking engine would explore ~2^50
	// splits (well past the 5s default timeout); RE2 dispatches it in linear time.
	input := strings.Repeat("a", 50)

	done := make(chan struct{})
	var matches int
	var eachErr error
	go func() {
		eachErr = re.EachSubmatchIndex(input, -1, func(_ []int) bool {
			matches++
			return true
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("EachSubmatchIndex did not return promptly; pattern likely ran on the backtracking engine")
	}
	require.NoError(t, eachErr)
	require.Zero(t, matches, "^(a+)+b must not match a run of only 'a'")
}
