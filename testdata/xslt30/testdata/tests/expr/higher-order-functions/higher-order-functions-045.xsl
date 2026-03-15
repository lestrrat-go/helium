<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Partial application of a literal function item -->
    
 
    
    <xsl:template name="main">
        <xsl:variable name="f" select="substring-before#2(?, '-')"/>
        <out>
            <xsl:value-of select="$f('the-end-of-the-world')"/>
        </out>
    </xsl:template>
    
     
 
</xsl:stylesheet>