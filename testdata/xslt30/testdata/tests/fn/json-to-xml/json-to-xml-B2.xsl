<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:j="http://www.w3.org/2005/xpath-functions"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="j xs">

    <xsl:param name="input-url" required="yes"/>
    
    <xsl:output indent="yes"/>

    <xsl:template name="xsl:initial-template">
        <xsl:variable name="input-as-xml">
            <array xmlns="http://www.w3.org/2005/xpath-functions">
                <xsl:for-each select="unparsed-text-lines($input-url)">
                    <xsl:copy-of select="json-to-xml(.)"/>
                </xsl:for-each>
            </array>
        </xsl:variable>
        <xsl:for-each-group select="$input-as-xml/j:array/*" group-by=".//j:string[@key='country']">
            <xsl:result-document href="weather-{current-grouping-key()}.xml">
                <city-list>
                    <xsl:apply-templates select="current-group()">
                        <xsl:sort select="(.//j:string[@key='sv'], .//j:string[@key='name'])[1]" lang="sv" case-order='upper-first'/>
                    </xsl:apply-templates>
                </city-list>    
            </xsl:result-document>
        </xsl:for-each-group>
    </xsl:template>
    
    <xsl:template match="j:map[not(@key)]">
        <record>
            <xsl:apply-templates/>
        </record>
    </xsl:template>
    
    <xsl:template match="j:map[not(@key)][count(*)=1]" priority="2">
        <xsl:apply-templates/>
    </xsl:template>
    
    <xsl:template match="*[@key[. castable as xs:NCName]]">
        <xsl:element name="{@key}">
            <xsl:apply-templates/>
        </xsl:element>
    </xsl:template>
    
    <xsl:template match="*[@key[not(. castable as xs:NCName)]]">
        <xsl:element name="_{@key}">
            <xsl:apply-templates/>
        </xsl:element>
    </xsl:template>
    
    <xsl:template match="*[@key='coord']" priority="3"/>


</xsl:stylesheet>
