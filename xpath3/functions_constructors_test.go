package xpath3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// requireXPathErrorCode evaluates expr and asserts it fails with a structured
// XPathError whose Code is exactly want.
func requireXPathErrorCode(t *testing.T, doc helium.Node, expr, want string) {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
	var xerr *xpath3.XPathError
	require.True(t, errors.As(err, &xerr), "error must be *xpath3.XPathError, got %T: %v", err, err)
	require.Equal(t, want, xerr.Code)
}

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

	t.Run("array atomizing to multiple values rejected", func(t *testing.T) {
		// An array argument that ATOMIZES to more than one value must be
		// rejected with XPTY0004 (a cardinality error), not FOTY0013.
		for _, expr := range []string{
			`xs:NMTOKENS(["a", "b"])`,
			`xs:IDREFS(["a", "b"])`,
			`xs:ENTITIES(["a", "b"])`,
		} {
			t.Run(expr, func(t *testing.T) {
				requireXPathErrorCode(t, doc, expr, "XPTY0004")
			})
		}
	})

	t.Run("non-atomizable later item surfaces FOTY0013", func(t *testing.T) {
		// A map cannot be atomized (FOTY0013). The whole argument is atomized
		// FIRST, so the un-atomizable item must surface its error even when it
		// appears after the second member — it must NOT be masked by the
		// XPTY0004 cardinality error.
		for _, expr := range []string{
			`xs:NMTOKENS(["a", "b", map{"x":"y"}])`,
			`xs:IDREFS(["a", "b", map{"x":"y"}])`,
			`xs:ENTITIES(["a", "b", map{"x":"y"}])`,
		} {
			t.Run(expr, func(t *testing.T) {
				requireXPathErrorCode(t, doc, expr, "FOTY0013")
			})
		}
	})

	t.Run("cardinality error names the list type", func(t *testing.T) {
		// The XPTY0004 message must name the list type (e.g. xs:NMTOKENS),
		// not the per-token item type (xs:NMTOKEN).
		for _, tc := range []struct {
			expr     string
			listType string
		}{
			{`xs:NMTOKENS(("a", "b"))`, "xs:NMTOKENS"},
			{`xs:IDREFS(("a", "b"))`, "xs:IDREFS"},
			{`xs:ENTITIES(("a", "b"))`, "xs:ENTITIES"},
		} {
			t.Run(tc.expr, func(t *testing.T) {
				compiled, err := xpath3.NewCompiler().Compile(tc.expr)
				require.NoError(t, err)
				_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.listType)
			})
		}
	})

	t.Run("empty string error names the list type", func(t *testing.T) {
		// An empty lexical list (e.g. xs:NMTOKENS("")) is a cast failure whose
		// message must name the LIST type (xs:NMTOKENS), not the per-token item
		// type (xs:NMTOKEN).
		for _, tc := range []struct {
			expr     string
			listType string
		}{
			{`xs:NMTOKENS("")`, "xs:NMTOKENS"},
			{`xs:IDREFS("")`, "xs:IDREFS"},
			{`xs:ENTITIES("")`, "xs:ENTITIES"},
		} {
			t.Run(tc.expr, func(t *testing.T) {
				compiled, err := xpath3.NewCompiler().Compile(tc.expr)
				require.NoError(t, err)
				_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.listType)
			})
		}
	})

	t.Run("empty array returns empty (not an error)", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`count(xs:NMTOKENS([]))`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(0), av.IntegerVal())
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

// TestXSScalarConstructorArrayCardinality verifies that scalar constructors,
// which share atomizeConstructorArg with the list constructors, reject an
// array argument that atomizes to more than one value with XPTY0004 rather
// than FOTY0013.
func TestXSScalarConstructorArrayCardinality(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	for _, expr := range []string{
		`xs:string(["a", "b"])`,
		`xs:dateTimeStamp(["2000-01-01T00:00:00Z", "2001-01-01T00:00:00Z"])`,
	} {
		t.Run(expr, func(t *testing.T) {
			requireXPathErrorCode(t, doc, expr, "XPTY0004")
		})
	}
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

// TestXSDateTimeStampSubtype verifies that xs:dateTimeStamp is a subtype of
// xs:dateTime: the constructor is idempotent for its own type, and xs:dateTime
// accepts an xs:dateTimeStamp value (subtype substitutability).
func TestXSDateTimeStampSubtype(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("dateTimeStamp of dateTimeStamp is idempotent", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTimeStamp(xs:dateTimeStamp("2000-01-01T00:00:00Z")) eq xs:dateTimeStamp("2000-01-01T00:00:00Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("dateTime accepts a dateTimeStamp value", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTime(xs:dateTimeStamp("2000-01-01T00:00:00Z")) eq xs:dateTime("2000-01-01T00:00:00Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("date accepts a dateTimeStamp value", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:date(xs:dateTimeStamp("2000-01-01T00:00:00Z")) eq xs:date("2000-01-01Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("time accepts a dateTimeStamp value", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:time(xs:dateTimeStamp("2000-01-01T00:00:00Z")) eq xs:time("00:00:00Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})
}

// TestXSDateTimeStampInvalidString verifies that an invalid string lexical
// passed to xs:dateTimeStamp reports FORG0001 naming xs:dateTimeStamp rather
// than the xs:dateTime fallback target.
func TestXSDateTimeStampInvalidString(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("invalid string names xs:dateTimeStamp", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTimeStamp("not-a-date")`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "error must be *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FORG0001", xerr.Code)
		require.Contains(t, xerr.Error(), "xs:dateTimeStamp")
		require.NotContains(t, xerr.Error(), "xs:dateTime\"")
	})
}

// TestXSDateTimeStampNodeValue verifies that xs:dateTimeStamp accepts an
// atomized node value (xs:untypedAtomic). A node whose text content is a valid
// dateTimeStamp lexical with a timezone succeeds; a node missing the mandatory
// timezone fails with FORG0001.
func TestXSDateTimeStampNodeValue(t *testing.T) {
	t.Run("node value with timezone accepted", func(t *testing.T) {
		doc := mustParseXML(t, "<root>2000-01-01T00:00:00Z</root>")
		compiled, err := xpath3.NewCompiler().Compile(`xs:dateTimeStamp(/root) eq xs:dateTimeStamp("2000-01-01T00:00:00Z")`)
		require.NoError(t, err)
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("node value without timezone rejected", func(t *testing.T) {
		doc := mustParseXML(t, "<root>2000-01-01T00:00:00</root>")
		requireXPathErrorCode(t, doc, `xs:dateTimeStamp(/root)`, "FORG0001")
	})
}
