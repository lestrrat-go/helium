<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="http://function-lookup-005a/"
    package-version="1.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
    
    <xsl:template name="sub" visibility="public">
        <sub>
            <multiply exists="{exists(function-lookup(QName('http://local/', 'multiply'), 2))}"/>
            <divide exists="{exists(function-lookup(QName('http://local/', 'divide'), 2))}"/>
            <add exists="{exists(function-lookup(QName('http://local/', 'add'), 2))}" effect="{function-lookup(QName('http://local/', 'add'), 2)(3,4)}"/>
            <subtract exists="{exists(function-lookup(QName('http://local/', 'subtract'), 2))}"/>
        </sub>
    </xsl:template>
 
    
    <xsl:function name="f:multiply" visibility="private">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x * $y"/>
    </xsl:function>
    
    <xsl:function name="f:add" visibility="public">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x + $y"/>
    </xsl:function>
    
    <xsl:function name="f:subtract" visibility="abstract">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
    </xsl:function>

</xsl:package>