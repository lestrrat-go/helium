<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- A test for xsl:sort with numeric keys -->
<xsl:template match="doc">
    <out>
      <xsl:for-each select="num">
        <xsl:sort data-type="number"/>
        <xsl:value-of select="."/><xsl:text> </xsl:text>
      </xsl:for-each>
      <xsl:for-each select="num">
        <xsl:sort data-type="number" order="descending"/>
        <xsl:value-of select="."/><xsl:text> </xsl:text>
      </xsl:for-each>
    </out>
</xsl:template>
</xsl:stylesheet>