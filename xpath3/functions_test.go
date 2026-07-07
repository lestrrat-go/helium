package xpath3_test

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// --- Boolean functions ---

func TestFnBoolean(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect bool
	}{
		{`boolean(1)`, true},
		{`boolean(0)`, false},
		{`boolean("")`, false},
		{`boolean("x")`, true},
		{`not(true())`, false},
		{`not(false())`, true},
		{`true()`, true},
		{`false()`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			require.Equal(t, tc.expect, av.BooleanVal())
		})
	}
}

// --- String functions ---

func TestFnString(t *testing.T) {
	doc := mustParseXML(t, `<root>hello</root>`)

	tests := []struct {
		expr   string
		expect string
	}{
		{`concat("a", "b", "c")`, "abc"},
		{`string-join(("a","b","c"), "-")`, "a-b-c"},
		{`upper-case("hello")`, "HELLO"},
		{`lower-case("HELLO")`, testHello},
		{`substring("12345", 2, 3)`, "234"},
		{`substring("12345", 2)`, "2345"},
		{`starts-with("hello", "he")`, lexicon.ValueTrue},
		{`ends-with("hello", "lo")`, lexicon.ValueTrue},
		{`contains("hello world", "world")`, lexicon.ValueTrue},
		{`substring-before("hello-world", "-")`, testHello},
		{`substring-after("hello-world", "-")`, "world"},
		{`translate("abc", "abc", "ABC")`, "ABC"},
		{`normalize-space("  hello   world  ")`, "hello world"},
		{`string-length("hello")`, "5"},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			s, _ := xpath3.CastAtomic(av, xpath3.TypeString)
			require.Equal(t, tc.expect, s.StringVal())
		})
	}
}

func TestFnMatches(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, `matches("hello", "^hel")`)
	require.Equal(t, 1, seq.Len())
	av := seq.Get(0).(xpath3.AtomicValue)
	require.True(t, av.BooleanVal())

	seq = evalExpr(t, doc, `matches(codepoints-to-string(983040), "[\p{IsPrivateUse}]")`)
	require.Equal(t, 1, seq.Len())
	av = seq.Get(0).(xpath3.AtomicValue)
	require.True(t, av.BooleanVal())
}

func TestFnReplace(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, `replace("hello world", "world", "Go")`)
	require.Equal(t, 1, seq.Len())
	av := seq.Get(0).(xpath3.AtomicValue)
	require.Equal(t, "hello Go", av.StringVal())
}

func TestFnTokenize(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("with pattern", func(t *testing.T) {
		seq := evalExpr(t, doc, `tokenize("a,b,c", ",")`)
		require.Equal(t, 3, seq.Len())
	})

	t.Run("whitespace", func(t *testing.T) {
		seq := evalExpr(t, doc, `tokenize("  a  b  c  ")`)
		require.Equal(t, 3, seq.Len())
	})
}

// --- Numeric functions ---

func TestFnNumeric(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect float64
	}{
		{`abs(-5)`, 5},
		{`abs(5)`, 5},
		{`ceiling(4.2)`, 5},
		{`floor(4.8)`, 4},
		{`round(4.5)`, 5},
		{`round(4.4)`, 4},
		{`round-half-to-even(2.5)`, 2},
		{`round-half-to-even(3.5)`, 4},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			require.InDelta(t, tc.expect, av.ToFloat64(), 0.001)
		})
	}
}

// TestFnRoundPrecision exercises fn:round and fn:round-half-to-even at
// negative, zero, and positive precision over positive and negative operands,
// including div-derived rationals. fn:round rounds the half toward positive
// infinity; a regression in the negative-precision path floored the rational to
// an integer before rounding, pushing values such as -249.9 past the boundary
// (yielding -300 instead of -200).
func TestFnRoundPrecision(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect string
	}{
		// Negative precision, half-up (toward +∞).
		{`round(-249.9, -2)`, "-200"},
		{`round(-250, -2)`, "-200"}, // tie -2.5 → -2 toward +∞
		{`round(250.1, -2)`, "300"},
		{`round(-1234.567, -2)`, "-1200"}, // QT3 fn-round-decimal-11
		{`round(8452, -2)`, "8500"},       // QT3 fn-round2args-2
		// Zero precision / no precision arg, half-up (toward +∞).
		{`round(-0.5)`, "0"},
		{`round(2.5)`, "3"},
		{`round(2.4)`, "2"},
		{`round(-2.5, 0)`, "-2"},
		// Positive precision, half-up.
		{`round(2.4567, 2)`, "2.46"},
		{`round(-2.345, 2)`, "-2.34"}, // tie toward +∞
		// Div-derived rational input (1 div 3 etc.).
		{`round((-2499 div 10), -2)`, "-200"},
		{`round((2501 div 10), -2)`, "300"},
		// Negative precision, half-to-even.
		{`round-half-to-even(-250, -2)`, "-200"},
		{`round-half-to-even(-350, -2)`, "-400"}, // tie → even
		{`round-half-to-even(-249.9, -2)`, "-200"},
		{`round-half-to-even(150, -2)`, "200"}, // tie → even
		{`round-half-to-even(250, -2)`, "200"}, // tie → even
		// Zero / positive precision, half-to-even.
		{`round-half-to-even(2.5)`, "2"},
		{`round-half-to-even(3.5)`, "4"},
		{`round-half-to-even(-2.5)`, "-2"},
		{`round-half-to-even(3.567, 2)`, "3.57"},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Evaluate(t.Context(), compiled, doc)
			require.NoError(t, err)
			require.Equal(t, tc.expect, result.StringValue())
		})
	}
}

// --- Aggregate functions ---

