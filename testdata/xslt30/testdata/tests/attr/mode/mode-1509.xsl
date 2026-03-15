<?xml version="1.0"?> 

<!-- A named mode cannot be abstract -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:mode name="m" visibility="abstract"/>
  
  <xsl:template match="unstreamable">
    <xsl:value-of select="a+b+c"/>
  </xsl:template>
  

  
    
</xsl:stylesheet>