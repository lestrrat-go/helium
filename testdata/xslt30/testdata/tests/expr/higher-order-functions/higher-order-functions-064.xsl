<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Partial apply, argument invalid for target function -->

    <xsl:variable name="ops" select="substring-before(?, 2)"/>
 
    <xsl:template name="main">
        <out>
            <xsl:value-of select="$ops('Michael Kay')" separator="~"/>
        </out>
    </xsl:template>
    
 
     
 
</xsl:stylesheet>