<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	
	<!-- Aliases for multiple namespaces with the same target; namespace declarations defined locally -->
	<?spec xslt#namespace-aliasing?>


	<xsl:namespace-alias stylesheet-prefix="a" result-prefix="A" xmlns:a="http://a.com/" xmlns:A="http://A.com/"/>
	<xsl:namespace-alias stylesheet-prefix="b" result-prefix="B" xmlns:b="http://b.com/" xmlns:B="http://A.com/"/>
	<xsl:namespace-alias stylesheet-prefix="c" result-prefix="C" xmlns:c="http://c.com/" xmlns:C="http://A.com/"/>
	<xsl:namespace-alias stylesheet-prefix="d" result-prefix="D" xmlns:d="http://d.com/" xmlns:D="http://A.com/"/>
	<xsl:namespace-alias stylesheet-prefix="e" result-prefix="E" xmlns:e="http://e.com/" xmlns:E="http://A.com/"/>
	
    <xsl:output method="xml"/>
	<xsl:template match="/">
		<out>
			<v xmlns="http://a.com/"/>
			<w xmlns="http://b.com/"/>
			<x xmlns="http://c.com/"/>
			<y xmlns="http://d.com/"/>
			<z xmlns="http://e.com/"/>
		</out>
	</xsl:template>
</xsl:stylesheet>
