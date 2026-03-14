<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:mode streamable="yes"/>
    
    <xsl:output indent="no"/>
    
    <xsl:template match="note">
        <xsl:copy>
            <xsl:for-each-group select="*/text()" group-starting-with="text()[ends-with(., ':')]">
                <group>
                    <xsl:copy-of select="snapshot(current-group())/.."/>
                </group>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>