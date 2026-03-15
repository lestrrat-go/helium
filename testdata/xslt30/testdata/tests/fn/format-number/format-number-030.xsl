<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT32 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number percent format with character being changed. -->

<xsl:decimal-format percent="c" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(0.4857,'###.###c')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
