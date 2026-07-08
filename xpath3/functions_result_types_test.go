package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFnResultTypes pins the spec-correct static/dynamic result types of
// fn:min/fn:max, fn:seconds-from-time/dateTime, and fn:namespace-uri
// (XPath 3.1 F&O §14.4.7/§14.4.8, §9.5.10/§9.5.14, §14.5.4).
func TestFnResultTypes(t *testing.T) {
	doc := mustParseXML(t, `<e xmlns="urn:x"/>`)

	// instanceOf cases assert the result type via the `instance of` operator.
	instanceOf := []struct {
		name string
		expr string
	}{
		// fn:min/fn:max retain the selected item's derived integer subtype.
		{"min retains positiveInteger", `min((xs:positiveInteger(123), xs:unsignedShort(124))) instance of xs:positiveInteger`},
		{"min integer common super", `min((xs:positiveInteger(123), xs:unsignedShort(124))) instance of xs:nonNegativeInteger`},
		{"max retains unsignedShort", `max((xs:positiveInteger(123), xs:unsignedShort(124))) instance of xs:unsignedShort`},
		{"min retains byte", `min((xs:byte(1), xs:byte(2))) instance of xs:byte`},
		{"max retains byte", `max((xs:byte(1), xs:byte(2))) instance of xs:byte`},
		// xs:anyURI is converted to xs:string when a plain string is present.
		{"min anyURI+string is string", `min((xs:anyURI("http://a.com"), "http://b.com")) instance of xs:string`},
		{"max anyURI+string is string", `max((xs:anyURI("http://c.com"), "http://b.com")) instance of xs:string`},
		// All-anyURI: no plain string, so anyURI is retained (not converted).
		{"min all anyURI stays anyURI", `min((xs:anyURI("http://a.com"), xs:anyURI("http://b.com"))) instance of xs:anyURI`},
		// A selected string subtype is retained (only anyURI is converted).
		{"min string subtype retained", `min((xs:token("http"), xs:anyURI("http://b.com"))) instance of xs:token`},
		// fn:seconds-from-time / fn:seconds-from-dateTime return xs:decimal.
		{"seconds-from-time is decimal", `seconds-from-time(xs:time("12:30:45.5")) instance of xs:decimal`},
		{"seconds-from-dateTime is decimal", `seconds-from-dateTime(xs:dateTime("2020-01-01T12:30:45")) instance of xs:decimal`},
		// fn:namespace-uri returns xs:anyURI.
		{"namespace-uri is anyURI", `namespace-uri(/*) instance of xs:anyURI`},
	}
	for _, tc := range instanceOf {
		t.Run(tc.name, func(t *testing.T) {
			r, err := evaluate(t.Context(), doc, tc.expr)
			require.NoError(t, err)
			b, ok := r.IsBoolean()
			require.True(t, ok, "expected boolean result for %q", tc.expr)
			require.True(t, b, "expected %q to be true", tc.expr)
		})
	}

	// Value checks: the selected item's value must be preserved unchanged.
	values := []struct {
		name string
		expr string
		want string
	}{
		{"min integer value", `string(min((xs:positiveInteger(123), xs:unsignedShort(124))))`, "123"},
		{"max integer value", `string(max((xs:positiveInteger(123), xs:unsignedShort(124))))`, "124"},
		{"min anyURI value", `min((xs:anyURI("http://a.com"), "http://b.com"))`, "http://a.com"},
		{"max anyURI value", `max((xs:anyURI("http://c.com"), "http://b.com"))`, "http://c.com"},
		{"seconds-from-time value", `string(seconds-from-time(xs:time("12:30:45.5")))`, "45.5"},
	}
	for _, tc := range values {
		t.Run(tc.name, func(t *testing.T) {
			got := evalString(t, tc.expr)
			require.Equal(t, tc.want, got)
		})
	}
}
