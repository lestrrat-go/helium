<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: test error condition: same symbol used in two different roles. -->

<xsl:import href="numberformat903err.xsl"/>

<xsl:decimal-format name="q" decimal-separator="." grouping-separator="."/>

<xsl:decimal-format name="r" decimal-separator="," grouping-separator=";"/>    

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(931.4857,'000.000', 'q')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
