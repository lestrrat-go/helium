<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:param name="z">
  <xsl:copy-of select="/unicorn"/>
</xsl:param>  

<xsl:template match="/">
  <out>
    <xsl:copy-of select="$z"/>
    <xsl:on-non-empty>
      <xsl:sequence select="23"/>
      <xsl:fallback>18.2</xsl:fallback>
    </xsl:on-non-empty>  
    <xsl:fallback>93.7</xsl:fallback>
  </out>
</xsl:template>

</xsl:stylesheet>
