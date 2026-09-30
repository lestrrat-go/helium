package xpath3_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// The atomic context item seeded via Evaluator.ContextItem / NewEvalState must
// survive across EvaluateReuse calls. A reuse call with a non-nil node clears
// the focus for that call; a later reuse call with a nil node must fall back to
// the NewEvalState-seeded base context item, and must never error with XPDY0002.
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
// re-read the clock on each EvaluateReuse call, never staying frozen at the
// time NewEvalState was constructed. The test runs in a synctest bubble, whose
// fake clock moves by exactly the slept duration, so the second call must
// report the first call's time plus one second.
func TestEvaluateReuse_CurrentTimeRefreshesWhenUnset(t *testing.T) {
	synctest.Test(t, testCurrentTimeRefreshesWhenUnset)
}

func testCurrentTimeRefreshesWhenUnset(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("current-dateTime()")
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	state := eval.NewEvalState(nil)

	first := evaluateReuseTime(t, compiled, state)
	time.Sleep(time.Second)
	second := evaluateReuseTime(t, compiled, state)

	require.Equal(t, time.Second, second.Sub(first),
		"current-dateTime() must follow the clock across reuse calls when CurrentTime is unset")
}

// An explicitly configured CurrentTime must stay pinned across EvaluateReuse
// calls: the refresh must not clobber a user-pinned clock. The bubble's fake
// clock moves by one second between the calls, so a refresh would show.
func TestEvaluateReuse_CurrentTimePinnedWhenSet(t *testing.T) {
	synctest.Test(t, testCurrentTimePinnedWhenSet)
}

func testCurrentTimePinnedWhenSet(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("current-dateTime()")
	require.NoError(t, err)

	pinned := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).CurrentTime(pinned)
	state := eval.NewEvalState(nil)

	first := evaluateReuseTime(t, compiled, state)
	time.Sleep(time.Second)
	second := evaluateReuseTime(t, compiled, state)

	require.True(t, first.Equal(pinned), "first call must report the pinned time, got %s", first)
	require.True(t, second.Equal(pinned), "pinned CurrentTime must remain fixed across reuse calls, got %s", second)
}

// evaluateReuseTime evaluates compiled against state and parses the resulting
// xs:dateTime string.
func evaluateReuseTime(t *testing.T, compiled *xpath3.Expression, state *xpath3.EvalState) time.Time {
	t.Helper()
	res, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	got, err := time.Parse(time.RFC3339Nano, res.StringValue())
	require.NoError(t, err)
	return got
}
