<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <!-- Test of schema-aware streamed inequality comparison -->
    
    <xsl:mode streamable="yes"/>
    <xsl:template match="/">
      <out result="{/*/transaction/@value > (xs:untypedAtomic(11.50), 11.51, 11.52)}"/>
    </xsl:template>   
    
</xsl:stylesheet>