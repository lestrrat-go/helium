<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test that a trailing decimalpoint is removed. -->
  
<xsl:decimal-format grouping-separator=" "/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(1234e0, '0000.####')"/>;
    <xsl:value-of select="format-number(1234.00, '0000.####')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
