<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- exponent-separator: XPath 3.1 -->
<xsl:decimal-format exponent-separator="E" />

<xsl:template name="main">
  <out>
    <xsl:value-of select="format-number(123.456, '0.0000E0')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
