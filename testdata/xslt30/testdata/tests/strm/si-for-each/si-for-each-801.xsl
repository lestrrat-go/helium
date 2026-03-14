<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
    <xsl:param name="node-name" required="yes"/>
    <xsl:mode streamable="yes"/>
    <xsl:template match="/">
        <root>
            <xsl:for-each select="//node()[name() = $node-name]">
                <found/>
            </xsl:for-each>
        </root>
    </xsl:template>
</xsl:stylesheet>