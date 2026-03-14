<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <!-- error if a non-map used in the sequence constructor of xsl:map -->
    
    <xsl:template name="main">
        <xsl:variable name="m" as="item()">
           <xsl:map>
              <xsl:map-entry key="1">red</xsl:map-entry>
              <xsl:map-entry key="2">blue</xsl:map-entry>
              <xsl:map-entry key="3">green</xsl:map-entry>
              <xsl:sequence select="17"/>
           </xsl:map>
        </xsl:variable>
        <e a="{$m(1)}"/>   
    </xsl:template>   
    
</xsl:stylesheet>