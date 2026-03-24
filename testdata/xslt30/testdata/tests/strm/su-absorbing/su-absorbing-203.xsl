<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    xmlns:mf="http://example.com/mf" 
    exclude-result-prefixes="xs math mf" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
    
    <!-- differs from su-absorbing-202 by declaring the required type of the result -->
    
    <xsl:function name="mf:group" as="element(Item)*" _streamability="{if ($STREAMABLE) then 'absorbing' else 'unclassified'}">
        <xsl:param name="items" as="element(Item)*"/>
        <xsl:param name="level" as="xs:integer"/>
        <xsl:for-each-group select="$items" group-starting-with="Item[string-length(@Value) eq $level]">
            <xsl:choose>
                <xsl:when test="self::Item[string-length(@Value) eq $level]">
                    <xsl:copy>
                        <xsl:copy-of select="@*"/>
                        <xsl:sequence select="mf:group(tail(current-group()), $level + 1)"/>
                    </xsl:copy>
                </xsl:when>
                <xsl:otherwise>
                    <xsl:apply-templates select="mf:group(current-group(), $level + 1)"/>
                </xsl:otherwise>
            </xsl:choose>
        </xsl:for-each-group>
    </xsl:function>
    
    <xsl:template match="Items">
        <xsl:copy>
            <xsl:copy-of select="@*"/>
            <xsl:sequence select="mf:group(Item, 2)"/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>