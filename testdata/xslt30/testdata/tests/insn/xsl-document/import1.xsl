<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

	<xsl:template match="doc">
	 	<out><xsl:document><xsl:sequence select="1 to 4"/></xsl:document></out>
	</xsl:template>
	
</xsl:transform>