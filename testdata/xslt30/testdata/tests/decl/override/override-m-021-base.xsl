<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    name="http://example.com/override-m-021-base" 
    package-version="1.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    default-mode="m1" 
    version="3.0">
    
    <xsl:mode name="m1" on-no-match="shallow-copy" visibility="public"/>
    
    <xsl:template match="foo">
        <bar>
            <xsl:apply-templates select="@*"/>
            <xsl:apply-templates/>
        </bar>
    </xsl:template>
    
</xsl:package>