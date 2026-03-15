<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:f="http://function"
    exclude-result-prefixes="xs math f"    
    version="3.0">
    
    <xsl:output method="text" />
    
    <xsl:template match="/" name="xsl:initial-template">
        <!--
            create text nodes that are concatenated together, intertwined with sequences of atomics. 
            Empty text nodes must be discarded. 
            Creating mixed atomics and text nodes through xsl:function, mark the last item, it ends with two atomics after unrolling the loop
        -->
        <xsl:sequence select="'-'"/>
        <xsl:sequence expand-text="yes">
            <xsl:for-each select="1 to 10">
                <xsl:sequence select="f:create()" />
            </xsl:for-each>
        </xsl:sequence>
        <xsl:sequence select="'-'"/>
    </xsl:template>
    
    <xsl:function name="f:create" as="item()+">
        <xsl:text expand-text="yes">{''}</xsl:text>
        <xsl:sequence select="''" />
    </xsl:function>
    
</xsl:stylesheet>