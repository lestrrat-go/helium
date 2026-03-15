<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="2.0">

    <!-- 
        Purpose: tests that use-when must not be in the xsl namespace on xsl elements
        xsl:fallback, normally ignored anyway, but use-when errors should be triggered
    -->


    <xsl:template match="/">
        <result>
            <xsl:value-of>
                <xsl:text>Next statement has an error</xsl:text>
                <xsl:fallback use-when="error">Nothing to fallback on, xsl:valuae-of existed since 1.0, but wrong attribs still raise an error</xsl:fallback>
            </xsl:value-of>
        </result>
    </xsl:template>
</xsl:transform>