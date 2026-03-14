<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:mf="http://example.com/mf"
    expand-text="yes"
    exclude-result-prefixes="#all">
    
    <xsl:function name="mf:test" as="item()*">
        <xsl:param name="i1" as="xs:integer*"/>
        <result>{sum($i1)}</result>
    </xsl:function>
    
    <xsl:template match="/" name="xsl:initial-template">
        <root>
            <xsl:merge>
                <xsl:merge-source select="1 to 10">
                    <xsl:merge-key select=". idiv 5"></xsl:merge-key>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:sequence select="(1 to 3) ! mf:test(current-merge-group())"/>
                </xsl:merge-action>
            </xsl:merge>
        </root>
    </xsl:template>
    
</xsl:stylesheet>