<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
    <!-- PURPOSE:  Test attribute default-mode on included stylesheet. -->
    <xsl:include href="mode-1616a.xsl"/>
    
    <xsl:template match="/">
        <out>
            <xsl:apply-templates select="doc/a" mode="a"/>
        </out>
    </xsl:template>
    
    <xsl:template match="a">
        <xsl:text>no-mode:</xsl:text>
        <xsl:value-of select="."/>
    </xsl:template>
    
</xsl:stylesheet>
