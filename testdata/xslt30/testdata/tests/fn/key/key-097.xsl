<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:key name="map" match="Message1/Response/CE//*[action = 'New']" composite="yes">
        <xsl:sequence select="ancestor::CE/id"/>
        <xsl:sequence select="node-name()"/>
        <xsl:variable name="pos" as="xs:integer">
            <xsl:number/>
        </xsl:variable>
        <xsl:sequence select="$pos"/>
    </xsl:key>
    
    <xsl:template match="Messages">
        <xsl:apply-templates select="Message2"/>
    </xsl:template>
    
    <xsl:template match="Message2/Response/CE//*[key('map', (ancestor::CE/id, node-name(), count((., preceding-sibling::*[node-name() = node-name(current())]))))]">
        <xsl:copy-of select="key('map', (ancestor::CE/id, node-name(), count((., preceding-sibling::*[node-name() = node-name(current())]))))"/>
    </xsl:template>
    
</xsl:stylesheet>