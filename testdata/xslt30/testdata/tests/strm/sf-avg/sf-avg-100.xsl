<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <!-- Test of schema-aware streamed call on fn:avg() -->
    
    <xsl:mode streamable="yes"/>
    <xsl:template match="/">
      <xsl:variable name="a" select="avg(/*/transaction/@value)"/>
      <out avg="{$a}" is-decimal="{$a instance of xs:decimal}"/>
    </xsl:template>   
    
</xsl:stylesheet>