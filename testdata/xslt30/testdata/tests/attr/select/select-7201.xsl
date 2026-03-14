<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Test //@* to get all attributes in the tree -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="/doc/b/bb/bbb" />
  </out>
</xsl:template>

<xsl:template match="bbb">
  <list><xsl:apply-templates select="//@*"/></list>
</xsl:template>

<xsl:template match="@*">
  <xsl:text>
Node </xsl:text>
  <xsl:value-of select="name(..)" />
  <xsl:text> has </xsl:text>
  <xsl:value-of select="." />
  <xsl:text> in @</xsl:text>
  <xsl:value-of select="name(.)" />
</xsl:template>

</xsl:stylesheet>