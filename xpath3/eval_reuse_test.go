package xpath3_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A zero-value Evaluator (cfg == nil) must not panic when building reusable
// state. newEvalCtx guards the nil cfg; NewEvalState must do the same.
func TestNewEvalState_ZeroValueEvaluator(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("1 + 1")
	require.NoError(t, err)

	var eval xpath3.Evaluator // zero value, cfg == nil
	require.NotPanics(t, func() {
		state := eval.NewEvalState(nil)
		result, evalErr := compiled.EvaluateReuse(t.Context(), state, nil)
		require.NoError(t, evalErr)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 2.0, n)
	})
}

// TraceWriter configured on the Evaluator must flow through to the reuse
// path so fn:trace writes to the caller's writer, not os.Stderr.
func TestNewEvalState_TraceWriter(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`trace(42, "label")`)
	require.NoError(t, err)

	var buf bytes.Buffer
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).TraceWriter(&buf)
	state := eval.NewEvalState(nil)

	_, err = compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	require.NotEmpty(t, buf.String(), "fn:trace output must go to the configured TraceWriter on the reuse path")
}

// A custom max-node limit configured on the Evaluator must be honored on the
// reuse path. With a tiny limit, a range expression exceeding it must error.
func TestNewEvalState_MaxNodesLimit(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile("1 to 100")
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).MaxNodesForTesting(10)
	state := eval.NewEvalState(nil)

	_, err = compiled.EvaluateReuse(t.Context(), state, nil)
	require.Error(t, err, "range exceeding the configured max-node limit must error on the reuse path")
	require.True(t, errors.Is(err, xpath3.ErrNodeSetLimit),
		"expected ErrNodeSetLimit, got %v", err)
}
