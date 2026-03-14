<?xml version="1.0"?> 

<!-- Conflicting visibility attributes, overridden by another at higher precedence --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  


  <xsl:mode name="X" visibility="final"/>  
  <xsl:mode name="X" visibility="private"/>
  
  <xsl:template match="/" mode="X">
    <out/>
  </xsl:template>

  
    
</xsl:stylesheet>