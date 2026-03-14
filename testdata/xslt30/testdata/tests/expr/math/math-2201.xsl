<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-numeric-multiply?>
<!-- PURPOSE:  Test of '*' operator. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="n1*n2"/>
</out>
</xsl:template>
</xsl:stylesheet>