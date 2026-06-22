package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// string() over the full XSD atomic type space drives atomicToString's
// per-type branches (gYear/gMonth/.., duration variants, base64/hex binary,
// QName, integer subtypes, date/time).
func TestAtomicToString_AllTypes(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		// gregorian types.
		{`string(xs:gYear("2020"))`, "2020"},
		{`string(xs:gMonth("--06"))`, "--06"},
		{`string(xs:gDay("---22"))`, "---22"},
		{`string(xs:gYearMonth("2020-06"))`, "2020-06"},
		{`string(xs:gMonthDay("--06-22"))`, "--06-22"},
		// date / time / dateTime.
		{`string(xs:date("2020-06-22"))`, "2020-06-22"},
		{`string(xs:time("12:34:56"))`, "12:34:56"},
		{`string(xs:dateTime("2020-06-22T12:34:56"))`, "2020-06-22T12:34:56"},
		// durations.
		{`string(xs:duration("P1Y2M3DT4H5M6S"))`, "P1Y2M3DT4H5M6S"},
		{`string(xs:dayTimeDuration("P1DT2H"))`, "P1DT2H"},
		{`string(xs:yearMonthDuration("P1Y2M"))`, "P1Y2M"},
		// binary.
		{`string(xs:base64Binary("aGk="))`, "aGk="},
		{`string(xs:hexBinary("48656C6C6F"))`, "48656C6C6F"},
		// integer subtypes.
		{`string(xs:byte("12"))`, "12"},
		{`string(xs:unsignedShort("300"))`, "300"},
		{`string(xs:positiveInteger("5"))`, "5"},
		// decimal / double / float / boolean / string-ish.
		{`string(xs:decimal("1.250"))`, "1.25"},
		{`string(xs:double("2.5"))`, "2.5"},
		{`string(xs:float("3.5"))`, "3.5"},
		{`string(true())`, wantTrue},
		{`string(xs:anyURI("http://x"))`, "http://x"},
		{`string(xs:NCName("abc"))`, "abc"},
		{`string(xs:token("  a  b  "))`, "a b"},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		require.Equal(t, tc.want, r.StringValue(), tc.expr)
	}

	// QName via fn:QName, then string().
	r, err := evaluate(t.Context(), nil, `string(fn:QName("http://x", "p:local"))`)
	require.NoError(t, err)
	require.Equal(t, "p:local", r.StringValue())
}