func TestFnAggregate(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("count", func(t *testing.T) {
		seq := evalExpr(t, doc, `count((1, 2, 3))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(3), av.IntegerVal())
	})

	t.Run("sum", func(t *testing.T) {
		seq := evalExpr(t, doc, `sum((1, 2, 3))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 6.0, av.ToFloat64(), 0.001)
	})

	t.Run("avg", func(t *testing.T) {
		seq := evalExpr(t, doc, `avg((2, 4, 6))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 4.0, av.ToFloat64(), 0.001)
	})

	t.Run("min", func(t *testing.T) {
		seq := evalExpr(t, doc, `min((3, 1, 2))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 1.0, av.ToFloat64(), 0.001)
	})

	t.Run("max", func(t *testing.T) {
		seq := evalExpr(t, doc, `max((3, 1, 2))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 3.0, av.ToFloat64(), 0.001)
	})
}

// --- Sequence functions ---

func TestFnSequence(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("empty", func(t *testing.T) {
		seq := evalExpr(t, doc, `empty(())`)
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("exists", func(t *testing.T) {
		seq := evalExpr(t, doc, `exists((1))`)
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("head", func(t *testing.T) {
		seq := evalExpr(t, doc, `head((1, 2, 3))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(1), av.IntegerVal())
	})

	t.Run("tail", func(t *testing.T) {
		seq := evalExpr(t, doc, `tail((1, 2, 3))`)
		require.Equal(t, 2, seq.Len())
	})

	t.Run("reverse", func(t *testing.T) {
		seq := evalExpr(t, doc, `reverse((1, 2, 3))`)
		require.Equal(t, 3, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(3), av.IntegerVal())
	})

	t.Run("subsequence", func(t *testing.T) {
		seq := evalExpr(t, doc, `subsequence((1, 2, 3, 4), 2, 2)`)
		require.Equal(t, 2, seq.Len())
	})

	t.Run("distinct-values", func(t *testing.T) {
		seq := evalExpr(t, doc, `distinct-values((1, 2, 1, 3, 2))`)
		require.Equal(t, 3, seq.Len())
	})

	t.Run("distinct-values mixed numerics", func(t *testing.T) {
		seq := evalExpr(t, doc, `distinct-values((xs:float('1'), xs:double('1'), xs:decimal('1.0'), xs:integer('1'), xs:float('0.1'), xs:decimal('0.1')))`)
		require.Equal(t, 2, seq.Len())
	})
}

// --- Node functions ---

