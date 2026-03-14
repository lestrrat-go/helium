<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:test="http://example.com/test"    
    exclude-result-prefixes="test">
    
    <!-- PURPOSE:  Test attribute default-mode (namespaced mode) on xsl:template element. -->
    <xsl:template match="/" default-mode="test:a">
        <out>
            <xsl:apply-templates select="doc/a"/>            
        </out>
    </xsl:template>
    
    <xsl:template match="a" mode="test:a">
        <xsl:text>mode-a:</xsl:text>
        <xsl:value-of select="."/>
    </xsl:template>
    
    <xsl:template match="a">
        <xsl:text>no-mode:</xsl:text>
        <xsl:value-of select="."/>
    </xsl:template>
    
</xsl:stylesheet>
