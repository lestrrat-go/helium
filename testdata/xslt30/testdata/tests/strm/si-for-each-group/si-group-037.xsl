<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:template match="p">
        <xsl:copy>
            <xsl:for-each-group select="node()" group-adjacent="node-name(.), @class" composite="true">
                <xsl:copy>
                    <xsl:copy-of select="@class"/>
                    <xsl:apply-templates select="current-group()/node()"/>
                </xsl:copy>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
 
    
</xsl:stylesheet>