<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT01 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number with 2 arguments, showing zeroes. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(2392.14*36.58,'000,000.000000')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
