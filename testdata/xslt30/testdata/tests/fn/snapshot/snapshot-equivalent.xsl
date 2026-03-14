<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:f="http://internal.snapshot.com/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs f" version="3.0">

    
    <xsl:function name="f:snapshot" as="item()*">
        <xsl:param name="input" as="item()*"/>
        <xsl:apply-templates select="$input" mode="snapshot"/>
    </xsl:function>
    
    <!-- for atomic values and function items, return the item unchanged -->
    
    <xsl:template match="." mode="snapshot">
        <xsl:sequence select="."/>
    </xsl:template>
    
    <!-- for a parentless node, return a deep copy -->
    
    <xsl:template match="/" mode="snapshot">
        <xsl:copy-of select="."/>
    </xsl:template>
    
    <!-- for an element, comment, or processing instruction: -->
    
    <xsl:template match="node()" mode="snapshot" as="node()">
        <xsl:sequence select="f:graft-to-parent(., .., function($n){$n/child::node()})"/>
    </xsl:template>
    
    <!-- for an attribute with a parent, graft it to a copy of its parent;
         then return the corresponding attribute this copied parent. -->
    
    <xsl:template match="@*" mode="snapshot" as="attribute()">
        <xsl:variable name="name" select="node-name(.)"/>
        <xsl:sequence select="f:graft-to-parent(., .., function($n){$n/@*[node-name(.) = $name]})"/>
    </xsl:template>
    
    <!-- for a namespace node with a parent, graft it to a copy of its parent;
         then return the corresponding namespace node of this copied parent. -->
    
    <xsl:template match="namespace-node()" mode="snapshot" as="namespace-node()">
        <xsl:variable name="name" select="local-name(.)"/>
        <xsl:sequence select="f:graft-to-parent(., .., function($n){$n/namespace-node()[local-name(.) = $name]})"/>
    </xsl:template>
    
    <!-- make a copy of the parent of a supplied node, with a copy of the supplied node attached
         as a child/attribute/namespace of the new parent as appropriate -->
    
    <xsl:function name="f:graft-to-parent" as="node()">
        <xsl:param name="n" as="node()"/>
        <xsl:param name="original-parent" as="node()?"/>
        <xsl:param name="down-function" as="function(node()) as node()"/>
        <!--<xsl:message>graft {path($n)} original-parent: {path($original-parent)}</xsl:message>-->
        <xsl:choose>
            <xsl:when test="$original-parent">
                <xsl:variable name="p" as="node()">
                    <xsl:copy select="$original-parent">
                        <xsl:copy-of select="@*"/>
                        <xsl:copy-of select="$n"/>
                    </xsl:copy>
                </xsl:variable>
                <xsl:sequence select="$down-function(f:graft-to-parent($p, $original-parent/.., function($n){$n/child::node()}))"/>
            </xsl:when>
            <xsl:otherwise>
                <xsl:sequence select="$n"/>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:function>
    

</xsl:stylesheet>
