<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-substring-before?>
<!-- PURPOSE:   Test of 'substring-before()' function. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="substring-before(foo, '/')"/>
</out>
</xsl:template>
</xsl:stylesheet>