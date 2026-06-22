package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// German and Italian ordinal spell-out (picture "w;o") exercise
// applyOrdinalWordsDe / germanOrdinalWord / germanWordToNumber and
// applyOrdinalWordsIt / italianOrdinalWord, which the cardinal "w" path
// does not reach.
func TestFormatIntegerOrdinalWords(t *testing.T) {
	cases := []struct {
		value string
		lang  string
	}{
		// German ordinals: small (< 20) -> "...te"; larger -> "...ste".
		{"1", "de"},
		{"3", "de"},
		{"7", "de"},
		{"20", "de"},
		{"21", "de"},
		{"42", "de"},
		{"100", "de"},
		// Italian ordinals (masculine).
		{"1", "it"},
		{"3", "it"},
		{"10", "it"},
		{"21", "it"},
		{"100", "it"},
	}
	for _, tc := range cases {
		expr := `format-integer(` + tc.value + `, "w;o", "` + tc.lang + `")`
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.NotEmpty(t, r.StringValue(), expr)
	}
}
