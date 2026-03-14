<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" >

    <xsl:template match="elem">
	    <xsl:apply-templates/>
    </xsl:template>
  
	<xsl:template match="b">
		<print_b><xsl:next-match/></print_b>
	</xsl:template>

</xsl:stylesheet>
