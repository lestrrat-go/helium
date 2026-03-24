<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://local-functions/"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:output method="xml" encoding="UTF-8" indent="no"  omit-xml-declaration="yes"/>

  <!-- Purpose: d-o-e works when streaming -->
  
  <xsl:mode streamable="true"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="*"/>
  </out>
</xsl:template>
  
  <xsl:template match="doc/*">
    <v><xsl:value-of select="." disable-output-escaping="true"/></v>
  </xsl:template>
  

</xsl:stylesheet>
