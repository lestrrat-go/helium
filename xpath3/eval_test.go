package xpath3_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func mustParseXML(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

func mustParseExpr(t *testing.T, s string) xpath3.Expr {
	t.Helper()
	expr, err := xpath3.Parse(s)
	require.NoError(t, err)
	return expr
}

func evalExpr(t *testing.T, node helium.Node, expr string) xpath3.Sequence {
	t.Helper()
	parsed := mustParseExpr(t, expr)
	result, err := xpath3.EvalForTesting(t.Context(), node, parsed)
	require.NoError(t, err)
	return result
}

func evalExprCtx(t *testing.T, ctx context.Context, node helium.Node, expr string) xpath3.Sequence {
	t.Helper()
	parsed := mustParseExpr(t, expr)
	result, err := xpath3.EvalForTesting(ctx, node, parsed)
	require.NoError(t, err)
	return result
}

// --- 3.1: Basic dispatch ---

func TestEvalLiteral(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("string literal", func(t *testing.T) {
		seq := evalExpr(t, doc, `"hello"`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, "hello", av.StringVal())
	})

	t.Run("numeric literal", func(t *testing.T) {
		seq := evalExpr(t, doc, "42")
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, int64(42), av.IntegerVal())
		require.Equal(t, xpath3.TypeInteger, av.TypeName)
	})
}

func TestEvalVariable(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	ctx := xpath3.WithVariables(t.Context(), map[string]xpath3.Sequence{
		"x": xpath3.SingleInteger(42),
	})
	seq := evalExprCtx(t, ctx, doc, "$x")
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, int64(42), av.IntegerVal())
}

func TestEvalUndefinedVariable(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	parsed := mustParseExpr(t, "$y")
	_, err := xpath3.EvalForTesting(t.Context(), doc, parsed)
	require.Error(t, err)
}

func TestEvalContextItem(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, ".")
	require.Len(t, seq, 1)
	ni := seq[0].(xpath3.NodeItem)
	require.Equal(t, doc, ni.Node)
}

func TestEvalSequenceExpr(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("empty sequence", func(t *testing.T) {
		seq := evalExpr(t, doc, "()")
		require.Len(t, seq, 0)
	})

	t.Run("comma sequence", func(t *testing.T) {
		seq := evalExpr(t, doc, `(1, 2, 3)`)
		require.Len(t, seq, 3)
	})
}

// --- 3.2: Location paths ---

func TestEvalLocationPath(t *testing.T) {
	doc := mustParseXML(t, `<root><a><b/></a><c/></root>`)
	root := doc.DocumentElement()

	t.Run("child::*", func(t *testing.T) {
		seq := evalExpr(t, root, "child::*")
		require.Len(t, seq, 2)
	})

	t.Run("abbreviated child", func(t *testing.T) {
		seq := evalExpr(t, root, "*")
		require.Len(t, seq, 2)
	})

	t.Run("absolute path /root/a", func(t *testing.T) {
		seq := evalExpr(t, doc, "/root/a")
		require.Len(t, seq, 1)
		ni := seq[0].(xpath3.NodeItem)
		require.Equal(t, "a", ni.Node.Name())
	})

	t.Run("descendant::b", func(t *testing.T) {
		seq := evalExpr(t, root, "descendant::b")
		require.Len(t, seq, 1)
	})

	t.Run("//b from doc", func(t *testing.T) {
		seq := evalExpr(t, doc, "//b")
		require.Len(t, seq, 1)
	})
}

func TestEvalPredicate(t *testing.T) {
	doc := mustParseXML(t, `<root><a v="1"/><a v="2"/><a v="3"/></root>`)
	root := doc.DocumentElement()

	t.Run("positional predicate", func(t *testing.T) {
		seq := evalExpr(t, root, "a[2]")
		require.Len(t, seq, 1)
		ni := seq[0].(xpath3.NodeItem)
		require.Equal(t, "a", ni.Node.Name())
	})

	t.Run("last position", func(t *testing.T) {
		seq := evalExpr(t, root, "a[3]")
		require.Len(t, seq, 1)
	})
}

func TestEvalAttribute(t *testing.T) {
	doc := mustParseXML(t, `<root id="42"/>`)
	root := doc.DocumentElement()

	seq := evalExpr(t, root, "@id")
	require.Len(t, seq, 1)
}

// --- 3.3: Binary operators ---

