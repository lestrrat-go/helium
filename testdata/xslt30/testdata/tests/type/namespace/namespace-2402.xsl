<xsl:stylesheet version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

	<!-- TEST namespace-uri-for-prefix() function -->
	<?spec fo#func-namespace-uri-for-prefix?>

	<xsl:template match="/">
     <z>
        <a emy="{namespace-uri-for-prefix('my', /*)}"/>
        <a edefault="{namespace-uri-for-prefix('', /*)}"/>
        <a fmy="{namespace-uri-for-prefix('my', //f)}"/>
        <a fdefault="{namespace-uri-for-prefix('', //f)}"/>
        <zero count="{count(namespace-uri-for-prefix('magic', //f))}"/>    
     </z>
	</xsl:template>

</xsl:stylesheet>
