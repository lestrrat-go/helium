<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
    
    <xsl:strip-space elements="*"/>
    
    <xsl:template match="contents">
        <xsl:copy>
            <xsl:for-each-group select="unnumbered" group-starting-with="unnumbered[@type = 'PT']">
                <xsl:copy>
                    <xsl:copy-of select="@*"/>
                    <xsl:fork>
                        <xsl:sequence>
                            <xsl:copy-of select="node()"/>
                        </xsl:sequence>
                        <xsl:sequence>
                            <xsl:apply-templates select="current-group() except ."/>
                        </xsl:sequence>
                    </xsl:fork>
                </xsl:copy>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="subtitle"/>
    
</xsl:stylesheet>