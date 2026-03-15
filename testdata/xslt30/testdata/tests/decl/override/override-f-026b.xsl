<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://localhost/fn"
    name="http://london2016/graph-idref"
    package-version="0.0.1"
    exclude-result-prefixes="xs g"
    expand-text="yes"
    version="3.0">
    
    <xsl:key name="id" match="*" use="@id"/>
    
    <xsl:use-package name="http://london2016/graph-lib">
        
        <xsl:accept component="function" names="g:refers-to-self#1" visibility="private"/>
        
        <xsl:override>
            <xsl:function name="g:neighbours" visibility="private" as="node()*">
                <xsl:param name="from" as="node()"/>
                <xsl:message>Neighbours: {$from/@idref} resolving to {key('id', tokenize($from/@idref, '\s+'), root($from))/@id}</xsl:message>
                <xsl:sequence select="key('id', tokenize($from/@idref, '\s+'), root($from))"/>
            </xsl:function>
        </xsl:override>
        
    </xsl:use-package>
    
    <xsl:variable name="test-data" visibility="private">
        <nodes>
            <node id="a" idref="b c d"/>
            <node id="b"/>
            <node id="c"/>
            <node id="d" idref="e f"/>
            <node id="e"/>
            <node id="f" idref="a"/>
        </nodes>
    </xsl:variable>
    
    <xsl:template name="xsl:initial-template" visibility="public">
        <results>
            <a-is-cyclic>{g:refers-to-self($test-data//node[@id='a'])}</a-is-cyclic>
            <e-is-cyclic>{g:refers-to-self($test-data//node[@id='e'])}</e-is-cyclic>
        </results>
    </xsl:template>
    
    
</xsl:package>