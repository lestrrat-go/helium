<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
    <xsl:for-each select="/*/item">
      <xsl:value-of select="name()"/>
    </xsl:for-each>
    <xsl:on-empty>
      <p>There are no items</p>
    </xsl:on-empty>  
  </out>
</xsl:template>

</xsl:stylesheet>
