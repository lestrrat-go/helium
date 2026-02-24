package xpath

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimplePath(t *testing.T) {
	expr, err := Parse("/a/b")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.True(t, lp.Absolute)
	require.Len(t, lp.Steps, 2)
	require.Equal(t, AxisChild, lp.Steps[0].Axis)
	require.Equal(t, "a", lp.Steps[0].NodeTest.(NameTest).Local)
	require.Equal(t, "b", lp.Steps[1].NodeTest.(NameTest).Local)
}

func TestParseRelativePath(t *testing.T) {
	expr, err := Parse("a/b")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.False(t, lp.Absolute)
	require.Len(t, lp.Steps, 2)
}

func TestParseDoubleSlash(t *testing.T) {
	expr, err := Parse("//a")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.True(t, lp.Absolute)
	// //a expands to /descendant-or-self::node()/child::a
	require.Len(t, lp.Steps, 2)
	require.Equal(t, AxisDescendantOrSelf, lp.Steps[0].Axis)
	require.Equal(t, AxisChild, lp.Steps[1].Axis)
	require.Equal(t, "a", lp.Steps[1].NodeTest.(NameTest).Local)
}

func TestParseAxis(t *testing.T) {
	expr, err := Parse("descendant::para")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Len(t, lp.Steps, 1)
	require.Equal(t, AxisDescendant, lp.Steps[0].Axis)
	require.Equal(t, "para", lp.Steps[0].NodeTest.(NameTest).Local)
}

func TestParseAttribute(t *testing.T) {
	expr, err := Parse("@id")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Len(t, lp.Steps, 1)
	require.Equal(t, AxisAttribute, lp.Steps[0].Axis)
	require.Equal(t, "id", lp.Steps[0].NodeTest.(NameTest).Local)
}

func TestParsePredicate(t *testing.T) {
	expr, err := Parse("item[3]")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Len(t, lp.Steps, 1)
	require.Len(t, lp.Steps[0].Predicates, 1)
	numExpr, ok := lp.Steps[0].Predicates[0].(NumberExpr)
	require.True(t, ok)
	require.Equal(t, 3.0, numExpr.Value)
}

func TestParseDot(t *testing.T) {
	expr, err := Parse(".")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Len(t, lp.Steps, 1)
	require.Equal(t, AxisSelf, lp.Steps[0].Axis)
}

func TestParseDotDot(t *testing.T) {
	expr, err := Parse("..")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Len(t, lp.Steps, 1)
	require.Equal(t, AxisParent, lp.Steps[0].Axis)
}

func TestParseWildcard(t *testing.T) {
	expr, err := Parse("*")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.Equal(t, "*", lp.Steps[0].NodeTest.(NameTest).Local)
}

func TestParseNodeTest(t *testing.T) {
	expr, err := Parse("node()")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	_, ok = lp.Steps[0].NodeTest.(TypeTest)
	require.True(t, ok)
}

func TestParseTextTest(t *testing.T) {
	expr, err := Parse("text()")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	tt, ok := lp.Steps[0].NodeTest.(TypeTest)
	require.True(t, ok)
	require.Equal(t, NodeTestText, tt.Type)
}

func TestParseFunctionCall(t *testing.T) {
	expr, err := Parse("count(//item)")
	require.NoError(t, err)
	fc, ok := expr.(FunctionCall)
	require.True(t, ok)
	require.Equal(t, "count", fc.Name)
	require.Len(t, fc.Args, 1)
}

func TestParseComparison(t *testing.T) {
	expr, err := Parse("a = 'hello'")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	require.Equal(t, TokenEquals, be.Op)
}

func TestParseArithmetic(t *testing.T) {
	expr, err := Parse("1 + 2")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	require.Equal(t, TokenPlus, be.Op)
}

func TestParseUnion(t *testing.T) {
	expr, err := Parse("a | b")
	require.NoError(t, err)
	_, ok := expr.(UnionExpr)
	require.True(t, ok)
}

func TestParseComplexExpr(t *testing.T) {
	expr, err := Parse("/bookstore/book[price>35.00]/title")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.True(t, lp.Absolute)
	require.Len(t, lp.Steps, 3)
	require.Equal(t, "bookstore", lp.Steps[0].NodeTest.(NameTest).Local)
	require.Equal(t, "book", lp.Steps[1].NodeTest.(NameTest).Local)
	require.Len(t, lp.Steps[1].Predicates, 1)
	require.Equal(t, "title", lp.Steps[2].NodeTest.(NameTest).Local)
}

func TestParseRootOnly(t *testing.T) {
	expr, err := Parse("/")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	require.True(t, lp.Absolute)
	require.Len(t, lp.Steps, 0)
}

func TestParseVariableRef(t *testing.T) {
	expr, err := Parse("$x + 1")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	_, ok = be.Left.(VariableExpr)
	require.True(t, ok)
}

func TestParseStringLiteral(t *testing.T) {
	expr, err := Parse(`"hello"`)
	require.NoError(t, err)
	lit, ok := expr.(LiteralExpr)
	require.True(t, ok)
	require.Equal(t, "hello", lit.Value)
}

func TestParseOr(t *testing.T) {
	expr, err := Parse("a or b")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	require.Equal(t, TokenOr, be.Op)
}

func TestParseAnd(t *testing.T) {
	expr, err := Parse("a and b")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	require.Equal(t, TokenAnd, be.Op)
}

func TestParseNegation(t *testing.T) {
	expr, err := Parse("-5")
	require.NoError(t, err)
	_, ok := expr.(UnaryExpr)
	require.True(t, ok)
}

func TestParseParenthesized(t *testing.T) {
	expr, err := Parse("(1 + 2)")
	require.NoError(t, err)
	be, ok := expr.(BinaryExpr)
	require.True(t, ok)
	require.Equal(t, TokenPlus, be.Op)
}

func TestParseFunctionMultipleArgs(t *testing.T) {
	expr, err := Parse("substring('hello', 2, 3)")
	require.NoError(t, err)
	fc, ok := expr.(FunctionCall)
	require.True(t, ok)
	require.Equal(t, "substring", fc.Name)
	require.Len(t, fc.Args, 3)
}

func TestParseQNameStep(t *testing.T) {
	expr, err := Parse("ns:elem")
	require.NoError(t, err)
	lp, ok := expr.(*LocationPath)
	require.True(t, ok)
	nt := lp.Steps[0].NodeTest.(NameTest)
	require.Equal(t, "ns", nt.Prefix)
	require.Equal(t, "elem", nt.Local)
}
