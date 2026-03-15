<xsl:package
    name="http://www.w3.org/xslt30tests/package-901"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="urn:functions">
    
    <!-- tests error XTDE3052 to be raised on abstract function -->

    <xsl:function name="f:abstract" visibility="abstract" as="xs:ID">
        <xsl:param name="one" as="xs:string" />
    </xsl:function>  

    <xsl:template name="xsl:initial-template" visibility="public">
        <not-ok><xsl:value-of select="f:abstract('test')" /></not-ok>
    </xsl:template>
    
</xsl:package>   