<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" >

	<xsl:template match="b" use-when="false()">
		<print_b><xsl:next-match/></print_b>
	</xsl:template>

</xsl:stylesheet>
