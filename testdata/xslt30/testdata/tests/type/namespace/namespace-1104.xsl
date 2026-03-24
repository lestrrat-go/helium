<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 	
>
	
	<!-- TEST xsl:namespace declaring the default namespace (XSLT 2.0) -->
	<?spec xslt#creating-namespace-nodes?>
	
	<xsl:param name="param" select="'http://a-namespace-uri/'"/>

	<xsl:template match="/">
		<x:out xmlns:x="http://some-uri/">
		  <xsl:namespace name=""><xsl:value-of select="$param"/></xsl:namespace>
		  <xsl:attribute name="new">at XSLT 2.0</xsl:attribute>
		</x:out>
	</xsl:template>

</xsl:stylesheet>
