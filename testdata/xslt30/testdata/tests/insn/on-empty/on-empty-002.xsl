<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>
    <xsl:copy-of select="/in/non-existent"/>
    <xsl:on-empty select="23"/>
  </out>
</xsl:template>

</xsl:stylesheet>
