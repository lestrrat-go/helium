<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position72 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test of position() and the ancestor-or-self axis. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select=".//f"/>
  </out>
</xsl:template>

<xsl:template match="f">
  <xsl:value-of select="ancestor-or-self::*[1]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[2]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[3]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[4]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[5]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[6]/text()[1]"/>
  <xsl:value-of select="ancestor-or-self::*[7]/text()[1]"/>
</xsl:template>

</xsl:stylesheet>
