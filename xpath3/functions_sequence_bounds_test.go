package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestSequenceBuiltinsRespectOpLimit(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).OpLimit(10).Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrOpLimit)
		})
	}
}

func TestSequenceBuiltinsRespectNodeLimit(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				MaxNodesForTesting(1000).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}
}

func TestSequenceBuiltinsPollCancellationDuringWork(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			ctx := heliumtest.NewPollContext(t.Context(), 10, context.Canceled)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.Canceled)
			require.Less(t, ctx.Polls(), 100, "the operation must stop polling soon after cancellation")
		})
	}
}

func TestSequenceBuiltinsHonorCancellation(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestEvaluateRejectsCancelledContext(t *testing.T) {
	t.Parallel()

	compiled, err := xpath3.NewCompiler().Compile("1")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
	require.ErrorIs(t, err, context.Canceled)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).EvaluateEBV(ctx, compiled, nil)
	require.ErrorIs(t, err, context.Canceled)
	state := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).NewEvalState(nil)
	_, err = compiled.EvaluateReuse(ctx, state, nil)
	require.ErrorIs(t, err, context.Canceled)
}
