<?xml version="1.0"?> 

<!-- A test for xsl:sort. Alphabetic strings restricted to lower-case a-z to avoid collation differences -->
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:template match="doc">
    <out>
    <asc>
      <xsl:for-each select="item">
        <xsl:sort/>
        <xsl:copy-of select="."/><xsl:text> </xsl:text>
      </xsl:for-each>
    </asc>
    <desc>
      <xsl:for-each select="item">
        <xsl:sort order="descending"/>
        <xsl:copy-of select="."/><xsl:text> </xsl:text>
      </xsl:for-each>
    </desc>
    </out>
  </xsl:template>
</xsl:stylesheet>