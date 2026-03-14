<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position70 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.4 -->
    <!-- Purpose: Test that set of nodes changes when strip-space is in effect. -->
  <!-- "The xsl:apply-templates instruction processes all children of the current node,
        including text nodes. However, text nodes that have been stripped as specified
        in 3.4 Whitespace Stripping will not be processed." -->

<xsl:strip-space elements="doc inner inner2"/>

<xsl:template match="/doc">
  <out>
    <xsl:apply-templates select="*|@*|comment()|text()"/>
  </out>
</xsl:template>

<xsl:template match="*"><!-- for child elements -->
  <xsl:text>E(</xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text>):</xsl:text>
  <xsl:value-of select="name()"/>
  <xsl:apply-templates select="@*"/>
  <xsl:apply-templates/>
  <xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="@*">
  <!-- The parser has freedom to present attributes in any order it wants.
     Input file should have only one attribute if you want consistent results across parsers. -->
  <xsl:text>A(</xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text>):</xsl:text>
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>T(</xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text>):</xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="comment()">
  <xsl:text>C(</xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text>):</xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