func TestFnNode(t *testing.T) {
	doc := mustParseXML(t, `<root><a/><b/></root>`)
	root := doc.DocumentElement()

	t.Run("local-name", func(t *testing.T) {
		seq := evalExpr(t, root, `local-name()`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "root", av.StringVal())
	})

	t.Run("name", func(t *testing.T) {
		seq := evalExpr(t, root, `name()`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "root", av.StringVal())
	})

	t.Run("has-children", func(t *testing.T) {
		seq := evalExpr(t, root, `has-children()`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("count children", func(t *testing.T) {
		seq := evalExpr(t, root, `count(*)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("position and last", func(t *testing.T) {
		seq := evalExpr(t, root, `*[position() = last()]`)
		require.Equal(t, 1, seq.Len())
		ni := seq.Get(0).(xpath3.NodeItem)
		require.Equal(t, "b", ni.Node.Name())
	})
}

// --- Math functions ---

func TestFnMath(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("math:pi", func(t *testing.T) {
		seq := evalExpr(t, doc, `math:pi()`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, math.Pi, av.DoubleVal(), 0.0001)
	})

	t.Run("math:sqrt", func(t *testing.T) {
		seq := evalExpr(t, doc, `math:sqrt(4)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 2.0, av.DoubleVal(), 0.001)
	})

	t.Run("math:pow", func(t *testing.T) {
		seq := evalExpr(t, doc, `math:pow(2, 10)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 1024.0, av.DoubleVal(), 0.001)
	})
}

// --- HOF functions ---

func TestFnHOF(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("for-each", func(t *testing.T) {
		seq := evalExpr(t, doc, `for-each((1, 2, 3), function($x) { $x + 10 })`)
		require.Equal(t, 3, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 11.0, av.ToFloat64(), 0.001)
	})

	t.Run("filter", func(t *testing.T) {
		seq := evalExpr(t, doc, `filter((1, 2, 3, 4, 5), function($x) { $x > 3 })`)
		require.Equal(t, 2, seq.Len())
	})

	t.Run("fold-left", func(t *testing.T) {
		seq := evalExpr(t, doc, `fold-left((1, 2, 3), 0, function($acc, $x) { $acc + $x })`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 6.0, av.ToFloat64(), 0.001)
	})
}

// --- Map functions ---

func TestFnMap(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("map:size", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:size(map { "a": 1, "b": 2 })`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("map:contains", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:contains(map { "a": 1 }, "a")`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("map:get", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:get(map { "x": 42 }, "x")`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 42.0, av.ToFloat64(), 0.001)
	})

	// Arithmetic-created fractional dayTimeDuration keys must canonicalize
	// identically to parsed ones so map:get/map:contains succeed. These use
	// NON-binary-exact fractions (0.1, 1.1) which would diverge if arithmetic
	// stored the fraction as a binary float64 instead of an exact rational.
	t.Run("map:get arithmetic fractional duration key (multiply)", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:get(map { xs:dayTimeDuration("PT1.1S"): "x" }, xs:dayTimeDuration("PT11S") * 0.1)`)
		require.Equal(t, 1, seq.Len())
		require.Equal(t, "x", seq.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("map:get arithmetic fractional duration key (add)", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:get(map { xs:dayTimeDuration("PT0.1S"): "y" }, xs:dayTimeDuration("PT0.05S") + xs:dayTimeDuration("PT0.05S"))`)
		require.Equal(t, 1, seq.Len())
		require.Equal(t, "y", seq.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("map:contains arithmetic fractional duration key", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:contains(map { xs:dayTimeDuration("PT1.1S"): "x" }, xs:dayTimeDuration("PT11S") * 0.1)`)
		require.Equal(t, 1, seq.Len())
		require.True(t, seq.Get(0).(xpath3.AtomicValue).BooleanVal())
	})

	t.Run("map:keys", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:keys(map { "a": 1, "b": 2 })`)
		require.Equal(t, 2, seq.Len())
	})

	t.Run("map:put", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:size(map:put(map { "a": 1 }, "b", 2))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	// Regression: two distinct xs:integer keys whose magnitude exceeds
	// MaxFloat64 (~1.8e308) must not collide. Before the fix, both were
	// normalized through ToFloat64 -> +Inf and shared one bucket, producing
	// a false XQDY0137 duplicate-key error / wrong map size.
	t.Run("huge integer keys do not collide", func(t *testing.T) {
		k1 := "1" + strings.Repeat("0", 400) // > MaxFloat64
		k2 := "2" + strings.Repeat("0", 400) // distinct, also > MaxFloat64
		expr := `map:size(map { ` + k1 + `: "a", ` + k2 + `: "b" })`
		seq := evalExpr(t, doc, expr)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("huge integer key lookup", func(t *testing.T) {
		k1 := "1" + strings.Repeat("0", 400)
		k2 := "2" + strings.Repeat("0", 400)
		expr := `map:get(map { ` + k1 + `: "a", ` + k2 + `: "b" }, ` + k2 + `)`
		seq := evalExpr(t, doc, expr)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "b", av.StringVal())
	})

	// Guard: normal small integer keys are unaffected.
	t.Run("small integer keys", func(t *testing.T) {
		seq := evalExpr(t, doc, `map:size(map { 1: "a", 2: "b" })`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())

		got := evalExpr(t, doc, `map:get(map { 1: "a", 2: "b" }, 2)`)
		require.Equal(t, 1, got.Len())
		require.Equal(t, "b", got.Get(0).(xpath3.AtomicValue).StringVal())
	})

	// Guard: integer and decimal keys that compare equal (1 and 1.0) are the
	// same key per XPath "same key" rules, so the map constructor must reject
	// them as a duplicate (XQDY0137). This confirms the fix preserves
	// cross-type numeric key equality for in-range values.
	t.Run("integer and decimal equal keys are duplicate", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`map { 1: "a", 1.0: "b" }`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "XQDY0137")
	})

	// Guard: a huge integer key and the equal huge decimal key are still the
	// same key (both > MaxFloat64), and must be rejected as a duplicate.
	t.Run("huge integer and decimal equal keys are duplicate", func(t *testing.T) {
		big := "1" + strings.Repeat("0", 400)
		compiled, err := xpath3.NewCompiler().Compile(`map { ` + big + `: "a", ` + big + `.0: "b" }`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "XQDY0137")
	})
}

// --- Array functions ---

func TestFnArray(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("array:size", func(t *testing.T) {
		seq := evalExpr(t, doc, `array:size([1, 2, 3])`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(3), av.IntegerVal())
	})

	t.Run("array:get", func(t *testing.T) {
		seq := evalExpr(t, doc, `array:get([10, 20, 30], 2)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 20.0, av.ToFloat64(), 0.001)
	})

	t.Run("array:head", func(t *testing.T) {
		seq := evalExpr(t, doc, `array:head([10, 20, 30])`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 10.0, av.ToFloat64(), 0.001)
	})

	t.Run("array:tail", func(t *testing.T) {
		seq := evalExpr(t, doc, `array:size(array:tail([1, 2, 3]))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("array:reverse", func(t *testing.T) {
		seq := evalExpr(t, doc, `array:get(array:reverse([1, 2, 3]), 1)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.InDelta(t, 3.0, av.ToFloat64(), 0.001)
	})
}

// --- array position-argument integer validation ---

func TestFnArrayIntegerPositions(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected error code %s for %q", code, expr)
	}

	// Non-integer or wrong-cardinality position arguments must raise XPTY0004.
	t.Run("XPTY0004", func(t *testing.T) {
		for _, expr := range []string{
			`array:remove([1, 2], 1.5)`,
			`array:subarray([1, 2, 3], 1.5)`,
			`array:subarray([1, 2, 3], 2, 1.5)`,
			`array:subarray([1, 2, 3], ())`,
			`array:insert-before([1, 2], 1.5, 9)`,
			`array:insert-before([1, 2], (), 9)`,
		} {
			t.Run(expr, func(t *testing.T) {
				evalErrCode(t, expr, "XPTY0004")
			})
		}
	})

	// Out-of-range integer positions must raise FOAY0001.
	t.Run("FOAY0001", func(t *testing.T) {
		for _, expr := range []string{
			`array:remove([1, 2], 5)`,
			`array:insert-before([1, 2], 0, 9)`,
			// Huge start+length must not overflow into a make() panic.
			`array:subarray([1], 6917529027641081856, 6917529027641081856)`,
		} {
			t.Run(expr, func(t *testing.T) {
				evalErrCode(t, expr, "FOAY0001")
			})
		}
	})

	// Valid invocations must continue to work (no regression).
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			expr   string
			expect []int64
		}{
			{`array:remove([1, 2], 1)`, []int64{2}},
			{`array:remove([1, 2, 3], (1, 2))`, []int64{3}},
			{`array:remove([1, 2, 3], ())`, []int64{1, 2, 3}},
			{`array:subarray([1, 2, 3], 2)`, []int64{2, 3}},
			{`array:subarray([1, 2, 3], 2, 1)`, []int64{2}},
			{`array:insert-before([1, 2], 1, 9)`, []int64{9, 1, 2}},
			{`array:insert-before([1, 2], 3, 9)`, []int64{1, 2, 9}},
		}
		for _, tc := range tests {
			t.Run(tc.expr, func(t *testing.T) {
				seq := evalExpr(t, doc, tc.expr)
				require.Equal(t, 1, seq.Len())
				arr, ok := seq.Get(0).(xpath3.ArrayItem)
				require.True(t, ok, "expected array result")
				require.Equal(t, len(tc.expect), arr.Size())
				for i, want := range tc.expect {
					member, err := arr.Get(i + 1)
					require.NoError(t, err)
					require.Equal(t, 1, member.Len())
					av := member.Get(0).(xpath3.AtomicValue)
					require.Equal(t, want, av.IntegerVal())
				}
			})
		}
	})
}

// --- URI functions ---

func TestFnURI(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("encode-for-uri", func(t *testing.T) {
		seq := evalExpr(t, doc, `encode-for-uri("hello world")`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "hello%20world", av.StringVal())
	})
}

// --- codepoints-to-string range checks ---

func TestFnCodepointsToStringRange(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected error code %s for %q", code, expr)
	}

	t.Run("non-integer is a type error", func(t *testing.T) {
		evalErrCode(t, `codepoints-to-string(65.9)`, "XPTY0004")
	})

	t.Run("high-precision fractional decimal is a type error", func(t *testing.T) {
		// xs:decimal keeps full precision (*big.Rat); this value rounds to 65.0
		// as float64 and must not slip past the integrality check.
		evalErrCode(t, `codepoints-to-string(65.000000000000000000000000001)`, "XPTY0004")
	})

	t.Run("value beyond int64 is FOCH0001", func(t *testing.T) {
		// 2^64 + 65 wraps to 65 ("A") via big.Int.Int64(); must error instead.
		evalErrCode(t, `codepoints-to-string(18446744073709551681)`, "FOCH0001")
	})

	t.Run("huge integer-valued float is FOCH0001", func(t *testing.T) {
		// 1e300 is integer-valued so it passes the fractional check, but is far
		// beyond the codepoint range; int(f) is implementation-defined, so the
		// range must be validated before the conversion.
		evalErrCode(t, `codepoints-to-string(1e300)`, "FOCH0001")
	})

	t.Run("negative integer-valued float is FOCH0001", func(t *testing.T) {
		evalErrCode(t, `codepoints-to-string(-1.0)`, "FOCH0001")
	})

	t.Run("integer codepoint still works", func(t *testing.T) {
		seq := evalExpr(t, doc, `codepoints-to-string(65)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "A", av.StringVal())
	})

	t.Run("integer-valued float still works", func(t *testing.T) {
		seq := evalExpr(t, doc, `codepoints-to-string(65.0)`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "A", av.StringVal())
	})

	t.Run("sequence of integers still works", func(t *testing.T) {
		seq := evalExpr(t, doc, `codepoints-to-string((72, 73))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "HI", av.StringVal())
	})

	t.Run("round-trip via string-to-codepoints", func(t *testing.T) {
		seq := evalExpr(t, doc, `codepoints-to-string(string-to-codepoints("hi"))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, "hi", av.StringVal())
	})
}

// --- argument cardinality/type validation in math/map/json/hof builtins ---

func TestBuiltinArgValidation(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err, "compile %q", expr)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected error code %s for %q", code, expr)
	}

	t.Run("type errors", func(t *testing.T) {
		cases := []string{
			`math:sin("abc")`,
			`math:sin((1,2))`,
			`math:pow(2,"x")`,
			// An empty $x must not excuse an invalid/empty required $y:
			// function-conversion validates each argument independently.
			`math:pow((),"x")`,
			`math:pow((),())`,
			`math:atan2(1,(2,3))`,
			// atan2's $x ($y) is required; an empty other arg is still XPTY0004.
			`math:atan2((),1)`,
			`math:atan2(1,())`,
			`map:put(map{},(1,2),"v")`,
			`map:entry((1,2),"v")`,
			`parse-json("{}","bad")`,
			`parse-json("{}",(map{},map{}))`,
			`abs((1,2))`,
			`function-lookup(QName("http://www.w3.org/2005/xpath-functions/math","sin"),1)(("a","b"))`,
		}
		for _, expr := range cases {
			t.Run(expr, func(t *testing.T) {
				evalErrCode(t, expr, "XPTY0004")
			})
		}
	})

	t.Run("valid controls still work", func(t *testing.T) {
		t.Run("math:sin(0)", func(t *testing.T) {
			seq := evalExpr(t, doc, `math:sin(0)`)
			require.Equal(t, 1, seq.Len())
			require.InDelta(t, 0.0, seq.Get(0).(xpath3.AtomicValue).DoubleVal(), 0.0001)
		})
		t.Run("math:pow(2,3)", func(t *testing.T) {
			seq := evalExpr(t, doc, `math:pow(2,3)`)
			require.Equal(t, 1, seq.Len())
			require.InDelta(t, 8.0, seq.Get(0).(xpath3.AtomicValue).DoubleVal(), 0.0001)
		})
		t.Run("math:pow empty x valid y is empty", func(t *testing.T) {
			// QT3 math-pow-001: math:pow((), 93.7) is the empty sequence
			// because $y is a valid number. The empty-$x short-circuit only
			// fires after $y validates.
			seq := evalExpr(t, doc, `math:pow((), 93.7)`)
			require.Nil(t, seq)
		})
		t.Run("map:put singleton key", func(t *testing.T) {
			seq := evalExpr(t, doc, `map:size(map:put(map{},1,"v"))`)
			require.Equal(t, 1, seq.Len())
			require.Equal(t, int64(1), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
		})
		t.Run("parse-json no options", func(t *testing.T) {
			seq := evalExpr(t, doc, `parse-json("{}")`)
			require.Equal(t, 1, seq.Len())
		})
		t.Run("parse-json empty-map options", func(t *testing.T) {
			seq := evalExpr(t, doc, `parse-json("{}", map{})`)
			require.Equal(t, 1, seq.Len())
		})
		t.Run("abs(-3)", func(t *testing.T) {
			seq := evalExpr(t, doc, `abs(-3)`)
			require.Equal(t, 1, seq.Len())
			require.Equal(t, int64(3), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
		})
	})
}

// --- Error function ---

func TestFnError(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	compiled, err := xpath3.NewCompiler().Compile(`error()`)
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
}

// --- DateTime functions ---

func TestFnDateTime(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("current-dateTime", func(t *testing.T) {
		seq := evalExpr(t, doc, `current-dateTime()`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, xpath3.TypeDateTime, av.TypeName)
	})

	t.Run("year-from-dateTime", func(t *testing.T) {
		seq := evalExpr(t, doc, `year-from-dateTime(current-dateTime())`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, av.IntegerVal() >= 2024)
	})

	t.Run("date and time combine", func(t *testing.T) {
		// fn:dateTime($arg1 as xs:date?, $arg2 as xs:time?) combines a date
		// and a time into an xs:dateTime.
		seq := evalExpr(t, doc, `dateTime(xs:date("2020-01-01"), xs:time("01:02:03"))`)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, xpath3.TypeDateTime, av.TypeName)

		strSeq := evalExpr(t, doc, `string(dateTime(xs:date("2020-01-01"), xs:time("01:02:03")))`)
		require.Equal(t, 1, strSeq.Len())
		require.Equal(t, "2020-01-01T01:02:03", strSeq.Get(0).(xpath3.AtomicValue).Value)
	})

	t.Run("empty arg yields empty sequence", func(t *testing.T) {
		require.Nil(t, evalExpr(t, doc, `dateTime((), xs:time("01:02:03"))`))
		require.Nil(t, evalExpr(t, doc, `dateTime(xs:date("2020-01-01"), ())`))
	})

	t.Run("dateTime as first arg is a type error", func(t *testing.T) {
		// $arg1 is declared xs:date?; passing an xs:dateTime must raise
		// XPTY0004 rather than being reinterpreted as a date.
		compiled, err := xpath3.NewCompiler().Compile(`dateTime(xs:dateTime("2020-01-01T12:00:00"), xs:time("01:02:03"))`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: "XPTY0004"})
	})

	t.Run("dateTime as second arg is a type error", func(t *testing.T) {
		// $arg2 is declared xs:time?; passing an xs:dateTime must raise
		// XPTY0004 rather than being reinterpreted as a time.
		compiled, err := xpath3.NewCompiler().Compile(`dateTime(xs:date("2020-01-01"), xs:dateTime("2020-01-01T12:00:00"))`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: "XPTY0004"})
	})
}

func TestFnAvgLexicalDecimal(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`avg((3.0, 4.0, 5.0))`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 4.0, n)
}

func TestFnImplicitTimezoneReturnsDuration(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`implicit-timezone()`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	require.True(t, result.IsAtomic())

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, xpath3.TypeDayTimeDuration, atomics[0].TypeName)
}

// TestBuiltinSignatureEnforcement verifies that static built-in function calls
// enforce their declared parameter signatures, raising XPTY0004 when an argument
// has the wrong cardinality or type, while spec-valid calls still succeed.
func TestBuiltinSignatureEnforcement(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("cardinality violations raise XPTY0004", func(t *testing.T) {
		// Each of these passes a sequence of length > 1 where the signature
		// requires xs:string? / xs:numeric? (zero-or-one).
		exprs := []string{
			`abs((1, 2))`,
			`upper-case(("a", "b"))`,
			`lower-case(("a", "b"))`,
			`string-length(("a", "b"))`,
			`normalize-space(("a", "b"))`,
			`ceiling((1, 2))`,
			`floor((1, 2))`,
		}
		for _, expr := range exprs {
			t.Run(expr, func(t *testing.T) {
				compiled, err := xpath3.NewCompiler().Compile(expr)
				require.NoError(t, err)
				_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
					Evaluate(t.Context(), compiled, doc)
				require.Error(t, err)
				var xpErr *xpath3.XPathError
				require.ErrorAs(t, err, &xpErr)
				require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
			})
		}
	})

	t.Run("type violations raise XPTY0004", func(t *testing.T) {
		// matches() arg 1 is xs:string?; a QName is not coercible to xs:string.
		compiled, err := xpath3.NewCompiler().Compile(`matches(current-dateTime(), "x")`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
	})

	t.Run("valid controls still pass", func(t *testing.T) {
		tests := []struct {
			expr   string
			expect string
		}{
			// Wrap numeric results in string() so the lexical form is asserted
			// uniformly via StringVal.
			{`string(abs(-3))`, "3"},
			{`upper-case("a")`, "A"},
			{`lower-case("A")`, "a"},
			{`string(string-length("abc"))`, "3"},
			{`string(ceiling(1.2))`, "2"},
			{`string(floor(1.8))`, "1"},
			{`normalize-space("  a  b ")`, "a b"},
			// fn:upper-case(()) returns the zero-length string (not empty seq).
			{`upper-case(())`, ""},
		}
		for _, tc := range tests {
			t.Run(tc.expr, func(t *testing.T) {
				seq := evalExpr(t, doc, tc.expr)
				require.Equal(t, 1, seq.Len())
				av := seq.Get(0).(xpath3.AtomicValue)
				require.Equal(t, tc.expect, av.StringVal())
			})
		}
	})

	t.Run("empty argument satisfies optional cardinality", func(t *testing.T) {
		// abs(()) returns the empty sequence; the call must not error.
		seq := evalExpr(t, doc, `abs(())`)
		if seq != nil {
			require.Equal(t, 0, seq.Len())
		}
	})

	t.Run("maps and arrays coerce to function predicate", func(t *testing.T) {
		// fn:filter's predicate is function(item()) as xs:boolean; maps and
		// arrays are arity-1 function items and must coerce successfully.
		for _, expr := range []string{
			`filter((4, 5, 6), map{4: true(), 5: false(), 6: true()})`,
			`filter((4, 5, 6), [1, 2, 3, true(), false(), true()])`,
		} {
			t.Run(expr, func(t *testing.T) {
				seq := evalExpr(t, doc, expr)
				require.Equal(t, 2, seq.Len())
			})
		}
	})
}

// integerIdentityFn is a custom TypedFunction declaring a single xs:integer
// parameter. It is used to verify that function-lookup enforces the declared
// parameter types, mirroring the direct named-reference path (f#1).
type integerIdentityFn struct{}

func (integerIdentityFn) MinArity() int { return 1 }
func (integerIdentityFn) MaxArity() int { return 1 }

func (integerIdentityFn) Call(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return args[0], nil
}

func (integerIdentityFn) FuncParamTypes() []xpath3.SequenceType {
	return []xpath3.SequenceType{{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "xs", Name: "integer"},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}}
}

func (integerIdentityFn) FuncReturnType() *xpath3.SequenceType {
	return &xpath3.SequenceType{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "xs", Name: "integer"},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
}

// TestFunctionLookupTypedParamValidation verifies that a function item obtained
// via function-lookup for a user-defined TypedFunction enforces the declared
// parameter types — the same XPTY0004 check the direct f#1 reference applies.
func TestFunctionLookupTypedParamValidation(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(map[string]xpath3.Function{"f": integerIdentityFn{}}, nil)

	t.Run("bad-typed arg raises XPTY0004", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)("bad")`)
		require.NoError(t, err)
		_, err = eval.Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: lexicon.ErrXPTY0004})
	})

	t.Run("good-typed arg works", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)(42)`)
		require.NoError(t, err)
		result, err := eval.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		require.Equal(t, int64(42), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("named-reference parity", func(t *testing.T) {
		// The direct f#1 path must reject the same bad arg.
		compiled, err := xpath3.NewCompiler().Compile(`f#1("bad")`)
		require.NoError(t, err)
		_, err = eval.Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: lexicon.ErrXPTY0004})
	})

	t.Run("untypedAtomic arg is coerced before invocation", func(t *testing.T) {
		// xs:untypedAtomic("7") satisfies the xs:integer parameter via the
		// function-conversion rules. The coerced value (an xs:integer) must be
		// what the function body observes — integerIdentityFn returns args[0],
		// so the result type proves whether coercion was applied.
		compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)(xs:untypedAtomic("7"))`)
		require.NoError(t, err)
		result, err := eval.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, xpath3.TypeInteger, av.TypeName)
		require.Equal(t, int64(7), av.IntegerVal())
	})

	t.Run("non-coercible untypedAtomic surfaces the cast error", func(t *testing.T) {
		// Function-conversion casts an xs:untypedAtomic argument to the xs:integer
		// parameter type; the cast of "not-an-int" fails with FORG0001. That real
		// dynamic error must surface unchanged — matching the direct f(...) and
		// f#1(...) call paths — rather than being collapsed into a generic
		// XPTY0004 type-mismatch by the function-item invocation path.
		compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)(xs:untypedAtomic("not-an-int"))`)
		require.NoError(t, err)
		_, err = eval.Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: "FORG0001"})
	})
}

