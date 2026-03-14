<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE:  Simple test of xsl:apply-templates with no mode. -->
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="doc/a"/>
    </out>
  </xsl:template>

  <xsl:template match="a" mode="a">
    <xsl:text>mode-a:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

  <xsl:template match="a">
    <xsl:text>no-mode:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

</xsl:stylesheet>
