package xpath3_test

import (
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
		require.Equal(t, "hello", string(nodes[0].Content()))
	})

	t.Run("with variables", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		vars := xpath3.NewVariables()
		vars.Set("x", xpath3.SingleString("test-value"))

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(vars).
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

		vars1 := xpath3.NewVariables()
		vars1.Set("x", xpath3.SingleString("one"))

		vars2 := xpath3.NewVariables()
		vars2.Set("x", xpath3.SingleString("two"))

		e1 := base.Variables(vars1)
		e2 := base.Variables(vars2)

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
