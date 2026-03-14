<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
  exclude-result-prefixes="map" 
  version="3.0">
  
  <!-- Saxon bug 3051 -->
  
  <xsl:variable name="mapP" select="map{'c':('something', 42, 3)}" as="map(*)"/>
  
  <!--<xsl:variable name="mapP" select="map{'c':('something', 42,3)}"/>-->
  
  <xsl:template name="main">
    <out>
      <xsl:sequence select="string-length($mapP?c[1]) + $mapP?c[2]"/>
    </out>
  </xsl:template>
</xsl:stylesheet>