<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="2.0">

    <!-- 
        Purpose: tests xsl:use-when inside xsl:fallback LRE without error
        Variant of previous, without an error in the xsl:use-when expression inside xsl:fallback
    -->


    <xsl:template match="/">
        <result>
            <xsl:value-of>
                <xsl:text>Success</xsl:text>
                <xsl:fallback>
                    <row xsl:use-when="function-available('string')">
                        Nothing to fallback on, xsl:value-of existed since 1.0
                    </row>
                </xsl:fallback>
            </xsl:value-of>
        </result>
    </xsl:template>
</xsl:transform>