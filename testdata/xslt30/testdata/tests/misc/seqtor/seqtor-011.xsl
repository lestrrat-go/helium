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
            Since "consecutive strings" from the spec does not apply here, we expect no spaces (w/o the text nodes, spaces would occur)
            Same as seqtor-008, but with expand-text and with some obvious programmer's mistakes not returning the empty sequence in the if-statements
            Note the difference with seqtor-010, which had opening/ending atomics, whereas here we have text nodes leading to surprising symmetry in the result
        -->
        <xsl:sequence>
            <xsl:for-each select="1 to 4">
                <xsl:text expand-text="yes">{ if(. = 1) then '-- START --' else ''}</xsl:text>
                <xsl:sequence select="if(. mod 2) then '|' else ''" />
                <xsl:sequence select="."/>
                <xsl:sequence select="if(. mod 2) then '' else '|'" />
                <xsl:text expand-text="yes">{ if(. = 4) then '-- END --' else ''}</xsl:text>
            </xsl:for-each>
        </xsl:sequence>
    </xsl:template>
    
</xsl:stylesheet>