<xsl:package
    name="http://www.w3.org/xslt30tests/package-905"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:templates">
    
    <!-- tests error XTDE3052 to be raised on abstract template -->

    <xsl:template name="t:abstract" visibility="abstract" as="xs:ID">
        <xsl:param name="one" as="xs:string" />
    </xsl:template>  

    <xsl:template name="xsl:initial-template" visibility="public">
        <not-ok><xsl:call-template name="t:abstract" /></not-ok>
    </xsl:template>
    
</xsl:package>   