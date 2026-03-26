package xpath1

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func parseDoc(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

func docRoot(t *testing.T, doc *helium.Document) helium.Node {
	t.Helper()
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if n.Type() == helium.ElementNode {
			return n
		}
	}
	t.Fatal("document has no root element")
	return nil
}

func TestFunctionFuncImplementsFunction(t *testing.T) {
	var called bool
	f := FunctionFunc(func(_ context.Context, _ []*Result) (*Result, error) {
		called = true
		return &Result{Type: StringResult, String: "ok"}, nil
	})

	var fn Function = f
	r, err := fn.Eval(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, StringResult, r.Type)
	require.Equal(t, "ok", r.String)
}

func TestFunctionContextAccessors(t *testing.T) {
	doc := parseDoc(t, `<root><item/></root>`)
	root := docRoot(t, doc)
	ctx := newEvalContextWithConfig(t.Context(), root, nil)
	ctx.position = 2
	ctx.size = 3
	ctx.namespaces = map[string]string{"ext": "urn:test"}
	ctx.variables = map[string]interface{}{"v": "hello"}

	var fctx FunctionContext = ctx
	require.Equal(t, root, fctx.Node())
	require.Equal(t, 2, fctx.Position())
	require.Equal(t, 3, fctx.Size())

	uri, ok := fctx.Namespace("ext")
	require.True(t, ok)
	require.Equal(t, "urn:test", uri)

	v, ok := fctx.Variable("v")
	require.True(t, ok)
	require.Equal(t, "hello", v)
}

func TestFunctionContextZeroValue(t *testing.T) {
	var fctx *evalContext

	require.Nil(t, fctx.Node())
	require.Equal(t, 0, fctx.Position())
	require.Equal(t, 0, fctx.Size())

	_, ok := fctx.Namespace("ext")
	require.False(t, ok)
	_, ok = fctx.Variable("v")
	require.False(t, ok)
}

func TestFunctionReceivesPreEvaluatedArgs(t *testing.T) {
	doc := parseDoc(t, `<root>hello</root>`)
	root := docRoot(t, doc)

	// Evaluate string(.) which should pre-evaluate the context node to "hello"
	// and pass it through concat which receives []*Result
	result, err := Evaluate(context.Background(), root, `concat(string(.), " world")`)
	require.NoError(t, err)
	require.Equal(t, "hello world", result.String)
}
