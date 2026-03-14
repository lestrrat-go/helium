<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:output indent="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:mode streamable="yes"/>
    
    <xsl:template match="/">
        <root>
            <xsl:for-each-group select="copy-of(root/items/item)" group-adjacent="foo">
                <group key="{current-grouping-key()}" group-nr="{position()}">
                    <xsl:copy-of select="current-group()"/>
                </group>
            </xsl:for-each-group>
        </root>
    </xsl:template>
    
</xsl:stylesheet>