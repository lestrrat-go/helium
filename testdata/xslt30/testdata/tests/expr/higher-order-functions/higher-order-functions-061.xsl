<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!--
         Pass a sequence of functions that can be coerced to the required type (same arity) 
         but have wrong argument type. Allows both XPTY0004 (if the param uses late binding after
         coercion and the error in the coercion wrapper is not detected during coercion) or
         XTTE0790 (if the wrong return type, xs:integer but xs:string expected, is found during
         coercion but prior to function evaluation in the stylesheet function body)
    -->
 
 
    <xsl:function name="local:apply" as="xs:string">
        <xsl:param name="fns" as="(function(xs:string) as xs:string)*"/>
        <xsl:param name="s" as="xs:string"/>
        <!-- this may or may not be evaluated with the coerced function wrapper and raise XPTY0004 -->
        <xsl:value-of select="for $f in $fns return $f($s)" />
    </xsl:function>
    
    <!-- third seq item contains function item with incorrect argument type, error may be 
        raised early during coercion (XTTE0790) or late during function body evaluation 
        (XPTY0004, but not XTTE0780, because it is not the return type of the stylesheet 
        function body that is wrong, but the evaluation of the function item) -->
    <xsl:variable name="ops" as="item()*" 
        select="(upper-case#1, lower-case#1, function($x as xs:double){string($x)})"/>
    
 
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:value-of select="local:apply($ops, 'Michael Kay')" separator="~"/>
        </out>
    </xsl:template>
    
 
     
 
</xsl:stylesheet>