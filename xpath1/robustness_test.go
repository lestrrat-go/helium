package xpath1_test

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
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

	// A node-set vs node-set comparison with one empty side is always false
	// and must short-circuit BEFORE computing any string values for the
	// non-empty side. A tight op limit (well below the n string values the
	// non-empty side would otherwise require) is honored without error,
	// proving the empty-side early return skips the per-node work.
	t.Run("empty right side short-circuits", func(t *testing.T) {
		compiled, err := xpath1.Compile("/root/a = /root/nonexistent")
		require.NoError(t, err)
		r, err := xpath1.NewEvaluator().OpLimit(2000).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.BooleanResult, r.Type)
		require.False(t, r.Bool)
	})

	t.Run("empty left side short-circuits", func(t *testing.T) {
		compiled, err := xpath1.Compile("/root/nonexistent = /root/b")
		require.NoError(t, err)
		r, err := xpath1.NewEvaluator().OpLimit(2000).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.BooleanResult, r.Type)
		require.False(t, r.Bool)
	})

	// Context cancellation is honored mid-compare: a context cancelled before
	// evaluation aborts the n*n comparison with context.Canceled rather than
	// running to completion.
	t.Run("context cancellation honored", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := xpath1.NewEvaluator().OpLimit(1_000_000).Evaluate(ctx, compiled, doc)
		require.ErrorIs(t, err, context.Canceled)
	})

	// The comparison path itself owns the string-value work: the right-hand
	// node-set comes from a custom function rather than a location path, so
	// eval()'s top-level cancellation check runs (uncancelled) BEFORE the
	// operands are evaluated. The function cancels the context as a side effect
	// just before handing back the node-set, so the cancellation is honored
	// only if compareNodeSet itself consults the context before materializing
	// right-hand string values. Before the lazy-materialization fix, the
	// compare eagerly built every right-hand string value with no context check
	// and returned a boolean, ignoring the cancellation.
	t.Run("comparison owns cancellation bound", func(t *testing.T) {
		rightPath, err := xpath1.Compile("/root/b")
		require.NoError(t, err)
		rhs, err := xpath1.NewEvaluator().Evaluate(t.Context(), rightPath, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, rhs.Type)
		require.Len(t, rhs.NodeSet, n)

		ctx, cancel := context.WithCancel(t.Context())
		boom := xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			// Cancel just before the comparison consumes the node-set.
			cancel()
			return rhs, nil
		})

		expr, err := xpath1.Compile("/root/a[1] = boom()")
		require.NoError(t, err)

		_, err = xpath1.NewEvaluator().Function("boom", boom).Evaluate(ctx, expr, doc)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// cancelNode wraps a real node and fires onContent the first time its
// string-value is materialized (StringValue reads a text node via Content()).
// It lets a node-set comparison cancel its own context mid-loop, deterministically
// and single-threaded, so the per-pair cancellation check can be exercised.
type cancelNode struct {
	helium.Node
	onContent func()
}

func (c *cancelNode) Content() []byte {
	if c.onContent != nil {
		c.onContent()
	}
	return c.Node.Content()
}

// textNodeAt returns the first text-node child of the element at path.
func textNodeAt(t *testing.T, doc *helium.Document, path string) helium.Node {
	t.Helper()
	el := nodeAt(t, doc, path)
	tn := el.FirstChild()
	require.NotNil(t, tn)
	require.Equal(t, helium.TextNode, tn.Type())
	return tn
}

// TestNodeSetComparisonCancelAfterN pins the per-pair cancellation check in the
// node-set-vs-node-set compare. The context is cancelled MID-loop (as a side
// effect of materializing the 3rd left node's string value) over a comparison of
// far fewer than the old 1024-pair re-check window. With only a pre-loop check
// plus an every-1024 re-check, that cancellation is missed entirely and the
// compare runs to completion returning (false, nil). A per-pair check catches it
// on the very next pair and returns the context error promptly.
func TestNodeSetComparisonCancelAfterN(t *testing.T) {
	const leftCount = 8
	const cancelAt = 2 // 0-based index of the left node whose Content() cancels

	var sb strings.Builder
	sb.WriteString("<root>")
	for i := range leftCount {
		sb.WriteString("<a>x")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("</a>")
	}
	sb.WriteString("<b>y</b></root>")
	doc := parseXML(t, sb.String())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var materialized int
	leftNodes := make([]helium.Node, leftCount)
	for i := range leftNodes {
		tn := textNodeAt(t, doc, "/root/a["+strconv.Itoa(i+1)+"]")
		idx := i
		leftNodes[i] = &cancelNode{Node: tn, onContent: func() {
			materialized++
			if idx == cancelAt {
				cancel()
			}
		}}
	}
	rightNode := textNodeAt(t, doc, "/root/b")

	lhs := nodeSetFunc(leftNodes)
	rhs := nodeSetFunc([]helium.Node{rightNode})

	expr, err := xpath1.Compile("lhs() = rhs()")
	require.NoError(t, err)

	_, err = xpath1.NewEvaluator().
		Function("lhs", lhs).
		Function("rhs", rhs).
		OpLimit(1_000_000).
		Evaluate(ctx, expr, doc)
	require.ErrorIs(t, err, context.Canceled)

	// The compare must have aborted promptly: only the nodes up to and including
	// the cancelling one were materialized, never all leftCount of them.
	require.Equal(t, cancelAt+1, materialized,
		"compare kept materializing left string values after cancellation")
}

