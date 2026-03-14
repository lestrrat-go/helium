<?xml version="1.0" ?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

	<xsl:template match="data">
		<xsl:param name="Ppar1" select="'Poor default'" />		
		<xsl:variable name="vpar1" select="'Poor default'" />
		<var>Variable Value: <xsl:value-of select="$vpar1" /></var>
		<par>Parameter Value: <xsl:value-of select="$Ppar1" /></par>
	</xsl:template>	

</xsl:stylesheet>