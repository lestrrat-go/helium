<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="input-uri1" select="'stream-211a.xml'"/>
    
    <xsl:param name="input-uri2" select="'stream-211b.xml'"/>
    <xsl:param name="doc2" select="doc($input-uri2)"/>
    
    <xsl:strip-space elements="*"/>
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:key name="change" match="change" use="@id"/>
    
    <xsl:template name="xsl:initial-template">
        <xsl:source-document streamable="true" href="{$input-uri1}">
            <xsl:apply-templates/>
        </xsl:source-document>
    </xsl:template>
    
    <xsl:template match="item[key('change', @id, $doc2)]/foo">
        <xsl:copy>
            <xsl:value-of select="key('change', ../@id, $doc2)"/>
        </xsl:copy>
    </xsl:template>
</xsl:stylesheet>