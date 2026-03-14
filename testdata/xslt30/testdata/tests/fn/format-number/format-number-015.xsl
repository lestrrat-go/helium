<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT16 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of decimal-format per-mille format with character being changed. -->

<xsl:decimal-format per-mille="m" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(0.4857,'###.###m')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
