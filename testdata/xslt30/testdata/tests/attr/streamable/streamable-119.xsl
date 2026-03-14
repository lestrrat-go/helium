<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, use last() -->
   
  <xsl:mode name="s" streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />

  <xsl:strip-space elements="*"/>
    
  <xsl:template match="/" name="main" mode="s">
    <out>
      <xsl:apply-templates select="*" mode="s"/>
    </out>
  </xsl:template>
  
  <xsl:template match="book" mode="s">
    <xsl:apply-templates select=".//chtitle"/>
  </xsl:template>
  
  <xsl:template match="chtitle" mode="s">
    <title chapter="{position()}" of="{last()}">
      <xsl:value-of select="."/>
    </title>
  </xsl:template>
  
    
</xsl:transform>

