<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- function expecting a function  -->
    
    <xsl:function name="local:case" as="xs:string">
        <xsl:param name="x" as="function(*)"/>
        <xsl:param name="y" as="xs:string"/>
        <xsl:sequence select="$x($y)"/>
    </xsl:function>
    
    
    <xsl:template name="main">
        <out><xsl:value-of select="local:case(upper-case#1, 'Mike'), local:case(lower-case#1, 'Mike')"/></out>
    </xsl:template>
    
</xsl:stylesheet>