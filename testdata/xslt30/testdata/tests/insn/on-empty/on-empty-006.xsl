<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>
    <xsl:copy-of select="/comment()"/>   
    <xsl:copy-of select="/processing-instruction()"/>
    <xsl:on-empty select="21, 23"/> 
  </out>
</xsl:template>

</xsl:stylesheet>
