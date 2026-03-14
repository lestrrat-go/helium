<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:scm="http://ns.saxonica.com/schema-component-model"
    xmlns:e="http://www.w3.org/1999/XSL/Spec/ElementSyntax"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs scm err e"
    version="3.0">
    
    <!-- Validate all examples in the spec -->
    
    
    <xsl:param name="spec" select="doc('xslt-expanded1.xml')"/>
    <xsl:import-schema namespace="http://www.w3.org/1999/XSL/Transform" schema-location="schema-for-xslt30.xsd"/>

    <xsl:template name="main">
        <out>
            <xsl:variable name="raw">
            <xsl:for-each select="$spec//eg">
                <xsl:try>
                    <xsl:apply-templates select="."/>
                    <xsl:catch>
                        <error processing="{path(.)} in {ancestor-or-self::*[@id]/@id}" description="{$err:description}">
                            <input><xsl:value-of select="." disable-output-escaping="yes"/></input>
                            <details><xsl:copy-of select="$err:value"/></details>
                        </error>
                    </xsl:catch>
                </xsl:try>
            </xsl:for-each>
            </xsl:variable>
            <xsl:copy-of select="$raw/error"/>
        </out>
    </xsl:template>
    
    <xsl:template match="eg[@role='xml']">
        <xsl:variable name="parsed" select="parse-xml-fragment(.)"/><!-- avoid validation-->
        <xsl:copy-of select="$parsed"/>
    </xsl:template>
    
    <xsl:template match="eg[@role='xslt-fragment']">
        <xsl:variable name="in" as="xs:string" expand-text="yes">
           &lt;z xmlns:xsl="http://www.w3.org/1999/XSL/Transform">{.}&lt;/z>
        </xsl:variable>   
        <xsl:variable name="parsed" select="parse-xml-fragment($in)"/><!-- avoid validation-->
        <xsl:copy-of select="$parsed" validation="strip"/>
    </xsl:template>
    
    <xsl:template match="eg[@role='xslt-document']">
        <xsl:variable name="parsed" select="parse-xml(.)"/>
        <xsl:copy-of select="$parsed" validation="strict"/>
    </xsl:template>
    
    <xsl:template match="eg[starts-with(@role,'xslt-declaration')]">
        <xsl:variable name="ns" select="substring-after(@role, ' ')"/>;
        <xsl:variable name="in" select="'&lt;xsl:transform xmlns:xsl=&quot;http://www.w3.org/1999/XSL/Transform&quot; xmlns:xs=&quot;http://www.w3.org/2001/XMLSchema&quot; ' 
            || $ns || '>' || string(.) || '&lt;/xsl:transform>'"/>
        <xsl:variable name="parsed" select="parse-xml($in)"/>
        <xsl:copy-of select="$parsed" validation="strict"/>
    </xsl:template>
    
    <xsl:template match="eg[starts-with(@role,'xslt-instruction')]">
        <xsl:variable name="ns" select="substring-after(@role, ' ')"/>;
        <xsl:variable name="in" select="'&lt;xsl:transform xmlns:xsl=&quot;http://www.w3.org/1999/XSL/Transform&quot;>&lt;xsl:template match=&quot;*&quot; ' 
            || $ns || '>' || string(.) || '&lt;/xsl:template>&lt;/xsl:transform>'"/>
        <xsl:variable name="parsed" select="parse-xml($in)"/>
        <xsl:copy-of select="$parsed" validation="strict"/>
    </xsl:template>

    
    
    <xsl:output indent="yes"/>
    
</xsl:stylesheet>