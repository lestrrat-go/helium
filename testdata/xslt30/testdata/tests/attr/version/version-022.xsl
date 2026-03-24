<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Show that a local variable is not visible within a sibling xsl:fallback element -->
<?spec xslt#forwards?>
<?spec xslt#fallback?>

    <xsl:variable name="x" select="42"/>

<xsl:template match="/" version="99.9">
    <xsl:explode>
        <xsl:variable name="x" select="99.9"/>
        <xsl:fallback>
            <o val="{$x}"/>
        </xsl:fallback>
    </xsl:explode>
</xsl:template>

</xsl:stylesheet>