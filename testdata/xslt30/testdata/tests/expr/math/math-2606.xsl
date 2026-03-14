<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-literals?>
<!-- PURPOSE:  Test of numeric literals containing "e" (XPath 2.0 only). -->
<xsl:template match="doc">
<out>
<xsl:value-of select="1.0e1 + 1.0e+01 + 1.0e-01 + 0.1e2 + .1e2"/>
</out>
</xsl:template>
</xsl:stylesheet>