// nodeAt resolves a location path to its first node, asserting the result is a
// non-empty node-set. Used to grab a real node for a synthetic node-set.
func nodeAt(t *testing.T, doc *helium.Document, path string) helium.Node {
	t.Helper()
	expr, err := xpath1.Compile(path)
	require.NoError(t, err)
	r, err := expr.Evaluate(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.NotEmpty(t, r.NodeSet)
	return r.NodeSet[0]
}

// nodeSetFunc returns a custom function that hands back the given nodes as a
// node-set result. A no-argument call costs exactly one op.
func nodeSetFunc(nodes []helium.Node) xpath1.Function {
	return xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.NodeSetResult, NodeSet: nodes}, nil
	})
}

// TestNodeSetComparisonNoUpfrontWork pins compareNodeSet's bound: every unit of
// work — the O(len(right)) right-hand cache AND each side's string-value
// materialization — must come AFTER the per-pair op charge. The regressions
// measure allocation across an Evaluate call: before the fix, a pre-exhausted
// budget (or a tight limit over a huge right-hand set) still allocated O(m)
// dense caches and/or materialized a huge left string value before charging.
func TestNodeSetComparisonNoUpfrontWork(t *testing.T) {
	allocDelta := func(fn func()) uint64 {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		fn()
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc
	}

	// Gap (i), left side: the op budget is spent before compareNodeSet runs, so
	// the very first per-pair charge must fail BEFORE the (huge) left node's
	// string value is materialized. OpLimit(2) is exactly the cost of the two
	// no-arg function operands, so compareNodeSet's first countOps trips.
	t.Run("pre-exhausted budget skips left materialization", func(t *testing.T) {
		const leftSize = 8 << 20 // 8 MiB string value
		doc := parseXML(t, "<root><a>"+strings.Repeat("x", leftSize)+"</a><b>y</b></root>")
		lhs := nodeSetFunc([]helium.Node{nodeAt(t, doc, "/root/a")})
		rhs := nodeSetFunc([]helium.Node{nodeAt(t, doc, "/root/b")})

		expr, err := xpath1.Compile("lhs() = rhs()")
		require.NoError(t, err)

		var cmpErr error
		delta := allocDelta(func() {
			_, cmpErr = xpath1.NewEvaluator().
				Function("lhs", lhs).
				Function("rhs", rhs).
				OpLimit(2).
				Evaluate(t.Context(), expr, doc)
		})
		require.ErrorIs(t, cmpErr, xpath1.ErrOpLimit)
		require.Less(t, delta, uint64(2<<20),
			"compare materialized the left string value before charging the op counter")
	})

	// Gap (i), right side: a huge right-hand node-set from a custom function with
	// a pre-exhausted budget must not trigger any O(m) up-front allocation.
	t.Run("pre-exhausted budget skips O(m) right allocation", func(t *testing.T) {
		doc := parseXML(t, "<root><a>x</a><b>y</b></root>")
		const m = 1_000_000
		big := make([]helium.Node, m)
		rn := nodeAt(t, doc, "/root/b")
		for i := range big {
			big[i] = rn
		}
		lhs := nodeSetFunc([]helium.Node{nodeAt(t, doc, "/root/a")})
		rhs := nodeSetFunc(big)

		expr, err := xpath1.Compile("lhs() = rhs()")
		require.NoError(t, err)

		var cmpErr error
		delta := allocDelta(func() {
			_, cmpErr = xpath1.NewEvaluator().
				Function("lhs", lhs).
				Function("rhs", rhs).
				OpLimit(2).
				Evaluate(t.Context(), expr, doc)
		})
		require.ErrorIs(t, cmpErr, xpath1.ErrOpLimit)
		require.Less(t, delta, uint64(4<<20),
			"compare allocated an O(len(right)) cache before charging the op counter")
	})

	// Gap (ii): a tight (non-zero) op limit over a huge right-hand set trips after
	// a bounded number of pairs. The work — and the sparse right-hand cache — must
	// stay proportional to the pairs actually reached, not to len(right).
	t.Run("tight limit trips after bounded work on huge right set", func(t *testing.T) {
		doc := parseXML(t, "<root><a>x</a><b>y</b></root>")
		const m = 1_000_000
		big := make([]helium.Node, m)
		rn := nodeAt(t, doc, "/root/b")
		for i := range big {
			big[i] = rn
		}
		lhs := nodeSetFunc([]helium.Node{nodeAt(t, doc, "/root/a")})
		rhs := nodeSetFunc(big)

		expr, err := xpath1.Compile("lhs() = rhs()")
		require.NoError(t, err)

		var cmpErr error
		delta := allocDelta(func() {
			// Two function operands cost 2 ops; the remaining 62 trip mid-loop.
			_, cmpErr = xpath1.NewEvaluator().
				Function("lhs", lhs).
				Function("rhs", rhs).
				OpLimit(64).
				Evaluate(t.Context(), expr, doc)
		})
		require.ErrorIs(t, cmpErr, xpath1.ErrOpLimit)
		require.Less(t, delta, uint64(4<<20),
			"compare allocated an O(len(right)) cache instead of growing lazily")
	})
}
