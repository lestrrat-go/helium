package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The ?keyword lookup-key syntax accepts XPath keyword tokens as NCName keys,
// exercising tokenAsNCName across its keyword cases.
func TestLookupKeyword_NCNameKeys(t *testing.T) {
	// Each keyword is used both as the map key and as the lookup key.
	keywords := []string{
		"div", "mod", "and", "or", "return", "else", "eq", "ne", "lt", "le",
		"gt", "ge", "idiv", "if", "then", "for", "let", "in", "some", "every",
		"satisfies", "is", "to", "union", "intersect", "except", "instance",
		"treat", "castable", "cast", "as", "of",
	}
	for _, kw := range keywords {
		expr := `map { "` + kw + `": 42 }?` + kw
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.Equal(t, "42", r.StringValue(), expr)
	}

	// Plain NCName key.
	r, err := evaluate(t.Context(), nil, `map { "plain": 7 }?plain`)
	require.NoError(t, err)
	require.Equal(t, "7", r.StringValue())

	// Array unary-lookup with a wildcard.
	r, err = evaluate(t.Context(), nil, `[1, 2, 3]?*`)
	require.NoError(t, err)
	require.Equal(t, "1 2 3", r.StringValue())
}
