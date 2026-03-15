<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test for node() in match patterns. Default axis is child. -->

<xsl:template match="/doc">
  <out>
    <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
  </out>
</xsl:template>

<xsl:template match="node()">
  <xsl:text>N-</xsl:text>
  <xsl:choose>
    <xsl:when test="name(.)!=''">
      <xsl:text>named:</xsl:text><xsl:value-of select="name()"/><xsl:text>...</xsl:text>
    </xsl:when>
    <xsl:when test=".!=''">
      <xsl:text>content:</xsl:text><xsl:value-of select="."/>
    </xsl:when>
    <xsl:otherwise>
      <xsl:text>UNKNOWN</xsl:text>
    </xsl:otherwise>
  </xsl:choose>
  <xsl:apply-templates select="@*"/>
  <xsl:apply-templates/>
  <xsl:text>,
</xsl:text>
</xsl:template>

<xsl:template match="attribute::*" priority="-1">
  <xsl:text>A:</xsl:text><xsl:value-of select="name()"/>
  <xsl:text>
  </xsl:text>
</xsl:template>

<xsl:template match="text()" priority="-2"><!-- Override built-in template -->
  <xsl:text>Incorrect text match:</xsl:text><xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
