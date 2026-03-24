<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Find text in a node with particular ID, via match pattern. -->

<xsl:strip-space elements="a b c"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc/a//text()"/>
    <xsl:text>&#10;</xsl:text>
  </out>
</xsl:template>

<xsl:template match="id('id6')/text()">
  <xsl:text>&#10;</xsl:text>
  <bee><xsl:value-of select="../@id"/></bee>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>&#10;</xsl:text>
  <other><xsl:value-of select="."/></other>
</xsl:template>

</xsl:stylesheet>
