<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Partial application of a inline function item -->
    

    <xsl:template name="main">
        <xsl:variable name="f" select="function($a as xs:string, $b as xs:string) { 
            starts-with($a, $b) and ends-with($a, $b)}(?, 'a')"/>
        <out>
            <xsl:value-of select="$f('abracadabra')"/>
        </out>
    </xsl:template>
    
     
 
</xsl:stylesheet>