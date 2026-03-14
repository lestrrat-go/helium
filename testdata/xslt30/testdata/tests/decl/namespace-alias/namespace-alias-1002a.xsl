<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 	
	xmlns:out="http://output.ns/"
	xmlns:new="http://wrong.namespace/">
	
	<!-- TEST namespace-alias - two declarations with different precedence -->


	<xsl:namespace-alias stylesheet-prefix="out" result-prefix="new"/>
	<xsl:template match="/">
		<html out:version="2.0"></html>
	</xsl:template>
</xsl:stylesheet>
