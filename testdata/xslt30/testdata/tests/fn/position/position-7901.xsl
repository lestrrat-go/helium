<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test of position() and last() in xsl:with-param. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/root">
  <out>
    <xsl:apply-templates select="a" />
  </out>
</xsl:template>

<xsl:template match="a">
  <xsl:text>
</xsl:text>
  <xsl:text>In a, position() is </xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text> and last() is </xsl:text>
  <xsl:value-of select="last()"/><xsl:text>
</xsl:text>
  <xsl:apply-templates select="b">
    <xsl:with-param name="a.pos" select="position()"/>
    <xsl:with-param name="a.last" select="last()"/>
  </xsl:apply-templates>
</xsl:template>

<xsl:template match="b">
  <xsl:param name="a.pos" select="pos-bad"/>
  <xsl:param name="a.last" select="last-bad"/>
  <xsl:text>In b number </xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text>, $a.pos=</xsl:text>
  <xsl:value-of select="$a.pos"/>
  <xsl:text> and $a.last=</xsl:text>
  <xsl:value-of select="$a.last"/>
  <xsl:text>
</xsl:text>
</xsl:template>

</xsl:stylesheet>