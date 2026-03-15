<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
    <xsl:for-each select="/*/thing">
      <item><xsl:value-of select="name()"/></item>
    </xsl:for-each>
    <xsl:on-empty>
      <p>There are no things</p>
    </xsl:on-empty>  
  </out>
</xsl:template>

</xsl:stylesheet>
