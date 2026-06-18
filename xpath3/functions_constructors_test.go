package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestXSTokenListCardinality verifies that the list constructors
// xs:NMTOKENS / xs:IDREFS / xs:ENTITIES enforce a single atomized
// argument and reject a sequence of more than one item with XPTY0004.
func TestXSTokenListCardinality(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("multiple items rejected", func(t *testing.T) {
		for _, expr := range []string{
			`xs:NMTOKENS(("a", "b"))`,
			`xs:IDREFS(("a", "b"))`,
			`xs:ENTITIES(("a", "b"))`,
		} {
			t.Run(expr, func(t *testing.T) {
				compiled, err := xpath3.NewCompiler().Compile(expr)
				require.NoError(t, err)
				_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
				require.Error(t, err)
				require.Contains(t, err.Error(), "XPTY0004")
			})
		}
	})

	t.Run("single whitespace-separated arg splits into tokens", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`count(xs:NMTOKENS("a b c")) eq 3`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})
}

// TestXSDateTimeStampWhitespace verifies that xs:dateTimeStamp accepts a
// lexical value with leading/trailing whitespace, since whitespace is
// collapsed before the timezone is checked.
func TestXSDateTimeStampWhitespace(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("whitespace-padded value accepted", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTimeStamp(" 2000-01-01T00:00:00Z ") eq xs:dateTimeStamp("2000-01-01T00:00:00Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("missing timezone still rejected", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTimeStamp(" 2000-01-01T00:00:00 ")`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "FORG0001")
	})
}
