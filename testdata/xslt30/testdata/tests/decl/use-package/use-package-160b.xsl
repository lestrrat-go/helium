<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:oupdtg="http://www.160.com/oupdtg" 
    version="3.0">
    
    <xsl:use-package name="http://www.160.com/runner" package-version="0.1"/>
    
    <xsl:template match="/">
        <xsl:apply-templates select="*" mode="transform"/>
    </xsl:template>
        
 
</xsl:stylesheet>