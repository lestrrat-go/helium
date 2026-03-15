<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION: 4.1 Node Set Functions -->
<!-- PURPOSE:  Test of 'count()'. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="count(*)"/>
</out>
</xsl:template>

</xsl:stylesheet>
