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
		plainErr := checkPatternForbiddenFunctions(plain.AST(), nil)
		prefErr := checkPatternForbiddenFunctions(pref.AST(), nil)
		require.Error(t, plainErr, "unprefixed current-group() must be forbidden in pattern")
		require.Error(t, prefErr,
			"fn:current-group() must be forbidden in pattern, same as unprefixed")
	})
}

// TestEQNameFnStreamabilityXSLTLayer verifies that EQName-spelled (Q{...}local)
// calls to special-cased built-ins classify the same as their unprefixed forms.
// The parser keeps the whole braced spelling in FunctionCall.Name with an empty
// Prefix, so the XSLT streamability gates must normalize it via the shared
// lexicon.StreamFnLocalName helper rather than comparing raw names.
func TestEQNameFnStreamabilityXSLTLayer(t *testing.T) {
	const fnNS = "http://www.w3.org/2005/xpath-functions"
	compile := func(t *testing.T, src string) *xpath3.Expression {
		t.Helper()
		expr, err := xpath3.NewCompiler().Compile(src)
		require.NoError(t, err, "compile %q", src)
		return expr
	}

	t.Run("last outside grounding", func(t *testing.T) {
		plain := compile(t, "child::a[last()]")
		eqname := compile(t, "child::a[Q{"+fnNS+"}last()]")
		require.True(t, exprUsesLastOutsideGrounding(plain), "unprefixed last() must be detected")
		require.Equal(t,
			exprUsesLastOutsideGrounding(plain),
			exprUsesLastOutsideGrounding(eqname),
			"EQName Q{...}last() must classify same as unprefixed last()")
	})

	t.Run("position outside grounding", func(t *testing.T) {
		plain := compile(t, "child::a[position() = 1]")
		eqname := compile(t, "child::a[Q{"+fnNS+"}position() = 1]")
		require.True(t, exprUsesFunctionOutsideGrounding(plain, "position"),
			"unprefixed position() must be detected")
		require.Equal(t,
			exprUsesFunctionOutsideGrounding(plain, "position"),
			exprUsesFunctionOutsideGrounding(eqname, "position"),
			"EQName Q{...}position() must classify same as unprefixed position()")
	})

	t.Run("string consuming context item", func(t *testing.T) {
		plain := compile(t, "string(.)")
		eqname := compile(t, "Q{"+fnNS+"}string(.)")
		require.Positive(t, countStreamingDownwardSelections(nil, plain.AST()),
			"unprefixed string(.) must count as a consuming selection")
		require.Equal(t,
			countStreamingDownwardSelections(nil, plain.AST()),
			countStreamingDownwardSelections(nil, eqname.AST()),
			"EQName Q{...}string(.) must count same as unprefixed string(.)")
	})

	t.Run("data consuming context item", func(t *testing.T) {
		plain := compile(t, "data(.)")
		eqname := compile(t, "Q{"+fnNS+"}data(.)")
		require.Positive(t, countStreamingDownwardSelections(nil, plain.AST()),
			"unprefixed data(.) must count as a consuming selection")
		require.Equal(t,
			countStreamingDownwardSelections(nil, plain.AST()),
			countStreamingDownwardSelections(nil, eqname.AST()),
			"EQName Q{...}data(.) must count same as unprefixed data(.)")
	})

	t.Run("current-group consuming", func(t *testing.T) {
		plain := compile(t, "current-group()/child::b")
		eqname := compile(t, "Q{"+fnNS+"}current-group()/child::b")
		require.Positive(t, countCurrentGroupConsumingInExpr(plain.AST()),
			"unprefixed current-group() must be counted as consuming")
		require.Equal(t,
			countCurrentGroupConsumingInExpr(plain.AST()),
			countCurrentGroupConsumingInExpr(eqname.AST()),
			"EQName Q{...}current-group() must count same as unprefixed current-group()")
	})

	t.Run("snapshot grounding", func(t *testing.T) {
		plain := compile(t, "snapshot(child::a)")
		eqname := compile(t, "Q{"+fnNS+"}snapshot(child::a)")
		require.True(t, isGroundingExpr(plain.AST()),
			"unprefixed snapshot() must be grounding")
		require.Equal(t,
			isGroundingExpr(plain.AST()),
			isGroundingExpr(eqname.AST()),
			"EQName Q{...}snapshot() must classify same as unprefixed snapshot()")
	})

	t.Run("copy-of grounding", func(t *testing.T) {
		plain := compile(t, "copy-of(child::a)")
		eqname := compile(t, "Q{"+fnNS+"}copy-of(child::a)")
		require.True(t, isGroundingExpr(plain.AST()),
			"unprefixed copy-of() must be grounding")
		require.Equal(t,
			isGroundingExpr(plain.AST()),
			isGroundingExpr(eqname.AST()),
			"EQName Q{...}copy-of() must classify same as unprefixed copy-of()")
	})
}
