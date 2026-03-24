<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:variable name="non-empty">
            <xsl:sequence select="''" />
            <xsl:sequence select="''" />
        </xsl:variable>
        
        <foo>
            <xsl:value-of>
                <xsl:copy-of select="$non-empty"/>
                <xsl:on-empty>WRONG! the document node contains a space, it is not empty!!!</xsl:on-empty>
            </xsl:value-of>
        </foo>
    </xsl:template>
    
</xsl:stylesheet>