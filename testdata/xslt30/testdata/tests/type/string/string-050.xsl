<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-starts-with?>
<!-- PURPOSE:   Test of 'starts-with()' function. -->
<xsl:template match="doc">
<out>
<!-- changed 29 May 2001 to do something useful... -->
<xsl:value-of select="starts-with(., 'ENCY')"/>
</out>
</xsl:template>
</xsl:stylesheet>