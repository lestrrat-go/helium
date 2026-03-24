<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NumberFormat40 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test specification of result pattern for infinity
       when quantity to be displayed is negative infinity. -->

<xsl:decimal-format infinity="huge" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-1 div 0e0,'###############################')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
