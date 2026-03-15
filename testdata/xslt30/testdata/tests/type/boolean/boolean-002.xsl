<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

<?spec fo#func-codepoint-equal?>
  <!-- PURPOSE: Test of codepoint-equal() function -->

<xsl:param name="p" select="'Rhubarb'"/>

<xsl:template match="/">
	<out>
	<t><xsl:value-of select="codepoint-equal($p, 'Rhubarb')"/></t>
	<f><xsl:value-of select="codepoint-equal($p, 'Rhubarb ')"/></f>	
	<f><xsl:value-of select="codepoint-equal($p, '')"/></f>		
	<t><xsl:value-of select="codepoint-equal(substring($p, 10, 1), '')"/></t>
	<t><xsl:value-of select="codepoint-equal($p, xs:anyURI('Rhubarb'))"/></t>						
	</out>
</xsl:template>


</xsl:stylesheet>
