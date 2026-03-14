<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#document?>
<!-- BUG: Saxon -->
<!-- DESCRIPTION: Documents loaded from URI's in a node-set -->

<xsl:variable name="docs"
              select="document(/xxx/ref/@file)" />

<xsl:template match="/">
<out note="results depend on arbitrary document order of different documents">
Size is <xsl:value-of select="count($docs)"/>;
<xsl:copy-of select="$docs" />
</out>
</xsl:template>

</xsl:stylesheet>



