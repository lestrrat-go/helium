<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT08 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number with prefix and suffix in format string. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(2.14*86.58,'PREFIX##00.000###SUFFIX')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
