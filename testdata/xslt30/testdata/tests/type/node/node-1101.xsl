<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test for node tests in match patterns (and union in select). -->

<xsl:template match="/doc">
  <out>
    <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
  </out>
</xsl:template>

<xsl:template match="*"><!-- for child elements -->
  <xsl:text>E:</xsl:text><xsl:value-of select="name()"/>
  <xsl:apply-templates select="@*"/>
  <xsl:apply-templates/>
  <xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="@*">
  <xsl:text>A:</xsl:text><xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>T:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="comment()">
  <xsl:text>C:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="processing-instruction()">
  <xsl:text>P:</xsl:text><xsl:value-of select="name()"/>
</xsl:template>

</xsl:stylesheet>
