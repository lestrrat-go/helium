<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" default-mode="a">
    
    <!-- PURPOSE:  Test attribute default-mode on nested elements. -->
    <xsl:template match="/" mode="#all">
        <out xsl:default-mode="b">
            <xsl:apply-templates select="doc/a" default-mode="a"/>
        </out>
    </xsl:template>
    
    <xsl:template match="a" default-mode="a">
        <xsl:text>element-mode-a:</xsl:text>
        <xsl:apply-templates select="@test"/>
    </xsl:template>
    
    <xsl:template match="a" default-mode="b">
        <xsl:text>element-mode-b:</xsl:text>
        <xsl:apply-templates select="@test"/>
    </xsl:template>
    
    <xsl:template match="@*" mode="a">
        <xsl:value-of select="."/>
    </xsl:template>
        
    <xsl:template match="@*" mode="b">
        <xsl:text>attribute-mode-b</xsl:text>
    </xsl:template>
    
</xsl:stylesheet>
