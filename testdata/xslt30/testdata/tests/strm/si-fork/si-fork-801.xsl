<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:mode name="toc" streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:output indent="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:template match="body">
        <xsl:copy>
            <xsl:fork>
                <xsl:sequence>
                    <heading>TOC</heading>                    
                    <ordered-list>
                        <xsl:apply-templates select="heading" mode="toc"/>
                    </ordered-list>                    
                </xsl:sequence>
                <xsl:sequence>
                    <xsl:apply-templates/>
                </xsl:sequence>
            </xsl:fork>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="heading" mode="toc">
        <list-item>
            <link href="#{generate-id()}">
                <xsl:apply-templates mode="#current"/>
            </link>
        </list-item>
    </xsl:template>
    
    <xsl:template match="heading">
        <xsl:copy>
            <xsl:attribute name="id" select="generate-id()"/>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>