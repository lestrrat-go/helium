<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- function expecting a function, caller supplies inline function. Needs function coercion  -->
    
    <xsl:function name="local:scramble" as="xs:string">
        <xsl:param name="x" as="function(xs:string) as xs:string"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
        
    <xsl:template name="main">
        <out><xsl:value-of select="local:scramble(
            function($x){translate($x, 'abcdefghijklmnopqrstuvwxyz', 'nopqrstuvwxyzabcdefghijklm')}, 'john')"/></out>
    </xsl:template>
    
</xsl:stylesheet>