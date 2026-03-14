<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns="http://www.w3.org/1999/XSL/Transform"
	xmlns:x="http://temp/" 	
>
	
	<!-- TEST xsl:namespace clashing with namespace of parent element -->
	<?spec xslt#creating-namespace-nodes?>

	<xsl:template match="/">
        <xsl:element name="ns:e" xmlns:ns="http://nsone.uri/">
          <xsl:namespace name="ns">http://nstwo.uri/</xsl:namespace>
        </xsl:element>
	</xsl:template>
</xsl:stylesheet>
