<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
    <xsl:for-each select="/*/item">
      <xsl:sort select="."/>
      <xsl:copy-of select="banana"/>
      <xsl:on-empty select="'apple'"/>
    </xsl:for-each>  
  </out>
</xsl:template>

</xsl:stylesheet>
