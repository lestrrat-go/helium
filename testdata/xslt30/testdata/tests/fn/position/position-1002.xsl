<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position13 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of last() function in value-of select. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="p"/>
  </out>
</xsl:template>

<xsl:template match="p">
  <xsl:value-of select="last()=3"/><xsl:text>,</xsl:text>
</xsl:template>

</xsl:stylesheet>
