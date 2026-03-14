<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT02 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number and # and 0 in format string. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(12792.14*96.58,'##,###,000.000###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
