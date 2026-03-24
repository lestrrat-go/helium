<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:out="http://output.ns/">

	<!-- TEST namespace-alias - two declarations with different precedence -->

	<xsl:import href="namespace-alias-1002a.xsl"/>
	
	<xsl:namespace-alias stylesheet-prefix="out" result-prefix="xsl"/>
	
	<xsl:output method="xml"/>
	
	<xsl:template match="/">
		<html out:version="2.0"/>
	</xsl:template>
	
</xsl:stylesheet>
