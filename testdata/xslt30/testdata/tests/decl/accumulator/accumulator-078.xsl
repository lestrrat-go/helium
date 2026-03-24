<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="#all">
    
    <accumulator name="a1" as="xs:integer" initial-value="0">
        <xsl:accumulator-rule match="foo" select="0"/>
        <xsl:accumulator-rule match="foo/*" select="$value + 1"/>
    </accumulator>
    
    <xsl:mode on-no-match="shallow-copy" use-accumulators="a1"/>
    
</xsl:stylesheet>