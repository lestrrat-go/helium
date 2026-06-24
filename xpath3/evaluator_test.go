package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestEvaluator(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>hello</a><b>world</b></root>`))
	require.NoError(t, err)

	compiler := xpath3.NewCompiler()

	t.Run("basic evaluation", func(t *testing.T) {
		expr, err := compiler.Compile("//a/text()")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, testHello, string(nodes[0].Content()))
	})

	t.Run("with variables", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("test-value")}).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, "test-value", s)
	})

	t.Run("with namespaces", func(t *testing.T) {
		nsDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com"><ns:item>found</ns:item></root>`))
		require.NoError(t, err)

		expr, err := compiler.Compile("//ns:item/text()")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"ns": "http://example.com"}).
			Evaluate(t.Context(), expr, nsDoc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "found", string(nodes[0].Content()))
	})

	t.Run("evaluator immutability", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		base := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

		e1 := base.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("one")})
		e2 := base.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("two")})

		r1, err := e1.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		s1, ok := r1.IsString()
		require.True(t, ok)
		require.Equal(t, "one", s1)

		r2, err := e2.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		s2, ok := r2.IsString()
		require.True(t, ok)
		require.Equal(t, "two", s2)
	})

	t.Run("zero value evaluator", func(t *testing.T) {
		expr, err := compiler.Compile("//a/text()")
		require.NoError(t, err)

		// A zero-value Evaluator must not panic.
		var ev xpath3.Evaluator
		result, err := ev.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, testHello, string(nodes[0].Content()))
	})

	t.Run("zero value evaluator with fluent methods", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		// Fluent methods on a zero-value Evaluator must not panic.
		var ev xpath3.Evaluator
		result, err := ev.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("from-zero")}).Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, "from-zero", s)
	})

	t.Run("nil expression returns error", func(t *testing.T) {
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), nil, doc)
		require.EqualError(t, err, "xpath3: expression has no compiled program")
	})

	t.Run("MustCompile", func(t *testing.T) {
		expr := compiler.MustCompile("1 + 2")
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		n, ok := result.IsNumber()
		require.True(t, ok)
		require.InDelta(t, 3.0, n, 0.001)
	})
}

func TestDocEmptyArgFragmentBaseURI(t *testing.T) {
	// doc("") resolves to the base URI verbatim. When that base URI carries a
	// fragment identifier the call must raise FODC0005, the same as a fragment
	// in an explicit argument.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	expr, err := xpath3.NewCompiler().Compile(`doc("")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("file:///tmp/doc.xml#frag").
		Evaluate(t.Context(), expr, doc)
	require.Error(t, err)

	var xerr *xpath3.XPathError
	require.ErrorAs(t, err, &xerr)
	require.Equal(t, "FODC0005", xerr.Code)
}

func TestEvaluatorBuilders(t *testing.T) {
	doc := mustParseXML(t, "<root><a/><b/></root>")
	root := doc.DocumentElement()

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Position(2).
		Size(5).
		PreservedIDAnnotations(map[helium.Node]string{}).
		AllowXML11Chars()

	compiled, err := xpath3.NewCompiler().Compile(`position()`)
	require.NoError(t, err)
	res, err := eval.Evaluate(t.Context(), compiled, root)
	require.NoError(t, err)
	n, ok := res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)

	compiledLast, err := xpath3.NewCompiler().Compile(`last()`)
	require.NoError(t, err)
	res, err = eval.Evaluate(t.Context(), compiledLast, root)
	require.NoError(t, err)
	n, ok = res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(5), n)
}

func TestVariableAndFunctionResolver(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		VariableResolver(varResolver{}).
		FunctionResolver(funcResolver{})

	compiled, err := xpath3.NewCompiler().Compile(`$dynamic`)
	require.NoError(t, err)
	res, err := eval.Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	n, ok := res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(99), n)
}

type varResolver struct{}

func (varResolver) ResolveVariable(_ context.Context, name string) (xpath3.Sequence, bool, error) {
	if name == "dynamic" {
		return atomicSeq(intAtomic(99)), true, nil
	}
	return nil, false, nil
}

type funcResolver struct{}

func (funcResolver) ResolveFunction(_ context.Context, _, _ string, _ int) (xpath3.Function, bool, error) {
	return nil, false, nil
}

func TestFnContextNode(t *testing.T) {
	doc := mustParseXML(t, "<root><child/></root>")
	root := doc.DocumentElement()

	captured := &capturingFn{}
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(map[string]xpath3.Function{"capture": captured}, nil)
	compiled, err := xpath3.NewCompiler().Compile(`capture()`)
	require.NoError(t, err)
	_, err = eval.Evaluate(t.Context(), compiled, root)
	require.NoError(t, err)
	require.NotNil(t, captured.node)
	require.Equal(t, root, captured.node)
	// Direct (non-dynamic) call: IsDynamicCall is false.
	require.False(t, captured.dynamic)
}

type capturingFn struct {
	node    helium.Node
	dynamic bool
}

func (*capturingFn) MinArity() int { return 0 }
func (*capturingFn) MaxArity() int { return 0 }
func (c *capturingFn) Call(ctx context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	c.node = xpath3.FnContextNode(ctx)
	c.dynamic = xpath3.IsDynamicCall(ctx)
	return atomicSeq(intAtomic(1)), nil
}
