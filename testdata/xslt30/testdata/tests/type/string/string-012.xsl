<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
<!-- PURPOSE:   Test of 'translate()' function. -->
<xsl:template match="/doc">
	<out>
       <xsl:value-of select='translate("bar","abc","ABC")'/>
	</out>
</xsl:template>
 
</xsl:stylesheet>
