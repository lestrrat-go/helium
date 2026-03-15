<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:strip-space elements="*"/>
    
    <xsl:mode streamable="yes"/>
    
    <xsl:template match="doc">
        <xsl:copy>
            <xsl:for-each-group select="*" group-adjacent="node-name(.)">
                <xsl:copy>
                    <xsl:value-of select="current-group()" separator=","/>
                </xsl:copy>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>