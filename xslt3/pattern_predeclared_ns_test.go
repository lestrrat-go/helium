package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestPatternPredeclaredFunctionNamespace verifies that match patterns may use
// the XPath 3.0 predeclared namespace prefixes (fn:, math:, map:, ...) without
// an explicit xmlns declaration in the stylesheet. The static context
// predeclares these bindings per XPath 3.0 / XSLT 3.0.
func TestPatternPredeclaredFunctionNamespace(t *testing.T) {
	t.Parallel()

	const tmpl = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"%s>
  <xsl:template match="%s"><out/></xsl:template>
</xsl:stylesheet>`

	tests := []struct {
		name    string
		extraNS string
		match   string
		wantErr bool
	}{
		{name: "fn-id-predeclared", match: "fn:id('x')"},
		{name: "id-unprefixed", match: "id('x')"},
		{
			name:    "fn-id-explicit-xmlns",
			extraNS: ` xmlns:fn="http://www.w3.org/2005/xpath-functions"`,
			match:   "fn:id('x')",
		},
		{name: "math-predeclared-predicate", match: "*[math:pi() > 3]"},
		{name: "map-predeclared-predicate", match: "*[map:size(map{}) = 0]"},
		{
			name:    "unknown-prefix-fails",
			match:   "bogus:id('x')",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := strings.Replace(tmpl, "%s", tc.extraNS, 1)
			src = strings.Replace(src, "%s", tc.match, 1)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
			if tc.wantErr {
				// An undeclared prefix must still be rejected at compile time
				// (XPST0081 at prefix resolution / XPST0017 at function check).
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
