<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<?spec xpath#eval_context?>
<!-- BUG: not a bug after all -->

<xsl:template name="nested-module">
	<xsl:value-of select="@identifier"/><xsl:text>--</xsl:text>
	<xsl:for-each select="ancestor::module">
		<xsl:sort select="position()" order="descending"/>
		<xsl:value-of select="current()/@identifier"/>
		<xsl:text>::</xsl:text>
	</xsl:for-each>
</xsl:template>



<xsl:template match="/">
<out>
<xsl:for-each select="//module">
--------------
    <xsl:variable name="nestedmodule">
        <xsl:call-template name="nested-module"/>
    </xsl:variable>
    <xsl:value-of select="$nestedmodule"/>
--------------
</xsl:for-each>
</out>
</xsl:template>

</xsl:stylesheet>

