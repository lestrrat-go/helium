<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0"
    expand-text="yes">
    
    <xsl:template name="main">
        <out>
            <xsl:sequence select="transform( map { 'stylesheet-location': 'non-existent.xsl', 'initial-match-selection' : 42})?*"/>
        </out>
    </xsl:template>
    
</xsl:stylesheet>