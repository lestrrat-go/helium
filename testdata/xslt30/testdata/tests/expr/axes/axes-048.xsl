<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xpath#axes?>
<!-- test preceding axis from an attribute. Bug report from Kurt Devlin -->

<xsl:variable name="allpubs"
    select="//pubtag[@list='yes']/@pub[
        not(.=preceding::pubtag[@list='yes']/@pub)]" />

<xsl:template match="/">
<out>
	<xsl:for-each select="$allpubs">
		<xsl:value-of select="." /><br />;
	</xsl:for-each>
</out>	
</xsl:template>
</xsl:stylesheet>
