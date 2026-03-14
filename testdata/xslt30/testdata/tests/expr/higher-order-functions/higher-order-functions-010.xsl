<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- sequence type function(x,z) as y  -->
    
    <xsl:function name="local:f" as="xs:integer">
        <xsl:param name="x" as="xs:integer"/>
        <xsl:param name="y" as="xs:long"/>
        <xsl:sequence select="$x + $y"/>
    </xsl:function>
    
    <xsl:param name="f" as="function(xs:integer, xs:long) as xs:integer" select="local:f#2"/>
    
    <xsl:template name="main">        
        <out><xsl:value-of select="$f(2, xs:long(5))"/></out>
    </xsl:template>
    
</xsl:stylesheet>