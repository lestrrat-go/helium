<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- inline function literal, constructor function, non default namespace  -->
    
    <xsl:param name="b" select="true()"/>
    
    <xsl:template name="main">
        <out><xsl:value-of select="let $f := (if ($b) then xs:date#1 else xs:time#1) return $f('2008-01-31')"/></out>
    </xsl:template>
    
</xsl:stylesheet>