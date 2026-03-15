<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://localhost/pkg"
    expand-text="yes">
    
    <xsl:use-package name="http://localhost/pkg">
        <xsl:accept component="function" names="local:foo#0" visibility="private"/>
        <xsl:override>
            <xsl:function name="local:foo">
                <xsl:sequence select="'foo2'"/>
            </xsl:function>
        </xsl:override>
    </xsl:use-package>
    
    <xsl:template name="xsl:initial-template">
        hello {local:foo()}
    </xsl:template>
    
</xsl:stylesheet>