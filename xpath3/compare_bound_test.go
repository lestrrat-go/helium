package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Finding XPATH3-108: a value comparison (eq/ne/lt/le/gt/ge) whose operand
// atomizes to more than one item must raise the cardinality error XPTY0004
// using an early-stop atomization — it must NOT materialize the whole
// (potentially huge or lazy) operand first. A counting lazy sequence proves
// that only a bounded number of items are ever produced.
func TestValueComparison_CardinalityEarlyStop(t *testing.T) {
	const n = 1_000_000
	doc := mustParseXML(t, "<root/>")

	var produced int
	big := sequence.NewRange[xpath3.Item](n, func(i int) xpath3.Item {
		produced++
		return xpath3.AtomicValue{TypeName: "xs:integer", Value: int64(i + 1)}
	})

	compiled, err := xpath3.NewCompiler().Compile("$big eq 0")
	require.NoError(t, err)

	// EvalBorrowing avoids the defensive variable-map clone, which would
	// otherwise materialize the whole lazy sequence before evaluation even
	// begins and mask the early-stop behavior under test.
	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(map[string]xpath3.Sequence{"big": big}).
		Evaluate(t.Context(), compiled, doc)

	require.Error(t, err)
	require.Contains(t, err.Error(), "XPTY0004")
	// Distinguishing empty / one / more-than-one needs at most two atoms; the
	// fix must early-stop rather than drain all n items.
	require.LessOrEqual(t, produced, 2,
		"value comparison must early-stop atomization, not materialize the whole operand")
}

// Finding XPATH3-101: a general comparison (= != < <= > >=) over two large
// sequences runs an O(N*M) left x right loop. That loop must consult the
// context so a context cancelled after evaluation begins aborts promptly,
// instead of scanning every pair. The cancel-after context lets evaluation
// start and only reports cancellation once the comparison loop is iterating.
func TestGeneralComparison_LargeLoopCancellable(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	// No pair is equal (1..1000 vs 2000..3000), so "=" must run the full
	// 1,000,000-pair loop unless it bails on cancellation.
	compiled, err := xpath3.NewCompiler().Compile("(1 to 1000) = (2000 to 3000)")
	require.NoError(t, err)

	const cancelAfter = 500
	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, doc)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, ctx.calls, 1000*1000,
		"general comparison must bail on cancellation, not scan every pair")
}

// Finding XPATH3-101: the general-comparison loop must charge ops against the
// configured op-limit so a large comparison cannot run unbounded.
func TestGeneralComparison_LargeLoopOpLimited(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("(1 to 1000) = (2000 to 3000)")
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		OpLimit(500).
		Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}
