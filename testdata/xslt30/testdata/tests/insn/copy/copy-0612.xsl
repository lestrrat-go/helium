<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:f="http://local-functions/"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="f xs">

    <!-- test combinations of copy-namespaces and inherit-namespaces -->
    
    <xsl:param name="INHERIT" static="yes" select="true()"/>
    <xsl:param name="COPY" static="yes" select="true()"/>
    
    <xsl:variable name="outer">
        <a xmlns="uri:a">
            <b:b xmlns:b="uri:b">
                <c xmlns="uri:c" xmlns:d="uri:d">
                    <graft xmlns=""/>
                </c>
            </b:b>
        </a>
    </xsl:variable>
    
    <xsl:variable name="inner">
        <p xmlns="uri:p">
            <q xmlns="uri:q">
                <r xmlns="uri:r" xmlns:s="uri:s"/>
            </q>
        </p>
    </xsl:variable>
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template match="*">
        <xsl:copy _inherit-namespaces="{$INHERIT}">
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="graft">
        <xsl:copy-of select="$inner" _copy-namespaces="{$COPY}"/>
    </xsl:template>

    <xsl:output method="xml" encoding="utf-8"/>
    <xsl:template name="main">
        <xsl:variable name="capture">
            <xsl:apply-templates select="$outer"/>
        </xsl:variable>
        <out>
            <xsl:for-each select="$capture//*">
                <element name="{local-name()}" uri="{namespace-uri()}" in-scope="{f:in-scope-namespaces(.)}"/>
            </xsl:for-each>
        </out>
    </xsl:template>

    <xsl:function name="f:in-scope-namespaces" as="xs:string">
        <xsl:param name="e" as="element(*)"/>
        <xsl:value-of>
            <xsl:for-each select="in-scope-prefixes($e)[. != 'xml']">
                <xsl:sort select="."/>
                <xsl:value-of select="concat('|', ., '=', namespace-uri-for-prefix(., $e))"/>
            </xsl:for-each> 
        </xsl:value-of>       
    </xsl:function>

</xsl:stylesheet>
