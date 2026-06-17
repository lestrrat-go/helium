package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Variables declared inside xsl:try (or its xsl:catch) must not leak into the
// surrounding scope. After the instruction completes, an outer variable of the
// same name must still resolve to its outer value.
func TestTryDoesNotLeakVariables(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			// Try body succeeds; inner $x must not shadow outer $x afterward.
			name: "success",
			body: `
      <xsl:try>
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
		{
			// Try body fails; catch runs and declares $x, which must not leak.
			name: "catch",
			body: `
      <xsl:try>
        <xsl:sequence select="1 div xs:integer('not-a-number')"/>
        <xsl:catch>
          <xsl:variable name="x" select="'inner'"/>
        </xsl:catch>
      </xsl:try>`,
		},
		{
			// rollback-output="no" with a successful try body.
			name: "no-rollback-success",
			body: `
      <xsl:try rollback-output="no">
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="x" select="'outer'"/>
    <out>`+tc.body+`<xsl:value-of select="$x"/></out>
  </xsl:template>
</xsl:stylesheet>`)

			result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, ">outer<")
			require.NotContains(t, result, ">inner<")
		})
	}
}
