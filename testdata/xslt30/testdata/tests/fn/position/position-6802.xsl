<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position79 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.2 -->
  <!-- Purpose: Look at preceding axis filtered by node test. Use of apply-templates causes document-order processing. -->
  <!-- Author: David Marston -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select=".//center" />
  </out>
</xsl:template>

<xsl:template match="center">
  <xsl:variable name="num" select="count(preceding::text())" />
  <xsl:text>There are </xsl:text>
  <xsl:value-of select="$num"/>
  <xsl:text> preceding text nodes
</xsl:text>
  <xsl:apply-templates select="preceding::text()"/>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>Position </xsl:text>
  <xsl:value-of select="position()"/>
  <xsl:text> is </xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
