<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Try 'ancestor-or-self::*' after walking down to it. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="/far-north/north/near-north/center/ancestor-or-self::*"/>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
</xsl:template>

</xsl:stylesheet>