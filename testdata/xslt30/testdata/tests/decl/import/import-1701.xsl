<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Show selection of templates from files with 1st and 2nd import precedence. -->

<xsl:import href="fragments/imp21b.xsl"/>
<xsl:import href="fragments/imp21d.xsl"/>

<xsl:template match="/doc">
  <out>
    <xsl:text>Match on /doc in top xsl&#xa;</xsl:text>
    <xsl:apply-templates select="*"/>
  </out>
</xsl:template>

<xsl:template match="foo">
  <A><xsl:apply-imports/></A><xsl:text>&#xa;</xsl:text>
</xsl:template>

<xsl:template match="goo">
  <A><xsl:apply-imports/></A><xsl:text>&#xa;</xsl:text>
</xsl:template>

<xsl:template match="bar">
  <xsl:text> - match on bar in top xsl</xsl:text>
</xsl:template>

</xsl:stylesheet>