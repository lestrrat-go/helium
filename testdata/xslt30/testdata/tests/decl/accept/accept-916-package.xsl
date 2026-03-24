<?xml version="1.0"?>
<xsl:package name="http://localhost/pkg" version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://localhost/pkg">
    
    <xsl:function name="local:foo" visibility="public">
        <xsl:sequence select="'foo'"/>
    </xsl:function>
    
</xsl:package>