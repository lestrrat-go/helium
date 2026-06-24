package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const qt3CaseblindCollationURI = "http://www.w3.org/2010/09/qt-fots-catalog/collation/caseblind"

func TestQT3CaseblindCollation(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("explicit collation argument", func(t *testing.T) {
		tests := []struct {
			name string
			expr string
			want func(t *testing.T, seq xpath3.Sequence)
		}{
			{
				name: "compare",
				expr: `compare("a", "A", "` + qt3CaseblindCollationURI + `")`,
				want: func(t *testing.T, seq xpath3.Sequence) {
					t.Helper()
					require.Len(t, seq, 1)
					require.Equal(t, int64(0), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
				},
			},
			{
				name: "deep-equal",
				expr: `deep-equal(("a", "A"), ("A", "a"), "` + qt3CaseblindCollationURI + `")`,
				want: func(t *testing.T, seq xpath3.Sequence) {
					t.Helper()
					require.Len(t, seq, 1)
					require.True(t, seq.Get(0).(xpath3.AtomicValue).BooleanVal())
				},
			},
			{
				name: "substring-after",
				expr: `substring-after("banana", "A", "` + qt3CaseblindCollationURI + `")`,
				want: func(t *testing.T, seq xpath3.Sequence) {
					t.Helper()
					require.Len(t, seq, 1)
					require.Equal(t, "nana", seq.Get(0).(xpath3.AtomicValue).StringVal())
				},
			},
			{
				name: "substring-before",
				expr: `substring-before("banana", "A", "` + qt3CaseblindCollationURI + `")`,
				want: func(t *testing.T, seq xpath3.Sequence) {
					t.Helper()
					require.Len(t, seq, 1)
					require.Equal(t, "b", seq.Get(0).(xpath3.AtomicValue).StringVal())
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tc.want(t, evalExpr(t, doc, tc.expr))
			})
		}
	})

	t.Run("default collation", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			DefaultCollation(qt3CaseblindCollationURI)
		seq := evalExprWithEval(t, eval, doc, `compare("a", "A")`)
		require.Len(t, seq, 1)
		require.Equal(t, int64(0), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
	})
}

const (
	collationCodepoint = "http://www.w3.org/2005/xpath-functions/collation/codepoint"
	collationHTMLASCII = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"
	collationUCA       = "http://www.w3.org/2013/collation/UCA"
)

func TestCollationHelpers(t *testing.T) {
	require.True(t, xpath3.IsCollationSupported(collationCodepoint))
	require.True(t, xpath3.IsCollationSupported(collationHTMLASCII))
	require.False(t, xpath3.IsCollationSupported("urn:bogus-collation"))

	keyFn, err := xpath3.ResolveCollationKeyFunc(collationCodepoint)
	require.NoError(t, err)
	require.Equal(t, keyFn("abc"), keyFn("abc"))
	require.NotEqual(t, keyFn("abc"), keyFn("abd"))

	_, err = xpath3.ResolveCollationKeyFunc("urn:bogus")
	require.Error(t, err)

	cmpFn, err := xpath3.ResolveCollationCompareFunc(collationCodepoint)
	require.NoError(t, err)
	require.Equal(t, 0, cmpFn("a", "a"))
	require.Negative(t, cmpFn("a", "b"))
	require.Positive(t, cmpFn("b", "a"))

	_, err = xpath3.ResolveCollationCompareFunc("urn:bogus")
	require.Error(t, err)

	require.False(t, xpath3.CollationHasUnsupportedOptions(collationCodepoint))
	require.False(t, xpath3.CollationHasUnsupportedOptions(collationUCA))
	require.True(t, xpath3.CollationHasUnsupportedOptions(collationUCA+"?alternate=shifted"))
}
