<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl" 
    version="3.0">
    
    <xsl:use-package name="http://www.oup.com/testParamOverride" package-version="0.1">
        <xsl:accept component="mode" names="transform" visibility="public"/>
        
        <xsl:override>
            <xsl:param name="newRootElement" select="'override-success'" as="xs:string"/>
        </xsl:override>
    </xsl:use-package>
    
    <xsl:template match="/">
        <xsl:apply-templates select="/" mode="transform"/>
    </xsl:template>
    
</xsl:stylesheet>