<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="streamable" static="true" required="true"/>
    <xsl:mode on-no-match="shallow-copy" _streamable="{$streamable}" use-accumulators="text-in-p-count"/>
    
    <xsl:accumulator name="text-in-p-count" as="xs:integer" _streamable="{$streamable}" initial-value="0">
        <xsl:accumulator-rule match="p" select="0"/>
        <xsl:accumulator-rule match="p/text()" select="$value + 1"/>
    </xsl:accumulator>
    
    <xsl:template match="p/text()[accumulator-before('text-in-p-count') eq 1]">
        <xsl:value-of select="replace(., '^\s+', '')"/>
    </xsl:template>
    
</xsl:stylesheet>