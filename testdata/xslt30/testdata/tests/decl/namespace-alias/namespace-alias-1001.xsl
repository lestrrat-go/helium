<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 	
	xmlns:out="http://output.ns/">
	
	<!-- TEST namespace-alias on attribute names -->
	<?spec xslt#namespace-aliasing?>
	<!-- BUG Saxon 6.3/013 -->

	<xsl:namespace-alias stylesheet-prefix="out" result-prefix="xsl"/>
    <xsl:output method="xml"/>
	<xsl:template match="/">
		<html out:version="2.0"></html>
	</xsl:template>
</xsl:stylesheet>
