<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	
	<!-- A circular chain of namespace aliases -->
	<?spec xslt#namespace-aliasing?>


	<xsl:namespace-alias stylesheet-prefix="a" result-prefix="b" xmlns:a="http://a.com/" xmlns:b="http://b.com/"/>
	<xsl:namespace-alias stylesheet-prefix="b" result-prefix="c" xmlns:b="http://b.com/" xmlns:c="http://c.com/"/>
	<xsl:namespace-alias stylesheet-prefix="c" result-prefix="d" xmlns:c="http://c.com/" xmlns:d="http://d.com/"/>
	<xsl:namespace-alias stylesheet-prefix="d" result-prefix="e" xmlns:d="http://d.com/" xmlns:e="http://e.com/"/>
	<xsl:namespace-alias stylesheet-prefix="e" result-prefix="a" xmlns:e="http://e.com/" xmlns:a="http://a.com/"/>
	
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
