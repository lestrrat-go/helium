<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- 
         Partial apply
         Pass a function item that can be coerced to the required type (same arity) but has wrong 
         argument type and return type. Allows both XPTY0004 (if the param uses late binding after
         coercion and the error in the coercion wrapper is not detected during coercion) or
         XTTE0790 (if the wrong return type, xs:integer but xs:string expected, is found during
         coercion but prior to function evaluation in the stylesheet function body)
    -->
    
    <xsl:function name="local:round" as="xs:double">
        <xsl:param name="x" as="xs:double"/>
        <xsl:param name="algorithm" as="(function(xs:double) as xs:double)"/>
        <xsl:sequence select="$algorithm($x)"/>
    </xsl:function>
    

    <xsl:variable name="roundToCeiling" select="local:round(?, upper-case#1)"/>
 
    <xsl:template name="main">
        <out>
            <xsl:value-of select="$roundToCeiling(12.4)"/>
        </out>
    </xsl:template>
    
 
     
 
</xsl:stylesheet>