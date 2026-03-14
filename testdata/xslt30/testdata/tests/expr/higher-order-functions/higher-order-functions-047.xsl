<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Repeated partial application of a function item -->
    
    <xsl:template name="main">
        <xsl:variable name="f" select="concat#3(?, '*', ?)"/>
        <xsl:variable name="g" select="$f('[', ?)"/>
        <out>
            <xsl:value-of select="$g(']')"/>
        </out>
    </xsl:template>
    
     
 
</xsl:stylesheet>