<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="f"
    exclude-result-prefixes="xs f"
    version="3.0"
    expand-text="yes">

    <xsl:template name="get-function">
        <xsl:sequence select="f:negative#1"/>
    </xsl:template>
    
    <xsl:function name="f:negative" as="xs:boolean">
        <xsl:param name="in" as="xs:integer"/>
        <xsl:sequence select="$in lt 0"/>
    </xsl:function>

</xsl:stylesheet>