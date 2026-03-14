<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, use of //x prevents streaming -->
   
  <xsl:mode streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />

  <xsl:strip-space elements="*"/>
    
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="mixed.xml"><xsl:apply-templates select="."/></xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="book">
    <!-- Can't use //chtitle in streaming template -->
    <xsl:apply-templates select="//chtitle"/>
  </xsl:template>
  
  <xsl:template match="chtitle">
    <title chapter="{.}"/>
  </xsl:template>
  
    
</xsl:transform>

