<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT06 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number per-mille format. -->

<xsl:output method="xml" encoding="ISO-8859-1"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(0.4857,'###.###&#8240;')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
