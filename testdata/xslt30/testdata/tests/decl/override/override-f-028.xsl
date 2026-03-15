<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs p"
    xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
    expand-text="yes"
    version="3.0">
    
    <xsl:use-package name="http://www.w3.org/xslt30tests/override-base-001"
        package-version="1.0">
        <xsl:override>
            <xsl:function name="p:f" as="xs:string" visibility="public">
                <xsl:param name="in" as="xs:string"/>
                <xsl:param name="count" as="xs:integer"/>
                <xsl:sequence select="$in || $count"/>
            </xsl:function>
            <xsl:function name="p:f-abstract" as="xs:string" visibility="private" >
                <xsl:param name="in" as="xs:string"/>
                <xsl:param name="count" as="xs:integer"/>
                {""}
            </xsl:function>
        </xsl:override>
    </xsl:use-package>

    <xsl:template name="xsl:initial-template" visibility="public">
        <out>{
            p:f('xxx', 3)
        }</out>
    </xsl:template>
    
</xsl:stylesheet>