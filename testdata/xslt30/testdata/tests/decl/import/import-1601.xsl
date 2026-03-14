<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Show that apply-imports applies on its matching node, not children. -->

<xsl:import href="fragments/imp20b.xsl"/>
<xsl:import href="fragments/imp20d.xsl"/>

<xsl:template match="/">
  <out>
    <xsl:text>Match on / in top xsl&#xa;</xsl:text>
    <xsl:apply-templates select="foo"/>
  </out>
</xsl:template>

<xsl:template match="foo">
  <A><xsl:apply-imports/></A>
</xsl:template>

<xsl:template match="bar">
  <xsl:text>match on bar in top xsl</xsl:text>
</xsl:template>

</xsl:stylesheet>