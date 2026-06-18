package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// map:merge's optional second argument is a single map. A non-map options
// argument (or a multi-item sequence) must raise XPTY0004 rather than being
// silently ignored.
func TestMapMergeRejectsNonMapOptions(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	cases := []string{
		`map:merge((map{"a": 1}), "not-a-map")`,
		`map:merge((map{"a": 1}), 42)`,
		`map:merge((map{"a": 1}), (map{"duplicates": "use-last"}, map{}))`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
		})
	}
}

// map:merge's "duplicates" option accepts the F&O 3.1 defined values:
// use-first, use-last, use-any, combine, reject.
func TestMapMergeAcceptsDuplicatesOptions(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	cases := []string{
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": "use-first"})`,
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": "use-last"})`,
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": "use-any"})`,
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": "combine"})`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.NoError(t, err, expr)
		})
	}
}

// An invalid "duplicates" option value must raise FOJS0005 rather than being
// silently accepted.
func TestMapMergeRejectsInvalidDuplicates(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	expr := `map:merge((map{"a": 1}), map{"duplicates": "bogus"})`
	_, err := evaluate(t.Context(), doc, expr)
	require.Error(t, err, expr)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOJS0005", xpErr.Code, expr)
}
