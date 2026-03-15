<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:f="http://functions"
    exclude-result-prefixes="xs math f"
    expand-text="true"
    version="3.0">
    
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="props1" select="f:available-system-properties()" as="xs:QName+" />
        <xsl:variable name="props2" select="f:available-system-properties()" as="xs:QName+" />
        <result>
            <xsl:for-each select="$props1">
                <xsl:text>Function returns value for property {.}: {let $p := . return exists($props2[. = $p])}&#xA;</xsl:text>
            </xsl:for-each>
        </result>
    </xsl:template>
    
    <xsl:function name="f:available-system-properties" as="xs:QName+">
        <xsl:sequence select="
            ('version',
            'vendor',
            'vendor-url',
            'product-name',
            'product-version',
            'is-schema-aware',
            'supports-serialization',
            'supports-backwards-compatibility',
            'supports-namespace-axis',
            'supports-streaming',
            'supports-dynamic-evaluation',
            'xpath-version',
            'xsd-version') !
            QName('http://www.w3.org/1999/XSL/Transform', .)" />        
    </xsl:function>
    
</xsl:stylesheet>