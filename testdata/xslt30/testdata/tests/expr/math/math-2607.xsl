<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-literals?>
<!-- PURPOSE:  Test of decimal numeric literals containing ".". -->
<xsl:template match="doc">
<out>
<xsl:value-of select="1.0 + 0.1 - 0.0 + 1.01 - 01.010 - 01.10"/>
</out>
</xsl:template>
</xsl:stylesheet>