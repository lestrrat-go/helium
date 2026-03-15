<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="word-type" as="xs:string" select="'adverb'"/>
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:template name="main">
        <xsl:source-document streamable="yes" href="si-copy-of-801.xml">
            <xsl:apply-templates
                select="copy-of(outermost(//p))[.//*[local-name() = $word-type]]">
                <xsl:with-param name="tagName" select="$word-type" tunnel="yes"/>
            </xsl:apply-templates>
        </xsl:source-document>      
    </xsl:template>
    
    <xsl:template match="*">
        <xsl:param name="tagName" tunnel="yes"/>
        <xsl:choose>
            <xsl:when test="local-name() = $tagName">
                <span class="highlight">
                    <xsl:apply-templates/>
                </span>
            </xsl:when>
            <xsl:otherwise>
                <xsl:next-match/>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:template>
    
</xsl:stylesheet>