<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- comments and PIs are stripped, then adjacent text nodes are merged -->
    
    <xsl:param name="p" select="'Invisible'"/>

    <xsl:template name="main">
        <out xsl:expand-text="yes">The Lord of the {str<!--please don't do this in real life-->ing($p)} Rings</out>
    </xsl:template>
    
</xsl:stylesheet>