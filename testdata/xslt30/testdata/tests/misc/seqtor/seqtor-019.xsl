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
            Nesting example with xsl:sequence and xsl:text and expand-text
        -->
        <xsl:text>before</xsl:text>
        <xsl:sequence expand-text="yes">
            <xsl:for-each select="1 to 100">
                <xsl:sequence expand-text="yes">{
                ''
                }<xsl:text expand-text="yes"></xsl:text><xsl:text expand-text="yes"></xsl:text><xsl:text expand-text="yes"></xsl:text>{
                (1,2,3)[false()]}<xsl:text expand-text="yes"></xsl:text><xsl:text expand-text="yes"></xsl:text><xsl:text expand-text="yes"></xsl:text></xsl:sequence>
            </xsl:for-each>
            <xsl:text expand-text="yes">{'-'}</xsl:text>
        </xsl:sequence>
        <xsl:text>after</xsl:text>
    </xsl:template>
    
</xsl:stylesheet>