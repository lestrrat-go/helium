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
                <xsl:text>Success</xsl:text>
                <xsl:fallback use-when="function-available('string')">Nothing to fallback on, xsl:value-of existed since 1.0</xsl:fallback>
            </xsl:value-of>
        </result>
    </xsl:template>
</xsl:transform>