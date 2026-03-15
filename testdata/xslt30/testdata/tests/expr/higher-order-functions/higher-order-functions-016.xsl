<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- function expecting a function, caller supplies local function  -->
    
    <xsl:function name="local:scramble" as="xs:string">
        <xsl:param name="x" as="function(xs:string) as xs:string"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
    
    <xsl:function name="local:rot13" as="xs:string">
        <xsl:param name="x" as="xs:string"/>
        <xsl:sequence select='translate($x, "abcdefghijklmnopqrstuvwxyz", "nopqrstuvwxyzabcdefghijklm")'/>
    </xsl:function>
        
    <xsl:template name="main">
        <out><xsl:value-of select="local:scramble(local:rot13#1, 'mike')"/></out>
    </xsl:template>
    
</xsl:stylesheet>