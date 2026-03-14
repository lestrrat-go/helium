<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-concat?>
<!-- PURPOSE:   Test of 'concat()' function. -->
<!-- spec changed to allow a numeric argument in 2.0 mode (21 Jun 2004) -->
<xsl:template match="doc">
<out>
<xsl:value-of select="concat(a, 34)"/>
</out>
</xsl:template>
</xsl:stylesheet>