<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:msp="urn:us:gov:ic:msp"
                version="2.0">

<xsl:template match="/">
<o>
  <xsl:text>
Test using attribute nodes:
</xsl:text>
  <xsl:for-each select="//msp:Source/@title">
    <xsl:text/>Found source <xsl:number level="any" count="msp:Source"/>
    <xsl:text>
</xsl:text>
  </xsl:for-each>
  <xsl:text>
Test using element nodes:
</xsl:text>
  <xsl:for-each select="//msp:Source/@title">
    <xsl:for-each select="..">
      <xsl:text/>Found source <xsl:number level="any"/>
      <xsl:text>
</xsl:text>
    </xsl:for-each>
  </xsl:for-each>
</o>  
</xsl:template>

</xsl:stylesheet>