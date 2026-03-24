<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- curry a system function -->
        
    <xsl:template name="main">
    <out>    
        <xsl:variable name="f" select="contains(?, 'e', default-collation())"/>
        <xsl:for-each select='("Mike", "John", "Dave", "Mary", "Jane")'>
            <xsl:sequence select="$f(.)"/>
        </xsl:for-each> 
    </out>
    </xsl:template>
    
</xsl:stylesheet>