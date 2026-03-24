package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestCallTemplateCoercesParamsToDeclaredTypes(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="show">
        <xsl:with-param name="a" select="xs:untypedAtomic('FOO')"/>
        <xsl:with-param name="c" select="xs:untypedAtomic('50')"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <xsl:param name="a" as="xs:string"/>
    <xsl:param name="c" as="xs:double"/>
    <q a="{$a instance of xs:string}" c="{$c instance of xs:double}"/>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, `a="true"`)
	require.Contains(t, result, `c="true"`)
}
