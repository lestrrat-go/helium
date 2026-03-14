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
        <xsl:variable name="non-empty" as="xs:string" select="''"/>        
        
        <bar>
            <xsl:value-of>
                <xsl:value-of select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:value-of select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:value-of select="$non-empty"/>
            </xsl:value-of>
        </bar>
    </xsl:template>
    
</xsl:stylesheet>