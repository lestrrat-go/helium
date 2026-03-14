<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<?spec xslt#creating-namespace-nodes?>
    <xsl:template match="/">
        <xsl:document>
            <root xsi:type="Root">
                <xsl:call-template name="namespaces">
                    <xsl:with-param name="nodes" select="./*"/>
                </xsl:call-template>
            </root>
        </xsl:document>
    </xsl:template>
    <xsl:template name="namespaces" as="node()*">
        <xsl:param name="nodes" as="node()*"/>
        <xsl:for-each select="$nodes//namespace::*[. != '']">
            <xsl:namespace name="{name()}">
                <xsl:attribute name="select">
                    <xsl:value-of select="."/>
                </xsl:attribute>
            </xsl:namespace>
        </xsl:for-each>
    </xsl:template>
</xsl:stylesheet>

