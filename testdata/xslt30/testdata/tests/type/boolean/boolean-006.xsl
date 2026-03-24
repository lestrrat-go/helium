<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-boolean?>
<!--  Test of boolean() function - conversion of empty string. -->
<xsl:template match="/">
<out>
<xsl:value-of select="boolean('')"/>
</out>
</xsl:template>
 
</xsl:stylesheet>
