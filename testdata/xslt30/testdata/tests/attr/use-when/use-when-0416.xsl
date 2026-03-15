<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- 
        Purpose: tests that use-when must not be in the xsl namespace on xsl elements
        Variant of previous, with second use-when and a different default namespace, has no effect on no-namespace of attribute
    -->


    <xsl:template match="/">
        <result>
            <xsl:value-of select="'raises no error'" xsl:use-when="no///error" use-when="function-available('no-such-function')" xmlns="http://www.w3.org/1999/XSL/Transform" />
        </result>
    </xsl:template>
</xsl:transform>