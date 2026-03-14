<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
    <xsl:template match="/">
        <root>
            <xsl:variable name="a-doc">
                <a-root/>
            </xsl:variable>
            <xsl:copy-of select="$a-doc"/>
            <xsl:apply-templates select="$a-doc" mode="test"/>
        </root>
    </xsl:template>
    <xsl:template match="/ | node()" mode="test">
        <xsl:copy/>
    </xsl:template>
</xsl:stylesheet>
