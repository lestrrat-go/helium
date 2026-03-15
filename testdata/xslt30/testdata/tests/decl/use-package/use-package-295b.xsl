<?xml version="1.0"?>
<xsl:package name="http://use-package-295b/pkg" version="3.0" exclude-result-prefixes="#all"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://localhost/pkg">
    
    <xsl:function name="local:foo" visibility="abstract"/>
    
    <xsl:function name="local:bar" visibility="final">
        <xsl:sequence select="local:foo()"/>
    </xsl:function>
    
</xsl:package>