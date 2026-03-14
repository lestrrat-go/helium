<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test for ancestor::*[...][...] and index of ancestors. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="text()">
  <xsl:choose>
    <xsl:when test="ancestor::*[@new='true'][not(text())]">
      <xsl:value-of select="name(ancestor::*[3])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="name(ancestor::*[2])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="name(ancestor::*[1])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="."/><xsl:text>&#010;</xsl:text>
    </xsl:when>
    <xsl:when test="ancestor::*[2][@new]">
      <xsl:value-of select="name(ancestor::*[3])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="name(ancestor::*[2])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="name(ancestor::*[1])"/><xsl:text>/</xsl:text>
      <xsl:value-of select="."/><xsl:text>&#010;</xsl:text>
    </xsl:when>
    <xsl:otherwise>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>

</xsl:stylesheet>
