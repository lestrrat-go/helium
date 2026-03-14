<?xml version="1.0"?> 

<!-- mode-onnomatch003 -->
<!-- Michael Kay -->
<!-- on-no-match="text-only-copy --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
 
  <xsl:mode on-no-match="text-only-copy"/>
  
  <xsl:template match="text()">
    <xsl:value-of select="upper-case(.)"/>
  </xsl:template>
   
</xsl:stylesheet>