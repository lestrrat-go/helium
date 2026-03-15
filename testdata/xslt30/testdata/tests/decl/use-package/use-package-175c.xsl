<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0"
    name="urn:175c">
    
    <xsl:use-package name="urn:175d">
        <xsl:override>
            <xsl:variable name="v" as="xs:string" select="'ccccc'" visibility="private"/>
        </xsl:override>
    </xsl:use-package>
    
    <xsl:template name="c" visibility="public">
        <c><xsl:value-of select="$v"/></c>
    </xsl:template>
    
    
</xsl:package>
