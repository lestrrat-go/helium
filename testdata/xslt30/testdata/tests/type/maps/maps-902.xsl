<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <!--
        tests the proper disappearance of map:for-each-entry
        empty function map:for-each-tenry used to exist, tests should throw static error XPST0017, even though the variable is not referenced
    -->
    
    <xsl:param name="test-case" static="yes" required="yes"/>
    
    
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="validmap" select='map{0:"Sonntag", 1:"Montag", 2:"Dienstag", 3:"Mittwoch", 4:"Donnerstag", 5:"Freitag", 6:"Samstag", 7:"Unbekannt"}' />
        <xsl:variable name="nomap" _select="{$test-case}" />
    </xsl:template>
    
</xsl:stylesheet>