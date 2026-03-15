<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <!--
        tests old syntax of maps: map { 1 := 'a' }
        this syntax existed in an earlier draft, tests should throw static error XPST0003, even though the variable is not referenced
    -->
    
    <xsl:param name="test-case" static="yes" required="yes" />
    
    
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="nomap" _select="{$test-case}" />
    </xsl:template>
    
</xsl:stylesheet>