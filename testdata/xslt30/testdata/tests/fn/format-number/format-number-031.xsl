<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Purpose: Test changing both digit and zero-digit in format string. -->

<xsl:output encoding="iso-8859-1"/>
<xsl:decimal-format digit="!" zero-digit="&#x0660;" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(4030201.0506,'#!!!,!!!,&#x0660;&#x0660;&#x0660;.&#x0660;&#x0660;&#x0660;&#x0660;&#x0660;&#x0660;0')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
