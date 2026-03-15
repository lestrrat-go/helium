<?xml version="1.0"?> 

<!-- mode-onnomatch005 -->
<!-- Michael Kay -->
<!-- on-no-match="shallow-copy" --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
 
  <xsl:mode name="s" on-no-match="shallow-copy"/>
  
  <xsl:template match="@length" mode="s">
    <xsl:attribute name="length" select=". + 1"/>
  </xsl:template>
  
  <xsl:template match="chtitle" mode="s">
    <chapter-title>
      <xsl:next-match/>
    </chapter-title>
  </xsl:template>
    
</xsl:stylesheet>