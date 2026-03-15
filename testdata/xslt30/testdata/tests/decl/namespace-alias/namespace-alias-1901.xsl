<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

	<!-- TEST xsl:namespace-alias - stylesheet-prefix="#default" -->
	<!-- See W3C bug 39307 for discussion of whether the result should be version="1.0" or xsl:version="1.0" -->

	<xsl:import href="namespace-alias-1002a.xsl"/>

	<xsl:namespace-alias stylesheet-prefix="#default" result-prefix="xsl"/>

	<xsl:template match="/">
		<stylesheet version="1.0">
		 <template name="x">
		  <apply-templates/>
		 </template>
		</stylesheet>
	</xsl:template>
</xsl:stylesheet>
