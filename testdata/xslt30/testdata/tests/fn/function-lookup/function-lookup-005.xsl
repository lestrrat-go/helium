<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="http://function-lookup-005/"
    package-version="1.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
    
    <xsl:use-package name="http://function-lookup-005a/">
        <xsl:override>
            <xsl:function name="f:add" visibility="public">
                <xsl:param name='x'/>
                <xsl:param name="y"/>
                <xsl:sequence select="$x + $y + 1"/>
            </xsl:function>
            <xsl:function name="f:subtract" visibility="public">
                <xsl:param name='x'/>
                <xsl:param name="y"/>
                <xsl:sequence select="$x + $y + 1"/>
            </xsl:function>
        </xsl:override>
    </xsl:use-package>
    
 
    <xsl:template name="main" visibility="public">
        <out>
            <xsl:call-template name="sub"/>
        </out>
    </xsl:template>
    
    <xsl:function name="f:divide" visibility="public">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x div $y"/>
    </xsl:function>
    

</xsl:package>