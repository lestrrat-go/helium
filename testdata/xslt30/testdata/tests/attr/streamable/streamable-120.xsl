<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, use sorting -->
   
  <xsl:mode streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />

  <xsl:strip-space elements="*"/>
    
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="mixed.xml"><xsl:apply-templates select="."/></xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="book">
    <xsl:apply-templates select=".//chtitle">
      <xsl:sort select="."/>
    </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="chtitle">
    <title chapter="{position()}">
      <xsl:value-of select="."/>
    </title>
  </xsl:template>
  
    
</xsl:transform>

