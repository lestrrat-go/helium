<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- numeric promotion works for various kinds of functions  -->
    

    
    <xsl:function name="local:round" as="xs:double">
        <xsl:param name="x" as="xs:double"/>
        <xsl:sequence select="floor($x)"/>
    </xsl:function>
    
   
    <xsl:function name="local:ops"  as="(function(xs:double) as xs:double)*">
        <xsl:sequence select="(
            abs#1, 
            local:round#1, 
            function($x){$x+1}, 
            round-half-to-even(?, 2))"/>
    </xsl:function>    
     
    <xsl:template name="main">
    <out>
        <xsl:value-of select="for $f in local:ops() return string(round-half-to-even($f(xs:decimal('123.456')), 4))" separator='~'/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>