<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION: 4.1 Node Set Functions -->
<!-- PURPOSE:  Test of 'last()' function with current context position. -->
<xsl:template match="doc">
<out>
<xsl:for-each select="*">
<xsl:value-of select="last()=4"/>
<xsl:text>,</xsl:text>
</xsl:for-each>
</out>
</xsl:template>
</xsl:stylesheet>