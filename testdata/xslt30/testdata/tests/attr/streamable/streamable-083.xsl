<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
    <!-- within a streaming template, use xsl:choose for conditional processing -->
    <!-- action depends on down-content, condition does not -->
    
     
    <xsl:mode name="s" streamable="yes"/>
    <xsl:strip-space elements="*"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
      
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="loans.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="Property" mode="s">
      <xsl:choose>
        <xsl:when test="lang('en')">
          <English>
            <xsl:copy-of select="Address"/>
          </English>
        </xsl:when>
        <xsl:when test="lang('de')">
          <German>
            <xsl:copy-of select="Address"/>
          </German>
        </xsl:when>
        <xsl:otherwise>
          <Unrecognized/>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:template>
    
    <xsl:template match="text()" mode="#all"/>
       
</xsl:transform>

