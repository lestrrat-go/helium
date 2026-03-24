<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
    
 
    <xsl:template name="xsl:initial-template">
        <out>
            <one xsl:use-when="exists(function-lookup(QName('http://local/', 'multiply'), 2))">
                <two/>
            </one>
        </out>
    </xsl:template>
    
    <xsl:function name="f:multiply" visibility="private">
        <xsl:param name='x'/>
        <xsl:param name="y"/>
        <xsl:sequence select="$x * $y"/>
    </xsl:function>

</xsl:stylesheet>