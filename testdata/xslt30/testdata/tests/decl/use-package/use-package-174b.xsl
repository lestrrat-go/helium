<xsl:package name="http://www.ctk.cz/B/" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:pA="http://www.ctk.cz/A/" xmlns:pB="http://www.ctk.cz/B/">
    <xsl:use-package name="http://www.ctk.cz/A/"/>
    <xsl:function name="pB:fB" as="xs:string" visibility="public">
        <xsl:sequence select="pA:fA()"/>
    </xsl:function>
</xsl:package>