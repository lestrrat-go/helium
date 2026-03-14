<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:output method="text" />
    
    <xsl:template match="/" name="xsl:initial-template">
        <!-- create text nodes that are concatenated together without spaces -->
        <xsl:sequence>
            <xsl:text expand-text="yes">{''}</xsl:text>
            <xsl:value-of select="1" />
            <xsl:text></xsl:text>
            <xsl:value-of select="'|'" />
            <xsl:text></xsl:text>
            <xsl:value-of select="2" />
            <xsl:text></xsl:text>
            <xsl:value-of select="'|'" />
            <xsl:text></xsl:text>
            <xsl:value-of select="3" />
        </xsl:sequence>
    </xsl:template>
    
</xsl:stylesheet>