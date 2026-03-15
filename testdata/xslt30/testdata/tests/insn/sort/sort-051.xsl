<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test xsl:sort using a sequence constructor -->

<xsl:template match="/">
  <xsl:variable name="numnodes" select="doc/num" />
  <out>
    Descending order....
    <xsl:for-each select="$numnodes">
      <xsl:sort order="descending">
        <xsl:apply-templates select="." mode="sk"/>
      </xsl:sort>
      <xsl:value-of select="."/><xsl:text> </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="num" mode="sk">
  <xsl:sequence select="round(.)"/>
</xsl:template>

</xsl:stylesheet>