package xpath3_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Lowering DefaultMaxRecursionDepth must cause deeply-nested expressions
// to terminate quickly with ErrRecursionLimit rather than continuing to
// consume goroutine stack.
func TestDefaultMaxRecursionDepth_Tunable(t *testing.T) {
	orig := xpath3.DefaultMaxRecursionDepth
	xpath3.DefaultMaxRecursionDepth = 8
	t.Cleanup(func() { xpath3.DefaultMaxRecursionDepth = orig })

	// A long chain of left-associative binary expressions
	//   ((((1+1)+1)+1)…+1)
	// forces the evaluator to recurse N-1 times to reach the leaf.
	// 50 operands → ~49 levels, comfortably above the limit of 8.
	const n = 50
	expr := "1" + strings.Repeat("+1", n-1)

	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)

	_, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, evalErr, "deeply-nested expression must trip the lowered recursion limit")
	require.True(t, errors.Is(evalErr, xpath3.ErrRecursionLimit),
		"expected ErrRecursionLimit, got %v", evalErr)
}
