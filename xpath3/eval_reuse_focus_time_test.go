package xpath3_test

import (
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// The atomic context item seeded via Evaluator.ContextItem / NewEvalState must
// survive across EvaluateReuse calls. A reuse call with a non-nil node clears
// the focus for that call; a later reuse call with a nil node must fall back to
// the NewEvalState-seeded base context item rather than erroring with XPDY0002.
func TestEvaluateReuse_RestoresSeededContextItem(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(".")
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		ContextItem(xpath3.AtomicValue{TypeName: "xs:integer", Value: int64(42)})
	state := eval.NewEvalState(nil)

	// First call against the seeded base focus.
	res, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Sequence().Len())

	// A reuse call with a non-nil node uses that node as focus.
	doc := mustParseXML(t, "<root/>")
	res, err = compiled.EvaluateReuse(t.Context(), state, doc)
	require.NoError(t, err)
	require.Equal(t, 1, res.Sequence().Len())

	// A later reuse call with a nil node must fall back to the seeded
	// atomic context item, NOT lose it and raise XPDY0002.
	res, err = compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err, "seeded context item must be restored on reuse with nil node")
	require.Equal(t, 1, res.Sequence().Len())
}

// Without an explicitly configured CurrentTime, fn:current-dateTime() must
// re-read the clock on each EvaluateReuse call rather than staying frozen at
// the time NewEvalState was constructed.
func TestEvaluateReuse_CurrentTimeRefreshesWhenUnset(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("current-dateTime()")
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	state := eval.NewEvalState(nil)

	first, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	firstStr := first.StringValue()

	// Advance the wall clock enough to guarantee a distinguishable timestamp.
	target := time.Now().Add(2 * time.Millisecond)
	for time.Now().Before(target) {
	}

	second, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	secondStr := second.StringValue()

	require.NotEqual(t, firstStr, secondStr,
		"current-dateTime() must not be frozen across reuse calls when CurrentTime is unset")
}

// An explicitly configured CurrentTime must stay pinned across EvaluateReuse
// calls — the refresh must not clobber a user-pinned clock.
func TestEvaluateReuse_CurrentTimePinnedWhenSet(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("current-dateTime()")
	require.NoError(t, err)

	pinned := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).CurrentTime(pinned)
	state := eval.NewEvalState(nil)

	first, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	firstStr := first.StringValue()

	target := time.Now().Add(2 * time.Millisecond)
	for time.Now().Before(target) {
	}

	second, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	secondStr := second.StringValue()

	require.Equal(t, firstStr, secondStr,
		"pinned CurrentTime must remain fixed across reuse calls")
}
