<?xml version="1.0"?> 

<!-- A test for xsl:sort -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:template match="doc">
    <out>
      <asc>
        <xsl:for-each select="item">
          <xsl:sort>
            <xsl:sequence select="."/>
          </xsl:sort>
          <xsl:copy-of select="."/>
          <xsl:text> </xsl:text>
        </xsl:for-each>
      </asc>
    </out>
  </xsl:template>
</xsl:stylesheet>
