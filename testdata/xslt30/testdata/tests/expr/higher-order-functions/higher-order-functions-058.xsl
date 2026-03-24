<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Heterogeneous sequence on rhs of "/" -->
 
    
    <xsl:variable name="f" as="element(*)"><a b="3"/></xsl:variable>
    
 
    <xsl:template name="main">
        <out>
            <xsl:value-of select="$f/(@b, upper-case#1)"/>
        </out>
    </xsl:template>
    
 
     
 
</xsl:stylesheet>