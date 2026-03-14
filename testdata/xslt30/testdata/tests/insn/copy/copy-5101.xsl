<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template match="bar" xpath-default-namespace="http://example.org">
        <xsl:variable name="bar-copy" as="element()">
            <xsl:copy-of select="." copy-namespaces="no"/>
        </xsl:variable>
        <xsl:value-of select="serialize($bar-copy)"/>
    </xsl:template>
    
</xsl:stylesheet>