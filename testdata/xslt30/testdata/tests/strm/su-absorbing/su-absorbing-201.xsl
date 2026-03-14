<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" xmlns:mf="http://example.com/mf" 
    exclude-result-prefixes="xs math mf" version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
    
    <xsl:param name="element-prefix" as="xs:string" select="'h'"/>
    
    <xsl:function name="mf:group" as="node()*" streamability="absorbing">
        <xsl:param name="input" as="node()*"/>
        <xsl:param name="level" as="xs:integer"/>
        <xsl:for-each-group select="$input" 
            group-starting-with="*[local-name() eq ($element-prefix || $level)]">
            <xsl:choose>
                <xsl:when test="self::*[local-name() eq ($element-prefix || $level)]">
                    <section>
                        <xsl:variable name="group-copy" select="copy-of(current-group())"/>
                        <xsl:copy-of select="$group-copy[1]"/>
                        <xsl:sequence select="mf:group($group-copy[position() gt 1], $level + 1)"/>
                    </section>
                </xsl:when>
                <xsl:otherwise>
                    <xsl:apply-templates select="current-group()"/>
                </xsl:otherwise>
            </xsl:choose>
        </xsl:for-each-group>
    </xsl:function>
    
    <xsl:template match="body">
        <xsl:copy>
            <xsl:apply-templates select="@*"/>
            <xsl:sequence select="mf:group(*, 1)"/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>