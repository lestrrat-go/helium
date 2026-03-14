<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- 
        purpose: to show that a sequence of two consecutive empty strings contains a space in-between and 
                 that adjacent text nodes are merged with no space in-between, but document nodes are not
    -->
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:variable name="non-empty" as="text()">
            <xsl:value-of>
                <xsl:text expand-text="yes">{''}</xsl:text>
                <xsl:text expand-text="yes">{''}</xsl:text>
            </xsl:value-of>
        </xsl:variable>
        
        <bar>
            <xsl:attribute name="foo">
                <xsl:sequence select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:sequence select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:sequence select="$non-empty"/>
            </xsl:attribute>
        </bar>
    </xsl:template>
    
</xsl:stylesheet>