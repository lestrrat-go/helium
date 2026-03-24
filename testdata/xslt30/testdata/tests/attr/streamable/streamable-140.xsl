<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
    exclude-result-prefixes="xs math map" 
    version="3.0">
    
    <xsl:output indent="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:template match="coordinates">
        <xsl:copy>
            <xsl:apply-templates
                select="map { 'lat' : xs:decimal(@lat), 'long' : xs:decimal(@long) }" 
            />
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match=".[. instance of map(xs:string, xs:decimal)]">
        <lat>
            <xsl:value-of select="map:get(., 'lat')"/>
        </lat>
        <long>
            <xsl:value-of select="map:get(., 'long')"/>
        </long>
    </xsl:template>
    
</xsl:stylesheet>