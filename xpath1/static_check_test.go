package xpath1_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func TestEvaluatorValidate(t *testing.T) {
	custom := xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
	})
	eval := xpath1.NewEvaluator().
		Namespaces(map[string]string{"item": "urn:item", "ext": "urn:ext"}).
		Variables(map[string]any{"bound": true}).
		Function("custom", custom).
		FunctionNS("urn:ext", "custom", custom)

	for _, source := range []string{
		`item:root/item:child[custom() and ext:custom()]`,
		`not(self::item:secret)`,
		`(item:a | item:b)/item:c`,
		`(item:a)[ext:custom()]`,
		`$bound`,
	} {
		expr, err := xpath1.Compile(source)
		require.NoError(t, err)
		require.NoError(t, eval.Validate(expr), source)
	}

	t.Run("unknown unqualified function", func(t *testing.T) {
		expr := xpath1.MustCompile(`false() and missing()`)
		require.ErrorIs(t, eval.Validate(expr), xpath1.ErrUnknownFunction)
	})

	t.Run("unknown namespaced function", func(t *testing.T) {
		expr := xpath1.MustCompile(`ext:missing()`)
		require.ErrorIs(t, eval.Validate(expr), xpath1.ErrUnknownFunction)
	})

	t.Run("unbound function prefix", func(t *testing.T) {
		expr := xpath1.MustCompile(`missing:call()`)
		require.ErrorIs(t, eval.Validate(expr), xpath1.ErrUnknownFunctionNamespace)
	})

	t.Run("unbound name test prefix", func(t *testing.T) {
		expr := xpath1.MustCompile(`not(self::missing:secret)`)
		require.ErrorIs(t, eval.Validate(expr), xpath1.ErrUnknownNamespacePrefix)
	})

	t.Run("undefined variable", func(t *testing.T) {
		expr := xpath1.MustCompile(`$missing`)
		require.ErrorIs(t, eval.Validate(expr), xpath1.ErrUndefinedVariable)
	})

	t.Run("xml prefix is predefined for name tests", func(t *testing.T) {
		expr := xpath1.MustCompile(`ancestor-or-self::*[@xml:lang]`)
		require.NoError(t, xpath1.NewEvaluator().Validate(expr))
	})

	t.Run("nil expression", func(t *testing.T) {
		require.ErrorIs(t, eval.Validate(nil), xpath1.ErrNilExpression)
	})
}
