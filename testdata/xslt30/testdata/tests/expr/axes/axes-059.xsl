
  <!-- PURPOSE: Test root() function. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="/">

    <xsl:variable name="t">
      <element-node/>
    </xsl:variable>
    <xsl:variable name="u">text-node</xsl:variable>
    <out>
      <xsl:value-of select="(/) is root()"/>
      <xsl:value-of select="(/) is root(//b)"/>
      <xsl:value-of select="$t is root($t/element-node)"/>
      <xsl:value-of select="$u is root($u/text())"/>
    </out>
  </xsl:template>


</xsl:stylesheet>
