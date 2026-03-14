<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">
    <xsl:strip-space elements="*"/>
    <xsl:output omit-xml-declaration="yes"/>
    <xsl:mode streamable="yes"/>
    <xsl:mode name="path" streamable="no"/>

    <xsl:variable name="q">"</xsl:variable>
    <xsl:param name="av" select="true()"/>
    <xsl:param name="a" select="true()"/>
    <xsl:param name="v" select="true()"/>
    <xsl:param name="dummy" select="true()"/>

    <xsl:template match="/">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>

    <xsl:template match="*">
        <p>
            <xsl:for-each select="ancestor-or-self::node()">
                <xsl:value-of select="name()"/>
                <xsl:if test="$a">
                    <xsl:for-each select="@*">
                        <xsl:value-of select="concat(&quot;[@&quot;, name())"/>
                        <xsl:if test="$av">
                            <xsl:value-of select="concat(&quot;=&quot;, $q, ., $q)"/>
                        </xsl:if>
                        <xsl:text>]</xsl:text>
                    </xsl:for-each>
                </xsl:if>
                <xsl:if test="position() != last()"><xsl:text>/</xsl:text></xsl:if>
            </xsl:for-each>
        </p>
        <xsl:apply-templates/>
    </xsl:template>
    
    <xsl:template match="c">
        <xsl:apply-templates select="text() ! snapshot(.)" mode="path"/>
    </xsl:template>
    

    <xsl:template match="text()" mode="path" priority="5">
        <t> 
           <xsl:next-match/>
        </t>
    </xsl:template>
    
    <xsl:template match="text()" mode="path" priority="4">
        <xsl:if test="$dummy">
            <xsl:for-each select="ancestor-or-self::node()">
                <xsl:value-of select="name()"/>
                <xsl:if test="$a">
                    <xsl:for-each select="@*">
                        <xsl:value-of select="concat(&quot;[@&quot;, name())"/>
                        <xsl:if test="$av">
                            <xsl:value-of select="concat(&quot;=&quot;, $q, ., $q)"/>
                        </xsl:if>
                        <xsl:text>]</xsl:text>
                    </xsl:for-each>
                </xsl:if>
                <xsl:if test="position() != last()"><xsl:text>/</xsl:text></xsl:if>
            </xsl:for-each>
            <xsl:text>text()</xsl:text>
            <xsl:if test="$v">
                <xsl:text>[.="</xsl:text><xsl:value-of select="."/><xsl:text>"]</xsl:text>
            </xsl:if>       
        </xsl:if>
    </xsl:template>

   

</xsl:stylesheet>
