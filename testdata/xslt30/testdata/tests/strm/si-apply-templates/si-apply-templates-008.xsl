<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode on-no-match="shallow-copy" _streamable="{$STREAMABLE}"/>
    
    <xsl:template match="/">
        <xsl:apply-templates>
            <xsl:with-param name="prefix" tunnel="no" as="xs:string" select="'['"/>
            <xsl:with-param name="suffix" tunnel="yes" as="xs:string" select="']'"/>
        </xsl:apply-templates>
    </xsl:template>
    
    <xsl:template match="w/@id">
        <xsl:param name="prefix" tunnel="no"/>
        <xsl:param name="suffix" tunnel="yes"/>
        <xsl:attribute name="{name()}" select="$prefix || . || $suffix"/>
    </xsl:template>
    
</xsl:stylesheet>