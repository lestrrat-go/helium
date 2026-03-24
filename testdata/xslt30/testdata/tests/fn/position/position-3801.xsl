<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position09 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 Node Set Functions -->
    <!-- Purpose: Test of position in match pattern predicates, both long and short versions. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="foo[3]/a[position()=3]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[3]/a[position()=2]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[3]/a[position()=1]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[2]/a[position()=3]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[2]/a[position()=2]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[2]/a[position()=1]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[1]/a[position()=3]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[1]/a[position()=2]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="foo[1]/a[position()=1]/num/@val">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="a/num">
  <xsl:apply-templates select="@val"/>
</xsl:template>

<!-- Override default template matching, otherwise the values 10, 11, 12 will be displayed -->

<xsl:template match="@val" priority="-1"/>

</xsl:stylesheet>
