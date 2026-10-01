package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestXPathEvaluatorCache covers the cases where the cached XPath evaluator
// must be rebuilt or kept apart: a change of the visible local variables, the
// XPath 1.0 compatibility-mode variant next to the normal one, and the
// variables xsl:evaluate adds for its own expression.
func TestXPathEvaluatorCache(t *testing.T) {
	t.Run("variable scopes", func(t *testing.T) {
		const ss = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="g" select="'G'"/>
  <xsl:template match="/">
    <out>
      <first><xsl:value-of select="$g"/></first>
      <xsl:variable name="a" select="'A1'"/>
      <second><xsl:value-of select="$g, $a"/></second>
      <xsl:for-each select="1 to 2">
        <xsl:variable name="b" select="concat('B', .)"/>
        <loop><xsl:value-of select="$a, $b, position(), last()"/></loop>
      </xsl:for-each>
      <xsl:call-template name="t">
        <xsl:with-param name="p" select="'P'"/>
      </xsl:call-template>
      <after><xsl:value-of select="$a"/></after>
    </out>
  </xsl:template>
  <xsl:template name="t">
    <xsl:param name="p"/>
    <xsl:variable name="a" select="'A2'"/>
    <called><xsl:value-of select="$p, $a, $g"/></called>
  </xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "<first>G</first>")
		require.Contains(t, out, "<second>G A1</second>")
		require.Contains(t, out, "<loop>A1 B1 1 2</loop><loop>A1 B2 2 2</loop>")
		require.Contains(t, out, "<called>P A2 G</called>")
		require.Contains(t, out, "<after>A1</after>")
	})

	t.Run("compatibility mode alternates with normal mode", func(t *testing.T) {
		// '10' < '9' compares strings in XPath 3.1 (true) but converts both
		// operands to numbers in XPath 1.0 compatibility mode (false).
		const ss = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <c1><xsl:value-of version="1.0" select="'10' &lt; '9'"/></c1>
      <c3><xsl:value-of select="'10' &lt; '9'"/></c3>
      <c1b><xsl:value-of version="1.0" select="'10' &lt; '9'"/></c1b>
      <c3b><xsl:value-of select="'10' &lt; '9'"/></c3b>
    </out>
  </xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "<c1>false</c1><c3>true</c3><c1b>false</c1b><c3b>true</c3b>")
	})

	t.Run("xsl:evaluate parameters stay local to the call", func(t *testing.T) {
		const ss = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="g" select="'G'"/>
  <xsl:template match="/">
    <out>
      <one><xsl:evaluate xpath="'$p'"><xsl:with-param name="p" select="'P1'"/></xsl:evaluate></one>
      <two><xsl:try><xsl:evaluate xpath="'$p'"/><xsl:catch>no-p</xsl:catch></xsl:try></two>
      <three><xsl:value-of select="$g"/></three>
    </out>
  </xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "<one>P1</one><two>no-p</two><three>G</three>")
	})
}
