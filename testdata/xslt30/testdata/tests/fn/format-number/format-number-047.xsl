<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test formatting of a high-precision decimal value. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(000123456789012345678901234567890.123456789012345678900000,
    '##0.0####################################################')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
