<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#format-number?>
<!-- PURPOSE:   Test of 'format-number()' function. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="format-number(0.25, '00%')"/>
</out>
</xsl:template>
</xsl:stylesheet>