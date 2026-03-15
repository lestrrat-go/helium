<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
       
  
  <!-- within a streaming template, use position() -->
   
  <xsl:mode streamable="yes"/>

  <xsl:strip-space elements="*"/>
    
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="mixed.xml"><xsl:apply-templates select="."/></xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="book">
    <xsl:apply-templates select="chapter/chtitle"/>
  </xsl:template>
  
  <xsl:template match="chtitle">
    <title chapter="{position()}">
      <xsl:value-of select="."/>
    </title>
  </xsl:template>
  
    
</xsl:transform>

