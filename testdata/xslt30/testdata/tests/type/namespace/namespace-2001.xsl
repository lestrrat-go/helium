<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns="http://www.w3.org/1999/XSL/Transform"
	xmlns:x="http://temp/">
	
	<!-- TEST copied namespace clashing with namespace of parent element -->

	<xsl:template match="/">
        <xsl:element name="ns:e" namespace="http://nsone.uri/">
          <xsl:copy-of select="doc/namespace::ns"/>
        </xsl:element>
	</xsl:template>
</xsl:stylesheet>
