package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// instance of against typed map / array / function / document-node tests
// exercises matchesItemType and isItemTypeSubtype's typed branches (key/value,
// member, param/return, inner node tests).
func TestInstanceOf_TypedItemTests(t *testing.T) {
	const xml = `<root><child/></root>`
	doc := mustParseXML(t, xml)
	root := doc.DocumentElement()

	cases := []struct {
		expr   string
		expect bool
	}{
		{`map { "a": 1 } instance of map(xs:string, xs:integer)`, true},
		{`map { "a": "v" } instance of map(xs:string, xs:integer)`, false},
		{`[1, 2] instance of array(xs:integer)`, true},
		{`["x"] instance of array(xs:integer)`, false},
		{`fn:abs#1 instance of function(*)`, true},
		{`function($x as xs:integer) as xs:integer { $x } instance of function(xs:integer) as xs:integer`, true},
		{`. instance of document-node()`, false},
		{`child::child instance of element(child)`, true},
		{`child::child instance of element(other)`, false},
		{`child::child instance of element(*)`, true},
		// typed function: arity mismatch -> false.
		{`fn:abs#1 instance of function(xs:double, xs:double) as xs:double`, false},
		// inline function with matching param/return.
		{`function($x as xs:integer) as xs:integer { $x } instance of function(xs:integer) as xs:integer`, true},
		// map as function(K) as V.
		{`map { "a": 1 } instance of function(xs:string) as xs:integer?`, true},
		{`map { "a": 1 } instance of function(xs:anyAtomicType) as item()*`, true},
		// array as function(xs:integer) as V.
		{`[1, 2] instance of function(xs:integer) as item()*`, true},
		// map with value type mismatch.
		{`map { "a": "s" } instance of map(xs:string, xs:integer)`, false},
		// empty map matches any typed map.
		{`map { } instance of map(xs:string, xs:integer)`, true},
		// array member type.
		{`[1, 2] instance of array(xs:integer)`, true},
		{`["s"] instance of array(xs:integer)`, false},
		{`[1, 2] instance of array(*)`, true},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := evaluate(t.Context(), root, tc.expr)
			require.NoError(t, err, tc.expr)
			b, ok := r.IsBoolean()
			require.True(t, ok, tc.expr)
			require.Equal(t, tc.expect, b, tc.expr)
		})
	}

	// document-node test against the actual document node.
	r, err := evaluate(t.Context(), doc, `. instance of document-node()`)
	require.NoError(t, err)
	b, ok := r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	r, err = evaluate(t.Context(), doc, `. instance of document-node(element(root))`)
	require.NoError(t, err)
	b, ok = r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)
}
