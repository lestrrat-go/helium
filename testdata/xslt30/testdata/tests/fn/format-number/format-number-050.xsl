<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Use non-BMP characters as separators -->

<xsl:output encoding="iso-8859-1"/>

<xsl:decimal-format decimal-separator="&#110000;" grouping-separator="&#110001;" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(1234567890.123456,'&#110001;000&#110000;000')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
