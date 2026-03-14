<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-count?>
<!-- Purpose: Test that current() returns a node-set suitable for count(). -->
<!-- Elaboration: There was a bug, masked by current()/sub-node -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="m"/>
    <xsl:text>&#10;</xsl:text>
    <xsl:for-each select="m">
      <for1><xsl:value-of select="count(current())"/></for1>
      <xsl:text>&#10;</xsl:text>
      <for2><xsl:value-of select="following-sibling::*[count(current())]"/></for2>
    </xsl:for-each>
    <xsl:text>&#10;</xsl:text>
  </out>
</xsl:template>

<xsl:template match="m">
  <apply><xsl:value-of select="count(current())"/></apply>
</xsl:template>

</xsl:stylesheet>