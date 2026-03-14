<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT26 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Designate a space as the grouping separator. -->

<xsl:decimal-format decimal-separator="," grouping-separator=" " />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(7654321.4857,'### ### ###,#####')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
