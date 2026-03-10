package xpath3_test

import (
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestParseLocationPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"root", "/"},
		{"absolute simple", "/a"},
		{"absolute multi", "/a/b/c"},
		{"relative", "a/b"},
		{"descendant-or-self", "//a"},
		{"absolute descendant", "/a//b"},
		{"self", "."},
		{"parent", ".."},
		{"attribute", "@id"},
		{"attribute axis", "attribute::id"},
		{"child axis", "child::a"},
		{"wildcard", "*"},
		{"prefix wildcard", "ns:*"},
		{"qname step", "ns:elem"},
		{"predicate", "a[1]"},
		{"multiple predicates", "a[1][2]"},
		{"node test", "node()"},
		{"text test", "text()"},
		{"comment test", "comment()"},
		{"pi test", "processing-instruction()"},
		{"pi with target", "processing-instruction('xml-stylesheet')"},
		{"complex path", "//div[@class='main']/p[position() > 1]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := xpath3.Parse(tc.input)
			require.NoError(t, err, "input: %s", tc.input)
			require.NotNil(t, expr)
		})
	}
}

func TestParseExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"number literal", "42"},
		{"string literal", `"hello"`},
		{"variable", "$x"},
		{"addition", "1 + 2"},
		{"subtraction", "3 - 1"},
		{"multiplication", "2 * 3"},
		{"division", "10 div 2"},
		{"modulo", "10 mod 3"},
		{"integer division", "10 idiv 3"},
		{"negation", "-1"},
		{"double negation", "--1"},
		{"and", "true() and false()"},
		{"or", "true() or false()"},
		{"equality", "1 = 1"},
		{"inequality", "1 != 2"},
		{"less than", "1 < 2"},
		{"greater equal", "1 >= 1"},
		{"value eq", "$x eq $y"},
		{"value ne", "$x ne $y"},
		{"value lt", "$x lt $y"},
		{"value le", "$x le $y"},
		{"value gt", "$x gt $y"},
		{"value ge", "$x ge $y"},
		{"parenthesized", "(1 + 2)"},
		{"function call", "count(//a)"},
		{"qname function", "fn:count(//a)"},
		{"function multi args", "substring('abc', 2, 1)"},
		{"nested functions", "concat(name(), ':', local-name())"},
		{"union pipe", "//a | //b"},
		{"union keyword", "//a union //b"},
		{"filter", "(//a)[1]"},
		{"path after filter", "(//a)[1]/b"},
		{"complex arithmetic", "($x + $y) * ($z - 1)"},
		{"comparison with path", "//a = //b"},
		{"empty sequence", "()"},
		{"sequence", "(1, 2, 3)"},
		{"context item", "."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := xpath3.Parse(tc.input)
			require.NoError(t, err, "input: %s", tc.input)
			require.NotNil(t, expr)
		})
	}
}

