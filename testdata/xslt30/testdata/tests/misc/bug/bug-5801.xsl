<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<?spec xpath#eval_context?>
<!-- (not actually a bug, just a suspected one) -->

<xsl:template match="/">
 <xsl:copy-of select="/a/b[last()-1]/c[last()-1]"/>
</xsl:template>



</xsl:transform>
