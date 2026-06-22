package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// try/catch exercises parseCatchCode (parse-time) and catchCodeMatches
// (eval-time) across the catch-code forms: "*", "err:LOCAL", "Q{uri}local",
// "Q{uri}*", "*:LOCAL", and "err:*".
func TestTryCatch_CodeForms(t *testing.T) {
	const errNS = "http://www.w3.org/2005/xqt-errors"

	cases := []struct {
		expr   string
		expect string
	}{
		// wildcard catch.
		{`try { 1 div 0 } catch * { "caught" }`, "caught"},
		// specific error code via err: prefix.
		{`try { xs:integer("x") } catch err:FORG0001 { "forg" }`, "forg"},
		// err:* wildcard.
		{`try { xs:integer("x") } catch err:* { "anyerr" }`, "anyerr"},
		// Q{uri}local form.
		{`try { xs:integer("x") } catch Q{` + errNS + `}FORG0001 { "q" }`, "q"},
		// Q{uri}* form.
		{`try { xs:integer("x") } catch Q{` + errNS + `}* { "qstar" }`, "qstar"},
		// *:LOCAL form.
		{`try { xs:integer("x") } catch *:FORG0001 { "star" }`, "star"},
		// successful body returns its value (no catch).
		{`try { 1 + 1 } catch * { "x" }`, "2"},
		// error variable access inside catch.
		{`try { xs:integer("x") } catch * { $err:code }`, "err:FORG0001"},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		require.Equal(t, tc.expect, r.StringValue(), tc.expr)
	}

	// Non-matching catch code re-raises the original error.
	_, err := evaluate(t.Context(), nil, `try { xs:integer("x") } catch err:XPDY0002 { "nope" }`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}

func TestCastableExprForms(t *testing.T) {
	cases := []struct {
		expr   string
		expect bool
	}{
		{`"1" castable as xs:integer`, true},
		{`"x" castable as xs:integer`, false},
		{`"1.5" castable as xs:decimal`, true},
		{`"2023-06-22" castable as xs:date`, true},
		{`"notadate" castable as xs:date`, false},
		{`"true" castable as xs:boolean`, true},
		{`() castable as xs:integer?`, true},
		{`() castable as xs:integer`, false},
		{`"abc" castable as xs:NCName`, true},
		{`"a:b:c" castable as xs:NCName`, false},
		// multi-item sequence is never castable to a single-item type.
		{`(1, 2) castable as xs:integer`, false},
		// numeric union type.
		{`1 castable as xs:numeric`, true},
		{`"1.5" castable as xs:numeric`, true},
		{`"x" castable as xs:numeric`, false},
		// list types: whitespace-separated members.
		{`"a b c" castable as xs:NMTOKENS`, true},
		{`"a b" castable as xs:IDREFS`, true},
		{`"" castable as xs:NMTOKENS`, false},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		b, ok := r.IsBoolean()
		require.True(t, ok, tc.expr)
		require.Equal(t, tc.expect, b, tc.expr)
	}
}
