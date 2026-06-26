package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A simple-map expression (E1 ! E2) whose right side yields a large lazy
// sequence must enforce the configured node-set/sequence limit BEFORE
// materializing the whole right-hand sequence. The right side is bound to a lazy
// Range that counts how many items it actually produces; the bound (20) is far
// below the range length (1000). The lazy accumulation must stop as soon as the
// running total would exceed the cap, so only a handful of items are ever
// produced. Before the fix the operator materialized the entire right-hand
// sequence first and only then checked the aggregate cap, producing all 1000
// items before erroring.
func TestEvalSimpleMapExpr_LazyRightHonorsMaxNodesBeforeMaterialize(t *testing.T) {
	const rangeLen = 1000
	const limit = 20

	var produced int
	big := sequence.NewRange[xpath3.Item](rangeLen, func(int) xpath3.Item {
		produced++
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x"}
	})

	compiled, err := xpath3.NewCompiler().Compile("1 ! $big")
	require.NoError(t, err)

	// EvalBorrowing keeps $big lazy (DefaultEvaluatorOptions would clone and thus
	// materialize the sequence up front, defeating the laziness probe).
	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(map[string]xpath3.Sequence{"big": big}).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
	// The cap must trip after producing only a bounded prefix, never the whole
	// 1000-item range.
	require.Less(t, produced, rangeLen, "right-hand sequence was fully materialized before the cap check")
	require.LessOrEqual(t, produced, limit+1)
}

// A simple-map whose accumulated result stays within the cap must succeed and
// return the full sequence, confirming the bound does not reject legitimate
// results.
func TestEvalSimpleMapExpr_WithinMaxNodes(t *testing.T) {
	const limit = 20

	compiled, err := xpath3.NewCompiler().Compile("(1, 2, 3) ! (1 to 2)")
	require.NoError(t, err)

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, 6, res.Sequence().Len())
}
