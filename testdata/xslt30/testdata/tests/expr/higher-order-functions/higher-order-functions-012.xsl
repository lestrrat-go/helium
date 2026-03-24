<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- function returning a function  -->
    
    <xsl:function name="local:case" as="function(*)">
        <xsl:param name="x" as="xs:boolean"/>
        <xsl:sequence select="if ($x) then upper-case#1 else lower-case#1"/>
    </xsl:function>
    
    
    <xsl:template name="main">
        <out><xsl:value-of select="local:case(true())('Mike'), local:case(false())('Mike')"/></out>
    </xsl:template>
    
</xsl:stylesheet>