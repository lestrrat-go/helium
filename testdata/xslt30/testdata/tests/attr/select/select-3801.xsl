<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the ancestor-or-self axis -->
  <!-- Heavily modified from Expression004 in NIST suite -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//child"><!-- 8 different ones -->
      <xsl:apply-templates select="ancestor-or-self::sub | ancestor-or-self::sub-sub"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="@pos"/><xsl:text>,</xsl:text>
</xsl:template>

</xsl:stylesheet>

