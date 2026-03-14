<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT28 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test what happens to minus sign embedded in second pattern. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(2392.14*(-36.58),'000,000.000###;-###,###.000###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
