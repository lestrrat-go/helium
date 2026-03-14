<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
    exclude-result-prefixes="xs math map" 
    version="3.0">
    
    <xsl:mode on-no-match="shallow-copy" streamable="yes" use-accumulators="item-group"/>

    
    <xsl:accumulator name="item-group" as="map(xs:string, xs:integer)" initial-value="map{}" streamable="yes">
        <xsl:accumulator-rule match="item" 
            select="if (map:contains($value, @class))
            then map:put($value, string(@class), $value(@class) + 1)
            else map:put($value, string(@class), 1)"/>
    </xsl:accumulator>
    
    <xsl:template match="item">
        <xsl:copy>
            <xsl:variable name="abefore" select="accumulator-before('item-group')" as="map(*)"/>
            <xsl:if test="$abefore(@class) eq 1">
                <xsl:attribute name="group-head" select="@class"/>
            </xsl:if>
            <xsl:apply-templates select="@*"/>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>