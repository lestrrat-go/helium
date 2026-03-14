<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position95 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test position in a union in a match pattern -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="descendant::*"/>
  </out>
</xsl:template>

<xsl:template match="alpha|z">
  <xsl:text>
</xsl:text>
  <xsl:value-of select="@key"/>
  <xsl:text> is at position </xsl:text>
  <xsl:value-of select="position()"/>
</xsl:template>

<xsl:template match="beta"/><!-- Squelch these -->

</xsl:stylesheet>
