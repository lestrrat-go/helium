package xpath1_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func TestParseSimplePath(t *testing.T) {
	doc := parseXML(t, `<a><b>hello</b></a>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "/a/b")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "b", result.NodeSet[0].Name())
}

func TestParseRelativePath(t *testing.T) {
	doc := parseXML(t, `<root><a><b>text</b></a></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a/b")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "b", result.NodeSet[0].Name())
}

func TestParseDoubleSlash(t *testing.T) {
	doc := parseXML(t, `<root><a><b><a>deep</a></b></a></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "//a")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 2)
}

func TestParseAxis(t *testing.T) {
	doc := parseXML(t, `<root><para>one</para><div/><para>two</para></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "descendant::para")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 2)
	require.Equal(t, "para", result.NodeSet[0].Name())
}

func TestParseAttribute(t *testing.T) {
	doc := parseXML(t, `<root id="42"/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "@id")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "42", string(result.NodeSet[0].Content()))
}

func TestParsePredicate(t *testing.T) {
	doc := parseXML(t, `<root><item>a</item><item>b</item><item>c</item></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "item[3]")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "c", string(result.NodeSet[0].Content()))
}

func TestParseDot(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, ".")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "root", result.NodeSet[0].Name())
}

func TestParseDotDot(t *testing.T) {
	doc := parseXML(t, `<root><child/></root>`)
	root := docElement(doc)

	nodes, err := xpath1.Find(t.Context(), root, "child")
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	result, err := xpath1.Evaluate(t.Context(), nodes[0], "..")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "root", result.NodeSet[0].Name())
}

func TestParseWildcard(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "*")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 3)
}

func TestParseNodeTest(t *testing.T) {
	doc := parseXML(t, `<root>text<child/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "node()")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.GreaterOrEqual(t, len(result.NodeSet), 2)
}

func TestParseTextTest(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "text()")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "hello", string(result.NodeSet[0].Content()))
}

func TestParseFunctionCall(t *testing.T) {
	doc := parseXML(t, `<root><item/><item/><item/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "count(//item)")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 3.0, result.Number)
}

func TestParseComparison(t *testing.T) {
	doc := parseXML(t, `<root><a>hello</a></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a = 'hello'")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.True(t, result.Bool)
}

func TestParseArithmetic(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "1 + 2")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 3.0, result.Number)
}

func TestParseUnion(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a | b")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 2)
}

func TestParseComplexExpr(t *testing.T) {
	doc := parseXML(t, `<bookstore>
		<book><title>A</title><price>30</price></book>
		<book><title>B</title><price>40</price></book>
		<book><title>C</title><price>50</price></book>
	</bookstore>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "/bookstore/book[price>35.00]/title")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 2)
	require.Equal(t, "title", result.NodeSet[0].Name())
}

func TestParseRootOnly(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	result, err := xpath1.Evaluate(t.Context(), doc, "/")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
}

func TestParseVariableRef(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	ev := xpath1.NewEvaluator().Variables(map[string]any{"x": 5.0})
	expr, err := xpath1.Compile("$x + 1")
	require.NoError(t, err)

	result, err := ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 6.0, result.Number)
}

func TestParseStringLiteral(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, `"hello"`)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "hello", result.String)
}

func TestParseOr(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a or b")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.True(t, result.Bool)
}

func TestParseAnd(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a and b")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.True(t, result.Bool)
}

func TestParseNegation(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "-5")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, -5.0, result.Number)
}

func TestParseParenthesized(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "(1 + 2)")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 3.0, result.Number)
}

func TestParseFunctionMultipleArgs(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "substring('hello', 2, 3)")
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "ell", result.String)
}

func TestParseQNameStep(t *testing.T) {
	doc := parseXML(t, `<root xmlns:ns="urn:test"><ns:elem>found</ns:elem></root>`)
	root := docElement(doc)

	ev := xpath1.NewEvaluator().Namespaces(map[string]string{"ns": "urn:test"})
	expr, err := xpath1.Compile("ns:elem")
	require.NoError(t, err)

	result, err := ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
}

func TestParseQNameFunctionCall(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	fn := xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.StringResult, String: "called"}, nil
	})

	ev := xpath1.NewEvaluator().
		Namespaces(map[string]string{"ext": "urn:ext"}).
		FunctionNS("urn:ext", "hello", fn)
	expr, err := xpath1.Compile("ext:hello('x')")
	require.NoError(t, err)

	result, err := ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "called", result.String)
}

func TestParseQNameFunctionCallNoArgs(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	fn := xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.NumberResult, Number: float64(len(args))}, nil
	})

	ev := xpath1.NewEvaluator().
		Namespaces(map[string]string{"ext": "urn:ext"}).
		FunctionNS("urn:ext", "now", fn)
	expr, err := xpath1.Compile("ext:now()")
	require.NoError(t, err)

	result, err := ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 0.0, result.Number)
}

func TestParseQNameStepNotFunction(t *testing.T) {
	doc := parseXML(t, `<root xmlns:ns="urn:test"><ns:elem>data</ns:elem></root>`)
	root := docElement(doc)

	ev := xpath1.NewEvaluator().Namespaces(map[string]string{"ns": "urn:test"})
	expr, err := xpath1.Compile("ns:elem")
	require.NoError(t, err)

	result, err := ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
}

func TestParseCompileError(t *testing.T) {
	_, err := xpath1.Compile("[[[")
	require.Error(t, err)
}
