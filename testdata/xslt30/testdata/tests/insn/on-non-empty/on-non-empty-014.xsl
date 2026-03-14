<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:param name="z">
  <xsl:copy-of select="/unicorn"/>
</xsl:param>  

<xsl:template match="/">
  <out>
    <xsl:copy-of select="$z"/>
    <xsl:on-non-empty>NON-EMPTY</xsl:on-non-empty>
    <xsl:on-empty>EMPTY</xsl:on-empty>    
  </out>
</xsl:template>

</xsl:stylesheet>
