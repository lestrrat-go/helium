<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Ensure that preceding::node() doesn't get namespace or attribute nodes -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//inner/preceding::node()">
      <xsl:value-of select="name(.)"/>
      <xsl:text> (</xsl:text>
      <xsl:value-of select="."/>
      <xsl:text>), </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
