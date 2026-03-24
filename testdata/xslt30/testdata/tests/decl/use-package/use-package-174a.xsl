<xsl:package name="http://www.ctk.cz/A/" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:pA="http://www.ctk.cz/A/">
    
    <xsl:function name="pA:helper" as="xs:string">
        <xsl:sequence select="'Prague'"/>
    </xsl:function>
    
    <xsl:function name="pA:fA" as="xs:string" visibility="public">
        <xsl:sequence select="pA:helper()"/>
    </xsl:function>
</xsl:package>