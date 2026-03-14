<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-true?>
<?spec xpath#id-logical-expressions?>
<!--  Test of boolean "or" operator with two true values -->
<xsl:template match="/">
<out>
<xsl:value-of select="true() or true()"/>
</out>
</xsl:template>
 
</xsl:stylesheet>
