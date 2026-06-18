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
		// Per F&O 3.1 option conventions the value is converted with the
		// function conversion rules, so xs:string subtypes, xs:anyURI, and a
		// single-item array all coerce to the string policy.
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": xs:NCName("use-last")})`,
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": xs:anyURI("use-first")})`,
		`map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": ["combine"]})`,
		// reject with no duplicate keys succeeds.
		`map:merge((map{"a": 1}, map{"b": 2}), map{"duplicates": "reject"})`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.NoError(t, err, expr)
		})
	}
}

// The "reject" duplicates policy raises FOJS0003 when an actual duplicate key
// is encountered.
func TestMapMergeRejectDuplicateKey(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	expr := `map:merge((map{"a": 1}, map{"a": 2}), map{"duplicates": "reject"})`
	_, err := evaluate(t.Context(), doc, expr)
	require.Error(t, err, expr)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOJS0003", xpErr.Code, expr)
}

// An invalid "duplicates" option value must raise FOJS0005 rather than being
// silently accepted. Per F&O 3.1 this covers empty, multi-item, non-string,
// and unknown values alike.
func TestMapMergeRejectsInvalidDuplicates(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	cases := []string{
		`map:merge((map{"a": 1}), map{"duplicates": "bogus"})`,
		`map:merge((map{"a": 1}), map{"duplicates": 42})`,
		`map:merge((map{"a": 1}), map{"duplicates": ()})`,
		`map:merge((map{"a": 1}), map{"duplicates": ("use-first", "use-last")})`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, "FOJS0005", xpErr.Code, expr)
		})
	}
}

// A "duplicates" value that is a custom atomic whose Go payload is a string but
// whose BaseType is NOT string-derived must still raise FOJS0005. The option is
// declared as xs:string, so a non-string-based atom is not convertible even when
// its underlying Go representation happens to be a string.
func TestMapMergeRejectsNonStringBasedDuplicates(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile(
		`map:merge((map{"a": 1}), map{"duplicates": $dup})`)
	require.NoError(t, err)

	// A user-defined atomic derived from xs:integer that carries a Go string
	// payload. The pre-fix coercion accepted any atom with a string payload.
	dup := xpath3.AtomicValue{
		TypeName: "Q{urn:x}strishInt",
		Value:    "use-last",
		BaseType: xpath3.TypeInteger,
	}
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
			"dup": xpath3.ItemSlice{dup},
		})).
		Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOJS0005", xpErr.Code)
}
