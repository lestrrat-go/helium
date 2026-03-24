<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <!-- map-entry can be used in its own right -->
    
    <xsl:template name="main">
        <xsl:variable name="m" as="item()">
           <xsl:map-entry key="1" select="2"/>
        </xsl:variable>
        <e a="{$m(1)}" b="{map:get($m, 1)}"/>   
    </xsl:template>   
    
</xsl:stylesheet>