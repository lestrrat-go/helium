<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:variable name="foo" as="text()*">
        <xsl:text expand-text="yes">{ (:hello:) 'empty'[false()]}</xsl:text>
        <xsl:text/>
        <xsl:text expand-text="yes">{ (:hello:) 'empty'[false()]}</xsl:text>
    </xsl:variable>
    
    <xsl:template match="/" name="xsl:initial-template">
        <!-- multiple text nodes are combined into one, no extra spaces -->
        <result><count><xsl:copy-of select="count($foo)" /></count><text><xsl:copy-of select="$foo" /></text></result>
    </xsl:template>
    
</xsl:stylesheet>