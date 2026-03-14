<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns="http://www.w3.org/1999/XSL/Transform"
	xmlns:x="http://temp/" 	
>
	
	<!-- TEST xsl:namespace-alias - result-prefix="#default" -->

    <xsl:import href="namespace-alias-1002a.xsl"/>
	
	<xsl:namespace-alias stylesheet-prefix="x" result-prefix="#default"/>
	
	<xsl:template match="/">
		<x:stylesheet version="1.0">
		 <x:template name="x">
		  <x:apply-templates/>
		 </x:template>
		</x:stylesheet>
	</xsl:template>
</xsl:stylesheet>
