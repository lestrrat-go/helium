package xpath1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// TestBuiltinArgErrors exercises the argument-count validation error branches
// of the built-in XPath functions, which are otherwise unreached by the
// happy-path tests.
func TestBuiltinArgErrors(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)

	// Each expression compiles fine but must fail at evaluation time because
	// the function is called with the wrong number of arguments.
	exprs := []string{
		"last(1)",
		"position(1)",
		"count()",
		"count(1, 2)",
		"id()",
		"id(1, 2)",
		"string(1, 2)",
		"concat('a')",
		"starts-with('a')",
		"contains('a')",
		"substring-before('a')",
		"substring-after('a')",
		"substring('a')",
		"substring('a', 1, 2, 3)",
		"string-length('a', 'b')",
		"normalize-space('a', 'b')",
		"translate('a', 'b')",
		"boolean()",
		"not()",
		"true(1)",
		"false(1)",
		"lang()",
		"number(1, 2)",
		"sum()",
		"floor()",
		"ceiling()",
		"round()",
		"local-name(1, 2)",
		"name(1, 2)",
		"namespace-uri(1, 2)",
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			compiled, err := xpath1.Compile(expr)
			require.NoError(t, err)
			_, err = compiled.Evaluate(t.Context(), doc)
			require.Error(t, err, "expected arg-count error for %q", expr)
		})
	}
}

// TestCountNonNodeSet covers the count() type-check error branch.
func TestCountNonNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	_, err := xpath1.Evaluate(t.Context(), doc, "count('not a node-set')")
	require.Error(t, err)
}

// TestSumNonNodeSet covers the sum() type-check error branch.
func TestSumNonNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	_, err := xpath1.Evaluate(t.Context(), doc, "sum(1)")
	require.Error(t, err)
}

// TestNodeArgNotNodeSet covers the non-node-set argument branch of
// nodeArgOrContext (used by name/local-name/namespace-uri).
func TestNodeArgNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	for _, expr := range []string{
		"name('x')",
		"local-name(1)",
		"namespace-uri(true())",
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := xpath1.Evaluate(t.Context(), doc, expr)
			require.Error(t, err)
		})
	}
}
