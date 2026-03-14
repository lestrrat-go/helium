<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
    
    <xsl:strip-space elements="*"/>
    
    <xsl:template match="Element">
        <xsl:copy>
            <xsl:for-each-group select="*" group-starting-with="h1">
                <xsl:choose>
                    <xsl:when test="self::h1">
                        <group>
                            <xsl:fork>
                                <xsl:sequence>
                                    <xsl:apply-templates select="current-group()"/>
                                </xsl:sequence>
                                <xsl:sequence>
                                    <xsl:if test="count(current-group()) eq 1">
                                        <para/>
                                    </xsl:if>
                                </xsl:sequence>
                            </xsl:fork>
                        </group>
                    </xsl:when>
                    <xsl:otherwise>
                        <xsl:apply-templates select="current-group()"/>
                    </xsl:otherwise>
                </xsl:choose>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>