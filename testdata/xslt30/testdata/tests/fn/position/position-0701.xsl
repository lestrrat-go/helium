<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: test position() with and without sorting -->

<xsl:template match="doc">
   <out>
      <xsl:for-each select="foo/a/num/@val">
        <xsl:value-of select="position()"/><xsl:value-of select="."/>
        <xsl:text> </xsl:text>
      </xsl:for-each>
      
      <xsl:for-each select="foo/a/num/@val">
        <xsl:sort select="."/>
        <xsl:value-of select="position()"/><xsl:value-of select="."/>
        <xsl:text> </xsl:text>
      </xsl:for-each>
   </out>
</xsl:template>
 
</xsl:stylesheet>
