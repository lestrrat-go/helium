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
