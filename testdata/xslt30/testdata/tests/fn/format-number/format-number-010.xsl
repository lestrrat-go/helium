<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT11 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test default decimal-format on pattern-only characters, positive number. -->

<xsl:decimal-format digit="!" pattern-separator="\" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(26931.4,'+!!!,!!!.!!!\-!!,!!!.!!!')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
