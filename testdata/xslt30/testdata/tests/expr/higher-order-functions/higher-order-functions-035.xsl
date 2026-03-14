<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Pass a sequence of functions that require coercion in different ways -->
    
    <xsl:function name="local:apply" as="xs:string*">
        <xsl:param name="fns" as="(function(xs:string) as xs:string)*"/>
        <xsl:param name="s" as="xs:string"/>
        <xsl:sequence select="for $f in $fns return $f($s)"/>
    </xsl:function>
    
    <xsl:variable name="ops"
        select="(upper-case#1, lower-case#1, function($x){translate($x, 'e', 'i')}, substring-before(?, ' '))"/>
    
        
    <xsl:template name="main">
    <out>    
        <xsl:value-of select="local:apply($ops, 'Michael Kay')" separator='~'/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>