func TestParseXPath3Extensions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Concat
		{"concat", `"a" || "b"`},
		{"concat chain", `"a" || "b" || "c"`},

		// Range
		{"range", "1 to 10"},

		// Simple map
		{"simple map", "//item ! name()"},

		// Arrow
		{"arrow", `$x => upper-case()`},
		{"arrow chain", `$x => upper-case() => substring(1, 3)`},

		// Intersect/except
		{"intersect", "$a intersect $b"},
		{"except", "$a except $b"},

		// For/let
		{"for", "for $x in //a return $x"},
		{"let", "let $x := 1 return $x + 1"},
		{"for multi binding", "for $x in //a, $y in //b return concat($x, $y)"},
		{"let multi binding", "let $x := 1, $y := 2 return $x + $y"},
		{"for let combined", "for $x in //a return let $y := name($x) return $y"},

		// Quantified
		{"some", "some $x in //a satisfies $x > 0"},
		{"every", "every $x in //a satisfies $x > 0"},

		// If/then/else
		{"if", "if (true()) then 1 else 2"},
		{"nested if", "if ($x > 0) then if ($x > 10) then 'big' else 'small' else 'zero'"},

		// Try/catch
		{"try catch wildcard", "try { 1 div 0 } catch * { 0 }"},

		// Instance of
		{"instance of", "$x instance of xs:integer"},
		{"instance of optional", "$x instance of xs:integer?"},
		{"instance of star", "$x instance of xs:integer*"},
		{"instance of plus", "$x instance of xs:integer+"},
		{"instance of node", "$x instance of node()"},
		{"instance of element", "$x instance of element()"},
		{"instance of item", "$x instance of item()"},

		// Cast/castable/treat
		{"cast as", "$x cast as xs:double"},
		{"cast as optional", "$x cast as xs:double?"},
		{"castable as", "$x castable as xs:integer"},
		{"treat as", "$x treat as xs:string"},

		// Named function ref
		{"named func ref", "fn:upper-case#1"},
		{"named func ref zero arity", "fn:true#0"},

		// Inline function
		{"inline function", "function($x) { $x + 1 }"},
		{"inline function typed", "function($x as xs:integer) as xs:integer { $x + 1 }"},
		{"inline function no params", "function() { 42 }"},

		// Partial application
		{"partial application", "substring(?, 2)"},

		// Map constructor
		{"empty map", "map { }"},
		{"map one entry", `map { "key": "value" }`},
		{"map multi entry", `map { "a": 1, "b": 2 }`},

		// Array constructors
		{"empty square array", "[]"},
		{"square array", "[1, 2, 3]"},
		{"curly array", "array { 1, 2, 3 }"},
		{"empty curly array", "array { }"},

		// Lookup
		{"lookup name", "$map?key"},
		{"lookup number", "$array?1"},
		{"lookup star", "$map?*"},
		{"lookup paren", "$map?(1 + 2)"},
		{"unary lookup", "?key"},

		// XPath 3.1 kind tests
		{"element test", "self::element()"},
		{"element test named", "self::element(foo)"},
		{"attribute test", "self::attribute()"},
		{"document node test", "self::document-node()"},
		{"namespace node test", "self::namespace-node()"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := xpath3.Parse(tc.input)
			require.NoError(t, err, "input: %s", tc.input)
			require.NotNil(t, expr)
		})
	}
}

