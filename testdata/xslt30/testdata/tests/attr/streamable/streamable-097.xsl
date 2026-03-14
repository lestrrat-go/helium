<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes=" xs" version="3.0">
       
  
  <!-- within a streaming template, evaluate a general comparison on streamed nodes -->
    
   
  <xsl:mode name="s" streamable="yes"/>
  <xsl:strip-space elements="*"/>
       
  <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="transactions.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="account" mode="s">
      <NotAllToday><xsl:value-of select="transaction/xs:date(@date) != current-date()"/></NotAllToday>
  </xsl:template>
  
  <xsl:template match="text()" mode="#all"/>
       
</xsl:transform>

