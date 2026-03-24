<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- get the name and arity of a constructor function -->
    
    <xsl:function name="local:scramble" as="xs:string">
        <xsl:param name="x" as="function(xs:string) as xs:string"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
        
    <xsl:template name="main">
        <out><xsl:value-of select="
            let $f := xs:dateTime#1, $n := function-name($f) 
            return (local-name-from-QName($n), namespace-uri-from-QName($n), function-arity($f))"/></out>
    </xsl:template>
    
</xsl:stylesheet>