func TestEvalArithmetic(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		name   string
		expr   string
		expect float64
	}{
		{"add", "2 + 3", 5},
		{"sub", "10 - 4", 6},
		{"mul", "3 * 4", 12},
		{"div", "10 div 3", 10.0 / 3.0},
		{"mod", "10 mod 3", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Len(t, seq, 1)
			av := seq[0].(xpath3.AtomicValue)
			require.InDelta(t, tc.expect, av.ToFloat64(), 0.001)
		})
	}
}

func TestEvalIdiv(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, "10 idiv 3")
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, int64(3), av.IntegerVal())
}

func TestEvalLogic(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect bool
	}{
		{`1 = 1 and 2 = 2`, true},
		{`1 = 1 and 2 = 3`, false},
		{`1 = 2 or 2 = 2`, true},
		{`1 = 2 or 2 = 3`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Len(t, seq, 1)
			av := seq[0].(xpath3.AtomicValue)
			require.Equal(t, tc.expect, av.BooleanVal())
		})
	}
}

func TestEvalComparison(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect bool
	}{
		// General comparisons
		{`1 = 1`, true},
		{`1 != 2`, true},
		{`1 < 2`, true},
		{`2 > 1`, true},
		{`1 <= 1`, true},
		{`1 >= 1`, true},
		{`"abc" = "abc"`, true},
		{`"abc" != "def"`, true},
		// Value comparisons
		{`1 eq 1`, true},
		{`1 ne 2`, true},
		{`1 lt 2`, true},
		{`2 gt 1`, true},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Len(t, seq, 1)
			av := seq[0].(xpath3.AtomicValue)
			require.Equal(t, tc.expect, av.BooleanVal())
		})
	}
}

func TestEvalUnaryMinus(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, "-5")
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, -5.0, av.ToFloat64())
}

func TestEvalConcat(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, `"hello" || " " || "world"`)
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, "hello world", av.StringVal())
}

func TestEvalRange(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("1 to 5", func(t *testing.T) {
		seq := evalExpr(t, doc, "1 to 5")
		require.Len(t, seq, 5)
		for i := 0; i < 5; i++ {
			av := seq[i].(xpath3.AtomicValue)
			require.Equal(t, int64(i+1), av.IntegerVal())
		}
	})

	t.Run("empty range", func(t *testing.T) {
		seq := evalExpr(t, doc, "5 to 1")
		require.Len(t, seq, 0)
	})
}

func TestEvalUnion(t *testing.T) {
	doc := mustParseXML(t, `<root><a/><b/><c/></root>`)
	root := doc.DocumentElement()

	seq := evalExpr(t, root, "a | c")
	require.Len(t, seq, 2)
}

// --- 3.3: Simple map ---

func TestEvalSimpleMap(t *testing.T) {
	doc := mustParseXML(t, `<root><a>1</a><a>2</a><a>3</a></root>`)
	root := doc.DocumentElement()

	seq := evalExpr(t, root, "a ! .")
	require.Len(t, seq, 3)
}

// --- 3.4: FLWOR ---

func TestEvalFLWOR(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	ctx := xpath3.WithVariables(t.Context(), map[string]xpath3.Sequence{
		"items": {
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(1)},
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(2)},
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(3)},
		},
	})

	t.Run("simple for", func(t *testing.T) {
		seq := evalExprCtx(t, ctx, doc, "for $x in $items return $x")
		require.Len(t, seq, 3)
	})

	t.Run("let binding", func(t *testing.T) {
		seq := evalExprCtx(t, ctx, doc, `let $x := 42 return $x`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, int64(42), av.IntegerVal())
	})
}

