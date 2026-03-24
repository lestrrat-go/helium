<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:use-package name="http://example.com/override-m-021-base" package-version="*">
        <xsl:override>
            <xsl:template match="items" mode="m1">
                <list>
                    <xsl:apply-templates mode="#current"/>
                </list>
            </xsl:template>
        </xsl:override>
    </xsl:use-package>
    
    <xsl:strip-space elements="*"/>
    
    <xsl:template match="/">
        <xsl:apply-templates select="." mode="m1"/>
    </xsl:template>
    
</xsl:stylesheet>