<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" xmlns:mf="http://example.com/mf" 
    exclude-result-prefixes="xs math mf" version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:param name="prefix" as="xs:string" select="'l'"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
    
    <xsl:mode name="num" _streamable="{$STREAMABLE}"/>
    
    <xsl:function name="mf:nest" as="node()*" 
        _streamability="{if ($STREAMABLE) then 'absorbing' else 'unclassified'}">
        <xsl:param name="input" as="node()*"/>
        <xsl:param name="level" as="xs:integer"/>
        <xsl:where-populated>
            <orderlist type="manual">
                <xsl:for-each-group select="$input" 
                    group-starting-with="*[@class = $prefix || $level]">
                    <item>
                        <xsl:fork>
                            <xsl:sequence>
                                <xsl:apply-templates select="node()[1]" mode="num"/>
                            </xsl:sequence>
                            <xsl:sequence>
                                <para>
                                    <xsl:apply-templates select="node()[position() gt 2]"/>
                                </para>                               
                            </xsl:sequence>
                            <xsl:sequence>
                                <rest size="{count(current-group())}"/>
                            </xsl:sequence>
                        </xsl:fork>                     
                    </item>
                </xsl:for-each-group>
            </orderlist>
        </xsl:where-populated>
    </xsl:function>
    
    <xsl:template match="root">
        <xsl:copy>
            <xsl:sequence select="mf:nest(*, 1)"/>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="*[matches(@class, $prefix || '[0-9]+')]/node()" mode="num">
        <xsl:attribute name="num" select="replace(., '^\s+', '')"/>
    </xsl:template>
    
</xsl:stylesheet>