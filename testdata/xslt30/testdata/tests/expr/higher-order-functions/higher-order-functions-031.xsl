<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Context item is a function item -->
        
    <xsl:template name="main">
    <out>    
        <xsl:value-of select="
            local-name-from-QName(function-name((upper-case#1, lower-case#1)[.('Mike') = 'MIKE']))"/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>