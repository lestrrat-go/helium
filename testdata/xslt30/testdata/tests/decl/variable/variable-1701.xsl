<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#local-variables?>

  <!-- Purpose: Verify that a variable in an inner template can take on a new value -->
  
  <!-- "Only the innermost binding of a variable is visible." -->

<xsl:template match="/">
  <out>
    <xsl:variable name="var" select="'level0'"/>
    <xsl:text>begin root template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
    <xsl:apply-templates/>
    <xsl:text>end root template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:variable name="var" select="'level1'"/>
  <xsl:text>begin doc template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
  <xsl:apply-templates/>
  <xsl:text>end doc template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="a">
  <xsl:variable name="var" select="@pos"/>
  <xsl:text>begin a template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
  <xsl:apply-templates/>
  <xsl:text>end a template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="b">
  <xsl:variable name="var" select="'level3'"/>
  <xsl:text>in b template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()"/><!-- suppress empty lines -->

</xsl:stylesheet>