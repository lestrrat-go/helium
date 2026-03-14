<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- return an inline function that accesses global variables from a user-defined function -->
    
    <xsl:variable name="sep" as="xs:string" select="'\s'"/>
    
    <xsl:function name="local:splitter" as="(function(xs:string) as xs:string*)?">
        <xsl:sequence select="function($x as xs:string) { for $i in tokenize($x, $sep) return upper-case($i) }"/> 
    </xsl:function>
        
    <xsl:template name="main">
    <out>    
        <xsl:value-of select="local:splitter()('A nice cup of tea')" separator="|"/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>