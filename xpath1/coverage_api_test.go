package xpath1_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// TestFunctionContextAccessors exercises the FunctionContext accessor methods
// (Node, Position, Size, Namespace, Variable) from within a custom function.
func TestFunctionContextAccessors(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	compiled, err := xpath1.Compile("/root/a[probe()]")
	require.NoError(t, err)

	var sawNode bool
	var sawSize int
	var nsURI string
	var nsOK bool
	var varVal any
	var varOK bool

	ev := xpath1.NewEvaluator().
		Namespaces(map[string]string{"p": "urn:p"}).
		Variables(map[string]any{"v": float64(7)}).
		Function("probe", xpath1.FunctionFunc(func(ctx context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			fctx := xpath1.GetFunctionContext(ctx)
			if fctx.Node() != nil {
				sawNode = true
			}
			sawSize = fctx.Size()
			nsURI, nsOK = fctx.Namespace("p")
			varVal, varOK = fctx.Variable("v")
			return &xpath1.Result{Type: xpath1.BooleanResult, Bool: fctx.Position() == 1}, nil
		}))

	r, err := ev.Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.True(t, sawNode)
	require.Equal(t, 3, sawSize)
	require.True(t, nsOK)
	require.Equal(t, "urn:p", nsURI)
	require.True(t, varOK)
	require.Equal(t, float64(7), varVal)
}

func TestAdditionalNamespacesAndVariables(t *testing.T) {
	doc := parseXML(t, `<root xmlns:p="urn:x"><p:foo>v</p:foo></root>`)

	t.Run("AdditionalNamespaces from empty", func(t *testing.T) {
		nodes, err := xpath1.NewEvaluator().
			AdditionalNamespaces(map[string]string{"p": nsURIX}).
			Find(t.Context(), xpath1.MustCompile("//p:foo"), doc)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("AdditionalNamespaces merges", func(t *testing.T) {
		nodes, err := xpath1.NewEvaluator().
			Namespaces(map[string]string{"q": "urn:q"}).
			AdditionalNamespaces(map[string]string{"p": nsURIX}).
			Find(t.Context(), xpath1.MustCompile("//p:foo"), doc)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("AdditionalVariables from empty", func(t *testing.T) {
		r, err := xpath1.NewEvaluator().
			AdditionalVariables(map[string]any{"x": float64(2)}).
			Evaluate(t.Context(), xpath1.MustCompile("$x + 1"), doc)
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number)
	})

	t.Run("AdditionalVariables merges", func(t *testing.T) {
		r, err := xpath1.NewEvaluator().
			Variables(map[string]any{"y": float64(10)}).
			AdditionalVariables(map[string]any{"x": float64(2)}).
			Evaluate(t.Context(), xpath1.MustCompile("$x + $y"), doc)
		require.NoError(t, err)
		require.Equal(t, 12.0, r.Number)
	})
}

func TestExpressionString(t *testing.T) {
	compiled := xpath1.MustCompile("/root/a")
	require.Equal(t, "/root/a", compiled.String())
}

func TestEvaluatorFind(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)

	t.Run("node-set", func(t *testing.T) {
		nodes, err := xpath1.NewEvaluator().Find(t.Context(), xpath1.MustCompile("/root/*"), doc)
		require.NoError(t, err)
		require.Len(t, nodes, 2)
	})

	t.Run("not a node-set", func(t *testing.T) {
		_, err := xpath1.NewEvaluator().Find(t.Context(), xpath1.MustCompile("1+1"), doc)
		require.ErrorIs(t, err, xpath1.ErrNotNodeSet)
	})
}

func TestNilExpression(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	t.Run("Expression.Evaluate nil receiver", func(t *testing.T) {
		var e *xpath1.Expression
		_, err := e.Evaluate(t.Context(), doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})

	t.Run("Evaluator.Evaluate nil expression", func(t *testing.T) {
		_, err := xpath1.NewEvaluator().Evaluate(t.Context(), nil, doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})
}

// TestCompareNodeSetWithBooleanAndNumberScalars exercises compareWithScalar's
// boolean and number branches when a node-set is compared to a scalar.
func TestCompareNodeSetScalarBranches(t *testing.T) {
	doc := parseXML(t, `<root><a>x</a><b></b></root>`)
	for _, tc := range []struct {
		expr string
		want bool
	}{
		// node-set vs boolean scalar via compareWithScalar boolean branch
		{`/root/a = true()`, true},
		{`/root/a != false()`, true},
		{`/root/a > false()`, true}, // non-empty string -> 1 > 0
		// node-set string vs number scalar (number branch)
		{`/root/a = 0`, false},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := xpath1.Evaluate(t.Context(), doc, tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, r.Bool)
		})
	}
}

// TestStringValueOfDocument exercises resultToString / resultToNumber on a
// node-set whose first node is the document.
func TestRootStringConversion(t *testing.T) {
	doc := parseXML(t, `<root>42</root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "string(/)")
	require.NoError(t, err)
	require.Equal(t, "42", r.String)

	r, err = xpath1.Evaluate(t.Context(), doc, "number(/) + 1")
	require.NoError(t, err)
	require.Equal(t, 43.0, r.Number)
}
