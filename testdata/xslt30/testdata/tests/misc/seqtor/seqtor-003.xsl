<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:output method="text" />
    
    <xsl:template match="/" name="xsl:initial-template">
        <!-- create text nodes that are concatenated together without spaces -->
        <xsl:for-each select="1 to 4">
            <xsl:value-of select="." />
            <xsl:value-of select="'|'" />
        </xsl:for-each>
    </xsl:template>
    
</xsl:stylesheet>