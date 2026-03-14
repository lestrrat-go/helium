<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 	
>
	
	<!-- TEST xsl:namespace (XSLT 2.0) -->
	
	<?spec xslt#creating-namespace-nodes?>
	
	<xsl:param name="param" select="'http://a-namespace-uri/'"/>

	<xsl:template match="/">
		<out>
		  <xsl:namespace name="zz"><xsl:value-of select="$param"/></xsl:namespace>
		  <xsl:attribute name="new">at XSLT 2.0</xsl:attribute>
		</out>
	</xsl:template>

</xsl:stylesheet>
