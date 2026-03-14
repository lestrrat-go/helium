<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT17 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test decimal-format output character for negative, 2 patterns. -->

<xsl:decimal-format minus-sign="_" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-26931.4,'+###,###.###;-###,###.###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
