<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:template match="/">
        <xsl:result-document href="test2016081202Result2.xml">
            <xsl:apply-templates/>
        </xsl:result-document>
    </xsl:template>
    
</xsl:stylesheet>