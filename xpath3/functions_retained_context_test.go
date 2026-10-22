package xpath3_test

import (
	"context"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestRetainedNamedFunctionUsesInvocationOpLimit(t *testing.T) {
	t.Parallel()

	created, err := xpath3.NewCompiler().Compile("for-each#2")
	require.NoError(t, err)
	lowLimitResult, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		OpLimit(10).
		Evaluate(t.Context(), created, nil)
	require.NoError(t, err)
	retained, ok := lowLimitResult.Sequence().Get(0).(xpath3.FunctionItem)
	require.True(t, ok)

	call, err := xpath3.NewCompiler().Compile("$f((1, 2, 3), function($x) { $x })")
	require.NoError(t, err)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"f": xpath3.ItemSlice{retained}}).
		OpLimit(100)
	for range 2 {
		result, err := eval.Evaluate(t.Context(), call, nil)
		require.NoError(t, err)
		require.Equal(t, 3, result.Sequence().Len())
	}
}

func TestRetainedNamedFunctionUsesInvocationBudget(t *testing.T) {
	t.Parallel()

	created, err := xpath3.NewCompiler().Compile("for-each#2")
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), created, nil)
	require.NoError(t, err)
	retained, ok := result.Sequence().Get(0).(xpath3.FunctionItem)
	require.True(t, ok)

	call, err := xpath3.NewCompiler().Compile("$f(1 to 100, function($x) { $x })")
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"f": xpath3.ItemSlice{retained}}).
		OpLimit(10).
		Evaluate(t.Context(), call, nil)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}

func TestRetainedFunctionLookupUsesInvocationContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	created, err := xpath3.NewCompiler().Compile(`function-lookup(QName("http://www.w3.org/2005/xpath-functions", "for-each"), 2)`)
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, created, nil)
	require.NoError(t, err)
	retained, ok := result.Sequence().Get(0).(xpath3.FunctionItem)
	require.True(t, ok)
	cancel()

	call, err := xpath3.NewCompiler().Compile("$f((1), function($x) { $x })")
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{"f": xpath3.ItemSlice{retained}}).
		Evaluate(t.Context(), call, nil)
	require.NoError(t, err)
}

func TestRetainedNamedFunctionConcurrentCallsUseSeparateBudgets(t *testing.T) {
	created, err := xpath3.NewCompiler().Compile("for-each#2")
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), created, nil)
	require.NoError(t, err)
	retained, ok := result.Sequence().Get(0).(xpath3.FunctionItem)
	require.True(t, ok)
	call, err := xpath3.NewCompiler().Compile("$f(1 to 100, function($x) { $x })")
	require.NoError(t, err)

	const calls = 2
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for range calls {
		wg.Go(func() {
			_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Variables(map[string]xpath3.Sequence{"f": xpath3.ItemSlice{retained}}).
				OpLimit(500).
				Evaluate(t.Context(), call, nil)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
