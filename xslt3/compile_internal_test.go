package xslt3

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// splitOriginIDs compiles a stylesheet source and returns, per mode, the
// non-zero splitOriginID values carried by union-split match templates.
func splitOriginIDs(t *testing.T, src string) []int64 {
	t.Helper()
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	ss, err := compile(context.Background(), doc, &compileConfig{})
	require.NoError(t, err)
	var ids []int64
	for _, tmpls := range ss.modeTemplates {
		for _, tmpl := range tmpls {
			if tmpl.splitOriginID != 0 {
				ids = append(ids, tmpl.splitOriginID)
			}
		}
	}
	return ids
}

// UNRES-8: the union-split origin id must be unique across the whole
// compilation, not merely within a single stylesheet/package compile. A
// per-stylesheet counter restarts at 0 for every compile(), so two independent
// compiles would hand their first union split the same id — and under
// xsl:use-package those splits can be merged into a single mode list where the
// on-multiple-match conflict check treats equal ids as one rule, wrongly
// suppressing a genuine cross-package XTDE0540. A process-global counter
// guarantees splits from different compiles never collide.
func TestSplitOriginIDUniqueAcrossCompiles(t *testing.T) {
	const src = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a | b"/>
</xsl:stylesheet>`

	idsA := splitOriginIDs(t, src)
	idsB := splitOriginIDs(t, src)
	require.NotEmpty(t, idsA, "union pattern should produce at least one split")
	require.NotEmpty(t, idsB)

	// Within one compile, every branch of the SAME union rule shares one id.
	for _, id := range idsA[1:] {
		require.Equal(t, idsA[0], id, "branches of one union rule must share an origin id")
	}
	for _, id := range idsB[1:] {
		require.Equal(t, idsB[0], id, "branches of one union rule must share an origin id")
	}

	// Across two independent compiles, the ids must NOT collide. With a
	// per-stylesheet counter both would be 1; with a global counter they differ.
	require.NotEqual(t, idsA[0], idsB[0],
		"split origin ids from independent compiles must be distinct so cross-package conflicts are not suppressed")
}

// Distinct union rules within a single compile must also receive distinct
// origin ids (one id per union rule, shared only among its own branches).
func TestSplitOriginIDDistinctPerUnionRule(t *testing.T) {
	const src = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a | b" mode="m1"/>
  <xsl:template match="c | d" mode="m2"/>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	ss, err := compile(context.Background(), doc, &compileConfig{})
	require.NoError(t, err)

	idByMode := map[string]int64{}
	for mode, tmpls := range ss.modeTemplates {
		for _, tmpl := range tmpls {
			if tmpl.splitOriginID == 0 {
				continue
			}
			if existing, ok := idByMode[mode]; ok {
				require.Equal(t, existing, tmpl.splitOriginID,
					"branches of the same union rule (mode %q) must share an id", mode)
				continue
			}
			idByMode[mode] = tmpl.splitOriginID
		}
	}
	require.Contains(t, idByMode, "m1")
	require.Contains(t, idByMode, "m2")
	require.NotEqual(t, idByMode["m1"], idByMode["m2"],
		"different union rules must receive distinct origin ids")
}

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
// lexicon.StreamFnLocalName helper, comparing no raw names.
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

// TestXSLTFunctionAritiesMatchRegistry guards xsltFunctionArities (the static
// table consulted by compile-time pattern validation) against drift from the
// runtime registries. Every fn:-namespace XSLT function registered at runtime
// must appear in the static table with identical min/max arity, and every
// static entry must be backed by a runtime registration.
func TestXSLTFunctionAritiesMatchRegistry(t *testing.T) {
	ec := &execContext{stylesheet: &Stylesheet{}}

	// Collect the runtime fn:-namespace XSLT functions from both registries:
	// the local-name map (xsltFunctions) lives in the fn: namespace, and the
	// QName map (xsltFunctionsNS) registers fn:-namespace entries explicitly.
	runtime := map[string][2]int{}
	for name, fn := range ec.xsltFunctions() {
		runtime[name] = [2]int{fn.MinArity(), fn.MaxArity()}
	}
	for qn, fn := range ec.xsltFunctionsNS() {
		if qn.URI != xpath3.NSFn {
			continue // skip schema constructors and other namespaces
		}
		// function-lookup is an XPath built-in (only specially registered in a
		// package context); it is not an XSLT-defined function for patterns.
		if qn.Name == "function-lookup" {
			continue
		}
		runtime[qn.Name] = [2]int{fn.MinArity(), fn.MaxArity()}
	}

	for name, bounds := range runtime {
		got, ok := xsltFunctionArities[name]
		require.Truef(t, ok, "xsltFunctionArities missing runtime fn:%s", name)
		require.Equalf(t, bounds, got, "arity mismatch for fn:%s", name)
	}
	for name := range xsltFunctionArities {
		_, ok := runtime[name]
		require.Truef(t, ok, "xsltFunctionArities has fn:%s with no runtime registration", name)
	}
}
