<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT09 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test default decimal-format on separator characters, changing both. -->

<xsl:decimal-format decimal-separator="|" grouping-separator="." />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(931.4857,'000.000|###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