// customPrefixParamFn is a TypedFunction whose single parameter declares an
// AtomicOrUnionType with a NON-xs/xsd prefix ("app"). Resolving such a type via
// function-lookup's Invoke closure formerly passed a nil eval context into
// coerceToSequenceType, which dereferenced ec.namespaces and panicked. The
// function must instead surface an XPTY0004 type error with no panic.
type customPrefixParamFn struct{}

func (customPrefixParamFn) MinArity() int { return 1 }
func (customPrefixParamFn) MaxArity() int { return 1 }

func (customPrefixParamFn) Call(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return args[0], nil
}

func (customPrefixParamFn) FuncParamTypes() []xpath3.SequenceType {
	return []xpath3.SequenceType{{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "app", Name: "id"},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}}
}

func (customPrefixParamFn) FuncReturnType() *xpath3.SequenceType {
	return &xpath3.SequenceType{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "app", Name: "id"},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
}

// TestFunctionLookupCustomPrefixParamNoPanic verifies that invoking a function
// item obtained via function-lookup for a TypedFunction whose parameter type has
// a non-xs prefix does not panic on a nil eval context, but returns an
// XPathError (XPTY0004) for the unresolvable custom type.
func TestFunctionLookupCustomPrefixParamNoPanic(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(map[string]xpath3.Function{"f": customPrefixParamFn{}}, nil)

	compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)(1)`)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		_, err = eval.Evaluate(t.Context(), compiled, doc)
	})
	require.Error(t, err)
	require.ErrorIs(t, err, &xpath3.XPathError{Code: lexicon.ErrXPTY0004})
}

// TestFnRoundScaleAware verifies that extreme but representable precisions
// produce the spec-correct result rather than being blindly clamped, and that
// each case returns promptly (no Exp on an astronomically large exponent). The
// huge-magnitude cases use integers far beyond float64 range, so results are
// compared as exact decimal strings.
func TestFnRoundScaleAware(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	// 5 followed by 4999 zeros — an exact half at precision -5000.
	bigHalf := "5" + strings.Repeat("0", 4999)
	// 1 followed by 5000 zeros — the rounded result.
	bigHalfRounded := "1" + strings.Repeat("0", 5000)

	tests := []struct {
		name   string
		expr   string
		expect string // exact decimal string of the result
	}{
		// Huge negative precision AT the magnitude boundary: must round the
		// half case up, not leave it unchanged (the old fixed clamp returned
		// the operand untouched).
		{"int half boundary up", `round(` + bigHalf + `, -5000)`, bigHalfRounded},
		// Half-to-even: the quotient (0) is already even, so the half rounds
		// down to 0 — distinct from half-up which rounds the same value up.
		{"int half boundary even", `round-half-to-even(` + bigHalf + `, -5000)`, "0"},
		// Huge negative precision PAST the magnitude: rounds to 0.
		{"int past magnitude", `round(` + bigHalf + `, -6000)`, "0"},
		// Huge positive precision PAST the operand's (zero) fractional scale:
		// integer is unchanged.
		{"int fine unchanged", `round(123, 5000)`, "123"},
		// Decimal with huge positive precision past its scale: unchanged.
		{"decimal fine unchanged", `round(1.236, 5000)`, "1.236"},
		// Decimal with huge negative precision past magnitude: zero.
		{"decimal past magnitude", `round(1.236, -5000)`, "0"},
		// Normal cases still correct.
		{"normal up", `round(2.5, 0)`, "3"},
		{"normal even", `round-half-to-even(2.5, 0)`, "2"},
		{"normal neg", `round(12345, -2)`, "12300"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan xpath3.Sequence, 1)
			go func() {
				done <- evalExpr(t, doc, tc.expr)
			}()
			var seq xpath3.Sequence
			select {
			case seq = <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("round(%q) did not return promptly", tc.expr)
			}
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			s, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tc.expect, s)
		})
	}
}

// TestFnRoundExtremePrecision guards against the precision argument (an
// xs:integer that may exceed int64 range) wrapping when converted to int, and
// against an astronomically large 10^|precision| scale hanging the computation.
func TestFnRoundExtremePrecision(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect float64
	}{
		{`round(1.236, 2)`, 1.24},
		{`round(123.456, -2)`, 100},
		{`round(1.23, 9223372036854775808)`, 1.23},
		{`round-half-to-even(1.23, 9223372036854775808)`, 1.23},
		{`round(1.23, -9223372036854775809)`, 0},
		{`round-half-to-even(1.23, -9223372036854775809)`, 0},
		{`round(125, -9223372036854775809)`, 0},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			require.InDelta(t, tc.expect, av.ToFloat64(), 0.001)
		})
	}
}

// TestFnRoundPrecisionArgValidation verifies the function-conversion rules for
// the required "$precision as xs:integer" parameter of fn:round and
// fn:round-half-to-even. The precision must be validated even when $arg is the
// empty sequence (the empty first argument does not excuse an absent or
// ill-typed precision); only xs:integer (and subtypes) and xs:untypedAtomic
// (cast to integer) are accepted, everything else raises XPTY0004.
func TestFnRoundPrecisionArgValidation(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected error code %s for %q", code, expr)
	}

	evalOK := func(t *testing.T, expr, want string) {
		t.Helper()
		seq := evalExpr(t, doc, expr)
		require.Equal(t, 1, seq.Len())
		av := seq.Get(0).(xpath3.AtomicValue)
		s, err := xpath3.AtomicToString(av)
		require.NoError(t, err)
		require.Equal(t, want, s)
	}

	t.Run("empty arg does not excuse bad precision", func(t *testing.T) {
		// An ill-typed precision is a type error even when $arg is ().
		evalErrCode(t, `round((), "bad")`, "XPTY0004")
		evalErrCode(t, `round-half-to-even((), "bad")`, "XPTY0004")
		// An empty precision in the 2-arg form violates exactly-one cardinality.
		evalErrCode(t, `round((), ())`, "XPTY0004")
	})

	t.Run("empty precision is a cardinality error", func(t *testing.T) {
		evalErrCode(t, `round(1, ())`, "XPTY0004")
		evalErrCode(t, `round-half-to-even(1.5, ())`, "XPTY0004")
	})

	t.Run("non-integer precision rejected", func(t *testing.T) {
		// No implicit numeric/boolean truncation to xs:integer.
		evalErrCode(t, `round(123.45, true())`, "XPTY0004")
		evalErrCode(t, `round(123.45, 1.9)`, "XPTY0004")
		evalErrCode(t, `round(123.45, 2.0)`, "XPTY0004")
		evalErrCode(t, `round(123.45, xs:double(2))`, "XPTY0004")
		evalErrCode(t, `round(123.45, xs:float(2))`, "XPTY0004")
		evalErrCode(t, `round(123.45, "2")`, "XPTY0004")
		evalErrCode(t, `round-half-to-even(123456e-2, "two")`, "XPTY0004")
	})

	t.Run("integer precision and subtypes accepted", func(t *testing.T) {
		evalOK(t, `round(123.456, 2)`, "123.46")
		evalOK(t, `round(123.456, xs:int(2))`, "123.46")
		evalOK(t, `round(123.456, xs:long(2))`, "123.46")
		evalOK(t, `round(123.456, xs:short(2))`, "123.46")
		// xs:untypedAtomic is cast to xs:integer.
		evalOK(t, `round(123.456, xs:untypedAtomic("2"))`, "123.46")
	})

	t.Run("empty arg with valid precision is empty", func(t *testing.T) {
		// A valid precision plus an empty $arg returns () (not an error).
		evalOK(t, `empty(round((), 3))`, wantTrue)
		evalOK(t, `empty(round((), 1))`, wantTrue)
		evalOK(t, `empty(round-half-to-even((), 3))`, wantTrue)
	})
}

// TestFnRoundDecimalHugePrecision verifies that the decimal half-to-even path
// computes huge-but-representable coarse precisions exactly, rather than being
// silently clamped to zero by a coarse downstream guard. The operand magnitude
// bounds the scale, so the result must be the exact large value and return
// promptly (no hang).
func TestFnRoundDecimalHugePrecision(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	// 6 followed by 4999 zeros, written as an xs:decimal (trailing ".0") so the
	// decimal/icu path is exercised. At precision -5000 the magnitude boundary
	// makes the value round up to 1 followed by 5000 zeros.
	bigDecimal := "6" + strings.Repeat("0", 4999) + ".0"
	bigRounded := "1" + strings.Repeat("0", 5000)

	tests := []struct {
		name   string
		expr   string
		expect string
	}{
		// Half-to-even decimal path, coarse precision past the clamp threshold.
		{"halfeven decimal boundary up", `round-half-to-even(` + bigDecimal + `, -5000)`, bigRounded},
		// Half-up decimal path with the same magnitude/precision.
		{"halfup decimal boundary up", `round(` + bigDecimal + `, -5000)`, bigRounded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan xpath3.Sequence, 1)
			go func() {
				done <- evalExpr(t, doc, tc.expr)
			}()
			var seq xpath3.Sequence
			select {
			case seq = <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("round(%q) did not return promptly", tc.expr)
			}
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			s, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tc.expect, s)
		})
	}
}

// TestFnRoundNonTerminatingHugePrecision guards against a DoS: a non-terminating
// xs:decimal (e.g. 1 div 3, whose reduced denominator has a prime factor other
// than 2 or 5) rounded at an astronomically large positive precision must NOT
// build 10^precision via Exp. Such an operand has an unbounded fractional-digit
// count, so a precision beyond roundMaxComputeScale (1<<20) is refused with
// FOAR0002 rather than silently rounding at a lower scale (which would return an
// observably wrong, lower-precision value). The refusal must be prompt, never a
// hang or absurd allocation.
func TestFnRoundNonTerminatingHugePrecision(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []string{
		// 1 div 3 = 0.333... (non-terminating). A billion fractional digits must
		// not trigger 10^1000000000; it exceeds the cap, so FOAR0002.
		`round-half-to-even(1 div 3, 1000000000)`,
		`round(1 div 3, 1000000000)`,
		// 2 div 3 = 0.666... rounds up at its last retained digit; still over the
		// cap, so FOAR0002 (never a billion-digit power).
		`round-half-to-even(2 div 3, 1000000000)`,
		// A precision just under the old non-terminating sentinel (1<<30) — well
		// past the cap, so still refused.
		`round-half-to-even(1 div 3, 1073741823)`,
		// One past the cap is the boundary that must error.
		`round(2 div 3, 1048577)`,
		// Precision exactly at the ratFracDigitNonTerminating sentinel (1<<30):
		// the "precision >= fracDigits" test must NOT short-circuit to
		// roundUnchanged for a non-terminating decimal; it exceeds the cap so it
		// must raise FOAR0002, not silently return the repeating operand.
		`round(1 div 3, 1073741824)`,
		`round-half-to-even(1 div 3, 1073741824)`,
		// Precision above the sentinel must also refuse, not return unchanged.
		`round(1 div 3, 2000000000)`,
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() {
				_, ferr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
				done <- ferr
			}()
			select {
			case ferr := <-done:
				require.Error(t, ferr, "expected FOAR0002 for %q", expr)
				require.ErrorIs(t, ferr, &xpath3.XPathError{Code: "FOAR0002"}, "expected FOAR0002 for %q", expr)
			case <-time.After(10 * time.Second):
				t.Fatalf("round(%q) did not return promptly (DoS)", expr)
			}
		})
	}
}

// TestFnRoundNonTerminatingAtCap verifies that a non-terminating xs:decimal
// rounded at exactly roundMaxComputeScale (1<<20) fractional digits computes the
// exact value (rather than erroring or silently clamping). 2/3 to 1<<20 digits is
// well-defined: every retained digit is 6 and the value rounds up in the last
// place, so the result is "0." followed by (1<<20 - 1) sixes and a trailing 7.
func TestFnRoundNonTerminatingAtCap(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	const maxScale = 1 << 20
	expr := `round(2 div 3, 1048576)`
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)

	done := make(chan xpath3.Sequence, 1)
	go func() {
		result, ferr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, ferr)
		done <- result.Sequence()
	}()
	var seq xpath3.Sequence
	select {
	case seq = <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("round(%q) did not return promptly", expr)
	}
	require.Equal(t, 1, seq.Len())
	av := seq.Get(0).(xpath3.AtomicValue)
	s, err := xpath3.AtomicToString(av)
	require.NoError(t, err)
	want := "0." + strings.Repeat("6", maxScale-1) + "7"
	require.Equal(t, want, s)
}

// TestFnRoundHalfToEvenNegativePrecision verifies negative-precision half-to-even
// rounding divides the full rational (never floors to an integer first), so a
// fractional part that breaks a tie is honoured.
func TestFnRoundHalfToEvenNegativePrecision(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect string
	}{
		// 250.1 at -2: 250.1/100 = 2.501 -> rounds to 3 -> 300 (not 200; the .1
		// breaks the tie upward, so flooring to 250 first would be wrong).
		{`round-half-to-even(250.1, -2)`, "300"},
		// -249.9 at -2: -2.499 -> rounds to -2 -> -200 (not -300).
		{`round-half-to-even(-249.9, -2)`, "-200"},
		// Exact even-tie cases at -1.
		{`round-half-to-even(248, -1)`, "250"},
		{`round-half-to-even(250, -1)`, "250"},
		{`round-half-to-even(2.5, -1)`, "0"},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			s, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tc.expect, s)
		})
	}
}

// TestFnRoundPrecisionCardinality verifies that the second ($precision) argument
// of fn:round / fn:round-half-to-even is a required singleton: a multi-item
// sequence must raise XPTY0004 rather than silently using its first item.
func TestFnRoundPrecisionCardinality(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	exprs := []string{
		`round(1, (1, 2))`,
		`round-half-to-even(1, (1, 2))`,
	}
	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
			require.Error(t, err, "expected error for %q", expr)
			require.ErrorIs(t, err, &xpath3.XPathError{Code: lexicon.ErrXPTY0004})
		})
	}
}

// TestFnSequenceIntegerCardinalityArgs verifies fn:remove/fn:insert-before/
// fn:subsequence enforce integer/cardinality rules on their position args
// instead of wrapping (big.Int) or silently ignoring extra/invalid items.
func TestFnSequenceIntegerCardinalityArgs(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected %s for %q", code, expr)
	}
	evalStrings := func(t *testing.T, expr string) []string {
		t.Helper()
		seq := evalExpr(t, doc, expr)
		out := make([]string, seq.Len())
		for i := range seq.Len() {
			out[i] = seq.Get(i).(xpath3.AtomicValue).StringVal()
		}
		return out
	}

	t.Run("XPTY0004", func(t *testing.T) {
		for _, expr := range []string{
			`insert-before(("a"), "not-a-position", "b")`,
			`insert-before(("a"), 2.9, "b")`,
			`insert-before(("a"), (1, 2), "b")`,
			`subsequence(("a","b","c"), (2, 3))`,
			`subsequence(("a","b","c"), 2, (1, 2))`,
		} {
			t.Run(expr, func(t *testing.T) { evalErrCode(t, expr, "XPTY0004") })
		}
	})

	t.Run("oversized integer leaves fn:remove unchanged", func(t *testing.T) {
		require.Equal(t, []string{"a", "b"}, evalStrings(t, `remove(("a","b"), 18446744073709551617)`))
		require.Equal(t, []string{"a", "b"}, evalStrings(t, `remove(("a","b"), -18446744073709551617)`))
	})

	t.Run("valid still works", func(t *testing.T) {
		require.Equal(t, []string{"b"}, evalStrings(t, `remove(("a","b"), 1)`))
		require.Equal(t, []string{"x", "a", "b"}, evalStrings(t, `insert-before(("a","b"), 1, "x")`))
		require.Equal(t, []string{"a", "b", "x"}, evalStrings(t, `insert-before(("a","b"), 99, "x")`))
		require.Equal(t, []string{"b", "c"}, evalStrings(t, `subsequence(("a","b","c"), 2)`))
		require.Equal(t, []string{"b"}, evalStrings(t, `subsequence(("a","b","c"), 2, 1)`))
		require.Equal(t, []string{"c"}, evalStrings(t, `subsequence(("a","b","c"), 2.6)`))
	})
}

// fn:index-of's $search-param is a single atomic item (xs:anyAtomicType), so a
// non-singleton search value is a type error (XPTY0004) rather than a silent
// search for just the first item.
func TestFnIndexOfSearchCardinality(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	evalErrCode := func(t *testing.T, expr, code string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err, "expected error for %q", expr)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: code}, "expected %s for %q", code, expr)
	}

	t.Run("multi-item search raises XPTY0004", func(t *testing.T) {
		for _, expr := range []string{
			`index-of((1, 2, 1), (1, 2))`,
			`index-of(("a", "b", "a"), ("a", "b"))`,
		} {
			t.Run(expr, func(t *testing.T) { evalErrCode(t, expr, "XPTY0004") })
		}
	})

	t.Run("empty search raises XPTY0004", func(t *testing.T) {
		evalErrCode(t, `index-of((1, 2, 1), ())`, "XPTY0004")
	})

	t.Run("single-item search still works", func(t *testing.T) {
		seq := evalExpr(t, doc, `index-of((10, 20, 30, 20), 20)`)
		require.Equal(t, 2, seq.Len())
		require.Equal(t, int64(2), seq.Get(0).(xpath3.AtomicValue).IntegerVal())
		require.Equal(t, int64(4), seq.Get(1).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("empty sequence with single search returns empty", func(t *testing.T) {
		seq := evalExpr(t, doc, `index-of((), 5)`)
		require.Equal(t, 0, seq.Len())
	})
}

func TestBuiltinFunctionQueries(t *testing.T) {
	require.True(t, xpath3.IsBuiltinFunction("abs"))
	require.False(t, xpath3.IsBuiltinFunction("definitely-not-a-builtin"))

	require.True(t, xpath3.IsBuiltinFunctionNS(xpath3.NSFn, "count"))
	require.False(t, xpath3.IsBuiltinFunctionNS("urn:nope", "count"))

	require.True(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "abs", 1))
	require.False(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "abs", 5))
	require.False(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "nope", 1))
}

func TestPredeclaredNamespaces(t *testing.T) {
	ns := xpath3.PredeclaredNamespaces()
	require.Equal(t, xpath3.NSFn, ns["fn"])
	require.NotEmpty(t, ns["xs"])

	// Mutating the returned copy must not affect package state.
	ns["fn"] = "tampered"
	ns2 := xpath3.PredeclaredNamespaces()
	require.Equal(t, xpath3.NSFn, ns2["fn"])
}
