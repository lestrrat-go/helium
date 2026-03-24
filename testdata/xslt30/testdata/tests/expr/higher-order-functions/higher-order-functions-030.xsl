<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- call a function computed in a parenthesized expression -->
    
 
        
    <xsl:template name="main">
    <out>    
        <xsl:value-of select="
            (if (current-date() gt xs:date('2000-12-31')) 
             then upper-case#1 else lower-case#1)('Mike')"/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>