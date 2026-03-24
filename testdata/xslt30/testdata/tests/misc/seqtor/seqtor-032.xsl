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
            Creating mixed atomics and text nodes through xsl:function, testing non tail-recursive function, tripling the spaces
        -->
        <xsl:sequence select="'start'"/>
        <xsl:sequence select="f:create(62)" />
    </xsl:template>
    
    <xsl:function name="f:create" as="item()+">
        <xsl:param name="i" as="xs:integer" />
        <xsl:sequence select="if($i = 0) then 'end' else (f:create($i - 1), codepoints-to-string($i + 64))" />
        <xsl:text expand-text="yes">{if($i = 62 or $i = 0) then '' else '|'}</xsl:text>
    </xsl:function>
    
    
    
</xsl:stylesheet>