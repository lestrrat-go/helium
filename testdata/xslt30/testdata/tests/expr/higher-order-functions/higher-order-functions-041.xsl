<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- return an inline function that references inner and outer local variables  -->
    
    <xsl:template name="main">
        <out>
            <xsl:for-each select="local:splitter('How nice! Thank you, I enjoyed that.')">
                <tokens>
                    <xsl:for-each select=".()">
                        <t><xsl:sequence select="."/></t>
                    </xsl:for-each>
                </tokens>
            </xsl:for-each>
        </out>
    </xsl:template>
    
    <xsl:function name="local:splitter" as="(function() as xs:string*)*">
        <xsl:param name="x" as="xs:string"/>
        <xsl:sequence select="for $sep in ('\s', ',', '!') 
            return function() { for $i in tokenize($x, $sep) return upper-case($i) } "/>
    </xsl:function>
    
    
     
 
</xsl:stylesheet>