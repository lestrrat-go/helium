<xsl:stylesheet 
    version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:f="urn:function" exclude-result-prefixes="xs f">
    
    <!-- current-output-uri() initial function, whole stylesheet is in temp output state -->
    
    <xsl:variable name="base" as="element()">
        <base>
            <xsl:value-of select="current-output-uri()" />
        </base>
    </xsl:variable>
    
    <xsl:function name="f:start" visibility="public" as="xs:string">
        <xsl:value-of>
            <xsl:apply-templates select="$base" />
            <xsl:call-template name="named" />
        </xsl:value-of>
    </xsl:function>
    
    <xsl:template match="$base">
        <xsl:text>|</xsl:text>
        <xsl:value-of select="." />
        <xsl:text>|</xsl:text>
        <xsl:text>|</xsl:text>
        <xsl:sequence select="current-output-uri()" />
        <xsl:text>|</xsl:text>
    </xsl:template>
    
    <xsl:template name="named">
        <xsl:text>|</xsl:text>
        <xsl:sequence select="current-output-uri()" />
        <xsl:text>|</xsl:text>
    </xsl:template>

</xsl:stylesheet>
