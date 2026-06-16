package xpath3_test

import (
	"math"
	"testing"

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
