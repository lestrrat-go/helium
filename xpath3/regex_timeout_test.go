package xpath3_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Patterns containing features Go's RE2 cannot handle (backreferences,
// character-class subtraction, large quantifiers) fall through to a
// backtracking regex engine that is vulnerable to catastrophic backtracking
// on adversary-supplied inputs. xpath3 sets a default match timeout on
// every such compilation so a pathological pattern + input does not pin
// a goroutine.
func TestRegexMatchTimeout_BoundsCatastrophicBacktracking(t *testing.T) {
	// regexp2's fastclock has ~100ms granularity, so a 150ms budget
	// realizes as ~150-300ms wall time. The elapsed bound is well below
	// the 5s default DefaultRegexMatchTimeout — a passing assertion
	// here proves the lowered budget actually took effect.
	const matchBudget = 150 * time.Millisecond
	const elapsedBound = 750 * time.Millisecond

	orig := xpath3.DefaultRegexMatchTimeout
	xpath3.DefaultRegexMatchTimeout = matchBudget
	t.Cleanup(func() { xpath3.DefaultRegexMatchTimeout = orig })

	// (.+)+\1 forces the regexp2 path (backreference) and exhibits
	// catastrophic backtracking: with 30 'a's plus a non-matching 'b'
	// the engine explores ~2^n splits. Empirically this runs many
	// seconds without a timeout; with matchBudget it must fail quickly
	// with regexp2's "match timeout after ..." error.
	input := strings.Repeat("a", 30) + "b"
	expr := `matches("` + input + `", "^(.+)+\1$")`

	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)

	start := time.Now()
	_, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	elapsed := time.Since(start)

	require.Error(t, evalErr,
		"expected regexp2 match timeout, got nil error; elapsed=%v", elapsed)
	require.Contains(t, evalErr.Error(), "match timeout",
		"expected regexp2 timeout error, got %v; elapsed=%v", evalErr, elapsed)
	require.Less(t, elapsed, elapsedBound,
		"timeout did not fire near %v budget; elapsed=%v err=%v",
		matchBudget, elapsed, evalErr)
}
