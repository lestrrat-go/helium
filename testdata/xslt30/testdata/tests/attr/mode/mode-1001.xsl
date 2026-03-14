<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: MODES17 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.7 Modes -->
    <!-- Purpose: Check that underscores are allowed in names. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc" mode="_1st_mode"/>
  </out>
</xsl:template>

<xsl:template match="a" mode="a.mode">
  <xsl:text>mode-a:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="a">
  <xsl:text>no-mode:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="doc" mode="_1st_mode">
  <xsl:text>_1st_mode: </xsl:text><xsl:apply-templates select="a" mode="a.mode"/>
</xsl:template>

</xsl:stylesheet>
