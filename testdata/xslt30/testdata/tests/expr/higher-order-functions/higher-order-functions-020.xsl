<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- get the name of a user-defined function -->
    
    <xsl:function name="local:scramble" as="xs:string">
        <xsl:param name="x" as="function(xs:string) as xs:string"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
        
    <xsl:template name="main">
        <out><xsl:value-of select="
            let $n := function-name(local:scramble#2) 
            return (local-name-from-QName($n), namespace-uri-from-QName($n), function-arity(local:scramble#2))"/></out>
    </xsl:template>
    
</xsl:stylesheet>