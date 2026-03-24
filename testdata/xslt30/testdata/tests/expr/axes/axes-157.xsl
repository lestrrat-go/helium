<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
  <!-- Section: 2.2 -->
  <!-- Purpose: Test for '//' followed by 'descendant::*' (i.e., all children and below). -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//near-north">
      <xsl:apply-templates select="center//descendant::*"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
</xsl:template>

</xsl:stylesheet>
