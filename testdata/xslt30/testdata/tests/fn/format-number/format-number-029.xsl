<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT31 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test output of altered minus, 2 patterns but no sign marker in pattern. -->

<xsl:decimal-format minus-sign="_" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-26931.4,'###,###.###;###,###.###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
