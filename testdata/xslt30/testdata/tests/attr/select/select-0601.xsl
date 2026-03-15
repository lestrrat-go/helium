<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#for-each?>
    <!-- Purpose: Test nesting of for-each.-->

<xsl:template match="/doc">
  <out>
    <xsl:for-each select="a">
      Inside an A node
      <xsl:for-each select="b">
        <xsl:for-each select="c">
          <xsl:for-each select="d">|
            <xsl:for-each select="e"><xsl:value-of select="title"/>,</xsl:for-each>
          </xsl:for-each>Finished C node: <xsl:value-of select="title"/>
        </xsl:for-each>
      </xsl:for-each>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
