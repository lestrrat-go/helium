<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://localhost/fn"
    name="http://london2016/graph-lib"
    package-version="0.0.1"
    exclude-result-prefixes="xs g"
    expand-text="true"
    version="3.0">
    
    <xsl:expose component="function" names="g:transitive-closure#1 g:refers-to-self#1" visibility="public"/>
    <xsl:expose component="function" names="g:neighbours#1" visibility="abstract"/>
    
    <xsl:function name="g:neighbours" visibility="abstract" as="node()*">
        <xsl:param name="from" as="node()"/>
    </xsl:function>
    
    <xsl:function name="g:transitive-closure" visibility="public" as="node()*">
        <xsl:param name="from" as="node()"/>
        <xsl:sequence select="g:transitive-closure($from, ())"/>
    </xsl:function>
    
    <xsl:function name="g:transitive-closure" visibility="private" as="node()*">
        <xsl:param name="from" as="node()"/>
        <xsl:param name="route" as="node()*"/>
        <xsl:variable name="direct" select="g:neighbours($from)"/>
        <xsl:message>Closure: from {$from/@id}, route = {$route/@id}, direct = {$direct/@id}, diff = {($direct except $route)/@id}</xsl:message>        
        <xsl:sequence select="$direct | (($direct except $route) ! g:transitive-closure(., ($route, .)))"/>
    </xsl:function>
    
    <xsl:function name="g:refers-to-self" as="xs:boolean" visibility="public">
        <xsl:param name="from" as="node()"/>
        <xsl:sequence select="exists($from intersect g:transitive-closure($from))"/>
    </xsl:function>
    
    
</xsl:package>