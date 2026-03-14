<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <!-- map-entry containing sequence constructor and xsl:fallback -->
    
    <xsl:template name="main">
        <xsl:variable name="m" as="item()">
           <xsl:map>
              <xsl:for-each select="1 to 10">
                 <xsl:map-entry key=".">
                   <xsl:variable name="key" select="."/>
                   <xsl:fallback/>
                   <xsl:for-each select="1 to 5">
                     <xsl:sequence select="$key * ."/>
                   </xsl:for-each>
                 </xsl:map-entry>  
              </xsl:for-each>
           </xsl:map>
        </xsl:variable>
        <e a="{$m(5)}"/>   
    </xsl:template>   
    
</xsl:stylesheet>