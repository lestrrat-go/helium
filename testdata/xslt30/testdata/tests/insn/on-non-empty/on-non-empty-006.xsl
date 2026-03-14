<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>
    <xsl:on-non-empty select="21"/>
    <xsl:copy-of select="/comment()"/>   
    <xsl:on-non-empty select="23"/>
    <xsl:copy-of select="/processing-instruction()"/> 
  </out>
</xsl:template>

</xsl:stylesheet>
