<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#eval_context?>
    <!-- Purpose: Test that for-each doesn't care about current node -->

<!-- Collect a node-set, outside any template -->
<xsl:param name="all" select="//OL"/>

<xsl:template match="/">
  <out>
    <xsl:for-each select="//comment()">
      <xsl:value-of select="."/><xsl:text>
</xsl:text>
    </xsl:for-each>
    <xsl:apply-templates select="OL//tag"/>
  </out>
</xsl:template>

<xsl:template match="tag">
  <xsl:text>Found the tag...
</xsl:text>
  <xsl:for-each select="$all/LI">
    <xsl:value-of select="."/><xsl:text>
</xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
