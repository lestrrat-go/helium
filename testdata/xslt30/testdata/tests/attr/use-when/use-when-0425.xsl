<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="2.0">

    <!-- 
        Purpose: tests that use-when must be in the xsl namespace on LREs
        Variant of previous, with one in the xsl namespace and one not, EBV is true, no error.
    -->


    <xsl:template match="/">
        <result>
            <row xsl:use-when="function-available('concat')" use-when="not///a///valid///xpath" />
        </result>
    </xsl:template>
</xsl:transform>
