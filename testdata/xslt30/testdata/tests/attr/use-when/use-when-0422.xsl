<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="2.0">

    <!-- 
        Purpose: tests that use-when must be in the xsl namespace on LREs
        In this case, must not throw XPST0003, but simply output the attribute
    -->


    <xsl:template match="/">
        <result>
            <row use-when="not///valid///xpath" />
        </result>
    </xsl:template>
</xsl:transform>
