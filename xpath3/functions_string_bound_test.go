package xpath3_test

import (
	"context"
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
