<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0"
    name="urn:176b">
    
    <xsl:use-package name="urn:176d" package-version="1.0.*"/>
    
    <xsl:template name="b" visibility="public">
        <b><xsl:value-of select="$v"/></b>
    </xsl:template>

    
</xsl:package>
