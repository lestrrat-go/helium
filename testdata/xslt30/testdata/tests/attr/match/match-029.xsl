<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<?spec xslt#patterns?>
  <!-- Purpose: // at start of match pattern should not affect selection of nodes. -->

<xsl:output method="xml" encoding="UTF-8" />

<xsl:template match="/">
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="//X" mode="unprefixed" />
    <xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="//X" mode="prefixed" />
  </out>
</xsl:template>

<xsl:template match="X" mode="unprefixed">
  <foundX><xsl:copy-of select="@level"/></foundX>
</xsl:template>

<xsl:template match="//X" mode="prefixed">
  <found-X><xsl:copy-of select="@level"/></found-X>
</xsl:template>

</xsl:stylesheet>