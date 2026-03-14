<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#copy-of?>
<!-- need to run this with -T option to demonstrate the bug -->

<!-- NullPointerException when using -T option -->

<xsl:variable name="t">
<xsl:for-each select="//node()">
<x/>
</xsl:for-each>
</xsl:variable>

<xsl:template match="/">
<out>
<xsl:copy-of select="$t"/>
</out>
</xsl:template>

</xsl:stylesheet>
