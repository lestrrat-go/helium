package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestPrefixedFnStreamabilityXSLTLayer verifies that the XSLT-layer
// streamability analysis treats fn:-prefixed calls to special-cased built-ins
// (last, current-group, ...) identically to their unprefixed forms. The "fn"
// prefix is the reserved binding for the XPath functions namespace, so both
// spellings name the same built-in (see isFnNamespacePrefix).
func TestPrefixedFnStreamabilityXSLTLayer(t *testing.T) {
	compile := func(t *testing.T, src string) *xpath3.Expression {
		t.Helper()
		expr, err := xpath3.NewCompiler().Compile(src)
		require.NoError(t, err, "compile %q", src)
		return expr
	}

	t.Run("last outside grounding", func(t *testing.T) {
		plain := compile(t, "child::a[last()]")
		pref := compile(t, "child::a[fn:last()]")
		require.Equal(t,
			exprUsesLastOutsideGrounding(plain),
			exprUsesLastOutsideGrounding(pref),
			"last() classification must match unprefixed vs fn:-prefixed")
		// Sanity: the unprefixed form is detected, so this is a meaningful assertion.
		require.True(t, exprUsesLastOutsideGrounding(plain), "unprefixed last() must be detected")
	})

	t.Run("current-group consuming", func(t *testing.T) {
		plain := compile(t, "current-group()/child::b")
		pref := compile(t, "fn:current-group()/child::b")
		require.Equal(t,
			countCurrentGroupConsumingInExpr(plain.AST()),
			countCurrentGroupConsumingInExpr(pref.AST()),
			"current-group() consuming count must match unprefixed vs fn:-prefixed")
		// Sanity: the unprefixed form is counted, so this is a meaningful assertion.
		require.Positive(t, countCurrentGroupConsumingInExpr(plain.AST()),
			"unprefixed current-group() must be counted as consuming")
	})

	t.Run("function outside grounding", func(t *testing.T) {
		plain := compile(t, "child::a[position() = 1]")
		pref := compile(t, "child::a[fn:position() = 1]")
		require.Equal(t,
			exprUsesFunctionOutsideGrounding(plain, "position"),
			exprUsesFunctionOutsideGrounding(pref, "position"),
			"position() classification must match unprefixed vs fn:-prefixed")
		require.True(t, exprUsesFunctionOutsideGrounding(plain, "position"),
			"unprefixed position() must be detected")
	})

	t.Run("higher-order filter with consuming arg", func(t *testing.T) {
		plain := compile(t, "filter(child::a, function($x) { true() })")
		pref := compile(t, "fn:filter(child::a, function($x) { true() })")
		require.Equal(t,
			exprHasHigherOrderWithConsumingArg(plain),
			exprHasHigherOrderWithConsumingArg(pref),
			"filter() HOF classification must match unprefixed vs fn:-prefixed")
		// Sanity: the unprefixed form is rejected, so this is a meaningful assertion.
		require.True(t, exprHasHigherOrderWithConsumingArg(plain),
			"unprefixed filter() with consuming arg must be detected")
	})

	t.Run("higher-order fold-right with consuming arg", func(t *testing.T) {
		plain := compile(t, "fold-right(child::a, 0, function($x, $y) { $y })")
		pref := compile(t, "fn:fold-right(child::a, 0, function($x, $y) { $y })")
		require.Equal(t,
			exprHasHigherOrderWithConsumingArg(plain),
			exprHasHigherOrderWithConsumingArg(pref),
			"fold-right() HOF classification must match unprefixed vs fn:-prefixed")
		require.True(t, exprHasHigherOrderWithConsumingArg(plain),
			"unprefixed fold-right() with consuming arg must be detected")
	})

	t.Run("forbidden current-group in pattern", func(t *testing.T) {
		plain := compile(t, "current-group()")
		pref := compile(t, "fn:current-group()")
		plainErr := checkPatternForbiddenFunctions(plain.AST())
		prefErr := checkPatternForbiddenFunctions(pref.AST())
		require.Error(t, plainErr, "unprefixed current-group() must be forbidden in pattern")
		require.Error(t, prefErr,
			"fn:current-group() must be forbidden in pattern, same as unprefixed")
	})
}
