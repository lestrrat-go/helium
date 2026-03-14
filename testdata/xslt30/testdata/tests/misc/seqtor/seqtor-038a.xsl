<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- 
        purpose: to show that a sequence of two consecutive empty strings contains a space in-between and 
                 that adjacent text nodes are merged with no space in-between, but document nodes are not
                 
        When the document node is used inside a simple constructor, it returns a single space (text node),
        because the document node is removed and its children returned, which are already constructed (i.e., not a sequence)
    -->
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:variable name="non-empty">
            <xsl:sequence select="''" />
            <xsl:sequence select="''" />
        </xsl:variable>
        
        <bar>
            <xsl:namespace name="foo">
                <xsl:copy-of select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:copy-of select="$non-empty"/>
                <xsl:value-of select="'|'" />
                <xsl:copy-of select="$non-empty"/>
            </xsl:namespace>
        </bar>
    </xsl:template>
    
    
</xsl:stylesheet>