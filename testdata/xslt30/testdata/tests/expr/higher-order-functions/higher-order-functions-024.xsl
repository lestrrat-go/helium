<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- get the name and arity of an inline function -->
    
    <xsl:function name="local:scramble" as="xs:string">
        <xsl:param name="x" as="function(xs:string) as xs:string"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
        
    <xsl:template name="main">
        <xsl:variable name="f" select="function($x as xs:string) as xs:string { upper-case($x) }"/>
        <xsl:variable name="n" select="function-name($f)"/>
        <out>
            <a loc="{local-name-from-QName($n)}" uri="{namespace-uri-from-QName($n)}"
            arity="{function-arity($f)}" eloc="{empty(local-name-from-QName($n))}" 
            euri="{empty(namespace-uri-from-QName($n))}"/></out>
    </xsl:template>
    
</xsl:stylesheet>