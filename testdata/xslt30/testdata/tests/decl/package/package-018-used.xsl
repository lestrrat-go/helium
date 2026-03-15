<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="0.1"
    name="urn:use-me"
    xmlns:me="urn:use-me"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">
    
    <!-- private to a package -->
    <xsl:mode on-no-match="fail" />
    
    <xsl:function name="me:function1" visibility="public" />
    <xsl:function name="me:function2" visibility="public" />
    
</xsl:package>