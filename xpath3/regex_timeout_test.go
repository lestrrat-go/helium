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
//
// The test lowers DefaultRegexMatchTimeout and checks that the compiled
// pattern carries the lowered budget, which is what bounds the match, and
// that the pathological match then fails with the engine's timeout error. It
// asserts nothing about how long the match took.
func TestRegexMatchTimeout_BoundsCatastrophicBacktracking(t *testing.T) {
	const matchBudget = 150 * time.Millisecond

	orig := xpath3.DefaultRegexMatchTimeout
	xpath3.DefaultRegexMatchTimeout = matchBudget
	t.Cleanup(func() { xpath3.DefaultRegexMatchTimeout = orig })

	// (.+)+\1 forces the regexp2 path (backreference) and exhibits
	// catastrophic backtracking: with 30 'a's plus a non-matching 'b'
	// the engine explores ~2^n splits. Empirically this runs many
	// seconds without a timeout; with matchBudget it must fail with
	// regexp2's "match timeout after ..." error. No other test compiles this
	// pattern, so the compilation cache cannot hand back one carrying a
	// different budget.
	const pattern = `^(.+)+\1$`
	re, err := xpath3.CompileRegex(pattern, "")
	require.NoError(t, err)
	require.Equal(t, matchBudget, re.MatchTimeoutForTesting(),
		"the backtracking compilation does not carry DefaultRegexMatchTimeout")

	input := strings.Repeat("a", 30) + "b"
	expr := `matches("` + input + `", "` + pattern + `")`

	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)

	_, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)

	require.Error(t, evalErr, "expected regexp2 match timeout, got nil error")
	require.Contains(t, evalErr.Error(), "match timeout",
		"expected regexp2 timeout error, got %v", evalErr)
}

func TestRegex_PublicAPI(t *testing.T) {
	re, err := xpath3.CompileRegex(`a(b+)c`, "")
	require.NoError(t, err)

	matched, err := re.MatchString("abbbc")
	require.NoError(t, err)
	require.True(t, matched)

	matched, err = re.MatchString("xyz")
	require.NoError(t, err)
	require.False(t, matched)

	idx, err := re.FindAllSubmatchIndex("abc and abbc", -1)
	require.NoError(t, err)
	require.Len(t, idx, 2)
	// First match "abc": full match + one capture group => 4 indices.
	require.Len(t, idx[0], 4)

	// Case-insensitive flag.
	rei, err := xpath3.CompileRegex(`abc`, "i")
	require.NoError(t, err)
	matched, err = rei.MatchString("ABC")
	require.NoError(t, err)
	require.True(t, matched)

	// Invalid pattern surfaces an error.
	_, err = xpath3.CompileRegex(`[`, "")
	require.Error(t, err)
}
