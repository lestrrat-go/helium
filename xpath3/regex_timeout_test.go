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
	// Tighten the timeout for the test so we don't have to wait a full
	// second to confirm bounding.
	orig := xpath3.DefaultRegexMatchTimeout
	xpath3.DefaultRegexMatchTimeout = 100 * time.Millisecond
	t.Cleanup(func() { xpath3.DefaultRegexMatchTimeout = orig })

	// Backreference forces the regexp2 path. (a+)\1+ over a long run of 'a'
	// followed by a non-matching tail explores an exponential split space.
	input := strings.Repeat("a", 30) + "!"
	expr := `matches("` + input + `", "^(a+)\1+$")`

	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)

	start := time.Now()
	_, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	elapsed := time.Since(start)

	// Either the engine returns a timeout error, or it returns a non-timeout
	// result/error quickly. The forbidden outcome is "runs much longer than
	// the timeout without erroring". Allow a generous margin for scheduling.
	require.Less(t, elapsed, 2*time.Second,
		"regex match exceeded budget; elapsed=%v err=%v", elapsed, evalErr)
}
