<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring?>
<!-- PURPOSE:   Test of 'substring()' function with Unicode surrogate pairs. -->


<xsl:template match="doc">
<out>
<xsl:variable name="test" select="'abc&#x50000;efg&#x50000;ijk'"/>
<xsl:value-of select="substring($test, 5, 5)"/>
</out>
</xsl:template>
</xsl:stylesheet>