func TestEvalQuantified(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	ctx := xpath3.WithVariables(t.Context(), map[string]xpath3.Sequence{
		"nums": {
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(1)},
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(2)},
			xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(3)},
		},
	})

	t.Run("some", func(t *testing.T) {
		seq := evalExprCtx(t, ctx, doc, "some $x in $nums satisfies $x = 2")
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("every", func(t *testing.T) {
		seq := evalExprCtx(t, ctx, doc, "every $x in $nums satisfies $x > 0")
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("every false", func(t *testing.T) {
		seq := evalExprCtx(t, ctx, doc, "every $x in $nums satisfies $x > 1")
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.False(t, av.BooleanVal())
	})
}

func TestEvalIf(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("true branch", func(t *testing.T) {
		seq := evalExpr(t, doc, `if (1 = 1) then "yes" else "no"`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, "yes", av.StringVal())
	})

	t.Run("false branch", func(t *testing.T) {
		seq := evalExpr(t, doc, `if (1 = 2) then "yes" else "no"`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, "no", av.StringVal())
	})
}

// --- 3.5: Type expressions ---

func TestEvalInstanceOf(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect bool
	}{
		{`42 instance of xs:integer`, true},
		{`42 instance of xs:decimal`, true},
		{`42 instance of xs:double`, false},
		{`"hello" instance of xs:string`, true},
		{`"hello" instance of xs:integer`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Len(t, seq, 1)
			av := seq[0].(xpath3.AtomicValue)
			require.Equal(t, tc.expect, av.BooleanVal())
		})
	}
}

func TestEvalCast(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("string to integer", func(t *testing.T) {
		seq := evalExpr(t, doc, `"42" cast as xs:integer`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, int64(42), av.IntegerVal())
	})

	t.Run("cast empty with ?", func(t *testing.T) {
		seq := evalExpr(t, doc, `() cast as xs:integer?`)
		require.Len(t, seq, 0)
	})
}

func TestEvalCastable(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("valid cast", func(t *testing.T) {
		seq := evalExpr(t, doc, `"42" castable as xs:integer`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.True(t, av.BooleanVal())
	})

	t.Run("invalid cast", func(t *testing.T) {
		seq := evalExpr(t, doc, `"abc" castable as xs:integer`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.False(t, av.BooleanVal())
	})
}

// --- 3.6: Function infrastructure ---

func TestEvalInlineFunction(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	// Inline function that returns its argument
	parsed := mustParseExpr(t, `let $f := function($x) { $x } return $f(42)`)
	result, err := xpath3.EvalForTesting(t.Context(), doc, parsed)
	require.NoError(t, err)
	require.Len(t, result, 1)
	av := result[0].(xpath3.AtomicValue)
	require.Equal(t, int64(42), av.IntegerVal())
}

func TestEvalMapConstructor(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	seq := evalExpr(t, doc, `map { "a": 1, "b": 2 }`)
	require.Len(t, seq, 1)
	m, ok := seq[0].(xpath3.MapItem)
	require.True(t, ok)
	require.Equal(t, 2, m.Size())
}

func TestEvalArrayConstructor(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("square bracket", func(t *testing.T) {
		seq := evalExpr(t, doc, `[1, 2, 3]`)
		require.Len(t, seq, 1)
		a, ok := seq[0].(xpath3.ArrayItem)
		require.True(t, ok)
		require.Equal(t, 3, a.Size())
	})
}

func TestEvalMapLookup(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	seq := evalExpr(t, doc, `map { "x": 42 }?x`)
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, int64(42), av.IntegerVal())
}

func TestEvalArrayLookup(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	seq := evalExpr(t, doc, `[10, 20, 30]?2`)
	require.Len(t, seq, 1)
	av := seq[0].(xpath3.AtomicValue)
	require.Equal(t, int64(20), av.IntegerVal())
}

// --- TryCatch ---

func TestEvalTryCatch(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("no error", func(t *testing.T) {
		seq := evalExpr(t, doc, `try { 42 } catch * { "error" }`)
		require.Len(t, seq, 1)
		av := seq[0].(xpath3.AtomicValue)
		require.Equal(t, int64(42), av.IntegerVal())
	})
}

// --- Empty sequence propagation ---

func TestEvalEmptySeqArithmetic(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, "() + 1")
	require.Len(t, seq, 0)
}

func TestEvalEmptySeqValueComparison(t *testing.T) {
	doc := mustParseXML(t, "<root/>")
	seq := evalExpr(t, doc, "() eq 1")
	require.Len(t, seq, 0)
}

// --- User functions ---

func TestEvalUserFunction(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	fn := &testFunc{
		minArity: 1,
		maxArity: 1,
		call: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return xpath3.SingleString("custom"), nil
		},
	}

	ctx := xpath3.WithFunctions(t.Context(), map[string]xpath3.Function{
		"myfunc": fn,
	})

	parsed := mustParseExpr(t, `myfunc("arg")`)
	result, err := xpath3.EvalForTesting(ctx, doc, parsed)
	require.NoError(t, err)
	require.Len(t, result, 1)
	av := result[0].(xpath3.AtomicValue)
	require.Equal(t, "custom", av.StringVal())
}

type testFunc struct {
	minArity int
	maxArity int
	call     func(context.Context, []xpath3.Sequence) (xpath3.Sequence, error)
}

func (f *testFunc) MinArity() int { return f.minArity }
func (f *testFunc) MaxArity() int { return f.maxArity }
func (f *testFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.call(ctx, args)
}
