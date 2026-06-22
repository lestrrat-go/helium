package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// instanceOf compiles and evaluates an `instance of` expression against a
// context node, returning its boolean result. It exercises matchesItemType /
// isItemTypeSubtype / matchNodeTest across many item-type shapes.
func instanceOf(t *testing.T, ctxXML, expr string) bool {
	t.Helper()
	doc := mustParseXML(t, ctxXML)
	root := doc.DocumentElement()
	r, err := evaluate(t.Context(), root, expr)
	require.NoError(t, err, expr)
	b, ok := r.IsBoolean()
	require.True(t, ok, "expected boolean for %q", expr)
	return b
}

func TestInstanceOf_ItemTypes(t *testing.T) {
	const xml = `<root att="v"><child/><!--c--><?pi data?></root>`

	cases := []struct {
		expr   string
		expect bool
	}{
		// atomic types & numeric hierarchy.
		{`1 instance of xs:integer`, true},
		{`1 instance of xs:decimal`, true},
		{`1 instance of xs:double`, false},
		{`1.5 instance of xs:decimal`, true},
		{`"x" instance of xs:string`, true},
		{`"x" instance of xs:integer`, false},
		{`true() instance of xs:boolean`, true},
		// cardinality.
		{`() instance of item()*`, true},
		{`() instance of item()`, false},
		{`() instance of item()?`, true},
		{`(1, 2) instance of xs:integer+`, true},
		{`(1, 2) instance of xs:integer`, false},
		{`1 instance of item()`, true},
		// node kind tests on the context tree.
		{`. instance of element()`, true},
		{`. instance of node()`, true},
		{`. instance of attribute()`, false},
		{`child::child[1] instance of element()`, true},
		{`@att instance of attribute()`, true},
		{`@att instance of node()`, true},
		{`comment() instance of comment()`, true},
		{`processing-instruction() instance of processing-instruction()`, true},
		{`. instance of element(root)`, true},
		{`. instance of element(other)`, false},
		// function / map / array item types.
		{`fn:abs#1 instance of function(*)`, true},
		{`map { "a": 1 } instance of map(*)`, true},
		{`[1, 2] instance of array(*)`, true},
		{`map { "a": 1 } instance of array(*)`, false},
		{`function($x) { $x } instance of function(*)`, true},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			require.Equal(t, tc.expect, instanceOf(t, xml, tc.expr))
		})
	}
}

func TestTreatAs(t *testing.T) {
	// treat as success returns the value; failure raises XPDY0050/XPTY0004.
	r, err := evaluate(t.Context(), nil, `1 treat as xs:integer`)
	require.NoError(t, err)
	n, ok := r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(1), n)

	_, err = evaluate(t.Context(), nil, `"x" treat as xs:integer`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}
