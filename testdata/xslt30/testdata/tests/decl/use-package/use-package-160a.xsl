<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:oupdtg="http://www.160.com/oupdtg"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:use-package name="http://www.160.com/common" package-version="0.1"/>
    
    <xsl:template match="/">
        <output>
            <xsl:call-template name="oupdtg:log"/>
            <xsl:copy-of select="oupdtg:getCity()"/>
        </output>
    </xsl:template>
        
 
</xsl:stylesheet>