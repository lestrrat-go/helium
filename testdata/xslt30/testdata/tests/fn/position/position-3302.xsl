<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position08 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() in match pattern. Used in predicate of name.
     Look for 1, last, others. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="a[position()=4]">
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="a[position()=3]">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="a[position()=2]">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="a[position()=1]">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

</xsl:stylesheet>
