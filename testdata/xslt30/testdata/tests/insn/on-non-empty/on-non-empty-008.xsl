<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
    <xsl:for-each select="/*/thing">
      <item><xsl:value-of select="name()"/></item>
    </xsl:for-each>
    <xsl:on-non-empty>
      <p>There are some things</p>
    </xsl:on-non-empty>  
  </out>
</xsl:template>

</xsl:stylesheet>
