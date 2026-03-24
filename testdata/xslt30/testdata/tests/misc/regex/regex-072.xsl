<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  >
  <xsl:template match="/" name="main">
    <out><xsl:value-of select="matches('','((a|b)cd?)')"/></out>
  </xsl:template>
</xsl:stylesheet>