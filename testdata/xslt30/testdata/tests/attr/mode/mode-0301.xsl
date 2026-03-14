<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: MODES10 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.7 Modes -->
    <!-- Purpose: Show that we only go into a mode via apply-templates.
     You can't put a mode on call-template, and the fact that you call a named
     template that has a mode specifier doesn't mean you are in that mode. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc" mode="a"/>
  </out>
</xsl:template>

<xsl:template match="doc" mode="a" priority="3">
  <xsl:text>Found doc...</xsl:text>
  <xsl:call-template name="scan"/>
</xsl:template>

<!-- The following template is both applied in mode a and called -->
<xsl:template name="scan" match="*" mode="a" priority="2">
  <xsl:text>Scanned </xsl:text><xsl:value-of select="name(.)"/><xsl:text>
</xsl:text>
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="x" priority="4">
  <xsl:text>Found x, no mode: </xsl:text><xsl:value-of select="."/><xsl:text>
</xsl:text>
  <xsl:apply-templates mode="a"/>
</xsl:template>

<xsl:template match="x" mode="a" priority="4">
  <xsl:text>Found x, mode a: </xsl:text><xsl:value-of select="@test"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>modeless text: </xsl:text><xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()" mode="a">
  <xsl:text>mode a text: </xsl:text><xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
