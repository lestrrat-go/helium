<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- inline function literal, imported used-defined function -->
    
    <xsl:import href="higher-order-functions-001.xsl"/>
    
    <xsl:param name="b" select="true()"/>
    
    <xsl:template name="main">
        <out><xsl:value-of select="let $f := (if ($b) then local:f#1 else abs#1) return $f(4)"/></out>
    </xsl:template>
    
</xsl:stylesheet>