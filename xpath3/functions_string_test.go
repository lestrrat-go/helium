package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestReplace_EdgeBranches(t *testing.T) {
	// Empty input string yields "".
	r, err := evaluate(t.Context(), nil, `replace((), "a", "b")`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())

	// 'q' literal flag: replacement is treated literally.
	r, err = evaluate(t.Context(), nil, `replace("a.b.c", ".", "X", "q")`)
	require.NoError(t, err)
	require.Equal(t, "aXbXc", r.StringValue())

	// Simple class with empty replacement -> rune-filter fast path.
	r, err = evaluate(t.Context(), nil, `replace("a1b2c3", "\p{Nd}", "")`)
	require.NoError(t, err)
	require.Equal(t, "abc", r.StringValue())

	// Negated class with empty replacement.
	r, err = evaluate(t.Context(), nil, `replace("a1b2c3", "\P{Nd}", "")`)
	require.NoError(t, err)
	require.Equal(t, "123", r.StringValue())

	// Backreference in replacement ($1).
	r, err = evaluate(t.Context(), nil, `replace("2023-06-22", "(\d+)-(\d+)-(\d+)", "$3/$2/$1")`)
	require.NoError(t, err)
	require.Equal(t, "22/06/2023", r.StringValue())

	// Pattern matching the empty string -> FORX0003.
	_, err = evaluate(t.Context(), nil, `replace("abc", "x*", "y")`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}

func TestTokenize_EdgeBranches(t *testing.T) {
	// Whitespace-normalizing single-arg tokenize.
	r, err := evaluate(t.Context(), nil, `tokenize("  a  b  c  ")`)
	require.NoError(t, err)
	atoms, err := r.Atomics()
	require.NoError(t, err)
	require.Len(t, atoms, 3)

	// Empty input -> empty sequence.
	r, err = evaluate(t.Context(), nil, `tokenize(())`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())

	// Case-insensitive flag.
	r, err = evaluate(t.Context(), nil, `tokenize("aXbxc", "x", "i")`)
	require.NoError(t, err)
	atoms, err = r.Atomics()
	require.NoError(t, err)
	require.Len(t, atoms, 3)
}

func TestAnalyzeString_EdgeBranches(t *testing.T) {
	// Mix of matches and non-matches.
	r, err := evaluate(t.Context(), nil,
		`fn:analyze-string("a1b2", "\d")//fn:match => count()`)
	require.NoError(t, err)
	n, ok := r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)

	r, err = evaluate(t.Context(), nil,
		`fn:analyze-string("a1b2", "\d")//fn:non-match => count()`)
	require.NoError(t, err)
	n, ok = r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)

	// Capture groups generate fn:group children.
	r, err = evaluate(t.Context(), nil,
		`fn:analyze-string("2023", "(\d)(\d)")//fn:group => count()`)
	require.NoError(t, err)
	n, ok = r.IsNumber()
	require.True(t, ok)
	require.Positive(t, n)

	// Empty input -> a result element with no match/non-match.
	r, err = evaluate(t.Context(), nil,
		`fn:analyze-string("", "\d")//fn:match => count()`)
	require.NoError(t, err)
	n, ok = r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(0), n)

	// Case-insensitive flag.
	r, err = evaluate(t.Context(), nil,
		`fn:analyze-string("aAbB", "a", "i")//fn:match => count()`)
	require.NoError(t, err)
	n, ok = r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)

	// Pattern matching empty string -> FORX0003.
	_, err = evaluate(t.Context(), nil, `fn:analyze-string("ab", "x*")`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}

