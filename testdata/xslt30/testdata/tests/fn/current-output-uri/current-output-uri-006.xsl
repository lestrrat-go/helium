<xsl:stylesheet 
    version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:f="urn:function" exclude-result-prefixes="xs f">
    
    <!-- current-output-uri() while evaluating a stylesheet function -->
    
    <xsl:function name="f:start" visibility="public">
        <xsl:sequence select="current-output-uri()"/>
    </xsl:function>

</xsl:stylesheet>
