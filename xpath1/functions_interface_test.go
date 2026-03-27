package xpath1_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func TestFunctionFuncImplementsFunction(t *testing.T) {
	var called bool
	f := xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		called = true
		return &xpath1.Result{Type: xpath1.StringResult, String: "ok"}, nil
	})

	var fn xpath1.Function = f
	r, err := fn.Eval(t.Context(), nil)
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, xpath1.StringResult, r.Type)
	require.Equal(t, "ok", r.String)
}

func TestFunctionPosition(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "count(a[position()=2])")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 1.0, result.Number)
}

func TestFunctionLast(t *testing.T) {
	doc := parseXML(t, `<root><a>1</a><a>2</a><a>3</a></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "a[last()]")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, result.Type)
	require.Len(t, result.NodeSet, 1)
	require.Equal(t, "3", string(result.NodeSet[0].Content()))
}

func TestFunctionNameContext(t *testing.T) {
	doc := parseXML(t, `<root><child/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "name()")
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "root", result.String)
}

func TestFunctionStringConversion(t *testing.T) {
	doc := parseXML(t, `<root>hello world</root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "string(.)")
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "hello world", result.String)
}

func TestFunctionNumberConversion(t *testing.T) {
	doc := parseXML(t, `<root>42</root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "number(.)")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 42.0, result.Number)
}

func TestFunctionBooleanConversion(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "boolean(a)")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.True(t, result.Bool)

	result, err = xpath1.Evaluate(t.Context(), root, "boolean(nonexistent)")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.False(t, result.Bool)
}

func TestFunctionConcatWithContext(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, `concat(string(.), " world")`)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "hello world", result.String)
}

func TestFunctionSumNodeSet(t *testing.T) {
	doc := parseXML(t, `<root><v>10</v><v>20</v><v>30</v></root>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "sum(v)")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, result.Type)
	require.Equal(t, 60.0, result.Number)
}

func TestFunctionTrueAndFalse(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, "true() and false()")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, result.Type)
	require.False(t, result.Bool)
}

func TestFunctionNormalizeSpaceCall(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, `normalize-space("  hello   world  ")`)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "hello world", result.String)
}

func TestFunctionTranslateCall(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	result, err := xpath1.Evaluate(t.Context(), root, `translate("abc", "abc", "ABC")`)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, result.Type)
	require.Equal(t, "ABC", result.String)
}

func TestGetFunctionContext(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	root := docElement(doc)

	var captured xpath1.FunctionContext
	capture := xpath1.FunctionFunc(func(ctx context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		captured = xpath1.GetFunctionContext(ctx)
		return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
	})

	ev := xpath1.NewEvaluator().Function("capture", capture)
	expr, err := xpath1.Compile("capture()")
	require.NoError(t, err)

	_, err = ev.Evaluate(t.Context(), expr, root)
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, "root", captured.Node().Name())
}
