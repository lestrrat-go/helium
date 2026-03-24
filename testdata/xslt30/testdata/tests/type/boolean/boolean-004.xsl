<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-not?>
<?spec fo#func-true?>
<!--  Test of boolean "not" operator with true value. -->
<xsl:template match="/">
<out>
<xsl:value-of select="not(true())"/>
</out>
</xsl:template>
 
</xsl:stylesheet>
