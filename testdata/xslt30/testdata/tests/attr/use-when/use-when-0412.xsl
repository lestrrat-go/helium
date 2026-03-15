<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- Purpose: tests that use-when must not be in the xsl namespace on xsl elements: XTSE0090 -->


    <xsl:template match="/">
        <result>
            <xsl:value-of select="'raises error'" xsl:use-when="false()" />
        </result>
    </xsl:template>
</xsl:transform>