func TestParseAST(t *testing.T) {
	t.Run("location path structure", func(t *testing.T) {
		expr, err := xpath3.Parse("/a/b")
		require.NoError(t, err)
		lp, ok := expr.(*xpath3.LocationPath)
		require.True(t, ok, "expected LocationPath, got %T", expr)
		require.True(t, lp.Absolute)
		require.Len(t, lp.Steps, 2)
		require.Equal(t, xpath3.AxisChild, lp.Steps[0].Axis)
		nt0, ok := lp.Steps[0].NodeTest.(xpath3.NameTest)
		require.True(t, ok)
		require.Equal(t, "a", nt0.Local)
	})

	t.Run("binary expr structure", func(t *testing.T) {
		expr, err := xpath3.Parse("1 + 2")
		require.NoError(t, err)
		be, ok := expr.(xpath3.BinaryExpr)
		require.True(t, ok, "expected BinaryExpr, got %T", expr)
		require.Equal(t, xpath3.TokenPlus, be.Op)
	})

	t.Run("concat expr structure", func(t *testing.T) {
		expr, err := xpath3.Parse(`"a" || "b"`)
		require.NoError(t, err)
		ce, ok := expr.(xpath3.ConcatExpr)
		require.True(t, ok, "expected ConcatExpr, got %T", expr)
		left, ok := ce.Left.(xpath3.LiteralExpr)
		require.True(t, ok)
		require.Equal(t, "a", left.Value)
	})

	t.Run("range expr structure", func(t *testing.T) {
		expr, err := xpath3.Parse("1 to 10")
		require.NoError(t, err)
		re, ok := expr.(xpath3.RangeExpr)
		require.True(t, ok, "expected RangeExpr, got %T", expr)
		start, ok := re.Start.(xpath3.LiteralExpr)
		require.True(t, ok)
		require.Equal(t, big.NewInt(1), start.Value)
	})

	t.Run("if expr structure", func(t *testing.T) {
		expr, err := xpath3.Parse("if (true()) then 1 else 2")
		require.NoError(t, err)
		ie, ok := expr.(xpath3.IfExpr)
		require.True(t, ok, "expected IfExpr, got %T", expr)
		require.NotNil(t, ie.Cond)
		require.NotNil(t, ie.Then)
		require.NotNil(t, ie.Else)
	})

	t.Run("for expr structure", func(t *testing.T) {
		expr, err := xpath3.Parse("for $x in //a return $x")
		require.NoError(t, err)
		fe, ok := expr.(xpath3.FLWORExpr)
		require.True(t, ok, "expected FLWORExpr, got %T", expr)
		require.Len(t, fe.Clauses, 1)
		fc, ok := fe.Clauses[0].(xpath3.ForClause)
		require.True(t, ok)
		require.Equal(t, "x", fc.Var)
	})

	t.Run("named function ref structure", func(t *testing.T) {
		expr, err := xpath3.Parse("fn:upper-case#1")
		require.NoError(t, err)
		nfr, ok := expr.(xpath3.NamedFunctionRef)
		require.True(t, ok, "expected NamedFunctionRef, got %T", expr)
		require.Equal(t, "fn", nfr.Prefix)
		require.Equal(t, "upper-case", nfr.Name)
		require.Equal(t, 1, nfr.Arity)
	})

	t.Run("map constructor structure", func(t *testing.T) {
		expr, err := xpath3.Parse(`map { "a": 1, "b": 2 }`)
		require.NoError(t, err)
		mc, ok := expr.(xpath3.MapConstructorExpr)
		require.True(t, ok, "expected MapConstructorExpr, got %T", expr)
		require.Len(t, mc.Pairs, 2)
	})

	t.Run("array square constructor", func(t *testing.T) {
		expr, err := xpath3.Parse("[1, 2, 3]")
		require.NoError(t, err)
		ac, ok := expr.(xpath3.ArrayConstructorExpr)
		require.True(t, ok, "expected ArrayConstructorExpr, got %T", expr)
		require.True(t, ac.SquareBracket)
		require.Len(t, ac.Items, 3)
	})

	t.Run("instance of structure", func(t *testing.T) {
		expr, err := xpath3.Parse("$x instance of xs:integer?")
		require.NoError(t, err)
		io, ok := expr.(xpath3.InstanceOfExpr)
		require.True(t, ok, "expected InstanceOfExpr, got %T", expr)
		require.Equal(t, xpath3.OccurrenceZeroOrOne, io.Type.Occurrence)
	})

	t.Run("cast as structure", func(t *testing.T) {
		expr, err := xpath3.Parse("$x cast as xs:double?")
		require.NoError(t, err)
		ce, ok := expr.(xpath3.CastExpr)
		require.True(t, ok, "expected CastExpr, got %T", expr)
		require.Equal(t, "xs", ce.Type.Prefix)
		require.Equal(t, "double", ce.Type.Name)
		require.True(t, ce.AllowEmpty)
	})

	t.Run("arrow desugars to function call", func(t *testing.T) {
		expr, err := xpath3.Parse("$x => upper-case()")
		require.NoError(t, err)
		fc, ok := expr.(xpath3.FunctionCall)
		require.True(t, ok, "expected FunctionCall, got %T", expr)
		require.Equal(t, "upper-case", fc.Name)
		require.Len(t, fc.Args, 1) // $x is prepended
	})

	t.Run("simple map structure", func(t *testing.T) {
		expr, err := xpath3.Parse("//item ! name()")
		require.NoError(t, err)
		sm, ok := expr.(xpath3.SimpleMapExpr)
		require.True(t, ok, "expected SimpleMapExpr, got %T", expr)
		require.NotNil(t, sm.Left)
		require.NotNil(t, sm.Right)
	})
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"unclosed paren", "(1 + 2"},
		{"unclosed bracket", "a[1"},
		{"trailing garbage", "1 + 2 )"},
		{"invalid axis", "bogus::a"},
		{"missing predicate close", "a["},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpath3.Parse(tc.input)
			require.Error(t, err, "input: %s", tc.input)
		})
	}
}
