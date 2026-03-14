<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- true#0 and false#0 -->
    
 
    
    <xsl:template name="main">
        <out>
            <xsl:value-of select="local:tf(0)(), local:tf(1)()"/>
        </out>
    </xsl:template>
    

    
    <xsl:function name="local:tf" as="function() as xs:boolean">
        <xsl:param name="i" as="xs:integer"/>
        <xsl:sequence select="if ($i) then true#0 else false#0"/>
    </xsl:function>
   
    
    
     
 
</xsl:stylesheet>