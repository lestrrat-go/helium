<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fn="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="#all"
    expand-text="yes">
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template match="example">
        <xsl:copy>{
            function-lookup(xs:QName(function), count(arguments/arg))
            =>
            apply(array { arguments/arg })
            }</xsl:copy>
    </xsl:template>
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:next-match/>
        <xsl:comment>Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')}</xsl:comment>
    </xsl:template>
    
</xsl:stylesheet>