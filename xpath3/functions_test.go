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
			`math:atan2(1,(2,3))`,
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

	lib := xpath3.NewFunctionLibrary()
	lib.Set("f", integerIdentityFn{})

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(lib)

	t.Run("bad-typed arg raises XPTY0004", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`function-lookup(QName("","f"), 1)("bad")`)
		require.NoError(t, err)
		_, err = eval.Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.ErrorIs(t, err, &xpath3.XPathError{Code: "XPTY0004"})
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
		require.ErrorIs(t, err, &xpath3.XPathError{Code: "XPTY0004"})
	})
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
			require.ErrorIs(t, err, &xpath3.XPathError{Code: "XPTY0004"})
		})
	}
}
