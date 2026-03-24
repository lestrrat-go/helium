<?xml version="1.0"?> 

<!-- The unnamed mode cannot be public -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:mode visibility="public" />
  
  <xsl:template match="unstreamable" name="main">
    <xsl:value-of select="a+b+c"/>
  </xsl:template>
  

  
    
</xsl:stylesheet>