// Nested capturing groups must produce nested fn:group elements (F&O 3.1
// §5.6.5): an outer group holds its own text plus the fn:group children for the
// groups nested within it, in document order, each with the correct @nr.
func TestAnalyzeString_NestedGroups(t *testing.T) {
	const fnNS = ` xmlns:fn="http://www.w3.org/2005/xpath-functions"`
	for _, tc := range []struct {
		name string
		expr string
		want string
	}{
		{
			name: "sibling groups nested under outer group",
			expr: `fn:serialize(fn:analyze-string("abc", "((a)(b))c"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:match><fn:group nr="1"><fn:group nr="2">a</fn:group><fn:group nr="3">b</fn:group></fn:group>c</fn:match></fn:analyze-string-result>`,
		},
		{
			// QT3 analyzeString-008: nested captured groups.
			name: "nested optional group with text before it",
			expr: `fn:serialize(fn:analyze-string("banana", "(a(n?))"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:non-match>b</fn:non-match><fn:match><fn:group nr="1">a<fn:group nr="2">n</fn:group></fn:group></fn:match><fn:match><fn:group nr="1">a<fn:group nr="2">n</fn:group></fn:group></fn:match><fn:match><fn:group nr="1">a<fn:group nr="2"/></fn:group></fn:match></fn:analyze-string-result>`,
		},
		{
			// QT3 analyzeString-017a: empty nested captured group is still nested.
			name: "empty nested group stays nested after text",
			expr: `fn:serialize(fn:analyze-string("banana", "(b(x?))"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:match><fn:group nr="1">b<fn:group nr="2"/></fn:group></fn:match><fn:non-match>anana</fn:non-match></fn:analyze-string-result>`,
		},
		{
			// "#" is a LITERAL regex character under the "x" flag (XPath 3.1 has
			// no "#" comments), and "(a)" is a real capturing group. This must not
			// panic (regression: a "#"-comment misread dropped the group, so the
			// derived group count fell short of the match's and indexed out of range).
			name: "x-flag hash is literal not a comment",
			expr: `fn:serialize(fn:analyze-string("#a", "#(a)", "x"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:match>#<fn:group nr="1">a</fn:group></fn:match></fn:analyze-string-result>`,
		},
		{
			// The "x" flag removes unescaped whitespace, so "( a )" nested in
			// "( (b) )" is the same as "((b))": group 2 nested in group 1.
			name: "x-flag whitespace nested groups",
			expr: `fn:serialize(fn:analyze-string("b", "( (b) )", "x"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:match><fn:group nr="1"><fn:group nr="2">b</fn:group></fn:group></fn:match></fn:analyze-string-result>`,
		},
		{
			// QT3 analyzeString-017: the sibling counterpart of 017a — identical
			// match spans but separate parentheses, so the empty group 2 is a
			// SIBLING of group 1, not nested. Nesting is a static property of the
			// pattern, not of the match positions.
			name: "empty sibling group is not nested",
			expr: `fn:serialize(fn:analyze-string("banana", "(b)(x?)"))`,
			want: `<fn:analyze-string-result` + fnNS + `><fn:match><fn:group nr="1">b</fn:group><fn:group nr="2"/></fn:match><fn:non-match>anana</fn:non-match></fn:analyze-string-result>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := evalString(t, tc.expr)
			require.Equal(t, tc.want, got)
		})
	}
}

// The "x" flag removes EXACTLY the four XML whitespace characters (#x9, #xA,
// #xD, #x20) outside character classes — never other Unicode spaces. A U+00A0
// (NBSP) in the pattern is a LITERAL character and must be preserved, across
// every regex function (fn:matches/replace/tokenize/analyze-string share
// stripFreeSpacing).
func TestRegexXFlagWhitespaceExactSet(t *testing.T) {
	const nbsp = " "
	boolResult := func(t *testing.T, expr string) bool {
		t.Helper()
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err)
		b, ok := r.IsBoolean()
		require.True(t, ok, "expected boolean for %q", expr)
		return b
	}

	// NBSP in pattern stays literal: it only matches an NBSP in the input.
	require.True(t, boolResult(t,
		`fn:matches("a`+nbsp+`b", "a`+nbsp+`b", "x")`),
		"NBSP in pattern must match NBSP in input under x flag")
	require.False(t, boolResult(t,
		`fn:matches("ab", "a`+nbsp+`b", "x")`),
		"NBSP in pattern must NOT be stripped under x flag")

	// An ordinary space IS stripped under x, so "a b" matches "ab".
	require.True(t, boolResult(t, `fn:matches("ab", "a b", "x")`))

	// analyze-string shares the same stripping: an NBSP-bearing pattern produces
	// a match (not a non-match) for the NBSP-bearing input.
	r, err := evaluate(t.Context(), nil,
		`fn:analyze-string("a`+nbsp+`b", "a`+nbsp+`b", "x")//fn:match => count()`)
	require.NoError(t, err)
	n, ok := r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(1), n)
}

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
