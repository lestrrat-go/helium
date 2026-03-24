<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="http://function-lookup-004a/"
    package-version="1.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
 
    
    <xsl:function name="f:multiply" visibility="private">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x * $y"/>
    </xsl:function>

</xsl:package>