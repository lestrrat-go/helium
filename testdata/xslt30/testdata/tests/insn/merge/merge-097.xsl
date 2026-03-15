<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:mf="http://example.com/mf"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="true" select="true()"/>
    
    <xsl:param name="input-uri" as="xs:string" select="'.'"/>
    
    <xsl:param name="file-pattern" as="xs:string" select="'merge-097-*.xml'"/>
    
    <xsl:param name="merge-select-expression" as="xs:string" static="yes" select="'*/*/*'"/>
    
    <xsl:param name="xslt-pattern-to-add-file-name" as="xs:string" static="yes" select="'item'"/>
    
    <xsl:output indent="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template name="xsl:initial-template">
      <out>
        <xsl:merge>
            <xsl:merge-source
                for-each-source="sort(uri-collection($input-uri || '?select=' || $file-pattern))"
                _select="{$merge-select-expression}"
                _streamable="{$STREAMABLE}">
                <xsl:merge-key select="true()"/>
            </xsl:merge-source>
            <xsl:merge-action>
                <xsl:message>Doing merge action</xsl:message>
                <xsl:sequence select="let $group-tail := tail(current-merge-group()) return mf:construct-doc(., $group-tail)"/>
            </xsl:merge-action>
        </xsl:merge>
      </out>        
    </xsl:template>
    
    <xsl:template _match="{$xslt-pattern-to-add-file-name}">
        <xsl:comment select="'Copied from ' || tokenize(document-uri(/), '/')[last()]"/>
        <xsl:next-match/>
    </xsl:template>
    
    <xsl:function name="mf:construct-doc" as="document-node()">
        <xsl:param name="first-node" as="node()"/>
        <xsl:param name="nodes" as="node()*"/>
        <xsl:apply-templates select="root($first-node)" mode="construct">
            <xsl:with-param name="nodes" select="$nodes"/>
        </xsl:apply-templates>
    </xsl:function>
    
    <xsl:mode name="construct" on-no-match="shallow-copy"/>
    
    <xsl:template _match="{$merge-select-expression}" mode="construct">
        <xsl:param name="nodes"/>
        <one><xsl:apply-templates select="."/></one>
        <two><xsl:apply-templates select="$nodes"/></two>
    </xsl:template>
    
</xsl:stylesheet>