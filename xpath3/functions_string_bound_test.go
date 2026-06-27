package xpath3_test

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Finding XPATH3-105: fn:string-to-codepoints builds the full codepoint
// sequence one item per input character. That construction must charge the
// evaluator op-limit so a huge input string cannot materialize an unbounded
// item sequence below the node-set cap but above the configured op budget.
func TestStringToCodepoints_LargeOpLimited(t *testing.T) {
	const n = 200_000
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("string-to-codepoints($s)")
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"s": xpath3.SingleString(strings.Repeat("x", n))}).
		OpLimit(1000).
		Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}

// Finding XPATH3-105: the codepoint-building loop must consult the context so a
// cancellation that lands after evaluation begins aborts promptly instead of
// scanning the whole input string.
func TestStringToCodepoints_LargeCancellable(t *testing.T) {
	const n = 200_000
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("string-to-codepoints($s)")
	require.NoError(t, err)

	const cancelAfter = 500
	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"s": xpath3.SingleString(strings.Repeat("x", n))}).
		Evaluate(ctx, compiled, doc)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, ctx.calls, n,
		"string-to-codepoints must bail on cancellation, not scan the whole string")
}

// Finding XPATH3-102: fn:analyze-string iterates the regex matches and builds a
// result tree element per match. That loop must charge the evaluator op-limit so
// an input with a huge number of matches cannot build an unbounded result tree
// below the node-set cap but above the configured op budget.
func TestAnalyzeString_LargeOpLimited(t *testing.T) {
	const n = 200_000
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("analyze-string($s, 'a')")
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"s": xpath3.SingleString(strings.Repeat("a", n))}).
		OpLimit(1000).
		Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}

// Finding XPATH3-102: the analyze-string match loop must consult the context so
// a cancellation that lands after evaluation begins aborts the tree build
// promptly instead of emitting an element for every match first.
func TestAnalyzeString_LargeCancellable(t *testing.T) {
	const n = 200_000
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("analyze-string($s, 'a')")
	require.NoError(t, err)

	const cancelAfter = 500
	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"s": xpath3.SingleString(strings.Repeat("a", n))}).
		Evaluate(ctx, compiled, doc)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, ctx.calls, n,
		"analyze-string must bail on cancellation, not emit an element per match")
}

// Finding XPATH3-102 (r2): rejecting on the op-limit is not enough — the match
// enumeration itself must STREAM so it never materializes the full O(matches)
// index slice up front. With a tight OpLimit over millions of matches the call
// must reject after charging ~OpLimit ops while allocating only a small bounded
// amount of memory, far below the tens of MB a FindAllStringSubmatchIndex over
// every match would need. Measuring the TotalAlloc delta proves the up-front
// match slice is gone: a pre-fix build that enumerates all matches first blows
// well past this ceiling regardless of where the op charge later fires.
func TestAnalyzeString_LargeOpLimitedDoesNotMaterializeAllMatches(t *testing.T) {
	const n = 4_000_000
	doc := mustParseXML(t, "<root/>")

	compiled, err := xpath3.NewCompiler().Compile("analyze-string($s, 'a')")
	require.NoError(t, err)

	input := strings.Repeat("a", n)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"s": xpath3.SingleString(input)}).
		OpLimit(1000).
		Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// FindAllStringSubmatchIndex over n single-char matches materializes an
	// [][]int with n entries (each a 2-int slice) — at least n*(24+16) bytes,
	// > 150MB for n=4M. Streaming and stopping at the OpLimit allocates only the
	// ~1000 result elements, well under this 32MB ceiling.
	const ceiling = 32 << 20
	delta := after.TotalAlloc - before.TotalAlloc
	require.Less(t, delta, uint64(ceiling),
		"analyze-string must stream matches, not materialize the full O(matches) index slice; allocated %d bytes over %d matches", delta, n)
}
