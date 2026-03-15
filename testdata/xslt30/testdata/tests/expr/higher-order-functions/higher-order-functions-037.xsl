<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Implicit atomization works for a variety of kinds of function -->
    
    <xsl:function name="local:lower-case" as="xs:string">
        <xsl:param name="x" as="xs:string"/>
        <xsl:sequence select="concat('''', lower-case($x), '''')"/>
    </xsl:function>
    
   
    <xsl:function name="local:ops" as="(function(xs:string) as xs:string)*">
        <xsl:sequence select="(
            upper-case#1, 
            local:lower-case#1, 
            function($x){translate($x, 'e', 'i')}, 
            substring-before(?, ' '))"/>
    </xsl:function>    
     
    <xsl:template name="main">
    <out>
        <xsl:variable name="a" as="element()">
            <a name="Michael Kay"/> 
        </xsl:variable>
        <xsl:value-of select="for $f in local:ops() return $f($a/@name)" separator='~'/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>