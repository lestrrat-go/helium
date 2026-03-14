<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NumberFormat37 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test result pattern for infinity, unchanged. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(1 div 0e0,'###############################')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
