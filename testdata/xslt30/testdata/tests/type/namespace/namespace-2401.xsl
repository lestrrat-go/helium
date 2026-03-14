<xsl:stylesheet version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:f="http://localhost/f"
	exclude-result-prefixes="f"
>

	<!-- TEST in-scope-prefixes() function -->
	<?spec fo#func-in-scope-prefixes?>

	<xsl:template match="/">
        <z>
            <e><xsl:value-of select="f:sort(in-scope-prefixes(/*))" separator=","/></e>
            <f><xsl:value-of select="f:sort(in-scope-prefixes(//f))" separator=","/></f>
        </z>
	</xsl:template>
	
	<xsl:function name="f:sort">
	  <xsl:param name="in"/>
	  <xsl:perform-sort select="$in">
	     <xsl:sort select="."/>
	  </xsl:perform-sort>
	</xsl:function>

</xsl:stylesheet>
