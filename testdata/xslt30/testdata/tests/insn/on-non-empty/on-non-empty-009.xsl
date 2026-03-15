<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
    <xsl:for-each select="/*/item">
      <xsl:value-of select="name()"/>
    </xsl:for-each>
    <xsl:on-non-empty>
      <p>There are some items</p>
    </xsl:on-non-empty>  
  </out>
</xsl:template>

</xsl:stylesheet>
