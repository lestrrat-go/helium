package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// ENG-001: a template rule matching an ATOMIC item with a required param
// supplied via xsl:with-param must succeed (no XTDE0700) and the param value
// must be visible in the template body.
func TestApplyTemplatesAtomicRequiredParamSupplied(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="1">
        <xsl:with-param name="p" select="'supplied'"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match=".[. instance of xs:integer]">
    <xsl:param name="p" as="xs:string" required="yes"/>
    <got><xsl:value-of select="$p"/></got>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, "<got>supplied</got>")
}

// ENG-002: a caller-supplied empty sequence () for a param with
// as="xs:string" (cardinality exactly-one) must raise XTTE0590, not pass
// silently.
func TestCallTemplateEmptySequenceForExactlyOneParamFails(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="show">
        <xsl:with-param name="p" select="()"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <xsl:param name="p" as="xs:string"/>
    <q><xsl:value-of select="$p"/></q>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	_, err := xslt3.TransformString(t.Context(), source, ss)
	require.Error(t, err)
	require.Contains(t, err.Error(), "XTTE0590")
}
