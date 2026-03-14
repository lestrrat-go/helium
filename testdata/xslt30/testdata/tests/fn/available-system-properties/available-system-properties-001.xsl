<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:xpath="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="xs math xpath"
    version="3.0">
    
    <xsl:param name="use-static" static="true" as="xs:boolean" />
    <xsl:variable name="available-sysprops-available" _static="{$use-static}" select="function-available('xpath:available-system-properties', 0)" />
    <xsl:variable name="available-sysprops1" _static="{$use-static}" select="available-system-properties#0" />
    <xsl:variable name="available-sysprops2" _static="{$use-static}" select="xpath:available-system-properties#0" />
    <xsl:variable name="available-sysprops3" _static="{$use-static}" select="Q{http://www.w3.org/2005/xpath-functions}available-system-properties#0" />
    <xsl:variable name="available-sysprops4" _static="{$use-static}" select="function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'available-system-properties'), 0)" />
    
    <xsl:template name="xsl:initial-template">
        <result>
            <xsl:value-of select="$available-sysprops-available" />
            <xsl:for-each select="$available-sysprops1, $available-sysprops2, $available-sysprops3, $available-sysprops4">
                <xsl:value-of select=". instance of function(*)" />
            </xsl:for-each>
        </result>
    </xsl:template>
    
</xsl:stylesheet>