<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- FileName: NumberFormat38 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test result pattern for not-a-number, unchanged. -->
  <!-- Note: uses version="1.0", gives an error under 2.0 -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number('foo','#############')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
