<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="xs"
  version="3.0">
   
   <!-- maps used in static variable -->
   
   <xsl:variable name="index" static="yes" as="map(xs:integer, xs:string)" select="map{1: 'x', 2: 'y', 3: 'z'}"/>
   
   <xsl:template name="xsl:initial-template">
      <out>
         <xsl:sequence select="22" use-when="$index(2) = 'y'"/>
      </out>
   </xsl:template>
</xsl:stylesheet>