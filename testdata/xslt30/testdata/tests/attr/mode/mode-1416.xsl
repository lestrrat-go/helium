<?xml version="1.0"?> 

<!-- mode-onnomatch006s -->
<!-- Michael Kay -->
<!-- on-no-match="shallow-copy" with streaming --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
 
  <xsl:mode on-no-match="shallow-copy" streamable="yes"/>
  
  <xsl:template match="text()">
    <xsl:value-of select="upper-case(.)"/>
  </xsl:template>
  
  <xsl:template match="v"/>
  
  <xsl:template match="chapter/text()"/>
     
</xsl:stylesheet>