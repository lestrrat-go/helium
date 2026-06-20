package xpath1_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// TestEvaluateNilExpression verifies that evaluating a nil or zero-value
// (uncompiled) Expression returns a clear error instead of panicking.
func TestEvaluateNilExpression(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	t.Run("nil pointer", func(t *testing.T) {
		var expr *xpath1.Expression
		_, err := expr.Evaluate(t.Context(), doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})

	t.Run("zero value", func(t *testing.T) {
		var expr xpath1.Expression
		_, err := expr.Evaluate(t.Context(), doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})

	t.Run("evaluator nil expr", func(t *testing.T) {
		ev := xpath1.NewEvaluator()
		_, err := ev.Evaluate(t.Context(), nil, doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})

	t.Run("evaluator zero expr", func(t *testing.T) {
		ev := xpath1.NewEvaluator()
		_, err := ev.Evaluate(t.Context(), &xpath1.Expression{}, doc)
		require.ErrorIs(t, err, xpath1.ErrNilExpression)
	})
}

// TestEvaluateContextCancelled verifies that a cancelled context aborts a
// long evaluation promptly with the context error rather than running to
// completion.
func TestEvaluateContextCancelled(t *testing.T) {
	// Build a reasonably large tree so descendant traversal does real work.
	var sb strings.Builder
	sb.WriteString("<root>")
	for range 5000 {
		sb.WriteString("<a><b><c/></b></a>")
	}
	sb.WriteString("</root>")
	doc := parseXML(t, sb.String())

	// A descendant-or-self traversal with a predicate exercises both the
	// axis-iteration loops and the per-node predicate evaluation.
	expr, err := xpath1.Compile("//*[. = .]")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // already cancelled before evaluation begins

	_, err = expr.Evaluate(ctx, doc)
	require.ErrorIs(t, err, context.Canceled)
}

// TestEvaluateContextOK confirms the cancellation guard does not break
// ordinary evaluation with a live context.
func TestEvaluateContextOK(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/></root>`)
	expr, err := xpath1.Compile("//a")
	require.NoError(t, err)

	r, err := expr.Evaluate(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 2)
}
