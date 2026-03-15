<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE:  Simple test of xsl:apply-templates with mode not found. -->
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="doc/a" mode="z"/>
    </out>
  </xsl:template>

  <xsl:template match="text()" mode="a">
    <xsl:text>mode-a:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

  <xsl:template match="text()">
    <xsl:text>no-mode:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

</xsl:stylesheet>
