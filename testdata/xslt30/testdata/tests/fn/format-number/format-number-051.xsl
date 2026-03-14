<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Use Osmanya digits (non-BMP) in formatted output -->

<xsl:output encoding="iso-8859-1"/>

<xsl:decimal-format zero-digit="&#x104a0;"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(1234567890.123456,'##########&#x104a0;.&#x104a0;#####')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
