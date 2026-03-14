<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE stylesheet [
<!ENTITY cr "<xsl:text>
</xsl:text>">
<!--  should be a named template but it cannot be used in streamable mode, so we define "a macro"   -->
<!ENTITY show-ancestors '
<xsl:for-each select="ancestor-or-self::node()">
            <xsl:value-of select="name()"/>
            <xsl:if test="$a">
                <xsl:for-each select="@*">
                    <xsl:value-of select="concat(&quot;[@&quot;,name())"/>
                    <xsl:if test="$av">
                        <xsl:value-of select="concat(&quot;=&quot;,$q,.,$q)"/>
                    </xsl:if>
                    <xsl:text>]</xsl:text>
                </xsl:for-each>
            </xsl:if>
            <xsl:if test="position()!=last()"><xsl:text>/</xsl:text></xsl:if>
        </xsl:for-each>'>
]>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">
    <xsl:strip-space elements="*"/>
    <xsl:output omit-xml-declaration="yes"/>
    <xsl:mode streamable="yes"/>
    
    <xsl:variable name="q">"</xsl:variable>
    <xsl:param name="av" select="true()"></xsl:param>
    <xsl:param name="a" select="true()"></xsl:param>
    <xsl:param name="v" select="true()"/>
    
    <xsl:template match="/">
        <out><xsl:apply-templates/></out>
    </xsl:template>
    
    <xsl:template match="*">
        <p>&show-ancestors;</p>
        <xsl:apply-templates/>
    </xsl:template>
    
    <xsl:template match="text()">
        <t>&show-ancestors;
        <xsl:text>text()</xsl:text>
        <xsl:if test="$v">
            <xsl:text>[.="</xsl:text><xsl:value-of select="."/><xsl:text>"]</xsl:text>
        </xsl:if>
        </t>
    </xsl:template>
    
</xsl:stylesheet>