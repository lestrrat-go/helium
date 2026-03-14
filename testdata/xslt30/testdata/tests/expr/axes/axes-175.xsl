<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Compound test where we bounce "horizontally" across the tree, using positions. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="preceding-sibling::*[2]/following-sibling::*[4]/preceding-sibling::*[5]/following-sibling::*[4]/following-sibling::*[2]"/>
      <!-- Nodes in order: west, east, far-west, near-east, far-east -->
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
