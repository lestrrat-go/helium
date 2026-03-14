<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, filter expression applied to computed values -->
   
  <xsl:mode streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />

  <xsl:strip-space elements="*"/>
    
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="outermost(.//bookcoll)"/>
    </out>
  </xsl:template>
  
  <xsl:template match="bookcoll">
    <xsl:apply-templates select="outermost(.//bktlong)"/>
  </xsl:template>
  
  <xsl:template match="bktlong">
    <title><xsl:value-of select="tokenize(., '\s+')[string-length(.) gt 5]"/></title>
  </xsl:template>
  
    
</xsl:transform>

