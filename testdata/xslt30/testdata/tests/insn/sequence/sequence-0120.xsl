<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
  <?spec fo#func-unordered?>              
  <xsl:template match="/">
    <xsl:variable name="u" select="unordered((1,2,3))" />
    <z><xsl:value-of select="$u"/></z>
  </xsl:template>
</xsl:stylesheet>
