<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate12 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.6 -->
    <!-- Purpose: Test use of passed-in value in an AVT.
     Derived from example code at end of 11.6 -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="OL">
  <xsl:text>OL!</xsl:text>
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="LI">
  <xsl:call-template name="numbered-block">
    <xsl:with-param name="format">A. </xsl:with-param>
  </xsl:call-template>
</xsl:template>

<xsl:template match="OL/LI">
  <xsl:call-template name="numbered-block">
    <xsl:with-param name="format">a. </xsl:with-param>
  </xsl:call-template>
</xsl:template>

<xsl:template name="numbered-block">
  <xsl:param name="format">1. </xsl:param>
  <xsl:number format="{$format}"/>
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>
