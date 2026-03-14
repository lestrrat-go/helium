<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- instance-of tests on user-defined function, varying the argument types -->
    
    <xsl:function name="local:f" as="element(e)?">
        <xsl:param name="x" as="xs:long"/>
        <xsl:param name="y" as="xs:NCName"/>
        <e x="{$x}" y="{$y}"/>
    </xsl:function>
        
    <xsl:template name="main">
    <out>    
        <xsl:value-of select="
            (:F:) local:f#2 instance of function(xs:long, xs:NCName) as element(), 
            (:F:) local:f#2 instance of function(xs:long, xs:NCName) as element()+, 
            (:T:) local:f#2 instance of function(xs:long, xs:NCName) as element()?, 
            (:T:) local:f#2 instance of function(xs:long, xs:NCName) as element()*, 
            (:T:) local:f#2 instance of function(xs:long, xs:NCName) as element(e)*, 
            (:F:) local:f#2 instance of function(xs:long, xs:NCName) as element(e, xs:anyType)*, 
            (:F:) local:f#2 instance of function(xs:long, xs:NCName) as element(*, xs:anyType)?, 
            (:F:) local:f#2 instance of function(xs:long, xs:NCName) as element(*, xs:untyped)?"/>
    </out>
    </xsl:template>
    
</xsl:stylesheet>