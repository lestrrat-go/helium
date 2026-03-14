<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:output method="text" />
    
    <xsl:template match="/" name="xsl:initial-template">
        <!--
            create text nodes that are concatenated together, intertwined with sequences of atomics. 
            Empty text nodes must be discarded. 
            Grouping by threes, balancing the spaces before/after the curlies by the magic of XSLT space-folding
            Same as previous, but using xsl:sequence with expand-text, returning text nodes
        -->
        <xsl:sequence>
            <xsl:for-each select="1 to 100">
                <xsl:sequence expand-text="yes">{ if(. = 1) then '-- START --' else ''}</xsl:sequence>
                <xsl:sequence select="if(. mod 3 = 1 and . != 1) then '}' else ()" />
                <xsl:sequence select="."/>
                <xsl:sequence select="if(. mod 3 = 2) then '{' else  ()" />
                <xsl:sequence expand-text="yes">{ if(. = 100) then '-- END --' else ''}</xsl:sequence>
            </xsl:for-each>
        </xsl:sequence>
    </xsl:template>
    
</xsl:stylesheet>