<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:out="urn:out"
    exclude-result-prefixes="xs out"
    version="3.0">

    <!-- 
        Purpose: tests that use-when must be in the xsl namespace on LREs.
        Variant of previous, with second use-when and a different default namespace, has no effect on no-namespace of attribute.
    -->


    <xsl:template match="/">
        <result>
            <out:row select="'raises no error'" xsl:use-when="function-available('concat')" use-when="not///a///valid///xpath" xmlns="http://www.w3.org/1999/XSL/Transform" />
        </result>
    </xsl:template>
</xsl:transform>