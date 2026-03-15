<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" 
        xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
        xmlns:xs="http://www.w3.org/2001/XMLSchema" 
        xmlns:xfn="http://www.w3.org/2003/11/xpath-functions"
        xmlns:zz="http://www.xxx.com/zz" 
        exclude-result-prefixes="xsl xs xfn zz"
        >
        
        <?spec xpath#id-path-expressions?>
        <?same-as-1.0 no?>

        <xsl:output method="xml" version="1.0" omit-xml-declaration="yes" encoding="UTF-8" indent="no"/>

        <xsl:variable name="dir">/</xsl:variable>
        
        <xsl:template match="/">
        <out>
                <a><xsl:value-of select="zz:build(//page[@title = 'D'])"/></a>
                <b><xsl:apply-templates mode="build" select="//page[@title = 'D']"/></b>
        </out>
        </xsl:template>
        
        <xsl:template match="page" mode="build">
                <xsl:variable name="ancestors" select="ancestor-or-self::page/@title"/>
                <xsl:variable name="levels" select="count($ancestors)"/>
                <xsl:for-each select="1 to $levels">
                        <xsl:value-of select="for $i in 1 to . return $ancestors[$i]" separator="_"/>
                        <xsl:if test=". &lt; $levels"><xsl:value-of select="$dir"/></xsl:if>
                </xsl:for-each>
        </xsl:template>
        
        <xsl:function name="zz:build">
                <xsl:param name="context"/>
                <xsl:variable name="ancestors" select="$context/ancestor-or-self::page/@title"/>
                <xsl:variable name="levels" select="count($ancestors)"/>
                <xsl:value-of>
                        <xsl:for-each select="1 to $levels">
                                <xsl:value-of>
                                        <xsl:value-of select="for $i in 1 to . return $ancestors[$i]" separator="_"/>
                                        <xsl:if test=". &lt; $levels"><xsl:value-of select="$dir"/></xsl:if>
                                </xsl:value-of>
                        </xsl:for-each>
                </xsl:value-of>
        </xsl:function>
        
</xsl:stylesheet>

