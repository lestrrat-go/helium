<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="input" as="xs:string">{ "prop1" : "value 1", "prop2" : "value 2" }</xsl:param>
    
    <xsl:output method="json" indent="yes"/>
    
    <xsl:mode on-multiple-match="fail"/>
    
    <xsl:template name="xsl:initial-template" match="/">
        <out>
            <xsl:apply-templates select="parse-json($input)"/>
        </out>
    </xsl:template>
    
    <xsl:template match=".[. instance of map(*)]">
        <A/>
    </xsl:template>
    
    <xsl:template match=".[. instance of map(*)]">
        <B/>
    </xsl:template>
    
</xsl:stylesheet>