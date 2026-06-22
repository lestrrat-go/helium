package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Casting to integer-derived and string-derived XSD types exercises
// CastAtomic's integer-range checking (integerTypeRange / checkBigIntRange) and
// validateStringDerivedType, reached through the `cast as` operator.
func TestCast_IntegerSubtypeRange(t *testing.T) {
	ok := []string{
		`"100" cast as xs:byte`,
		`"100" cast as xs:short`,
		`"100" cast as xs:int`,
		`"100" cast as xs:long`,
		`"100" cast as xs:unsignedByte`,
		`"100" cast as xs:unsignedShort`,
		`"100" cast as xs:unsignedInt`,
		`"100" cast as xs:unsignedLong`,
		`"1" cast as xs:positiveInteger`,
		`"0" cast as xs:nonNegativeInteger`,
		`"0" cast as xs:nonPositiveInteger`,
		`"-1" cast as xs:negativeInteger`,
		// numeric sources, exercising the numeric->subtype path.
		`100 cast as xs:byte`,
		`100 cast as xs:int`,
	}
	for _, e := range ok {
		_, err := evaluate(t.Context(), nil, e)
		require.NoError(t, err, e)
	}

	outOfRange := []string{
		`"1000" cast as xs:byte`,       // > 127
		`"-1" cast as xs:unsignedByte`, // < 0
		`"0" cast as xs:positiveInteger`,
		`"1" cast as xs:negativeInteger`,
		`"1" cast as xs:nonPositiveInteger`,
		`"-1" cast as xs:nonNegativeInteger`,
		`"100000" cast as xs:short`, // > 32767
		`"notanint" cast as xs:byte`, // unparseable
	}
	for _, e := range outOfRange {
		_, err := evaluate(t.Context(), nil, e)
		require.Error(t, err, e)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr, e)
	}
}

func TestCast_StringDerivedTypes(t *testing.T) {
	ok := []string{
		`"abc" cast as xs:NCName`,
		`"abc" cast as xs:Name`,
		`"abc" cast as xs:NMTOKEN`,
		`"a b c" cast as xs:NMTOKENS`,
		`"abc" cast as xs:ID`,
		`"abc" cast as xs:IDREF`,
		`"a b" cast as xs:IDREFS`,
		`"abc" cast as xs:ENTITY`,
	}
	for _, e := range ok {
		_, err := evaluate(t.Context(), nil, e)
		require.NoError(t, err, e)
	}

	invalid := []string{
		`"a:b:c" cast as xs:NCName`, // colon not allowed in NCName
		`"" cast as xs:NMTOKENS`,    // empty list
		`"" cast as xs:IDREFS`,      // empty list
	}
	for _, e := range invalid {
		_, err := evaluate(t.Context(), nil, e)
		require.Error(t, err, e)
	}
}

// cast as xs:numeric (union) and xs:QName exercise the special-cased target
// branches in evalCastExpr.
func TestCast_UnionAndQName(t *testing.T) {
	// numeric: already-numeric returns as-is.
	r, err := evaluate(t.Context(), nil, `1 cast as xs:numeric instance of xs:integer`)
	require.NoError(t, err)
	b, ok := r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	// numeric: string casts to double.
	r, err = evaluate(t.Context(), nil, `"2.5" cast as xs:numeric instance of xs:double`)
	require.NoError(t, err)
	b, ok = r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	// QName cast from string.
	r, err = evaluate(t.Context(), nil, `string(fn:local-name-from-QName("foo" cast as xs:QName))`)
	require.NoError(t, err)
	require.Equal(t, "foo", r.StringValue())

	// Empty-sequence with ? cast modifier yields empty.
	r, err = evaluate(t.Context(), nil, `() cast as xs:integer?`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())

	// Multi-item cast -> XPTY0004.
	_, err = evaluate(t.Context(), nil, `(1, 2) cast as xs:integer`)
	require.Error(t, err)
}

// fn:string-to-codepoints / xs:double special values exercise atomicToString and
// formatting branches through the public string conversion surface.
func TestStringValueConversions(t *testing.T) {
	cases := map[string]string{
		`string(1)`:           "1",
		`string(1.5)`:         "1.5",
		`string(true())`:      "true",
		`string(xs:double("INF"))`:  "INF",
		`string(xs:double("-INF"))`: "-INF",
		`string(xs:double("NaN"))`:  "NaN",
	}
	for expr, want := range cases {
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.Equal(t, want, r.StringValue(), expr)
	}
}
