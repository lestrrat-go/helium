<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT07 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number currency symbol, which is not supposed to be there. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(95.4857,'&#164;###.####')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
