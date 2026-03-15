<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://localhost/fn"
    name="http://london2016/override-f-027/package-1"
    package-version="1.0"
    exclude-result-prefixes="xs f"
    version="3.0">
    
    <xsl:expose component="function" names="*" visibility="public"/>
 
    <xsl:function name="f:b">
        <xsl:param name="x"/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x + $y"/>
    </xsl:function> 
    
    
</xsl:package>