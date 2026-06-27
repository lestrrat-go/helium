package xpath1_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func TestEvaluateContext(t *testing.T) {
	// verifies that evaluating a nil or zero-value (uncompiled) Expression
	// returns a clear error instead of panicking.
	t.Run("nil expression", func(t *testing.T) {
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
	})

	// verifies that a cancelled context aborts a long evaluation promptly with
	// the context error rather than running to completion.
	t.Run("cancelled", func(t *testing.T) {
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
	})

	// verifies that a context cancelled before evaluation aborts even the
	// "simple" (bounded result size) axes promptly with context.Canceled,
	// rather than materializing the full node-set. child::* and attribute::*
	// route through TraverseAxisSimple, which previously never consulted ctx —
	// so a wide node could yield a full result with a nil error after
	// cancellation occurred.
	t.Run("cancelled simple axis", func(t *testing.T) {
		const width = 5000

		t.Run("child::*", func(t *testing.T) {
			var sb strings.Builder
			sb.WriteString("<root>")
			for range width {
				sb.WriteString("<c/>")
			}
			sb.WriteString("</root>")
			doc := parseXML(t, sb.String())

			expr, err := xpath1.Compile("/root/child::*")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err = expr.Evaluate(ctx, doc)
			require.ErrorIs(t, err, context.Canceled)
		})

		t.Run("attribute::*", func(t *testing.T) {
			var sb strings.Builder
			sb.WriteString("<root><e ")
			for i := range width {
				sb.WriteString("a")
				sb.WriteString(strconv.Itoa(i))
				sb.WriteString(`="v" `)
			}
			sb.WriteString("/></root>")
			doc := parseXML(t, sb.String())

			expr, err := xpath1.Compile("/root/e/attribute::*")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err = expr.Evaluate(ctx, doc)
			require.ErrorIs(t, err, context.Canceled)
		})
	})

	// confirms the cancellation guard does not break ordinary evaluation with a
	// live context.
	t.Run("ok", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/></root>`)
		expr, err := xpath1.Compile("//a")
		require.NoError(t, err)

		r, err := expr.Evaluate(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 2)
	})
}

// verifies that a node-set vs node-set general comparison (an O(n*m) string
// comparison) charges the operation limit instead of running an uncancellable
// quadratic loop. Two large, non-matching node-sets are compared under an op
// limit sized to clear the cheap path traversal but trip during the quadratic
// comparison.
func TestNodeSetComparisonBounded(t *testing.T) {
	const n = 300

	var sb strings.Builder
	sb.WriteString("<root>")
	for i := range n {
		sb.WriteString("<a>x")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("</a>")
	}
	for i := range n {
		sb.WriteString("<b>y")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("</b>")
	}
	sb.WriteString("</root>")
	doc := parseXML(t, sb.String())

	// /root/a and /root/b never share a string value, forcing the full
	// n*n comparison before returning false.
	compiled, err := xpath1.Compile("/root/a = /root/b")
	require.NoError(t, err)

	// The path traversal costs roughly 2*(1+2n) ~= 1200 ops; the comparison
	// then attempts n*n == 90000 string comparisons. A limit between the two
	// is honored only if the comparison loop charges the op counter.
	t.Run("op limit honored", func(t *testing.T) {
		_, err := xpath1.NewEvaluator().OpLimit(5000).Evaluate(t.Context(), compiled, doc)
		require.ErrorIs(t, err, xpath1.ErrOpLimit)
	})

	// A generous limit still produces the correct boolean result.
	t.Run("generous limit ok", func(t *testing.T) {
		r, err := xpath1.NewEvaluator().OpLimit(1_000_000).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.BooleanResult, r.Type)
		require.False(t, r.Bool)
	})
}
